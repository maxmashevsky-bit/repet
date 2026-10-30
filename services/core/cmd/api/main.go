package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/httpapi"
	"tutor-platform/services/core/internal/objectstorage"
	"tutor-platform/services/core/internal/pdfextract"
	"tutor-platform/services/core/internal/store"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "apply migrations and exit")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		log.Error("configuration failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database pool failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	migrationsDir := filepath.Join("migrations")
	if err := store.Migrate(ctx, db, migrationsDir); err != nil {
		log.Error("migration failed", "error", err)
		os.Exit(1)
	}
	if *migrateOnly {
		log.Info("migrations applied")
		return
	}

	st := store.New(db)
	if cfg.PDFExtractorURL != "" {
		objects, err := objectstorage.NewFilesystem(cfg.ObjectStoragePath)
		if err != nil {
			log.Error("private storage initialization failed", "error", err)
			os.Exit(1)
		}
		defer objects.Close()
		extractor, err := pdfextract.NewClient(cfg.PDFExtractorURL)
		if err != nil {
			log.Error("PDF extractor configuration failed", "error", err)
			os.Exit(1)
		}
		st = store.NewWithPDF(db, objects, extractor)
		done := make(chan struct{})
		go func() { defer close(done); st.RunPDFQueue(ctx, log) }()
		defer func() { stop(); <-done }()
	}
	provider := auth.DevProvider{Enabled: cfg.DevAuthEnabled}
	handler := httpapi.NewServer(cfg, log, st, provider)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	preview := &http.Server{Addr: cfg.HTMLPreviewAddr, Handler: httpapi.NewHTMLPreviewServer(cfg, st), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	if cfg.HTMLPreviewAddr != "" {
		go func() {
			if err := preview.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Error("HTML preview server failed", "error", err)
				stop()
			}
		}()
	}

	go func() {
		log.Info("http server starting", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	if cfg.HTMLPreviewAddr != "" {
		if err := preview.Shutdown(shutdownCtx); err != nil {
			log.Error("HTML preview shutdown failed", "error", err)
		}
	}
	log.Info("http server stopped")
}
