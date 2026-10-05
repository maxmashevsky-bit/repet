package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

func TestHTTPAuthorizationPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := store.Migrate(t.Context(), pool, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(NewServer(config.Config{CORSAllowedOrigins: []string{"http://example.test"}, RequestBodyLimitByte: 1 << 20}, slog.New(slog.NewTextHandler(io.Discard, nil)), store.New(pool), auth.DevProvider{Enabled: false}))
	defer server.Close()
	client := server.Client()
	unique, err := platform.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	post := func(path string, body any, cookie *http.Cookie) (int, []byte, *http.Cookie) {
		t.Helper()
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req, err := http.NewRequest(http.MethodPost, server.URL+path, bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			req.AddCookie(cookie)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		payload, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		var session *http.Cookie
		for _, item := range res.Cookies() {
			if item.Name == store.SessionCookieName {
				session = item
			}
		}
		return res.StatusCode, payload, session
	}
	get := func(path string, cookie *http.Cookie) int {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		return res.StatusCode
	}
	register := func(prefix string, role domain.Role) (domain.User, *http.Cookie) {
		t.Helper()
		status, payload, cookie := post("/api/v1/auth/register", map[string]any{"display_name": prefix, "email": prefix + unique + "@example.test", "password": "password123", "role": role, "accept_terms": true}, nil)
		if status != http.StatusCreated || cookie == nil || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
			t.Fatalf("register %s: %d %s", prefix, status, payload)
		}
		var envelope struct {
			User domain.User `json:"user"`
		}
		if err := json.Unmarshal(payload, &envelope); err != nil {
			t.Fatal(err)
		}
		return envelope.User, cookie
	}
	tutor, tutorCookie := register("tutor", domain.RoleTutor)
	student, studentCookie := register("student", domain.RoleStudent)
	_, otherCookie := register("other", domain.RoleStudent)
	if status := get("/api/v1/dashboard", nil); status != http.StatusUnauthorized {
		t.Fatalf("anonymous dashboard status %d", status)
	}
	if status, _, _ := post("/api/v1/invitations", map[string]any{"student_email": student.Email}, studentCookie); status != http.StatusForbidden {
		t.Fatalf("student invitation status %d", status)
	}
	status, payload, _ := post("/api/v1/invitations", map[string]any{"student_email": student.Email}, tutorCookie)
	if status != http.StatusCreated {
		t.Fatalf("tutor invitation: %d %s", status, payload)
	}
	var invitation struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(payload, &invitation); err != nil {
		t.Fatal(err)
	}
	if status, _, _ := post("/api/v1/invitations/"+invitation.Token+"/accept", map[string]any{}, otherCookie); status != http.StatusForbidden {
		t.Fatalf("wrong student accepted invitation: %d", status)
	}
	status, payload, _ = post("/api/v1/invitations/"+invitation.Token+"/accept", map[string]any{}, studentCookie)
	if status != http.StatusCreated {
		t.Fatalf("accept invitation: %d %s", status, payload)
	}
	var relation domain.Relation
	if err := json.Unmarshal(payload, &relation); err != nil {
		t.Fatal(err)
	}
	lessonInput := map[string]any{"relation_id": relation.ID, "title": "Lesson", "starts_at": "2030-01-01T10:00:00Z", "ends_at": "2030-01-01T11:00:00Z"}
	if status, _, _ := post("/api/v1/lessons", lessonInput, studentCookie); status != http.StatusForbidden {
		t.Fatalf("student created lesson: %d", status)
	}
	status, payload, _ = post("/api/v1/lessons", lessonInput, tutorCookie)
	if status != http.StatusCreated {
		t.Fatalf("tutor lesson: %d %s", status, payload)
	}
	var lesson domain.Lesson
	if err := json.Unmarshal(payload, &lesson); err != nil {
		t.Fatal(err)
	}
	if status := get("/api/v1/lessons/"+lesson.ID, studentCookie); status != http.StatusOK {
		t.Fatalf("student lesson status %d", status)
	}
	if status := get("/api/v1/lessons/"+lesson.ID, otherCookie); status != http.StatusNotFound {
		t.Fatalf("foreign lesson status %d", status)
	}
	if status := get("/api/v1/lessons/not-a-uuid", tutorCookie); status != http.StatusBadRequest {
		t.Fatalf("invalid lesson ID status %d", status)
	}
	if status, _, _ := post("/api/v1/profile", map[string]any{"display_name": "Changed", "timezone": "Europe/Moscow", "locale": "ru", "role": "admin"}, tutorCookie); status != http.StatusBadRequest {
		t.Fatalf("client-owned role accepted: %d", status)
	}
	if status, _, _ := post("/api/v1/auth/logout", map[string]any{}, tutorCookie); status != http.StatusOK {
		t.Fatalf("logout status %d", status)
	}
	if status := get("/api/v1/dashboard", tutorCookie); status != http.StatusUnauthorized {
		t.Fatalf("revoked session status %d (user %s)", status, tutor.ID)
	}
}
