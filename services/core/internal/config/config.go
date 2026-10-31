package config

import (
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ObjectStoragePath        string
	PDFExtractorURL          string
	AppEnv                   string
	HTTPAddr                 string
	HTMLPreviewAddr          string
	HTMLPreviewPublicURL     string
	DatabaseURL              string
	CORSAllowedOrigins       []string
	DevAuthEnabled           bool
	CookieSecure             bool
	RequestBodyLimitByte     int64
	SMTPAddr                 string
	SMTPFrom                 string
	SMTPUser                 string
	SMTPPassword             string
	AppPublicURL             string
	RequireEmailVerification bool
}

func Load() (Config, error) {
	verification := getEnv("REQUIRE_EMAIL_VERIFICATION", "true")
	if verification != "true" && verification != "false" {
		return Config{}, errors.New("REQUIRE_EMAIL_VERIFICATION must be true or false")
	}
	cfg := Config{
		ObjectStoragePath:        getEnv("OBJECT_STORAGE_PATH", ".data/objects"),
		PDFExtractorURL:          getEnv("PDF_EXTRACTOR_URL", ""),
		AppEnv:                   getEnv("APP_ENV", "development"),
		HTTPAddr:                 getEnv("HTTP_ADDR", ":8080"),
		HTMLPreviewAddr:          getEnv("HTML_PREVIEW_ADDR", ":8081"),
		HTMLPreviewPublicURL:     getEnv("HTML_PREVIEW_PUBLIC_URL", "http://html.localhost:8081"),
		DatabaseURL:              getEnv("DATABASE_URL", "postgres://tutor:tutor@localhost:5432/tutor_platform?sslmode=disable"),
		CORSAllowedOrigins:       splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000")),
		DevAuthEnabled:           getEnv("DEV_AUTH_ENABLED", "false") == "true",
		CookieSecure:             getEnv("COOKIE_SECURE", "false") == "true",
		RequestBodyLimitByte:     parseInt64(getEnv("REQUEST_BODY_LIMIT_BYTES", "1048576"), 1048576),
		SMTPAddr:                 getEnv("SMTP_ADDR", "127.0.0.1:1025"),
		SMTPFrom:                 getEnv("SMTP_FROM", "no-reply@repet.local"),
		SMTPUser:                 getEnv("SMTP_USER", ""),
		SMTPPassword:             getEnv("SMTP_PASSWORD", ""),
		AppPublicURL:             getEnv("APP_PUBLIC_URL", "http://127.0.0.1:3000"),
		RequireEmailVerification: verification == "true",
	}
	for key, target := range map[string]*string{"DATABASE_URL": &cfg.DatabaseURL, "SMTP_PASSWORD": &cfg.SMTPPassword} {
		if path := getEnv(key+"_FILE", ""); path != "" {
			if os.Getenv(key) != "" {
				return Config{}, fmt.Errorf("set only %s or %s_FILE", key, key)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return Config{}, fmt.Errorf("read %s_FILE: %w", key, err)
			}
			*target = strings.TrimSpace(string(data))
			if *target == "" {
				return Config{}, fmt.Errorf("%s_FILE is empty", key)
			}
		}
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.HTMLPreviewPublicURL != "" {
		u, err := url.Parse(c.HTMLPreviewPublicURL)
		if err != nil || u.Hostname() == "" || u.User != nil || u.Path != "" && u.Path != "/" || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") || c.AppEnv == "production" && u.Scheme != "https" {
			return errors.New("invalid isolated HTML preview URL")
		}
		for _, origin := range c.CORSAllowedOrigins {
			app, err := url.Parse(origin)
			if err != nil || strings.EqualFold(app.Hostname(), u.Hostname()) {
				return errors.New("HTML preview requires a distinct hostname from every application origin")
			}
		}
	}
	if c.AppEnv == "" {
		return errors.New("APP_ENV is required")
	}
	if c.AppEnv == "production" && c.DevAuthEnabled {
		return errors.New("development auth cannot be enabled in production")
	}
	if c.AppEnv == "production" && !c.CookieSecure {
		return errors.New("COOKIE_SECURE must be enabled in production")
	}
	if c.AppEnv == "production" && !c.RequireEmailVerification {
		return errors.New("email verification is required in production")
	}
	if c.RequireEmailVerification {
		u, err := url.Parse(c.AppPublicURL)
		if err != nil || u.Hostname() == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && (u.Scheme != "http" || c.AppEnv == "production")) {
			return errors.New("invalid APP_PUBLIC_URL")
		}
		found := false
		for _, origin := range c.CORSAllowedOrigins {
			if strings.TrimRight(origin, "/") == strings.TrimRight(c.AppPublicURL, "/") {
				found = true
			}
		}
		if !found {
			return errors.New("APP_PUBLIC_URL must be an allowed application origin")
		}
		if _, _, err := net.SplitHostPort(c.SMTPAddr); err != nil {
			return errors.New("SMTP_ADDR must be host:port (STARTTLS port, normally 587)")
		}
		if address, err := mail.ParseAddress(c.SMTPFrom); err != nil || address.Address != c.SMTPFrom || strings.ContainsAny(c.SMTPFrom, "\r\n") {
			return errors.New("invalid SMTP_FROM")
		}
	}
	if _, err := url.Parse(c.DatabaseURL); err != nil {
		return fmt.Errorf("invalid DATABASE_URL: %w", err)
	}
	if len(c.CORSAllowedOrigins) == 0 {
		return errors.New("at least one CORS origin is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	var out []string
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func parseInt64(value string, fallback int64) int64 {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
