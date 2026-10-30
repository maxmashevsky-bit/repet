package config

import "testing"

func TestProductionEmailConfig(t *testing.T) {
	cfg := Config{AppEnv: "production", DatabaseURL: "postgres://example", CookieSecure: true, CORSAllowedOrigins: []string{"https://example.test"}, AppPublicURL: "https://example.test", HTMLPreviewPublicURL: "https://lessons.example.net", RequireEmailVerification: true, SMTPAddr: "smtp.example.test:587", SMTPFrom: "no-reply@example.test"}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	cfg.RequireEmailVerification = false
	if cfg.Validate() == nil {
		t.Fatal("production allowed verification bypass")
	}
	cfg.RequireEmailVerification = true
	for _, value := range []string{"http://example.test", "https://evil.test", "https://example.test/path"} {
		cfg.AppPublicURL = value
		if cfg.Validate() == nil {
			t.Fatal("invalid public URL accepted", value)
		}
	}
	cfg.AppPublicURL = "https://example.test"
	cfg.SMTPFrom = "a@example.test\r\nBcc: b@example.test"
	if cfg.Validate() == nil {
		t.Fatal("sender header injection accepted")
	}
}

func TestHTMLPreviewIsolationConfig(t *testing.T) {
	cfg := Config{AppEnv: "development", DatabaseURL: "postgres://example", CORSAllowedOrigins: []string{"http://localhost:3000"}, HTMLPreviewPublicURL: "http://html.localhost:8081"}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"http://localhost:8081", "javascript:alert(1)", "https://user:pass@html.example.test", "https://html.example.test/path", "https://html.example.test?token=x"} {
		cfg.HTMLPreviewPublicURL = value
		if cfg.Validate() == nil {
			t.Fatal("accepted", value)
		}
	}
}

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

func TestProductionRequiresSecureCookie(t *testing.T) {
	cfg := Config{
		AppEnv:               "production",
		DatabaseURL:          "postgres://example",
		CORSAllowedOrigins:   []string{"https://example.test"},
		RequestBodyLimitByte: 1,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production config with insecure cookie to fail")
	}
}
