package pdfextract

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const MaxPDFBytes = 5 << 20
const MaxPages = 10
const MaxResponseBytes = 20 << 20

var ErrInvalidPDF = errors.New("invalid, encrypted, oversized or unsupported PDF")
var pagesPattern = regexp.MustCompile(`(?m)^Pages:\s+(\d+)\s*$`)

type Page struct {
	Number      int    `json:"number"`
	Text        string `json:"text"`
	PNG         []byte `json:"png"`
	OCRRequired bool   `json:"ocr_required"`
}
type Figure struct {
	Number int    `json:"number"`
	Page   int    `json:"page"`
	PNG    []byte `json:"png"`
}
type Result struct {
	Pages    []Page   `json:"pages"`
	Figures  []Figure `json:"figures"`
	Warnings []string `json:"warnings"`
}
type Runner interface {
	Run(context.Context, string, string, ...string) ([]byte, error)
}
type Commands struct{}

func (Commands) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	switch name {
	case "pdfinfo", "pdftotext", "pdftoppm", "pdfimages":
	default:
		return nil, errors.New("unsupported PDF tool")
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "LC_ALL=C", "HOME=" + dir, "XDG_CACHE_HOME=" + dir, "TMPDIR=" + dir}
	out := limitedBuffer{limit: 1 << 20}
	stderr := limitedBuffer{limit: 8192}
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("PDF tool %s failed: %w", name, err)
	}
	return out.Bytes(), nil
}

type limitedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.Len() {
		return 0, errors.New("PDF output limit")
	}
	return b.Buffer.Write(p)
}

type Extractor struct{ Runner Runner }

func (e Extractor) Extract(ctx context.Context, source []byte) (Result, error) {
	out := Result{Pages: []Page{}, Figures: []Figure{}, Warnings: []string{"Автоматическое извлечение может содержать ошибки. Сверьте условия, числа и рисунки с оригиналом."}}
	if len(source) > MaxPDFBytes || !bytes.HasPrefix(source, []byte("%PDF-")) {
		return out, ErrInvalidPDF
	}
	dir, err := os.MkdirTemp("", "repet-pdf-")
	if err != nil {
		return out, err
	}
	defer os.RemoveAll(dir)
	input := filepath.Join(dir, "source.pdf")
	if err = os.WriteFile(input, source, 0600); err != nil {
		return out, err
	}
	runner := e.Runner
	if runner == nil {
		runner = Commands{}
	}
	info, err := runner.Run(ctx, dir, "pdfinfo", input)
	if err != nil {
		return out, ErrInvalidPDF
	}
	match := pagesPattern.FindSubmatch(info)
	if len(match) != 2 || regexp.MustCompile(`(?mi)^Encrypted:\s+yes`).Match(info) {
		return out, ErrInvalidPDF
	}
	pages, _ := strconv.Atoi(string(match[1]))
	if pages < 1 || pages > MaxPages {
		return out, ErrInvalidPDF
	}
	total := 0
	for number := 1; number <= pages; number++ {
		n := strconv.Itoa(number)
		textPath := filepath.Join(dir, "text-"+n+".txt")
		if _, err = runner.Run(ctx, dir, "pdftotext", "-f", n, "-l", n, "-layout", "-enc", "UTF-8", input, textPath); err != nil {
			return out, ErrInvalidPDF
		}
		text, err := readBounded(textPath, 100000)
		if err != nil {
			return out, err
		}
		prefix := filepath.Join(dir, "page-"+n)
		if _, err = runner.Run(ctx, dir, "pdftoppm", "-f", n, "-l", n, "-singlefile", "-scale-to", "1600", "-png", input, prefix); err != nil {
			return out, ErrInvalidPDF
		}
		image, err := readPNG(prefix + ".png")
		if err != nil {
			return out, err
		}
		total += len(image)
		if total > 12<<20 {
			return out, ErrInvalidPDF
		}
		out.Pages = append(out.Pages, Page{Number: number, Text: string(text), PNG: image, OCRRequired: strings.TrimSpace(string(text)) == ""})
	}
	list, err := runner.Run(ctx, dir, "pdfimages", "-list", input)
	if err != nil {
		return out, ErrInvalidPDF
	}
	imagePages := map[int]int{}
	for _, line := range strings.Split(string(list), "\n") {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		page, pe := strconv.Atoi(f[0])
		index, ie := strconv.Atoi(f[1])
		if pe == nil && ie == nil && page >= 1 && page <= pages {
			imagePages[index] = page
		}
	}
	if len(imagePages) > 20 {
		out.Warnings = append(out.Warnings, "В PDF более 20 изображений. Для рисунков доступны полные изображения страниц.")
		return out, nil
	}
	if len(imagePages) > 0 {
		prefix := filepath.Join(dir, "figure")
		if _, err = runner.Run(ctx, dir, "pdfimages", "-png", input, prefix); err != nil {
			return out, ErrInvalidPDF
		}
		files, err := filepath.Glob(prefix + "-*.png")
		if err != nil {
			return out, err
		}
		sort.Strings(files)
		for _, file := range files {
			index, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(file), "figure-"), ".png"))
			if err != nil {
				continue
			}
			page := imagePages[index]
			if page == 0 {
				continue
			}
			image, err := readPNG(file)
			if err != nil {
				out.Warnings = append(out.Warnings, "Некоторые встроенные рисунки слишком велики; используйте изображение страницы.")
				continue
			}
			if total+len(image) > 12<<20 {
				out.Warnings = append(out.Warnings, "Достигнут лимит рисунков; используйте изображения страниц.")
				break
			}
			total += len(image)
			out.Figures = append(out.Figures, Figure{Number: index, Page: page, PNG: image})
		}
	}
	return out, nil
}
func readBounded(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, ErrInvalidPDF
	}
	return data, nil
}
func readPNG(path string) ([]byte, error) {
	data, err := readBounded(path, 2<<20)
	if err != nil {
		return nil, err
	}
	image, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || image.Width < 1 || image.Height < 1 || image.Width > 3000 || image.Height > 3000 || image.Width*image.Height > 4_000_000 {
		return nil, ErrInvalidPDF
	}
	return data, nil
}
