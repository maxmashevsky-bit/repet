package pdfextract

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

type Client struct {
	endpoint string
	http     *http.Client
}

func NewClient(base string) (*Client, error) {
	u, err := url.Parse(base)
	if err != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" || u.Path != "" && u.Path != "/" {
		return nil, errors.New("invalid PDF extractor endpoint")
	}
	return &Client{endpoint: strings.TrimRight(base, "/") + "/extract", http: &http.Client{Timeout: 45 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) Extract(ctx context.Context, source []byte) (Result, error) {
	var out Result
	if len(source) > MaxPDFBytes || !bytes.HasPrefix(source, []byte("%PDF-")) {
		return out, ErrInvalidPDF
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(source))
	if err != nil {
		return out, err
	}
	r.Header.Set("Content-Type", "application/pdf")
	response, err := c.http.Do(r)
	if err != nil {
		return out, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		if response.StatusCode == 422 || response.StatusCode == 413 {
			return out, ErrInvalidPDF
		}
		return out, errors.New("PDF extraction service did not complete the document")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, MaxResponseBytes+1))
	if err != nil {
		return out, err
	}
	if len(data) > MaxResponseBytes {
		return out, ErrInvalidPDF
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&out); err != nil {
		return out, err
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		return out, ErrInvalidPDF
	}
	if len(out.Pages) < 1 || len(out.Pages) > MaxPages || len(out.Figures) > 20 || len(out.Warnings) > 25 {
		return out, ErrInvalidPDF
	}
	for i := range out.Pages {
		p := &out.Pages[i]
		if p.Number != i+1 || len(p.Text) > 100000 || !utf8.ValidString(p.Text) || strings.ContainsRune(p.Text, 0) {
			return out, ErrInvalidPDF
		}
		p.OCRRequired = strings.TrimSpace(p.Text) == ""
		p.PNG, err = NormalizePNG(p.PNG)
		if err != nil {
			return out, err
		}
	}
	for i := range out.Figures {
		f := &out.Figures[i]
		if f.Page < 1 || f.Page > len(out.Pages) || f.Number < 0 {
			return out, ErrInvalidPDF
		}
		f.PNG, err = NormalizePNG(f.PNG)
		if err != nil {
			return out, err
		}
	}
	for _, warning := range out.Warnings {
		if len(warning) > 2000 || !utf8.ValidString(warning) {
			return out, ErrInvalidPDF
		}
	}
	return out, nil
}

// Re-encoding strips ancillary content before a derived image reaches the application origin.
func NormalizePNG(data []byte) ([]byte, error) {
	if len(data) > 2<<20 {
		return nil, ErrInvalidPDF
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 3000 || config.Height > 3000 || config.Width*config.Height > 4_000_000 {
		return nil, ErrInvalidPDF
	}
	image, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalidPDF
	}
	var out bytes.Buffer
	if err = png.Encode(&out, image); err != nil {
		return nil, err
	}
	if out.Len() > 2<<20 {
		return nil, ErrInvalidPDF
	}
	return out.Bytes(), nil
}
