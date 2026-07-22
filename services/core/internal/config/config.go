package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv               string
	HTTPAddr             string
	DatabaseURL          string
	CORSAllowedOrigins   []string
	DevAuthEnabled       bool
	CookieSecure         bool
	RequestBodyLimitByte int64
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		HTTPAddr:             getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://tutor:tutor@localhost:5432/tutor_platform?sslmode=disable"),
		CORSAllowedOrigins:   splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		DevAuthEnabled:       getEnv("DEV_AUTH_ENABLED", "true") == "true",
		CookieSecure:         getEnv("COOKIE_SECURE", "false") == "true",
		RequestBodyLimitByte: parseInt64(getEnv("REQUEST_BODY_LIMIT_BYTES", "1048576"), 1048576),
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.AppEnv == "" {
		return errors.New("APP_ENV is required")
	}
	if c.AppEnv == "production" && c.DevAuthEnabled {
		return errors.New("development auth cannot be enabled in production")
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
