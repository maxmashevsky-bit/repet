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
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

func TestTutorPlannerPostgres(t *testing.T) {
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
	st := store.New(pool)
	server := httptest.NewServer(NewServer(config.Config{RequestBodyLimitByte: 1 << 20}, slog.New(slog.NewTextHandler(io.Discard, nil)), st, auth.DevProvider{}))
	defer server.Close()
	id := func() string {
		v, e := platform.NewUUID()
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	register := func(role domain.Role) (domain.User, *http.Cookie) {
		u, token, e := st.RegisterUser(t.Context(), id()+"@example.test", "Planner test", "password123", role, "test", "127.0.0.1")
		if e != nil {
			t.Fatal(e)
		}
		return u, &http.Cookie{Name: store.SessionCookieName, Value: token}
	}
	tutor, tc := register(domain.RoleTutor)
	_, sc := register(domain.RoleStudent)
	_, other := register(domain.RoleTutor)
	request := func(method string, body any, cookie *http.Cookie) (int, []byte, error) {
		data, e := json.Marshal(body)
		if e != nil {
			return 0, nil, e
		}
		r, e := http.NewRequestWithContext(t.Context(), method, server.URL+"/api/v1/tutor-planner", bytes.NewReader(data))
		if e != nil {
			return 0, nil, e
		}
		r.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		res, e := server.Client().Do(r)
		if e != nil {
			return 0, nil, e
		}
		defer res.Body.Close()
		payload, e := io.ReadAll(res.Body)
		return res.StatusCode, payload, e
	}
	call := func(method string, body any, cookie *http.Cookie, want int) domain.TutorPlan {
		t.Helper()
		status, data, e := request(method, body, cookie)
		if e != nil {
			t.Fatal(e)
		}
		if status != want {
			t.Fatalf("%s: %d want %d: %s", method, status, want, data)
		}
		var p domain.TutorPlan
		if want == 200 {
			if e := json.Unmarshal(data, &p); e != nil {
				t.Fatal(e)
			}
		}
		return p
	}
	p := call("GET", nil, tc, 200)
	if p.Revision != 0 || p.Students == nil || p.Lessons == nil || p.Windows == nil {
		t.Fatal("empty plan shape")
	}
	call("GET", nil, nil, 401)
	call("GET", nil, sc, 403)
	call("PUT", p, sc, 403)
	p.Students = []domain.PlanStudent{{ID: id(), Name: "Private learner", Subject: "7 / math", PriceKopecks: 120000, Duration: 60, Contact: "@private_learner"}}
	p.Lessons = []domain.PlanLesson{{ID: id(), StudentID: p.Students[0].ID, Weekday: 1, StartMinute: 540, Duration: 60}}
	p.Windows = []domain.PlanWindow{{Weekday: 1, StartMinute: 540}, {Weekday: 1, StartMinute: 570}}
	saved := call("PUT", p, tc, 200)
	if saved.Revision != 1 {
		t.Fatal("revision")
	}
	read := call("GET", nil, tc, 200)
	if read.Students[0].Contact != p.Students[0].Contact || len(read.Lessons) != 1 || len(read.Windows) != 2 {
		t.Fatal("persistence")
	}
	peer := call("GET", nil, other, 200)
	if len(peer.Students) != 0 || peer.Revision != 0 {
		t.Fatal("other tutor saw plan")
	}
	call("PUT", saved, other, 409)
	call("PUT", p, tc, 409)
	call("PUT", map[string]any{"revision": 1, "owner_id": tutor.ID}, tc, 400)
	call("PUT", map[string]any{"revision": 1, "calculator": map[string]any{"role": "admin"}}, tc, 400)
	bad := saved
	bad.Lessons = []domain.PlanLesson{{ID: id(), StudentID: id(), Weekday: 1, StartMinute: 540, Duration: 60}}
	call("PUT", bad, tc, 400)
	bad = saved
	bad.Students = []domain.PlanStudent{}
	call("PUT", bad, tc, 400)
	// Both requests carry the same revision: exactly one may commit.
	statuses := make(chan int, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); status, _, e := request("PUT", saved, tc); statuses <- status; errs <- e }()
	}
	wg.Wait()
	close(statuses)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatal("lost-update guard", counts)
	}
	p = call("GET", nil, tc, 200)
	if p.Revision != 2 {
		t.Fatal("race revision")
	}
	foreign := tutor
	foreign.OrganizationID = id()
	if got, e := st.GetTutorPlan(t.Context(), foreign); e != nil || got.Revision != 0 {
		t.Fatal("org isolation", e)
	}
	if _, e := st.SaveTutorPlan(t.Context(), foreign, p); e != apperr.ErrConflict {
		t.Fatal("foreign org update", e)
	}
	p.Students = []domain.PlanStudent{}
	p.Lessons = []domain.PlanLesson{}
	p = call("PUT", p, tc, 200)
	if len(p.Students) != 0 || len(p.Lessons) != 0 {
		t.Fatal("delete")
	}
}
