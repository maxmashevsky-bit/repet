package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

func testSMTP(t *testing.T) (string, <-chan string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	messages := make(chan string, 16)
	done := make(chan struct{})
	t.Cleanup(func() { listener.Close(); <-done })
	go func() {
		defer close(done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			func() {
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(5 * time.Second))
				c := textproto.NewConn(conn)
				c.PrintfLine("220 local test SMTP")
				for {
					line, err := c.ReadLine()
					if err != nil {
						return
					}
					switch {
					case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"), strings.HasPrefix(line, "MAIL"), strings.HasPrefix(line, "RCPT"):
						c.PrintfLine("250 OK")
					case line == "DATA":
						c.PrintfLine("354 send message")
						data, err := c.ReadDotBytes()
						if err != nil {
							return
						}
						messages <- string(data)
						c.PrintfLine("250 accepted")
					case line == "QUIT":
						c.PrintfLine("221 bye")
						return
					default:
						c.PrintfLine("500 unsupported")
					}
				}
			}()
		}
	}()
	return listener.Addr().String(), messages
}

func TestMailRejectsPlaintextProduction(t *testing.T) {
	addr, _ := testSMTP(t)
	err := sendVerificationEmail(context.Background(), config.Config{AppEnv: "production", SMTPAddr: addr, SMTPFrom: "no-reply@example.test", AppPublicURL: "https://example.test"}, "student@example.test", "secret")
	if err == nil || !strings.Contains(err.Error(), "STARTTLS") {
		t.Fatalf("plaintext production SMTP accepted: %v", err)
	}
}

func TestEmailVerificationPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err = store.Migrate(t.Context(), pool, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	addr, messages := testSMTP(t)
	cfg := config.Config{CORSAllowedOrigins: []string{"http://example.test"}, RequestBodyLimitByte: 1 << 20, RequireEmailVerification: true, AppPublicURL: "http://example.test", SMTPAddr: addr, SMTPFrom: "no-reply@example.test"}
	server := httptest.NewServer(NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), store.New(pool), auth.DevProvider{}))
	defer server.Close()
	call := func(method, path string, body any, cookie *http.Cookie, want int) ([]byte, *http.Cookie) {
		t.Helper()
		data, _ := json.Marshal(body)
		req, _ := http.NewRequest(method, server.URL+path, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			req.AddCookie(cookie)
		}
		res, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		payload, _ := io.ReadAll(res.Body)
		if res.StatusCode != want {
			t.Fatalf("%s %s: %d want %d: %s", method, path, res.StatusCode, want, payload)
		}
		var session *http.Cookie
		for _, c := range res.Cookies() {
			if c.Name == store.SessionCookieName {
				session = c
			}
		}
		return payload, session
	}
	unique, _ := platform.NewUUID()
	var firstToken string
	for _, role := range []string{"student", "tutor"} {
		payload, cookie := call("POST", "/api/v1/auth/register", map[string]any{"display_name": "Проверка почты", "email": role + unique + "@example.test", "password": "Password123!", "role": role, "accept_terms": true}, nil, 201)
		if !bytes.Contains(payload, []byte(`"verification_email_sent":true`)) || !bytes.Contains(payload, []byte(`"email_verified":false`)) {
			t.Fatalf("wrong registration response %s", payload)
		}
		var message string
		select {
		case message = <-messages:
		case <-time.After(time.Second):
			t.Fatal("SMTP did not receive confirmation")
		}
		match := regexp.MustCompile(`http://example.test/verify-email#token=([A-Za-z0-9_-]+)`).FindStringSubmatch(message)
		if len(match) != 2 {
			t.Fatal("missing verification URL", message)
		}
		token := match[1]
		if bytes.Contains(payload, []byte(token)) {
			t.Fatal("token leaked in API")
		}
		var count int
		if err = pool.QueryRow(t.Context(), `select count(*) from email_verification_tokens where token_hash=$1`, platform.TokenHash(token)).Scan(&count); err != nil || count != 1 {
			t.Fatal("token was not stored hashed", err)
		}
		call("GET", "/api/v1/me", nil, cookie, 200)
		call("GET", "/api/v1/dashboard", nil, cookie, 403)
		call("GET", "/api/v1/lessons", nil, cookie, 403)
		call("POST", "/api/v1/auth/verification-email", map[string]any{}, cookie, 409)
		call("POST", "/api/v1/auth/verify-email", map[string]any{"token": token, "email_verified": true}, cookie, 400)
		call("POST", "/api/v1/auth/verify-email", map[string]any{"token": token}, nil, 401)
		if firstToken != "" {
			call("POST", "/api/v1/auth/verify-email", map[string]any{"token": firstToken}, cookie, 400)
		} else {
			firstToken = token
		}
		call("POST", "/api/v1/auth/verify-email", map[string]any{"token": "bad"}, cookie, 400)
		if _, err = pool.Exec(t.Context(), `update email_verification_tokens set expires_at=clock_timestamp()-interval '1 second' where token_hash=$1`, platform.TokenHash(token)); err != nil {
			t.Fatal(err)
		}
		call("POST", "/api/v1/auth/verify-email", map[string]any{"token": token}, cookie, 400)
		if _, err = pool.Exec(t.Context(), `update email_verification_tokens set expires_at=clock_timestamp()+interval '1 hour' where token_hash=$1`, platform.TokenHash(token)); err != nil {
			t.Fatal(err)
		}
		call("POST", "/api/v1/auth/verify-email", map[string]any{"token": token}, cookie, 200)
		call("POST", "/api/v1/auth/verify-email", map[string]any{"token": token}, cookie, 200)
		call("GET", "/api/v1/dashboard", nil, cookie, 200)
		payload, _ = call("POST", "/api/v1/auth/login", map[string]any{"email": role + unique + "@example.test", "password": "Password123!"}, nil, 200)
		if !bytes.Contains(payload, []byte(`"email_verified":true`)) {
			t.Fatal("verification did not persist")
		}
	}
	cfg.SMTPAddr = "127.0.0.1:1"
	failed := httptest.NewServer(NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), store.New(pool), auth.DevProvider{}))
	defer failed.Close()
	req, _ := http.NewRequest("POST", failed.URL+"/api/v1/auth/register", strings.NewReader(`{"display_name":"Mail failure","email":"fail`+unique+`@example.test","password":"Password123!","role":"student","accept_terms":true}`))
	req.Header.Set("Content-Type", "application/json")
	res, err := failed.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if res.StatusCode != 201 || !bytes.Contains(data, []byte(`"verification_email_sent":false`)) {
		t.Fatalf("SMTP failure falsely reported success: %s", data)
	}
}
