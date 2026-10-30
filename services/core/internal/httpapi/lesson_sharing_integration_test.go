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
	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

func TestLessonSharingPostgres(t *testing.T) {
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
		value, err := platform.NewUUID()
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	register := func(role domain.Role) (domain.User, string) {
		u, s, err := st.RegisterUser(t.Context(), id()+"@example.test", "Sharing fixture", "Password123!", role, "test", "127.0.0.1")
		if err != nil {
			t.Fatal(err)
		}
		return u, s
	}
	a, as := register(domain.RoleTutor)
	b, bs := register(domain.RoleTutor)
	c, cs := register(domain.RoleTutor)
	student, ss := register(domain.RoleStudent)
	orgID, foreignID := id(), id()
	if _, err = pool.Exec(t.Context(), `insert into organizations(id,name) values($1,'Sharing isolated org')`, orgID); err != nil {
		t.Fatal(err)
	}
	foreignEmail := id() + "@example.test"
	if _, err = pool.Exec(t.Context(), `insert into users(id,organization_id,email,display_name,role)values($1,$2,$3,'Foreign tutor','tutor')`, foreignID, orgID, foreignEmail); err != nil {
		t.Fatal(err)
	}
	foreign, err := st.GetUser(t.Context(), foreignID)
	if err != nil {
		t.Fatal(err)
	}
	fs, err := st.CreateSession(t.Context(), foreign, "test", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(NewServer(config.Config{RequestBodyLimitByte: 1 << 20}, slog.New(slog.NewTextHandler(io.Discard, nil)), st, auth.DevProvider{}))
	defer srv.Close()
	call := func(session, method, path string, input any, want int, out any) []byte {
		t.Helper()
		data, _ := json.Marshal(input)
		req, _ := http.NewRequest(method, srv.URL+path, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		if session != "" {
			req.AddCookie(&http.Cookie{Name: store.SessionCookieName, Value: session})
		}
		res, err := srv.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		if res.StatusCode != want {
			t.Fatalf("%s %s: %d want %d %s", method, path, res.StatusCode, want, body)
		}
		if out != nil && json.Unmarshal(body, out) != nil {
			t.Fatalf("invalid JSON %s", body)
		}
		return body
	}
	structure := func(actor domain.User) (domain.Course, domain.Topic) {
		course, err := st.CreateCourse(t.Context(), actor, domain.CourseInput{Title: "Sharing course", Track: "school", GradeMin: 5, GradeMax: 8, Difficulty: 1, Visibility: "linked_students"})
		if err != nil {
			t.Fatal(err)
		}
		course, err = st.SetCourseStatus(t.Context(), actor, course.ID, "published")
		if err != nil {
			t.Fatal(err)
		}
		topic, err := st.CreateTopic(t.Context(), actor, course.ID, domain.TopicInput{Title: "Fractions"})
		if err != nil {
			t.Fatal(err)
		}
		topic, err = st.SetTopicStatus(t.Context(), actor, topic.ID, "published")
		if err != nil {
			t.Fatal(err)
		}
		return course, topic
	}
	ac, at := structure(a)
	bc, bt := structure(b)
	_, ct := structure(c)
	original, err := st.CreateLearningLesson(t.Context(), a, ac.ID, domain.LearningLessonInput{TopicID: at.ID, Title: "Pinned sharing example", Content: "Original explanation", Track: "school", GradeMin: 5, GradeMax: 8, Difficulty: 2})
	if err != nil {
		t.Fatal(err)
	}
	privateAnswer := "1234567/89"
	exercise, err := st.SaveExercise(t.Context(), a, original.ID, "", domain.ExerciseInput{Kind: "numeric", Prompt: "An authored fraction", Options: []string{}, Difficulty: 2, AnswerKey: domain.AnswerKeyInput{Answer: privateAnswer, Explanation: "Private original solution", Hints: []string{"A useful hint"}}})
	if err != nil {
		t.Fatal(err)
	}
	sharePath := "/api/v1/catalog/lessons/" + original.ID + "/shares"
	grant := domain.LessonShareInput{RecipientEmail: b.Email, AllowCopy: true, RightsConfirmed: true}
	call(as, "POST", sharePath, grant, 404, nil)
	original, err = st.PublishLearningLesson(t.Context(), a, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, session := range []string{bs, cs, fs} {
		call(session, "POST", sharePath, grant, 404, nil)
	}
	call(ss, "POST", sharePath, grant, 403, nil)
	call("", "POST", sharePath, grant, 401, nil)
	for _, email := range []string{a.Email, student.Email, foreignEmail, id() + "@example.test"} {
		bad := grant
		bad.RecipientEmail = email
		call(as, "POST", sharePath, bad, 404, nil)
	}
	bad := grant
	bad.RightsConfirmed = false
	call(as, "POST", sharePath, bad, 400, nil)
	for _, field := range []string{"owner_id", "role", "organization_id", "created_by", "permissions", "recipient_id", "version_id"} {
		call(as, "POST", sharePath, map[string]any{"recipient_email": b.Email, "allow_copy": true, "rights_confirmed": true, field: b.ID}, 400, nil)
	}
	var shared, repeated domain.LessonShare
	raw := call(as, "POST", sharePath, grant, 201, &shared)
	if bytes.Contains(raw, []byte(privateAnswer)) {
		t.Fatal("private answer leaked")
	}
	call(as, "POST", sharePath, grant, 201, &repeated)
	if shared.ID != repeated.ID {
		t.Fatal("duplicate grant")
	}
	var received domain.SharedLesson
	raw = call(bs, "GET", "/api/v1/catalog/shared-lessons/"+shared.ID, nil, 200, &received)
	if bytes.Contains(raw, []byte(privateAnswer)) || bytes.Contains(raw, []byte("Private original solution")) {
		t.Fatal("shared public DTO leaked keys")
	}
	if received.Lesson.Version.ID != original.Version.ID || received.Share.VersionNumber != 1 {
		t.Fatal("wrong pinned version")
	}
	for _, session := range []string{as, cs, fs} {
		call(session, "GET", "/api/v1/catalog/shared-lessons/"+shared.ID, nil, 404, nil)
	}
	call(ss, "GET", "/api/v1/catalog/shared-lessons/"+shared.ID, nil, 403, nil)
	call(bs, "GET", "/api/v1/catalog/courses/"+ac.ID, nil, 404, nil)
	call(bs, "GET", "/api/v1/catalog/courses/"+ac.ID+"/topics", nil, 404, nil)
	call(bs, "GET", "/api/v1/catalog/lessons/"+original.ID, nil, 404, nil)
	call(bs, "GET", "/api/v1/catalog/lessons/"+original.ID+"/answer-keys", nil, 404, nil)
	var page domain.CatalogPage[domain.LessonShare]
	call(bs, "GET", "/api/v1/catalog/shared-lessons?limit=1&q=Pinned", nil, 200, &page)
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatal(page)
	}
	call(bs, "GET", "/api/v1/catalog/shared-lessons?limit=1&offset=1", nil, 200, &page)
	if page.Total != 1 || len(page.Items) != 0 {
		t.Fatal(page)
	}
	call(cs, "GET", "/api/v1/catalog/shared-lessons", nil, 200, &page)
	if page.Total != 0 {
		t.Fatal("incoming grants leaked")
	}
	call(as, "GET", sharePath, nil, 200, &page)
	if page.Total != 1 {
		t.Fatal(page)
	}
	call(bs, "GET", sharePath, nil, 404, nil)
	call(bs, "GET", "/api/v1/catalog/shared-lessons?owner_id="+a.ID, nil, 400, nil)
	input := original.Version.LearningLessonInput
	input.Title = "New publication, not the shared snapshot"
	if _, err = st.UpdateLearningLesson(t.Context(), a, original.ID, input); err != nil {
		t.Fatal(err)
	}
	if _, err = st.PublishLearningLesson(t.Context(), a, original.ID); err != nil {
		t.Fatal(err)
	}
	call(bs, "GET", "/api/v1/catalog/shared-lessons/"+shared.ID, nil, 200, &received)
	if received.Lesson.Version.Title != "Pinned sharing example" || received.Lesson.Version.Number != 1 {
		t.Fatal("share changed retroactively")
	}
	call(as, "POST", sharePath, grant, 409, nil)
	copyPath := "/api/v1/catalog/shared-lessons/" + shared.ID + "/copy"
	copyInput := domain.LessonCopyInput{RequestID: id(), CourseID: bc.ID, TopicID: bt.ID, Title: "My personal draft"}
	call(cs, "POST", copyPath, copyInput, 404, nil)
	call(ss, "POST", copyPath, copyInput, 403, nil)
	for _, target := range []domain.LessonCopyInput{{RequestID: id(), CourseID: ac.ID, TopicID: at.ID, Title: "Foreign course"}, {RequestID: id(), CourseID: bc.ID, TopicID: ct.ID, Title: "Foreign topic"}} {
		call(bs, "POST", copyPath, target, 404, nil)
	}
	var copied, again domain.LearningLesson
	raw = call(bs, "POST", copyPath, copyInput, 201, &copied)
	if bytes.Contains(raw, []byte(privateAnswer)) {
		t.Fatal("copy response leaked answer")
	}
	if copied.Status != "draft" || copied.AuthorID != b.ID || copied.CourseID != bc.ID || copied.Version.TopicID != bt.ID || copied.Version.Exercises[0].ID == exercise.ID || copied.Provenance == nil || copied.Provenance.SourceVersionNumber != 1 || copied.Provenance.OriginalAuthorID != a.ID {
		t.Fatalf("bad copy: %+v", copied)
	}
	call(bs, "POST", copyPath, copyInput, 201, &again)
	if copied.ID != again.ID {
		t.Fatal("copy replay duplicated lesson")
	}
	changed := copyInput
	changed.Title = "Different payload"
	call(bs, "POST", copyPath, changed, 409, nil)
	call(as, "GET", "/api/v1/catalog/lessons/"+copied.ID, nil, 404, nil)
	var keys struct {
		AnswerKeys []domain.ExerciseAnswerKey `json:"answer_keys"`
	}
	private := call(bs, "GET", "/api/v1/catalog/lessons/"+copied.ID+"/answer-keys", nil, 200, &keys)
	if !bytes.Contains(private, []byte(privateAnswer)) {
		t.Fatal("private key was not copied server side")
	}
	if _, err = pool.Exec(t.Context(), `update learning_lesson_provenance set source_title='tampered' where lesson_id=$1`, copied.ID); err == nil {
		t.Fatal("mutable attribution")
	}
	if _, err = st.PublishLearningLesson(t.Context(), b, copied.ID); err != nil {
		t.Fatal(err)
	}
	call(bs, "POST", "/api/v1/catalog/lessons/"+copied.ID+"/shares", domain.LessonShareInput{RecipientEmail: c.Email, AllowCopy: true, RightsConfirmed: true}, 404, nil)
	call(bs, "POST", sharePath+"/"+shared.ID+"/revoke", map[string]any{}, 404, nil)
	call(as, "POST", sharePath+"/"+shared.ID+"/revoke", map[string]any{}, 200, nil)
	call(as, "POST", sharePath+"/"+shared.ID+"/revoke", map[string]any{}, 200, nil)
	call(bs, "GET", "/api/v1/catalog/shared-lessons/"+shared.ID, nil, 404, nil)
	copyFresh := copyInput
	copyFresh.RequestID = id()
	call(bs, "POST", copyPath, copyFresh, 404, nil)
	call(bs, "POST", copyPath, copyInput, 201, &again)
	if again.ID != copied.ID {
		t.Fatal("legal existing copy lost after revocation")
	}
	grant.AllowCopy = false
	call(as, "POST", sharePath, grant, 201, &shared)
	copyPath = "/api/v1/catalog/shared-lessons/" + shared.ID + "/copy"
	call(bs, "POST", copyPath, copyFresh, 403, nil)
	if _, err = st.ArchiveLearningLesson(t.Context(), a, original.ID); err != nil {
		t.Fatal(err)
	}
	call(bs, "GET", "/api/v1/catalog/shared-lessons/"+shared.ID, nil, 404, nil)
	// Concurrent identical copy requests create one draft, not two.
	second, err := st.CreateLearningLesson(t.Context(), a, ac.ID, domain.LearningLessonInput{TopicID: at.ID, Title: "Concurrent source", Content: "Original", Track: "school", GradeMin: 5, GradeMax: 8, Difficulty: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = st.PublishLearningLesson(t.Context(), a, second.ID); err != nil {
		t.Fatal(err)
	}
	g, err := st.CreateLessonShare(t.Context(), a, second.ID, domain.LessonShareInput{RecipientEmail: b.Email, AllowCopy: true, RightsConfirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	parallel := domain.LessonCopyInput{RequestID: id(), CourseID: bc.ID, TopicID: bt.ID, Title: "Concurrent copy"}
	var wg sync.WaitGroup
	results := make(chan string, 4)
	failures := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, e := st.CopySharedLesson(t.Context(), b, g.ID, parallel)
			if e != nil {
				failures <- e
			} else {
				results <- out.ID
			}
		}()
	}
	wg.Wait()
	close(failures)
	close(results)
	for e := range failures {
		t.Fatal(e)
	}
	var only string
	for value := range results {
		if only != "" && only != value {
			t.Fatal("concurrent duplicate")
		}
		only = value
	}
}
