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
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

func TestStudySessionsPostgres(t *testing.T) {
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
	server := httptest.NewServer(NewServer(config.Config{RequestBodyLimitByte: 1 << 20}, slog.New(slog.NewTextHandler(io.Discard, nil)), store.New(pool), auth.DevProvider{}))
	defer server.Close()
	id := func() string {
		v, e := platform.NewUUID()
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	call := func(method, path string, body any, cookie *http.Cookie, want int, out any) []byte {
		t.Helper()
		data, e := json.Marshal(body)
		if e != nil {
			t.Fatal(e)
		}
		req, e := http.NewRequest(method, server.URL+path, bytes.NewReader(data))
		if e != nil {
			t.Fatal(e)
		}
		req.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			req.AddCookie(cookie)
		}
		res, e := server.Client().Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		raw, e := io.ReadAll(res.Body)
		if e != nil {
			t.Fatal(e)
		}
		if res.StatusCode != want {
			t.Fatalf("%s %s = %d want %d: %s", method, path, res.StatusCode, want, raw)
		}
		if out != nil {
			if e = json.Unmarshal(raw, out); e != nil {
				t.Fatal(e)
			}
		}
		return raw
	}
	register := func(role string) (domain.User, *http.Cookie) {
		t.Helper()
		data, _ := json.Marshal(map[string]any{"email": id() + "@example.test", "display_name": "Study " + role, "password": "StudyTesting123!", "role": role, "accept_terms": true})
		r, e := server.Client().Post(server.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(data))
		if e != nil {
			t.Fatal(e)
		}
		defer r.Body.Close()
		var out struct {
			User domain.User `json:"user"`
		}
		if r.StatusCode != 201 {
			t.Fatal("registration", r.StatusCode)
		}
		if e = json.NewDecoder(r.Body).Decode(&out); e != nil {
			t.Fatal(e)
		}
		for _, c := range r.Cookies() {
			if c.Name == store.SessionCookieName {
				return out.User, c
			}
		}
		t.Fatal("cookie")
		return domain.User{}, nil
	}
	tutor, tc := register("tutor")
	_, other := register("tutor")
	student, sc := register("student")
	_, peer := register("student")
	var invite struct {
		Token string `json:"token"`
	}
	call("POST", "/api/v1/invitations", map[string]any{"student_email": student.Email}, tc, 201, &invite)
	call("POST", "/api/v1/invitations/"+invite.Token+"/accept", map[string]any{}, sc, 201, nil)
	var course domain.Course
	call("POST", "/api/v1/catalog/courses", domain.CourseInput{Title: "Own study demo", Track: "school", GradeMin: 5, GradeMax: 11, Difficulty: 1, Visibility: "linked_students"}, tc, 201, &course)
	cp := "/api/v1/catalog/courses/" + course.ID
	var topic domain.Topic
	call("POST", cp+"/topics", domain.TopicInput{Title: "Числа"}, tc, 201, &topic)
	call("POST", cp+"/publish", map[string]any{}, tc, 200, nil)
	call("POST", "/api/v1/catalog/topics/"+topic.ID+"/publish", map[string]any{}, tc, 200, nil)
	ids := []string{}
	var source domain.LearningLesson
	for i := 0; i < 3; i++ {
		var lesson domain.LearningLesson
		call("POST", cp+"/lessons", domain.LearningLessonInput{TopicID: topic.ID, Title: "Own exercise", Track: "school", GradeMin: 5, GradeMax: 11, Difficulty: 1}, tc, 201, &lesson)
		lp := "/api/v1/catalog/lessons/" + lesson.ID
		kind, key, prompt, number := "numeric", "7", "Вычислите 3 + 4.", 7
		if i == 2 {
			kind, key, prompt, number = "long_text", "[manual]", "Объясните решение.", 20
		}
		var ex domain.Exercise
		call("POST", lp+"/exercises", domain.ExerciseInput{Kind: kind, Prompt: prompt, Difficulty: 1, AnswerKey: domain.AnswerKeyInput{Answer: key, Explanation: "PRIVATE_STUDY_SOLUTION", Hints: []string{"PRIVATE_STUDY_HINT"}}}, tc, 201, &ex)
		call("POST", lp+"/publish", map[string]any{}, tc, 200, &lesson)
		source = lesson
		var bank domain.BankExercise
		call("POST", "/api/v1/exercise-bank", domain.BankExerciseInput{LessonID: lesson.ID, ExerciseID: ex.ID, BankMetadata: domain.BankMetadata{Grade: 9, Direction: "oge", Topic: "Числа", ExamNumber: &number, Source: "Own demonstration", Attribution: tutor.DisplayName, RightsStatus: "own", License: "Permission to use"}}, tc, 201, &bank)
		ids = append(ids, bank.ID)
	}
	var collection domain.BankCollection
	call("POST", "/api/v1/bank-collections", domain.BankCollectionInput{RequestID: id(), Title: "Study collection", CourseID: course.ID, TopicID: topic.ID, BankIDs: ids}, tc, 201, &collection)
	in := domain.ExamSpecificationInput{RequestID: id(), CollectionID: collection.ID, Title: "Own mock exam", Direction: "oge", Year: 2026, DurationSeconds: 60, Instructions: "Own criteria, not official scores", Slots: []domain.ExamSlot{{Number: 7, Count: 2, Points: 2}, {Number: 20, Count: 1, Points: 3}}}
	var spec domain.ExamSpecification
	call("POST", "/api/v1/exam-specifications", in, sc, 403, nil)
	call("POST", "/api/v1/exam-specifications", in, tc, 201, &spec)
	var replay domain.ExamSpecification
	call("POST", "/api/v1/exam-specifications", in, tc, 201, &replay)
	if spec.ID != replay.ID {
		t.Fatal("spec replay")
	}
	bad := in
	bad.Title = "changed"
	call("POST", "/api/v1/exam-specifications", bad, tc, 409, nil)
	sp := "/api/v1/exam-specifications/" + spec.ID
	call("GET", sp, nil, sc, 404, nil)
	call("GET", sp, nil, other, 404, nil)
	call("POST", sp+"/publish", map[string]any{}, tc, 200, nil)
	call("GET", sp, nil, peer, 404, nil)
	var specs domain.CatalogPage[domain.ExamSpecification]
	call("GET", "/api/v1/exam-specifications?limit=1", nil, sc, 200, &specs)
	if specs.Total != 1 || specs.Items[0].ExerciseCount != 3 {
		t.Fatal("spec visibility/count", specs)
	}
	start := domain.StudyStartInput{RequestID: id(), SpecificationID: spec.ID, Mode: "exam", Seed: "repeatable"}
	var session, again domain.StudySession
	call("POST", "/api/v1/study-sessions", start, tc, 403, nil)
	call("POST", "/api/v1/study-sessions", start, sc, 201, &session)
	call("POST", "/api/v1/study-sessions", start, sc, 201, &again)
	if again.ID != session.ID || session.DeadlineAt == nil || session.MaxPoints != 7 {
		t.Fatal("start/deadline/replay", session)
	}
	sessionPath := "/api/v1/study-sessions/" + session.ID
	call("GET", sessionPath, nil, peer, 404, nil)
	call("GET", sessionPath, nil, tc, 403, nil)
	lp := "/api/v1/catalog/lessons/" + session.State.Attempt.LessonID
	runPath := lp + "/learning-attempts/" + session.State.Attempt.ID
	var latest domain.ExerciseAttempt
	for i, ex := range session.State.Version.Exercises {
		call("POST", runPath+"/navigate", domain.LearningNavigateInput{RequestID: id(), Step: "exercise:" + ex.ID}, sc, 200, nil)
		value := "7"
		if i == 1 {
			value = "0"
		}
		if ex.Kind == "long_text" {
			value = "Моё подробное решение"
		}
		answer := domain.ExerciseAttemptInput{RequestID: id(), VersionID: session.State.Version.ID, LearningAttemptID: session.State.Attempt.ID, Answer: domain.ExerciseAnswer{Value: &value}}
		path := lp + "/exercises/" + ex.ID + "/attempts"
		raw := call("POST", path, answer, sc, 201, &latest)
		if latest.Status != "submitted" || strings.Contains(string(raw), "PRIVATE_") {
			t.Fatal("exam result leak")
		}
		var saved domain.ExerciseAttempt
		call("POST", path, answer, sc, 201, &saved)
		if saved.ID != latest.ID {
			t.Fatal("answer replay")
		}
		if ex.Kind == "numeric" && i == 0 {
			answer.RequestID = id()
			value = "7"
			answer.Answer = domain.ExerciseAnswer{Value: &value}
			call("POST", path, answer, sc, 201, nil)
		}
	}
	call("GET", sessionPath, nil, sc, 200, &again)
	if again.Points != 0 || again.State.Attempt.PendingReview != 0 {
		t.Fatal("hidden exam scores")
	}
	var analytics domain.AnalyticsReport
	analyticsPath := "/api/v1/learning-analytics/" + student.ID
	call("GET", analyticsPath, nil, sc, 200, &analytics)
	if analytics.Summary.Attempts != 0 || analytics.Summary.Accuracy != nil || analytics.Exams.Total != 1 || analytics.Exams.Items[0].Points != nil {
		t.Fatal("active exam analytics leaked", analytics)
	}
	var reviewPlan domain.ReviewPlan
	call("GET", "/api/v1/learning-reviews/"+student.ID+"?mode=all", nil, sc, 200, &reviewPlan)
	if reviewPlan.Items.Total != 0 || reviewPlan.Pending != 0 || reviewPlan.MistakeTotal != 0 || reviewPlan.Unanswered != 0 {
		t.Fatal("active exam review plan leaked", reviewPlan)
	}
	var queue domain.CatalogPage[domain.StudyReview]
	call("GET", "/api/v1/study-reviews", nil, tc, 200, &queue)
	if queue.Total != 0 {
		t.Fatal("early review")
	}
	var state domain.LearningState
	call("POST", runPath+"/complete", domain.LearningActionInput{RequestID: id()}, sc, 200, &state)
	call("GET", sessionPath, nil, sc, 200, &again)
	if again.Points != 2 || again.State.Attempt.PendingReview != 1 {
		t.Fatal("score", again.Points, again.State.Attempt)
	}
	call("GET", "/api/v1/study-reviews", nil, tc, 200, &queue)
	if queue.Total != 1 {
		t.Fatal("review queue", queue)
	}
	call("GET", "/api/v1/study-reviews", nil, sc, 403, nil)
	call("GET", "/api/v1/study-reviews", nil, other, 200, &queue)
	if queue.Total != 0 {
		t.Fatal("foreign review")
	}
	reviewPath := lp + "/attempts/" + latest.ID + "/review"
	call("PUT", reviewPath, domain.ExerciseReviewInput{Status: "correct", Feedback: "Хорошее решение"}, other, 404, nil)
	call("PUT", reviewPath, domain.ExerciseReviewInput{Status: "correct", Feedback: "Хорошее решение"}, tc, 200, nil)
	call("GET", sessionPath, nil, sc, 200, &again)
	if again.Points != 5 || again.State.Attempt.PendingReview != 0 {
		t.Fatal("manual score", again.Points)
	}
	call("GET", analyticsPath, nil, tc, 200, &analytics)
	if analytics.Summary.Graded != 3 || analytics.Summary.Correct != 2 || analytics.Summary.Pending != 0 || len(analytics.Topics.Items) != 1 || analytics.Topics.Items[0].Topic != "Числа" || analytics.Exams.Items[0].Points == nil || *analytics.Exams.Items[0].Points != 5 {
		t.Fatal("completed bank exam analytics", analytics)
	}
	start.RequestID = id()
	call("POST", "/api/v1/study-sessions", start, sc, 201, &again)
	for i, ex := range session.State.Version.Exercises {
		if again.State.Version.Exercises[i].ID != ex.ID {
			t.Fatal("not reproducible")
		}
	}
	// Only this isolated fixture is moved past its deadline; normal updates remain forbidden.
	if _, err = pool.Exec(t.Context(), `update study_sessions set seed='forged' where id=$1`, again.ID); err == nil {
		t.Fatal("mutable session")
	}
	tx, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(t.Context())
	if _, err = tx.Exec(t.Context(), `alter table study_sessions disable trigger immutable_study_session`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(t.Context(), `update study_sessions set deadline_at=clock_timestamp()-interval '1 second' where id=$1`, again.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(t.Context(), `alter table study_sessions enable trigger immutable_study_session`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	ex := again.State.Version.Exercises[0]
	value := "7"
	call("POST", lp+"/exercises/"+ex.ID+"/attempts", domain.ExerciseAttemptInput{RequestID: id(), VersionID: again.State.Version.ID, LearningAttemptID: again.State.Attempt.ID, Answer: domain.ExerciseAnswer{Value: &value}}, sc, 422, nil)
	call("GET", "/api/v1/study-reviews", nil, tc, 200, nil)
	call("GET", "/api/v1/study-sessions/"+again.ID, nil, sc, 200, &again)
	if again.State.Attempt.Status != "completed" || again.Points != 0 {
		t.Fatal("expiry did not finalize")
	}
	practice := domain.StudyStartInput{RequestID: id(), SpecificationID: spec.ID, Mode: "practice", Seed: "retry-errors", Count: 1, RepeatSessionID: session.ID}
	call("POST", "/api/v1/study-sessions", practice, sc, 201, &again)
	if len(again.State.Version.Exercises) != 1 {
		t.Fatal("repeat selection")
	}
	call("GET", lp+"/learning-attempts/"+again.State.Attempt.ID, nil, peer, 404, nil)
	call("GET", "/api/v1/catalog/lessons/"+source.ID+"/answer-keys", nil, sc, 403, nil)
	in.RequestID = id()
	in.ParentID = spec.ID
	in.Title = "Version two"
	call("POST", "/api/v1/exam-specifications", in, tc, 201, &replay)
	if replay.Version != 2 {
		t.Fatal("versioning")
	}
	call("GET", sessionPath, nil, sc, 200, &session)
	if session.Specification.Title != "Own mock exam" {
		t.Fatal("changed pinned config")
	}
	if _, err = pool.Exec(t.Context(), `update exam_specifications set duration_seconds=120 where id=$1`, spec.ID); err == nil {
		t.Fatal("mutable spec")
	}
}
