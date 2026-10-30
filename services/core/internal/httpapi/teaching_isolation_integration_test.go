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

func TestTeachingIsolationPostgres(t *testing.T) {
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
		u, token, e := st.RegisterUser(t.Context(), id()+"@example.test", "Teaching fixture", "Password123!", role, "test", "127.0.0.1")
		if e != nil {
			t.Fatal(e)
		}
		return u, token
	}
	a, as := register(domain.RoleTutor)
	b, bs := register(domain.RoleTutor)
	sa, sas := register(domain.RoleStudent)
	sb, sbs := register(domain.RoleStudent)
	connect := func(teacher, student domain.User) domain.Relation {
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
	ra := connect(a, sa)
	connect(b, sb)
	srv := httptest.NewServer(NewServer(config.Config{RequestBodyLimitByte: 1 << 20}, slog.New(slog.NewTextHandler(io.Discard, nil)), st, auth.DevProvider{}))
	defer srv.Close()
	call := func(session, method, path string, input any, want int, out any) {
		t.Helper()
		data, e := json.Marshal(input)
		if e != nil {
			t.Fatal(e)
		}
		req, e := http.NewRequest(method, srv.URL+path, bytes.NewReader(data))
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
		body, e := io.ReadAll(res.Body)
		if e != nil {
			t.Fatal(e)
		}
		if res.StatusCode != want {
			t.Fatalf("%s %s=%d want%d %s", method, path, res.StatusCode, want, body)
		}
		if out != nil {
			if e = json.Unmarshal(body, out); e != nil {
				t.Fatal(e)
			}
		}
	}
	path := "/api/v1/teaching-preferences"
	var first, other domain.TeachingPreferences
	call(as, "GET", path, nil, 200, &first)
	if first.Revision != 0 || len(first.Grades) != 11 {
		t.Fatal(first)
	}
	call("", "GET", path, nil, 401, nil)
	call(sas, "GET", path, nil, 403, nil)
	in := domain.TeachingPreferences{Subjects: []string{"math", "physics"}, Grades: []int{7, 8, 9}, Duration: 45}
	call(as, "PUT", path, in, 200, &first)
	if first.Revision != 1 || first.Duration != 45 {
		t.Fatal(first)
	}
	call(bs, "GET", path+"?tutor_id="+a.ID, nil, 200, &other)
	if other.Revision != 0 || other.Duration != 60 {
		t.Fatal("preferences leaked", other)
	}
	call(as, "PUT", path, in, 409, nil)
	call(sas, "PUT", path, in, 403, nil)
	for _, field := range []string{"role", "owner_id", "organization_id", "permissions", "created_by", "tutor_id"} {
		body := map[string]any{"revision": 1, "subjects": []string{"math"}, "grades": []int{7}, "default_duration_minutes": 45, field: b.ID}
		call(as, "PUT", path, body, 400, nil)
	}
	for _, bad := range []domain.TeachingPreferences{{Revision: 1, Subjects: []string{"math", "math"}, Grades: []int{7}, Duration: 45}, {Revision: 1, Subjects: []string{"math"}, Grades: []int{7, 7}, Duration: 45}, {Revision: 1, Subjects: []string{"unknown"}, Grades: []int{7}, Duration: 45}, {Revision: 1, Subjects: []string{"math"}, Grades: []int{12}, Duration: 45}, {Revision: 1, Subjects: []string{"math"}, Grades: []int{7}, Duration: 241}} {
		call(as, "PUT", path, bad, 400, nil)
	}
	foreignID, orgID := id(), id()
	if _, err = pool.Exec(t.Context(), `insert into organizations(id,name)values($1,'Teaching isolation fixture')`, orgID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(t.Context(), `insert into users(id,organization_id,email,display_name,role)values($1,$2,$3,'Foreign fixture','tutor')`, foreignID, orgID, id()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	foreign, err := st.GetUser(t.Context(), foreignID)
	if err != nil {
		t.Fatal(err)
	}
	foreignSession, err := st.CreateSession(t.Context(), foreign, "test", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	call(foreignSession, "GET", path, nil, 200, &other)
	if other.Revision != 0 {
		t.Fatal("organization preference leak")
	}
	call(foreignSession, "PUT", path, in, 200, &other)
	call(as, "GET", path, nil, 200, &first)
	if first.Revision != 1 {
		t.Fatal("foreign write changed settings")
	}
	course, err := st.CreateCourse(t.Context(), a, domain.CourseInput{Title: "Public teaching fixture", Track: "school", GradeMin: 5, GradeMax: 11, Difficulty: 1, Visibility: "organization"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = st.SetCourseStatus(t.Context(), a, course.ID, "published"); err != nil {
		t.Fatal(err)
	}
	topic, err := st.CreateTopic(t.Context(), a, course.ID, domain.TopicInput{Title: "Teaching isolation"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = st.SetTopicStatus(t.Context(), a, topic.ID, "published"); err != nil {
		t.Fatal(err)
	}
	lesson, err := st.CreateLearningLesson(t.Context(), a, course.ID, domain.LearningLessonInput{Title: "Public template", Content: "Read only source", TopicID: topic.ID, Track: "school", GradeMin: 5, GradeMax: 11, Difficulty: 1})
	if err != nil {
		t.Fatal(err)
	}
	objective, err := st.SaveExercise(t.Context(), a, lesson.ID, "", domain.ExerciseInput{Kind: "numeric", Prompt: "2 + 2", Difficulty: 1, Position: 0, AnswerKey: domain.AnswerKeyInput{Answer: "4"}})
	if err != nil {
		t.Fatal(err)
	}
	manual, err := st.SaveExercise(t.Context(), a, lesson.ID, "", domain.ExerciseInput{Kind: "long_text", Prompt: "Explain", Difficulty: 1, Position: 1, AnswerKey: domain.AnswerKeyInput{Answer: "[manual]"}})
	if err != nil {
		t.Fatal(err)
	}
	lesson, err = st.PublishLearningLesson(t.Context(), a, lesson.ID)
	if err != nil {
		t.Fatal(err)
	}
	lp := "/api/v1/catalog/lessons/" + lesson.ID
	call(bs, "GET", lp, nil, 200, nil)
	call(bs, "PUT", lp, lesson.Version.LearningLessonInput, 404, nil)
	call(foreignSession, "GET", lp, nil, 404, nil)
	input := func(value string) domain.ExerciseAttemptInput {
		return domain.ExerciseAttemptInput{VersionID: lesson.Version.ID, RequestID: id(), Answer: domain.ExerciseAnswer{Value: &value}}
	}
	call(sbs, "POST", lp+"/exercises/"+objective.ID+"/attempts", input("4"), 201, nil)
	var page domain.CatalogPage[domain.ExerciseAttempt]
	call(as, "GET", lp+"/attempts", nil, 200, &page)
	if page.Total != 0 {
		t.Fatal("public author saw another tutor's student", page)
	}
	notes, err := st.ListNotifications(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	call(sbs, "POST", lp+"/exercises/"+manual.ID+"/attempts", input("PRIVATE_OTHER_STUDENT_SOLUTION"), 404, nil)
	after, err := st.ListNotifications(t.Context(), a)
	if err != nil || len(after) != len(notes) {
		t.Fatal("unrelated student's manual answer leaked via notification", err)
	}
	var own domain.ExerciseAttempt
	call(sas, "POST", lp+"/exercises/"+manual.ID+"/attempts", input("OWN_SOLUTION"), 201, &own)
	call(as, "GET", lp+"/attempts", nil, 200, &page)
	if page.Total != 1 || page.Items[0].StudentID != sa.ID {
		t.Fatal("own relation missing", page)
	}
	if _, err = pool.Exec(t.Context(), `update relations set status='ended' where id=$1`, ra.ID); err != nil {
		t.Fatal(err)
	}
	call(as, "GET", lp+"/attempts", nil, 200, &page)
	if page.Total != 0 {
		t.Fatal("revoked child's attempt remained visible")
	}
	call(as, "PUT", lp+"/attempts/"+own.ID+"/review", domain.ExerciseReviewInput{Status: "correct"}, 404, nil)
}
