package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

func TestLearningCatalogPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := store.Migrate(ctx, pool, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	st := store.New(pool)
	server := httptest.NewServer(NewServer(config.Config{CORSAllowedOrigins: []string{"http://example.test"}, RequestBodyLimitByte: 1 << 20}, slog.New(slog.NewTextHandler(io.Discard, nil)), st, auth.DevProvider{Enabled: false}))
	defer server.Close()
	client := server.Client()
	request := func(method, path string, body any, cookie *http.Cookie) (int, []byte, *http.Cookie) {
		t.Helper()
		var reader io.Reader
		if body != nil {
			data, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			reader = bytes.NewReader(data)
		}
		req, err := http.NewRequestWithContext(ctx, method, server.URL+path, reader)
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
		data, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		var session *http.Cookie
		for _, c := range res.Cookies() {
			if c.Name == store.SessionCookieName {
				session = c
			}
		}
		return res.StatusCode, data, session
	}
	call := func(method, path string, body any, cookie *http.Cookie, want int) []byte {
		t.Helper()
		status, data, _ := request(method, path, body, cookie)
		if status != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, status, want, data)
		}
		return data
	}
	decode := func(data []byte, out any) {
		t.Helper()
		if err := json.Unmarshal(data, out); err != nil {
			t.Fatal(err)
		}
	}
	unique, err := platform.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	register := func(prefix, role string) (domain.User, *http.Cookie) {
		t.Helper()
		status, data, cookie := request("POST", "/api/v1/auth/register", map[string]any{"display_name": prefix, "email": prefix + unique + "@example.test", "password": "password123", "role": role, "accept_terms": true}, nil)
		if status != 201 || cookie == nil {
			t.Fatalf("register: %d %s", status, data)
		}
		var result struct {
			User domain.User `json:"user"`
		}
		decode(data, &result)
		return result.User, cookie
	}
	tutor, tutorCookie := register("catalog-tutor", "tutor")
	student, studentCookie := register("catalog-student", "student")
	_, otherTutorCookie := register("catalog-other-tutor", "tutor")
	_, otherStudentCookie := register("catalog-other-student", "student")
	base := "/api/v1/catalog"
	call("GET", base+"/courses", nil, nil, 401)
	courseInput := domain.CourseInput{Title: "Олимпиадная математика", Description: "Курс 1–11", Track: "olympiad", GradeMin: 1, GradeMax: 11, Difficulty: 3}
	call("POST", base+"/courses", courseInput, studentCookie, 403)
	for _, field := range []string{"role", "owner_id", "organization_id", "created_by", "permissions", "author_id", "status"} {
		data, _ := json.Marshal(courseInput)
		var forged map[string]any
		decode(data, &forged)
		forged[field] = tutor.ID
		call("POST", base+"/courses", forged, tutorCookie, 400)
	}
	badCourse := courseInput
	badCourse.GradeMax = 12
	call("POST", base+"/courses", badCourse, tutorCookie, 400)
	for _, track := range []string{"school", "oge", "ege"} {
		input := courseInput
		input.Track = track
		call("POST", base+"/courses", input, tutorCookie, 201)
	}
	var course domain.Course
	decode(call("POST", base+"/courses", courseInput, tutorCookie, 201), &course)
	if course.AuthorID != tutor.ID || course.Visibility != "linked_students" || course.Status != "draft" {
		t.Fatalf("course defaults: %+v", course)
	}
	cp := base + "/courses/" + course.ID
	call("GET", cp, nil, studentCookie, 404)
	call("GET", cp, nil, otherTutorCookie, 404)
	call("POST", cp+"/publish", map[string]any{}, otherTutorCookie, 404)
	courseInput.Description = "Обновлённое описание"
	decode(call("PUT", cp, courseInput, tutorCookie, 200), &course)
	if course.Description != courseInput.Description {
		t.Fatal("course edit not saved")
	}
	var root, topic domain.Topic
	decode(call("POST", cp+"/topics", domain.TopicInput{Title: "Счёт"}, tutorCookie, 201), &root)
	decode(call("POST", cp+"/topics", domain.TopicInput{Title: "Чётность", ParentID: &root.ID}, tutorCookie, 201), &topic)
	var foreignCourse domain.Course
	decode(call("POST", base+"/courses", courseInput, tutorCookie, 201), &foreignCourse)
	var foreignTopic domain.Topic
	decode(call("POST", base+"/courses/"+foreignCourse.ID+"/topics", domain.TopicInput{Title: "Другая тема"}, tutorCookie, 201), &foreignTopic)
	call("POST", cp+"/topics", domain.TopicInput{Title: "Чужой родитель", ParentID: &foreignTopic.ID}, tutorCookie, 400)
	tp := base + "/topics/" + topic.ID
	call("PUT", base+"/topics/"+root.ID, domain.TopicInput{Title: "Цикл", ParentID: &topic.ID}, tutorCookie, 400)
	decode(call("PUT", tp, domain.TopicInput{Title: "Чётность и инварианты", ParentID: &root.ID, Position: 2}, tutorCookie, 200), &topic)
	var topics domain.CatalogPage[domain.Topic]
	decode(call("GET", cp+"/topics?limit=1&offset=1", nil, tutorCookie, 200), &topics)
	if topics.Total != 2 || len(topics.Items) != 1 || topics.Items[0].ID != topic.ID {
		t.Fatalf("topic pagination: %+v", topics)
	}
	lessonInput := domain.LearningLessonInput{TopicID: topic.ID, Title: "Секрет чётности", Summary: "Исследуем числа", Content: "Исходный текст", Track: "olympiad", GradeMin: 4, GradeMax: 7, Difficulty: 3}
	var lesson domain.LearningLesson
	badLesson := lessonInput
	badLesson.TopicID = foreignTopic.ID
	call("POST", cp+"/lessons", badLesson, tutorCookie, 400)
	badLesson = lessonInput
	badLesson.Content = "\x00"
	call("POST", cp+"/lessons", badLesson, tutorCookie, 400)
	decode(call("POST", cp+"/lessons", lessonInput, tutorCookie, 201), &lesson)
	lp := base + "/lessons/" + lesson.ID
	call("POST", lp+"/publish", map[string]any{}, tutorCookie, 422)
	call("POST", lp+"/publish", map[string]any{"created_by": tutor.ID}, tutorCookie, 400)
	call("POST", cp+"/publish", map[string]any{}, tutorCookie, 200)
	call("POST", tp+"/publish", map[string]any{}, tutorCookie, 422)
	call("POST", base+"/topics/"+root.ID+"/publish", map[string]any{}, tutorCookie, 200)
	call("POST", tp+"/publish", map[string]any{}, tutorCookie, 200)
	call("GET", lp, nil, studentCookie, 404)
	call("PUT", lp, lessonInput, otherTutorCookie, 404)
	call("POST", lp+"/publish", map[string]any{}, studentCookie, 403)
	key1 := "private-answer-one-" + unique
	key2 := "private-answer-two-" + unique
	exInput := domain.ExerciseInput{Kind: "text", Prompt: "Объясни закономерность", Difficulty: 2, AnswerKey: domain.AnswerKeyInput{Answer: key1, Explanation: "Закрытое объяснение"}}
	var exercise domain.Exercise
	badExercise := exInput
	badExercise.Kind = "single_choice"
	badExercise.Options = []string{"1", "2"}
	call("POST", lp+"/exercises", badExercise, tutorCookie, 400)
	badExercise.AnswerKey.Answer = "1"
	badJSONExercise := badExercise
	badJSONExercise.Options = []string{"\x00", "2"}
	badJSONExercise.AnswerKey.Answer = "\x00"
	call("POST", lp+"/exercises", badJSONExercise, tutorCookie, 400)
	var choiceExercise domain.Exercise
	decode(call("POST", lp+"/exercises", badExercise, tutorCookie, 201), &choiceExercise)
	data := call("POST", lp+"/exercises", exInput, tutorCookie, 201)
	decode(data, &exercise)
	assertNoCatalogKeys(t, data, key1, key2)
	ep := lp + "/exercises/" + exercise.ID
	call("GET", lp+"/answer-keys", nil, nil, 401)
	call("GET", lp+"/answer-keys", nil, studentCookie, 403)
	call("GET", lp+"/answer-keys", nil, otherTutorCookie, 404)
	if !bytes.Contains(call("GET", lp+"/answer-keys", nil, tutorCookie, 200), []byte(key1)) {
		t.Fatal("owner cannot read key")
	}
	var invitation struct {
		Token string `json:"token"`
	}
	decode(call("POST", "/api/v1/invitations", map[string]any{"student_email": student.Email}, tutorCookie, 201), &invitation)
	call("POST", "/api/v1/invitations/"+invitation.Token+"/accept", map[string]any{}, studentCookie, 201)
	call("GET", cp, nil, studentCookie, 200)
	call("GET", tp, nil, studentCookie, 200)
	decode(call("GET", cp+"/topics", nil, studentCookie, 200), &topics)
	if topics.Total != 2 {
		t.Fatal("published topic tree not visible")
	}
	call("GET", lp, nil, studentCookie, 404)
	decode(call("POST", lp+"/publish", map[string]any{}, tutorCookie, 200), &lesson)
	if lesson.Version.Number != 1 || lesson.HasUnpublishedChanges {
		t.Fatalf("publication: %+v", lesson)
	}
	var published domain.LearningLesson
	data = call("GET", lp, nil, studentCookie, 200)
	decode(data, &published)
	assertNoCatalogKeys(t, data, key1, key2)
	if published.Version.Content != "Исходный текст" || len(published.Version.Exercises) != 2 {
		t.Fatalf("published content: %+v", published)
	}
	call("GET", lp, nil, otherStudentCookie, 404)
	call("GET", lp+"/versions/0", nil, studentCookie, 404)
	call("GET", lp+"/versions/1", nil, studentCookie, 200)
	assertNoCatalogKeys(t, call("GET", ep, nil, studentCookie, 200), key1, key2)
	var exercises domain.CatalogPage[domain.Exercise]
	decode(call("GET", lp+"/exercises?limit=1", nil, studentCookie, 200), &exercises)
	if exercises.Total != 2 || len(exercises.Items) != 1 {
		t.Fatalf("exercise list: %+v", exercises)
	}
	lessonInput.Title = "Новая редакция"
	lessonInput.Content = "Изменённый текст"
	decode(call("PUT", lp, lessonInput, tutorCookie, 200), &lesson)
	if !lesson.HasUnpublishedChanges || lesson.Version.Number != 0 {
		t.Fatal("draft changes not tracked")
	}
	exInput.Prompt = "Новый вопрос"
	exInput.AnswerKey.Answer = key2
	call("PUT", ep, exInput, tutorCookie, 200)
	var stillPublished domain.LearningLesson
	decode(call("GET", lp, nil, studentCookie, 200), &stillPublished)
	if !reflect.DeepEqual(published.Version, stillPublished.Version) {
		t.Fatal("editing changed published snapshot")
	}
	assertNoCatalogKeys(t, call("GET", lp+"/versions", nil, studentCookie, 200), key1, key2)
	var noKeySearch domain.CatalogPage[domain.LearningLesson]
	decode(call("GET", base+"/lessons?course_id="+course.ID+"&q="+key1, nil, studentCookie, 200), &noKeySearch)
	if noKeySearch.Total != 0 {
		t.Fatal("search indexed private answers")
	}
	decode(call("GET", base+"/lessons?course_id="+course.ID+"&status=draft", nil, studentCookie, 200), &noKeySearch)
	if noKeySearch.Total != 0 {
		t.Fatal("draft status filter leaked")
	}
	var codes [2]int
	var responses [2][]byte
	var wg sync.WaitGroup
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i], responses[i], _ = request("POST", lp+"/publish", map[string]any{}, tutorCookie)
		}(i)
	}
	wg.Wait()
	if !(codes[0] == 200 && codes[1] == 422 || codes[1] == 200 && codes[0] == 422) {
		t.Fatalf("concurrent publish: %v %s %s", codes, responses[0], responses[1])
	}
	var oldVersion, newVersion domain.LearningLesson
	decode(call("GET", lp+"/versions/1", nil, tutorCookie, 200), &oldVersion)
	decode(call("GET", lp, nil, studentCookie, 200), &newVersion)
	updatedPrompt := false
	for _, item := range newVersion.Version.Exercises {
		if item.ID == exercise.ID && item.Prompt == exInput.Prompt {
			updatedPrompt = true
		}
	}
	if !reflect.DeepEqual(oldVersion.Version, published.Version) || newVersion.Version.Number != 2 || newVersion.Version.Content != lessonInput.Content || !updatedPrompt {
		t.Fatal("versions not isolated")
	}
	call("GET", lp+"/versions/1", nil, studentCookie, 404)
	if !bytes.Contains(call("GET", lp+"/answer-keys?version=1", nil, tutorCookie, 200), []byte(key1)) {
		t.Fatal("old answer key was changed")
	}
	if !bytes.Contains(call("GET", lp+"/answer-keys?version=2", nil, tutorCookie, 200), []byte(key2)) {
		t.Fatal("new key not published")
	}
	for _, sql := range []string{
		`update lesson_versions set content='corrupt' where id=$1`,
		`update lesson_versions set sealed=false where id=$1`,
		`delete from lesson_versions where id=$1`,
		`update exercises set prompt='corrupt' where version_id=$1`,
		`delete from exercises where version_id=$1`,
		`update exercise_answer_keys set answer='corrupt' where exercise_id in(select id from exercises where version_id=$1)`,
		`delete from exercise_answer_keys where exercise_id in(select id from exercises where version_id=$1)`,
		`insert into exercises(version_id,kind,prompt,difficulty) values($1,'text','corrupt',1)`,
	} {
		_, err := pool.Exec(ctx, sql, oldVersion.Version.ID)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
			t.Fatalf("snapshot SQL should be blocked: %s: %v", sql, err)
		}
	}
	var keyRowID string
	if err := pool.QueryRow(ctx, `select id::text from exercises where version_id=$1 limit 1`, oldVersion.Version.ID).Scan(&keyRowID); err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `insert into exercise_answer_keys(exercise_id,answer) values($1,'corrupt')`, keyRowID)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Fatalf("key insertion should be blocked: %v", err)
	}
	var versions domain.CatalogPage[domain.LessonVersion]
	decode(call("GET", lp+"/versions?limit=1&offset=1", nil, tutorCookie, 200), &versions)
	if versions.Total != 2 || len(versions.Items) != 1 || versions.Items[0].Number != 1 {
		t.Fatalf("versions: %+v", versions)
	}
	secondInput := lessonInput
	secondInput.Title = "Подготовка ЕГЭ"
	secondInput.Content = "Экзаменационная практика"
	secondInput.Track = "ege"
	secondInput.GradeMin = 11
	secondInput.GradeMax = 11
	secondInput.Difficulty = 5
	var second domain.LearningLesson
	decode(call("POST", cp+"/lessons", secondInput, tutorCookie, 201), &second)
	call("POST", base+"/lessons/"+second.ID+"/publish", map[string]any{}, tutorCookie, 200)
	for _, query := range []string{"?course_id=" + course.ID + "&q=редакция", "?course_id=" + course.ID + "&track=olympiad&grade=5&difficulty=3", "?author_id=" + tutor.ID + "&course_id=" + course.ID + "&topic_id=" + root.ID + "&q=Изменённый"} {
		var page domain.CatalogPage[domain.LearningLesson]
		data = call("GET", base+"/lessons"+query, nil, studentCookie, 200)
		decode(data, &page)
		assertNoCatalogKeys(t, data, key1, key2)
		if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != lesson.ID {
			t.Fatalf("filter %s: %+v", query, page)
		}
	}
	for _, query := range []string{"?limit=1&offset=1&course_id=" + course.ID, "?limit=1&offset=999&course_id=" + course.ID} {
		var page domain.CatalogPage[domain.LearningLesson]
		decode(call("GET", base+"/lessons"+query, nil, studentCookie, 200), &page)
		if page.Total != 2 || page.Items == nil || page.Offset == 1 && len(page.Items) != 1 || page.Offset == 999 && len(page.Items) != 0 {
			t.Fatalf("pagination: %+v", page)
		}
	}
	for _, query := range []string{"?limit=0", "?limit=101", "?offset=-1", "?grade=0", "?grade=12", "?difficulty=0", "?difficulty=6", "?status=hidden", "?course_id=not-uuid", "?author_id=bad", "?track=", "?limit=1&limit=2", "?organization_id=" + tutor.OrganizationID} {
		call("GET", base+"/lessons"+query, nil, tutorCookie, 400)
	}
	call("GET", base+"/lessons/not-uuid", nil, tutorCookie, 400)
	call("GET", lp+"/versions/bad", nil, tutorCookie, 400)
	call("GET", lp+"/versions/2147483648", nil, tutorCookie, 400)
	call("GET", lp+"/answer-keys?version=2147483648", nil, tutorCookie, 400)
	tooFar := exInput
	tooFar.Position = 1000001
	call("PUT", ep, tooFar, tutorCookie, 400)
	call("PUT", tp, domain.TopicInput{Title: "Слишком большая позиция", Position: 1000001}, tutorCookie, 400)
	call("GET", lp+"/answer-keys?version=-1", nil, tutorCookie, 400)
	call("GET", lp+"/answer-keys?owner_id="+tutor.ID, nil, tutorCookie, 400)
	call("POST", lp+"/exercises", exInput, studentCookie, 403)
	call("PUT", ep, exInput, otherTutorCookie, 404)
	var foundCourses domain.CatalogPage[domain.Course]
	decode(call("GET", base+"/courses?author_id="+tutor.ID+"&track=olympiad&grade=11&q=математика", nil, studentCookie, 200), &foundCourses)
	if foundCourses.Total != 1 {
		t.Fatalf("course search: %+v", foundCourses)
	}
	call("POST", base+"/topics/"+root.ID+"/archive", map[string]any{}, tutorCookie, 200)
	call("GET", tp, nil, studentCookie, 404)
	decode(call("GET", cp+"/topics", nil, studentCookie, 200), &topics)
	if topics.Total != 0 {
		t.Fatal("archived ancestor leaked through topic list")
	}
	call("GET", lp, nil, studentCookie, 404)
	call("POST", base+"/topics/"+root.ID+"/publish", map[string]any{}, tutorCookie, 200)
	call("GET", lp, nil, studentCookie, 200)
	call("POST", lp+"/archive", map[string]any{}, tutorCookie, 200)
	call("GET", lp, nil, studentCookie, 404)
	call("GET", lp+"/versions/2", nil, studentCookie, 404)
	call("GET", lp, nil, tutorCookie, 200)
	call("POST", lp+"/publish", map[string]any{}, tutorCookie, 200)
	call("POST", cp+"/archive", map[string]any{}, tutorCookie, 200)
	call("GET", cp, nil, studentCookie, 404)
	call("GET", lp, nil, studentCookie, 404)
	call("POST", cp+"/publish", map[string]any{}, tutorCookie, 200)
	courseInput.Visibility = "organization"
	call("PUT", cp, courseInput, tutorCookie, 200)
	call("GET", lp, nil, otherStudentCookie, 200)
	call("GET", lp, nil, otherTutorCookie, 200)
	organizationID, err := platform.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	foreignUserID, err := platform.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into organizations(id,name) values($1,'Catalog foreign test')`, organizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into users(id,organization_id,email,display_name,role) values($1,$2,$3,'Foreign student','student')`, foreignUserID, organizationID, "foreign"+unique+"@example.test"); err != nil {
		t.Fatal(err)
	}
	foreignUser, err := st.GetUser(ctx, foreignUserID)
	if err != nil {
		t.Fatal(err)
	}
	foreignToken, err := st.CreateSession(ctx, foreignUser, "test", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	foreignCookie := &http.Cookie{Name: store.SessionCookieName, Value: foreignToken}
	call("GET", cp, nil, foreignCookie, 404)
	call("GET", tp, nil, foreignCookie, 404)
	call("GET", lp, nil, foreignCookie, 404)
	call("GET", ep, nil, foreignCookie, 404)
	var foreignPage domain.CatalogPage[domain.LearningLesson]
	decode(call("GET", base+"/lessons?course_id="+course.ID, nil, foreignCookie, 200), &foreignPage)
	if foreignPage.Total != 0 || len(foreignPage.Items) != 0 {
		t.Fatal("cross-organization list leaked")
	}
	call("GET", lp+"/versions/0", nil, otherTutorCookie, 404)
	call("GET", lp+"/answer-keys", nil, otherTutorCookie, 404)
	call("POST", lp+"/publish", nil, otherStudentCookie, 403)
	courseInput.Visibility = "linked_students"
	call("PUT", cp, courseInput, tutorCookie, 200)
	if _, err := pool.Exec(ctx, `update relations set status='ended' where tutor_id=$1 and student_id=$2`, tutor.ID, student.ID); err != nil {
		t.Fatal(err)
	}
	call("GET", lp, nil, studentCookie, 404)
	// A known resource ID must not bypass the organization boundary, including for its author.
	wrongOrg := tutor
	wrongOrg.OrganizationID = "00000000-0000-4000-8000-000000000099"
	if _, err := st.GetCourse(ctx, wrongOrg, course.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-org course: %v", err)
	}
	if _, err := st.GetLearningLesson(ctx, wrongOrg, lesson.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-org lesson: %v", err)
	}
	if _, err := st.UpdateLearningLesson(ctx, wrongOrg, lesson.ID, lessonInput); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-org edit: %v", err)
	}
	if _, err := st.GetExerciseAnswerKeys(ctx, wrongOrg, lesson.ID, 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-org keys: %v", err)
	}
	adminActor := tutor
	adminActor.Role = domain.RoleAdmin
	if _, err := st.GetCourse(ctx, adminActor, course.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("implicit admin read bypass: %v", err)
	}
	if _, err := st.UpdateCourse(ctx, adminActor, course.ID, courseInput); !errors.Is(err, ErrForbidden) {
		t.Fatalf("implicit admin write bypass: %v", err)
	}
	// No learning-catalog write created a calendar appointment.
	var appointments int
	if err := pool.QueryRow(ctx, `select count(*) from lessons where tutor_id=$1`, tutor.ID).Scan(&appointments); err != nil || appointments != 0 {
		t.Fatalf("calendar collision: %d %v", appointments, err)
	}
}

func assertNoCatalogKeys(t *testing.T, data []byte, secrets ...string) {
	t.Helper()
	for _, word := range append(secrets, `"answer"`, `"answer_key"`, `"explanation"`) {
		if bytes.Contains(data, []byte(word)) {
			t.Fatalf("private key leaked: %s", data)
		}
	}
}
