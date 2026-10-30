package pdfextract

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

type fakeRunner struct{ info string }

func (f fakeRunner) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch name {
	case "pdfinfo":
		if f.info != "" {
			return []byte(f.info), nil
		}
		return []byte("Pages: 2\nEncrypted: no\n"), nil
	case "pdftotext":
		text := ""
		if args[1] == "1" {
			text = "1. Найдите 5 + 2.\n2. Запишите число 11.\n\f"
		}
		return nil, os.WriteFile(args[len(args)-1], []byte(text), 0600)
	case "pdftoppm":
		return nil, writeTestPNG(args[len(args)-1] + ".png")
	case "pdfimages":
		if args[0] == "-list" {
			return []byte("page num type width height\n1 0 image 2 2\n"), nil
		}
		return nil, writeTestPNG(args[len(args)-1] + "-000.png")
	}
	panic(name)
}
func writeTestPNG(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{G: 100, A: 255})
	return png.Encode(f, img)
}

func TestExtractionPreservesTextAndPages(t *testing.T) {
	result, err := (Extractor{Runner: fakeRunner{}}).Extract(t.Context(), []byte("%PDF-1.7 fixture"))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Pages) != 2 || result.Pages[0].Number != 1 || result.Pages[0].Text != "1. Найдите 5 + 2.\n2. Запишите число 11.\n\f" || result.Pages[0].OCRRequired || !result.Pages[1].OCRRequired {
		t.Fatal(result.Pages)
	}
	if len(result.Figures) != 1 || result.Figures[0].Page != 1 || len(result.Figures[0].PNG) == 0 {
		t.Fatal(result.Figures)
	}
	for _, info := range []string{"Pages: 0\n", "Pages: " + strconv.Itoa(MaxPages+1) + "\n", "Pages: 1\nEncrypted: yes\n", "not a PDF"} {
		if _, err = (Extractor{Runner: fakeRunner{info: info}}).Extract(t.Context(), []byte("%PDF-1.7")); err == nil {
			t.Fatal("accepted", info)
		}
	}
	if _, err = (Extractor{Runner: fakeRunner{}}).Extract(t.Context(), []byte("not PDF")); err == nil {
		t.Fatal("accepted non-PDF")
	}
	if _, err = (Extractor{Runner: fakeRunner{}}).Extract(t.Context(), bytes.Repeat([]byte("x"), MaxPDFBytes+1)); err == nil {
		t.Fatal("accepted oversized PDF")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err = (Extractor{Runner: fakeRunner{}}).Extract(ctx, []byte("%PDF-1.7")); err == nil {
		t.Fatal("ignored cancellation")
	}
}
func TestWorkerHTTP(t *testing.T) {
	h := Handler(Extractor{Runner: fakeRunner{}})
	for _, tc := range []struct {
		method, path, kind, body string
		status                   int
	}{{"GET", "/healthz", "", "", 200}, {"GET", "/extract", "", "", 404}, {"POST", "/extract", "text/html", "<html>", 415}, {"POST", "/extract", "application/pdf", "bad", 422}, {"POST", "/extract", "application/pdf", "%PDF-1.7 fixture", 200}} {
		t.Run(strconv.Itoa(tc.status)+tc.method, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.kind)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatal(w.Code, w.Body.String())
			}
		})
	}
}

func TestClientValidatesWorkerOutput(t *testing.T) {
	server := httptest.NewServer(Handler(Extractor{Runner: fakeRunner{}}))
	defer server.Close()
	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Extract(t.Context(), []byte("%PDF-1.7 fixture"))
	if err != nil || len(result.Pages) != 2 {
		t.Fatal(result, err)
	}
	for _, base := range []string{"file:///tmp/pdf", "http://user:pass@host", "http://host/extract", "http://host?key=x"} {
		if _, err := NewClient(base); err == nil {
			t.Fatal("unsafe worker URL", base)
		}
	}
	if _, err := NormalizePNG([]byte("<script>bad</script>")); err == nil {
		t.Fatal("accepted non-PNG")
	}
}
