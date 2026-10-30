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
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

func TestExerciseEnginePostgres(t *testing.T) {
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
	if err = store.Migrate(ctx, pool, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	st := store.New(pool)
	server := httptest.NewServer(NewServer(config.Config{RequestBodyLimitByte: 1 << 20}, slog.New(slog.NewTextHandler(io.Discard, nil)), st, auth.DevProvider{Enabled: false}))
	defer server.Close()
	client := server.Client()
	call := func(method, path string, body any, cookie *http.Cookie, want int) []byte {
		t.Helper()
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req, err := http.NewRequestWithContext(ctx, method, server.URL+path, bytes.NewReader(data))
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
		if res.StatusCode != want {
			t.Fatalf("%s %s: %d want %d: %s", method, path, res.StatusCode, want, payload)
		}
		return payload
	}
	decode := func(data []byte, out any) {
		t.Helper()
		if err := json.Unmarshal(data, out); err != nil {
			t.Fatal(err)
		}
	}
	newID := func() string {
		t.Helper()
		id, err := platform.NewUUID()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	register := func(role string) (domain.User, *http.Cookie) {
		t.Helper()
		var result struct {
			User domain.User `json:"user"`
		}
		decode(call("POST", "/api/v1/auth/register", map[string]any{"display_name": "engine-" + role, "email": newID() + "@example.test", "password": "password123", "role": role, "accept_terms": true}, nil, 201), &result)
		// Registration creates the cookie; use an independent login to obtain it without a shared jar.
		body, _ := json.Marshal(map[string]string{"email": result.User.Email, "password": "password123"})
		res, err := client.Post(server.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatal("login", res.StatusCode)
		}
		for _, cookie := range res.Cookies() {
			if cookie.Name == store.SessionCookieName {
				return result.User, cookie
			}
		}
		t.Fatal("session missing")
		return result.User, nil
	}
	tutor, tc := register("tutor")
	student, sc := register("student")
	_, outsider := register("student")
	_, otherTutor := register("tutor")
	var invitation struct {
		Token string `json:"token"`
	}
	decode(call("POST", "/api/v1/invitations", map[string]string{"student_email": student.Email}, tc, 201), &invitation)
	call("POST", "/api/v1/invitations/"+invitation.Token+"/accept", map[string]any{}, sc, 201)
	base := "/api/v1/catalog"
	var course domain.Course
	decode(call("POST", base+"/courses", domain.CourseInput{Title: "Engine", Track: "school", GradeMin: 1, GradeMax: 11, Difficulty: 2}, tc, 201), &course)
	var topic domain.Topic
	decode(call("POST", base+"/courses/"+course.ID+"/topics", domain.TopicInput{Title: "Five types"}, tc, 201), &topic)
	var lesson domain.LearningLesson
	input := domain.LearningLessonInput{TopicID: topic.ID, Title: "Five types", Content: "Math", Track: "school", GradeMin: 1, GradeMax: 11, Difficulty: 2}
	decode(call("POST", base+"/courses/"+course.ID+"/lessons", input, tc, 201), &lesson)
	lp := base + "/lessons/" + lesson.ID
	keys := []string{"b", `["a","c"]`, "1/2", "равные углы", ""}
	kinds := []string{"single_choice", "multiple_choice", "numeric", "text", "long_text"}
	exercises := make([]domain.Exercise, 5)
	for i, kind := range kinds {
		options := []string{}
		if strings.Contains(kind, "choice") {
			options = []string{"a", "b", "c"}
		}
		ex := domain.ExerciseInput{Kind: kind, Prompt: "Prompt " + kind, Options: options, Difficulty: 2, Position: i, AnswerKey: domain.AnswerKeyInput{Answer: keys[i], Explanation: "PRIVATE_KEY_EXPLANATION"}}
		decode(call("POST", lp+"/exercises", ex, tc, 201), &exercises[i])
	}
	call("POST", base+"/courses/"+course.ID+"/publish", map[string]any{}, tc, 200)
	call("POST", base+"/topics/"+topic.ID+"/publish", map[string]any{}, tc, 200)
	decode(call("POST", lp+"/publish", map[string]any{}, tc, 200), &lesson)
	if len(lesson.Version.Exercises) != 5 {
		t.Fatal("five types not published")
	}
	value := func(s string) domain.ExerciseAnswer { return domain.ExerciseAnswer{Value: &s} }
	answers := []domain.ExerciseAnswer{{Selected: []string{"b"}}, {Selected: []string{"c", "a"}}, value(" +2 / 4 "), value("  РАВНЫЕ\t\nуглы "), value("Подробное решение ученика.")}
	attempts := make([]domain.ExerciseAttempt, 5)
	for i, answer := range answers {
		path := lp + "/exercises/" + exercises[i].ID + "/attempts"
		in := domain.ExerciseAttemptInput{VersionID: lesson.Version.ID, RequestID: newID(), Answer: answer}
		if i == 0 {
			call("POST", path, in, nil, 401)
			call("POST", path, in, tc, 403)
			call("POST", path, in, outsider, 404)
		}
		payload := call("POST", path, in, sc, 201)
		decode(payload, &attempts[i])
		want := "correct"
		if i == 4 {
			want = "needs_review"
		}
		if attempts[i].Status != want {
			t.Fatalf("%s: %s", kinds[i], attempts[i].Status)
		}
		if strings.Contains(string(payload), "PRIVATE_KEY") || strings.Contains(string(payload), "answer_key") {
			t.Fatal("leaked key")
		}
		var replay domain.ExerciseAttempt
		decode(call("POST", path, in, sc, 201), &replay)
		if replay.ID != attempts[i].ID {
			t.Fatal("retry duplicated attempt")
		}
		in.Answer = value("different")
		call("POST", path, in, sc, 409)
	}
	for _, bad := range []domain.ExerciseAnswer{value("1/0"), value("--2"), value("1 2"), value("1e3"), {Selected: []string{"b"}}} {
		call("POST", lp+"/exercises/"+exercises[2].ID+"/attempts", domain.ExerciseAttemptInput{VersionID: lesson.Version.ID, RequestID: newID(), Answer: bad}, sc, 400)
	}
	call("POST", lp+"/exercises/"+exercises[1].ID+"/attempts", domain.ExerciseAttemptInput{VersionID: lesson.Version.ID, RequestID: newID(), Answer: domain.ExerciseAnswer{Selected: []string{"a", "a"}}}, sc, 400)
	call("POST", lp+"/exercises/"+exercises[0].ID+"/attempts", map[string]any{"version_id": lesson.Version.ID, "request_id": newID(), "answer": map[string]any{"selected": []string{"b"}, "status": "correct"}}, sc, 400)
	var wrong domain.ExerciseAttempt
	decode(call("POST", lp+"/exercises/"+exercises[2].ID+"/attempts", domain.ExerciseAttemptInput{VersionID: lesson.Version.ID, RequestID: newID(), Answer: value("-1/2")}, sc, 201), &wrong)
	if wrong.Status != "incorrect" {
		t.Fatal("incorrect not recorded")
	}
	var page domain.CatalogPage[domain.ExerciseAttempt]
	decode(call("GET", lp+"/attempts?limit=2&offset=2", nil, tc, 200), &page)
	if page.Total != 6 || len(page.Items) != 2 {
		t.Fatal("pagination", page)
	}
	call("GET", lp+"/attempts", nil, outsider, 404)
	call("GET", lp+"/attempts", nil, otherTutor, 404)
	decode(call("GET", lp+"/attempts?offset=999", nil, sc, 200), &page)
	if page.Total != 6 || len(page.Items) != 0 {
		t.Fatal("empty page")
	}
	var count int
	if err = pool.QueryRow(ctx, `select count(*) from notifications where user_id=$1 and idempotency_key=$2`, tutor.ID, "exercise-attempt:"+attempts[4].ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("teacher notification", count, err)
	}
	reviewPath := lp + "/attempts/" + attempts[4].ID + "/review"
	review := domain.ExerciseReviewInput{Status: "correct", Feedback: "Хорошее обоснование."}
	call("PUT", reviewPath, review, sc, 403)
	call("PUT", reviewPath, review, otherTutor, 404)
	var reviewed domain.ExerciseAttempt
	decode(call("PUT", reviewPath, review, tc, 200), &reviewed)
	if reviewed.ReviewedAt == nil || reviewed.Status != "correct" {
		t.Fatal("review not persisted")
	}
	call("PUT", reviewPath, review, tc, 200)
	call("PUT", reviewPath, domain.ExerciseReviewInput{Status: "incorrect"}, tc, 409)
	call("PUT", lp+"/attempts/"+attempts[0].ID+"/review", review, tc, 422)
	if err = pool.QueryRow(ctx, `select count(*) from notifications where user_id=$1 and idempotency_key=$2`, student.ID, "exercise-review:"+attempts[4].ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("review notification", count, err)
	}
	// A repeated concurrent request creates one row, even when the first response was lost.
	requestID := newID()
	var wg sync.WaitGroup
	ids := make(chan string, 4)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := st.SubmitExerciseAnswer(ctx, student, lesson.ID, exercises[2].ID, domain.ExerciseAttemptInput{VersionID: lesson.Version.ID, RequestID: requestID, Answer: value(".5")})
			if err != nil {
				t.Error(err)
				return
			}
			ids <- out.ID
		}()
	}
	wg.Wait()
	close(ids)
	first := ""
	for id := range ids {
		if first != "" && first != id {
			t.Error("concurrent duplicate")
		}
		first = id
	}
	if first == "" {
		t.Fatal("no concurrent result")
	}
	oldVersion := lesson.Version.ID
	call("PUT", lp+"/exercises/"+exercises[2].ID, domain.ExerciseInput{Kind: "numeric", Prompt: "New numeric prompt", Difficulty: 2, Position: 2, AnswerKey: domain.AnswerKeyInput{Answer: "2/3"}}, tc, 200)
	var unchanged domain.ExerciseAttempt
	decode(call("POST", lp+"/exercises/"+exercises[2].ID+"/attempts", domain.ExerciseAttemptInput{VersionID: oldVersion, RequestID: newID(), Answer: value(".5")}, sc, 201), &unchanged)
	if unchanged.Status != "correct" {
		t.Fatal("draft key changed published checking")
	}
	input.Content = "New content"
	call("PUT", lp, input, tc, 200)
	decode(call("POST", lp+"/publish", map[string]any{}, tc, 200), &lesson)
	call("POST", lp+"/exercises/"+exercises[2].ID+"/attempts", domain.ExerciseAttemptInput{VersionID: oldVersion, RequestID: newID(), Answer: value(".5")}, sc, 409)
	call("POST", lp+"/exercises/"+exercises[2].ID+"/attempts", domain.ExerciseAttemptInput{VersionID: oldVersion, RequestID: requestID, Answer: value(".5")}, sc, 201)
	var newAttempt domain.ExerciseAttempt
	decode(call("POST", lp+"/exercises/"+exercises[2].ID+"/attempts", domain.ExerciseAttemptInput{VersionID: lesson.Version.ID, RequestID: newID(), Answer: value(".5")}, sc, 201), &newAttempt)
	if newAttempt.Status != "incorrect" {
		t.Fatal("new version ignored new key")
	}
	wrongOrg := student
	wrongOrg.OrganizationID = newID()
	if _, err = st.ListExerciseAttempts(ctx, wrongOrg, lesson.ID, domain.CatalogFilter{}); !errors.Is(err, ErrNotFound) {
		t.Fatal("tenant access")
	}
	if _, err = st.SubmitExerciseAnswer(ctx, wrongOrg, lesson.ID, exercises[2].ID, domain.ExerciseAttemptInput{VersionID: lesson.Version.ID, RequestID: newID(), Answer: value("2/3")}); !errors.Is(err, ErrNotFound) {
		t.Fatal("tenant submit", err)
	}
	wrongTeacher := tutor
	wrongTeacher.OrganizationID = wrongOrg.OrganizationID
	if _, err = st.ReviewExerciseAttempt(ctx, wrongTeacher, lesson.ID, attempts[4].ID, review); !errors.Is(err, ErrNotFound) {
		t.Fatal("tenant review", err)
	}
	_, peerCookie := register("student")
	var peer struct {
		User domain.User `json:"user"`
	}
	decode(call("GET", "/api/v1/me", nil, peerCookie, 200), &peer.User)
	decode(call("POST", "/api/v1/invitations", map[string]string{"student_email": peer.User.Email}, tc, 201), &invitation)
	call("POST", "/api/v1/invitations/"+invitation.Token+"/accept", map[string]any{}, peerCookie, 201)
	decode(call("GET", lp+"/attempts", nil, peerCookie, 200), &page)
	if page.Total != 0 || len(page.Items) != 0 {
		t.Fatal("another student's attempts leaked")
	}
	publicInput := course.CourseInput
	publicInput.Visibility = "organization"
	call("PUT", base+"/courses/"+course.ID, publicInput, tc, 200)
	call("GET", lp, nil, otherTutor, 200)
	call("GET", lp+"/attempts", nil, otherTutor, 403)
	call("POST", base+"/topics/"+topic.ID+"/archive", map[string]any{}, tc, 200)
	call("GET", lp+"/attempts", nil, sc, 404)
	call("POST", lp+"/exercises/"+exercises[2].ID+"/attempts", domain.ExerciseAttemptInput{VersionID: oldVersion, RequestID: requestID, Answer: value(".5")}, sc, 404)
	testVisualHTTP(t, call, decode, tc, sc)
	testLearningHTTP(t, call, decode, tc, sc, peerCookie, otherTutor)
	testPlayerHTTP(t, call, decode, tc, sc, peerCookie)
	testMaterialAssignmentsHTTP(t, call, decode, tc, sc, outsider, otherTutor, tutor, student, pool)
}
