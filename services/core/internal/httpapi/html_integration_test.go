package httpapi

import (
	"bytes"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

func TestHTMLLessonsPostgres(t *testing.T) {
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
	cfg := config.Config{RequestBodyLimitByte: 1 << 20, CORSAllowedOrigins: []string{"http://app.test"}, HTMLPreviewPublicURL: "http://html.localhost:8081"}
	app := httptest.NewServer(NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), st, auth.DevProvider{}))
	defer app.Close()
	preview := httptest.NewServer(NewHTMLPreviewServer(cfg, st))
	defer preview.Close()
	id := func() string {
		v, e := platform.NewUUID()
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	register := func(role domain.Role) (domain.User, *http.Cookie) {
		u, token, e := st.RegisterUser(t.Context(), id()+"@example.test", "HTML test", "password123", role, "test", "127.0.0.1")
		if e != nil {
			t.Fatal(e)
		}
		return u, &http.Cookie{Name: store.SessionCookieName, Value: token}
	}
	tutor, tc := register(domain.RoleTutor)
	student, sc := register(domain.RoleStudent)
	_, peer := register(domain.RoleStudent)
	_, other := register(domain.RoleTutor)
	inv, token, err := st.CreateInvitation(t.Context(), tutor, student.Email, "Student")
	_ = inv
	if err != nil {
		t.Fatal(err)
	}
	rel, err := st.AcceptInvitation(t.Context(), student, token)
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path string, input any, cookie *http.Cookie, want int) []byte {
		t.Helper()
		data, e := json.Marshal(input)
		if e != nil {
			t.Fatal(e)
		}
		r, e := http.NewRequestWithContext(t.Context(), method, app.URL+path, bytes.NewReader(data))
		if e != nil {
			t.Fatal(e)
		}
		r.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		res, e := app.Client().Do(r)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		body, e := io.ReadAll(res.Body)
		if e != nil {
			t.Fatal(e)
		}
		if res.StatusCode != want {
			t.Fatalf("%s %s: %d want %d: %s", method, path, res.StatusCode, want, body)
		}
		return body
	}
	decode := func(data []byte, out any) {
		t.Helper()
		if err := json.Unmarshal(data, out); err != nil {
			t.Fatal(err)
		}
	}
	var course domain.Course
	decode(call("POST", "/api/v1/catalog/courses", domain.CourseInput{Title: "HTML course", Track: "school", GradeMin: 1, GradeMax: 11, Difficulty: 1, Visibility: "linked_students"}, tc, 201), &course)
	var topic domain.Topic
	decode(call("POST", "/api/v1/catalog/courses/"+course.ID+"/topics", domain.TopicInput{Title: "HTML topic"}, tc, 201), &topic)
	source := `<!doctype html><html lang="ru"><head><title>Сложение HTML</title></head><body><h1>5 + 2</h1><button onclick="this.textContent='7'">Решить</button><script>const PRIVATE_LEGACY_KEY=7;</script></body></html>`
	in := domain.HTMLImportInput{RequestID: id(), CourseID: course.ID, TopicID: topic.ID, Track: "school", GradeMin: 5, GradeMax: 7, Difficulty: 1, Filename: "lesson.html", Source: source, RightsStatus: "own"}
	call("GET", "/api/v1/html-lessons", nil, nil, 401)
	call("POST", "/api/v1/html-lessons", in, sc, 403)
	var lesson domain.HTMLLesson
	payload := call("POST", "/api/v1/html-lessons", in, tc, 201)
	decode(payload, &lesson)
	if strings.Contains(string(payload), "PRIVATE_LEGACY_KEY") || lesson.Version.Title != "Сложение HTML" || lesson.Version.Number != 1 {
		t.Fatal("metadata/source leak", string(payload))
	}
	lp := "/api/v1/html-lessons/" + lesson.ID
	v1 := lesson.Version.ID
	var again domain.HTMLLesson
	decode(call("POST", "/api/v1/html-lessons", in, tc, 201), &again)
	if again.ID != lesson.ID || again.Version.ID != v1 {
		t.Fatal("upload replay")
	}
	call("GET", lp, nil, sc, 404)
	call("GET", lp, nil, other, 404)
	call("POST", lp+"/publish", htmlVersionAction{v1}, tc, 422)
	call("POST", "/api/v1/catalog/courses/"+course.ID+"/publish", map[string]any{}, tc, 200)
	call("POST", "/api/v1/catalog/topics/"+topic.ID+"/publish", map[string]any{}, tc, 200)
	call("POST", lp+"/publish", htmlVersionAction{v1}, tc, 200)
	call("GET", lp, nil, sc, 200)
	call("GET", lp, nil, peer, 404)
	call("GET", lp+"/source?version_id="+v1, nil, sc, 403)
	var original map[string]string
	decode(call("GET", lp+"/source?version_id="+v1, nil, tc, 200), &original)
	if original["source"] != source {
		t.Fatal("source changed")
	}
	var grant struct {
		URL string `json:"url"`
	}
	decode(call("POST", lp+"/preview", map[string]string{"version_id": v1}, sc, 200), &grant)
	parsed, _ := url.Parse(grant.URL)
	req, _ := http.NewRequestWithContext(t.Context(), "GET", preview.URL+parsed.Path, nil)
	req.Host = "html.localhost:8081"
	res, err := preview.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || string(body) != source {
		t.Fatal("preview source", res.StatusCode)
	}
	csp := res.Header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "sandbox allow-scripts") || strings.Contains(csp, "allow-same-origin") || !strings.Contains(csp, "connect-src 'none'") || res.Header.Get("Set-Cookie") != "" {
		t.Fatal("unsafe preview headers", csp)
	}
	res, err = app.Client().Get(app.URL + parsed.Path)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode == 200 {
		t.Fatal("HTML exposed on app origin")
	}
	var assignment domain.Assignment
	assign := domain.HTMLAssignInput{RequestID: id(), StudentID: student.ID, DueAt: time.Now().Add(24 * time.Hour)}
	decode(call("POST", lp+"/assign", assign, tc, 201), &assignment)
	if assignment.HTML == nil || assignment.HTML.VersionID != v1 {
		t.Fatal("assignment snapshot")
	}
	var replay domain.Assignment
	decode(call("POST", lp+"/assign", assign, tc, 201), &replay)
	if replay.ID != assignment.ID {
		t.Fatal("assign replay")
	}
	call("GET", "/api/v1/assignments/"+assignment.ID, nil, peer, 404)
	call("GET", "/api/v1/assignments/"+assignment.ID, nil, other, 404)
	in.RequestID = id()
	in.Source = strings.ReplaceAll(source, "5 + 2", "8 + 1")
	in.Title = "Изменённый HTML"
	decode(call("POST", lp+"/versions", in, tc, 201), &lesson)
	if lesson.Version.Number != 2 {
		t.Fatal("version")
	}
	decode(call("GET", lp, nil, sc, 200), &again)
	if again.Version.ID != v1 {
		t.Fatal("draft overwrote publication")
	}
	if _, err = pool.Exec(t.Context(), `update html_lesson_versions set source='changed' where id=$1`, v1); err == nil {
		t.Fatal("mutable source")
	}
	call("POST", lp+"/publish", htmlVersionAction{lesson.Version.ID}, tc, 200)
	var pinned domain.HTMLLesson
	decode(call("GET", lp+"?assignment_id="+assignment.ID, nil, sc, 200), &pinned)
	if pinned.Version.ID != v1 {
		t.Fatal("assignment changed after publish")
	}
	call("POST", lp+"/archive", map[string]any{}, tc, 200)
	call("GET", lp, nil, sc, 404)
	call("GET", lp+"?assignment_id="+assignment.ID, nil, sc, 200)
	var converted domain.LearningLesson
	decode(call("POST", lp+"/convert", htmlVersionAction{v1}, tc, 201), &converted)
	if converted.Status != "draft" || len(converted.Version.Exercises) != 0 || strings.Contains(converted.Version.Content, "PRIVATE_LEGACY_KEY") {
		t.Fatal("conversion trusted legacy code")
	}
	call("POST", "/api/v1/catalog/lessons/"+converted.ID+"/publish", map[string]any{}, tc, 422)
	call("POST", "/api/v1/catalog/lessons/"+converted.ID+"/exercises", domain.ExerciseInput{Kind: "numeric", Prompt: "Сколько будет 5 + 2?", Difficulty: 1, AnswerKey: domain.AnswerKeyInput{Answer: "7", Explanation: "Складываем числа."}}, tc, 201)
	call("POST", "/api/v1/catalog/lessons/"+converted.ID+"/publish", map[string]any{}, tc, 200)
	var native domain.LearningLesson
	decode(call("GET", "/api/v1/catalog/lessons/"+converted.ID, nil, sc, 200), &native)
	if len(native.Version.Exercises) != 1 {
		t.Fatal("converted native exercises unavailable")
	}
	call("POST", "/api/v1/assignments/"+assignment.ID+"/submit", map[string]string{"answer": "Решил самостоятельно, прошу проверить"}, sc, 200)
	var runs int
	if err = pool.QueryRow(t.Context(), `select count(*) from exercise_attempts where student_id=$1`, student.ID).Scan(&runs); err != nil || runs != 0 {
		t.Fatal("legacy generated trusted statistics", err)
	}
	if _, err = pool.Exec(t.Context(), `update relations set status='blocked' where id=$1`, rel.ID); err != nil {
		t.Fatal(err)
	}
	call("GET", lp+"?assignment_id="+assignment.ID, nil, sc, 404)
	call("POST", "/api/v1/assignments/"+assignment.ID+"/submit", map[string]string{"answer": "replay"}, sc, 404)
	req, _ = http.NewRequestWithContext(t.Context(), "GET", preview.URL+parsed.Path, nil)
	req.Host = "html.localhost:8081"
	res, err = preview.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 404 {
		t.Fatal("revoked preview still readable")
	}
	call("POST", "/api/v1/html-lessons", map[string]any{"owner_id": tutor.ID}, tc, 400)
}
