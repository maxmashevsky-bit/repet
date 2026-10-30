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
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

func TestPlannerCalendarPostgres(t *testing.T) {
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
	st := store.New(pool)
	id := func() string {
		v, e := platform.NewUUID()
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	register := func(role domain.Role) (domain.User, string) {
		u, token, e := st.RegisterUser(t.Context(), id()+"@example.test", "Planner integration", "Password123!", role, "test", "127.0.0.1")
		if e != nil {
			t.Fatal(e)
		}
		return u, token
	}
	tutor, ts := register(domain.RoleTutor)
	student, ss := register(domain.RoleStudent)
	other, osession := register(domain.RoleTutor)
	connect := func(teacher domain.User) domain.Relation {
		_, token, e := st.CreateInvitation(t.Context(), teacher, student.Email, "")
		if e != nil {
			t.Fatal(e)
		}
		r, e := st.AcceptInvitation(t.Context(), student, token)
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	relation, foreign := connect(tutor), connect(other)
	plan := domain.EmptyTutorPlan()
	plan.Students = []domain.PlanStudent{{ID: id(), RelationID: foreign.ID, Name: "Linked", Subject: "Fractions", Duration: 60, PriceKopecks: 120000}}
	if _, err = st.SaveTutorPlan(t.Context(), tutor, plan); err == nil {
		t.Fatal("foreign relation allowed")
	}
	plan.Students[0].RelationID = relation.ID
	plan.Students = append(plan.Students, domain.PlanStudent{ID: id(), Name: "Offline", Duration: 60})
	day := time.Now().In(time.FixedZone("MSK", 3*60*60)).AddDate(0, 0, 2)
	weekday := int(day.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	plan.Lessons = []domain.PlanLesson{{ID: id(), StudentID: plan.Students[0].ID, Weekday: weekday, StartMinute: 600, Duration: 60}, {ID: id(), StudentID: plan.Students[1].ID, Weekday: weekday, StartMinute: 720, Duration: 60}, {ID: id(), StudentID: plan.Students[0].ID, Weekday: weekday, StartMinute: 840, Duration: 60, Paused: true}}
	plan, err = st.SaveTutorPlan(t.Context(), tutor, plan)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(NewServer(config.Config{RequestBodyLimitByte: 1 << 20}, slog.New(slog.NewTextHandler(io.Discard, nil)), st, auth.DevProvider{}))
	defer srv.Close()
	call := func(session string, body any, want int) domain.PlanCalendarResult {
		t.Helper()
		raw, e := json.Marshal(body)
		if e != nil {
			t.Fatal(e)
		}
		req, e := http.NewRequest("POST", srv.URL+"/api/v1/tutor-planner/calendar", bytes.NewReader(raw))
		if e != nil {
			t.Fatal(e)
		}
		req.Header.Set("Content-Type", "application/json")
		if session != "" {
			req.AddCookie(&http.Cookie{Name: store.SessionCookieName, Value: session})
		}
		res, e := srv.Client().Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		data, e := io.ReadAll(res.Body)
		if e != nil {
			t.Fatal(e)
		}
		if res.StatusCode != want {
			t.Fatalf("publish=%d want%d %s", res.StatusCode, want, data)
		}
		var out domain.PlanCalendarResult
		if want == 200 {
			if e = json.Unmarshal(data, &out); e != nil {
				t.Fatal(e)
			}
		}
		return out
	}
	in := domain.PlanCalendarInput{Revision: plan.Revision, From: day.Format("2006-01-02"), To: day.Format("2006-01-02"), RequestID: id()}
	call("", in, 401)
	call(ss, in, 403)
	call(osession, in, 404)
	out := call(ts, in, 200)
	if out.Created != 1 || out.Unlinked != 1 || out.Paused != 1 {
		t.Fatal(out)
	}
	if replay := call(ts, in, 200); replay != out {
		t.Fatal("replay changed", replay, out)
	}
	for _, actor := range []domain.User{tutor, student, other} {
		lessons, e := st.ListLessons(t.Context(), actor)
		if e != nil {
			t.Fatal(e)
		}
		want := 1
		if actor.ID == other.ID {
			want = 0
		}
		if len(lessons) != want {
			t.Fatal("calendar visibility", actor.ID, lessons)
		}
	}
	changed := in
	changed.To = day.AddDate(0, 0, 1).Format("2006-01-02")
	call(ts, changed, 409)
	in.RequestID = id()
	out = call(ts, in, 200)
	if out.Created != 0 || out.Existing != 1 {
		t.Fatal("duplicate calendar entries", out)
	}
	bad := in
	bad.RequestID = id()
	bad.Revision++
	call(ts, bad, 409)
	bad = in
	bad.RequestID = id()
	bad.To = day.AddDate(0, 0, 84).Format("2006-01-02")
	call(ts, bad, 400)
	call(ts, map[string]any{"revision": in.Revision, "from": in.From, "to": in.To, "request_id": id(), "owner_id": other.ID}, 400)
	plan.Students[0].Subject = "Changed after publication"
	plan, err = st.SaveTutorPlan(t.Context(), tutor, plan)
	if err != nil {
		t.Fatal(err)
	}
	in.RequestID = id()
	in.Revision = plan.Revision
	call(ts, in, 200)
	lessons, err := st.ListLessons(t.Context(), student)
	if err != nil || len(lessons) != 1 || lessons[0].Title != "Fractions" {
		t.Fatal("calendar changed retroactively", lessons, err)
	}
	plan.Lessons = append(plan.Lessons, domain.PlanLesson{ID: id(), StudentID: plan.Students[0].ID, Weekday: weekday, StartMinute: 630, Duration: 60})
	plan, err = st.SaveTutorPlan(t.Context(), tutor, plan)
	if err != nil {
		t.Fatal(err)
	}
	in.RequestID = id()
	in.Revision = plan.Revision
	call(ts, in, 409)
	lessons, err = st.ListLessons(t.Context(), student)
	if err != nil || len(lessons) != 1 {
		t.Fatal("conflicting export partly committed", lessons, err)
	}
	if _, err = pool.Exec(t.Context(), `update relations set status='ended' where id=$1`, relation.ID); err != nil {
		t.Fatal(err)
	}
	in.RequestID = id()
	in.From = day.AddDate(0, 0, 7).Format("2006-01-02")
	in.To = in.From
	call(ts, in, 404)
}
