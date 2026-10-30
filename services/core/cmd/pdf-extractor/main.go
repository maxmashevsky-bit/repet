package main

import (
	"log"
	"net/http"
	"os/exec"
	"time"
	"tutor-platform/services/core/internal/pdfextract"
)

func main() {
	for _, tool := range []string{"pdfinfo", "pdftotext", "pdftoppm", "pdfimages"} {
		if _, err := exec.LookPath(tool); err != nil {
			log.Fatal("missing PDF tool: ", tool)
		}
	}
	server := &http.Server{Addr: ":8092", Handler: pdfextract.Handler(pdfextract.Extractor{}), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 45 * time.Second, IdleTimeout: 30 * time.Second}
	log.Fatal(server.ListenAndServe())
}
