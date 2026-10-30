package chatfiles

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxBytes = 4 << 20
const MaxPixels = 4 << 20

var ErrContent = errors.New("unsupported or invalid chat file")

type Content struct {
	Bytes          []byte
	Filename, Type string
}

// Positive allowlist and reconstruction, not a general-purpose antivirus scanner.
// Original image bytes, metadata and trailing payloads never reach object storage.
func Normalize(filename string, source []byte) (Content, error) {
	var out Content
	if len(source) == 0 || len(source) > MaxBytes || !utf8.ValidString(filename) || strings.ContainsAny(filename, "/\\") {
		return out, ErrContent
	}
	filename = strings.TrimSpace(filename)
	for _, r := range filename {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return out, ErrContent
		}
	}
	if utf8.RuneCountInString(filename) > 120 || filename == "" {
		return out, ErrContent
	}
	ext := strings.ToLower(path.Ext(filename))
	if ext == ".txt" {
		if !utf8.Valid(source) {
			return out, ErrContent
		}
		for _, r := range string(source) {
			if (unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t') || unicode.Is(unicode.Cf, r) {
				return out, ErrContent
			}
		}
		out = Content{Bytes: []byte(strings.ReplaceAll(string(source), "\r\n", "\n")), Filename: filename, Type: "text/plain; charset=utf-8"}
		return out, nil
	}
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		return out, ErrContent
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(source))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 4096 || cfg.Height > 4096 || int64(cfg.Width)*int64(cfg.Height) > MaxPixels {
		return out, ErrContent
	}
	if (ext == ".png" && format != "png") || ((ext == ".jpg" || ext == ".jpeg") && format != "jpeg") {
		return out, ErrContent
	}
	im, _, err := image.Decode(bytes.NewReader(source))
	if err != nil {
		return out, ErrContent
	}
	var buf boundedBuffer
	if format == "png" {
		err = png.Encode(&buf, im)
		out.Type = "image/png"
	} else {
		err = jpeg.Encode(&buf, im, &jpeg.Options{Quality: 85})
		out.Type = "image/jpeg"
	}
	if err != nil {
		return Content{}, ErrContent
	}
	out.Bytes = buf.Bytes()
	out.Filename = filename
	return out, nil
}

type boundedBuffer struct{ bytes.Buffer }

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > MaxBytes {
		return 0, ErrContent
	}
	return b.Buffer.Write(p)
}
