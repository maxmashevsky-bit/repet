package config

import "testing"

func TestProductionRejectsDevAuth(t *testing.T) {
	cfg := Config{
		AppEnv:               "production",
		DatabaseURL:          "postgres://example",
		CORSAllowedOrigins:   []string{"https://example.test"},
		DevAuthEnabled:       true,
		RequestBodyLimitByte: 1,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production config with dev auth to fail")
	}
}
