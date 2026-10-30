package pdfextract

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"time"
)

func Handler(extractor Extractor) http.Handler {
	busy := make(chan struct{}, 1)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Method == http.MethodGet && r.URL.Path == "/healthz" {
			w.WriteHeader(200)
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/extract" {
			http.NotFound(w, r)
			return
		}
		kind, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || kind != "application/pdf" {
			http.Error(w, "PDF content type required", 415)
			return
		}
		select {
		case busy <- struct{}{}:
			defer func() { <-busy }()
		default:
			http.Error(w, "extractor busy", 429)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
		defer cancel()
		source, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxPDFBytes))
		if err != nil {
			http.Error(w, "PDF size limit exceeded", 413)
			return
		}
		result, err := extractor.Extract(ctx, source)
		if err != nil {
			http.Error(w, "PDF extraction failed; check format, encryption and page limits", 422)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	})
}
