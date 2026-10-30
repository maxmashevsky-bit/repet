package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

func TestLearningReviewsPostgres(t *testing.T) {
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
	srv := httptest.NewServer(NewServer(config.Config{RequestBodyLimitByte: 1 << 20}, slog.New(slog.NewTextHandler(io.Discard, nil)), store.New(pool), auth.DevProvider{}))
	defer srv.Close()
	id := func() string {
		v, e := platform.NewUUID()
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	client := func() *http.Client {
		j, e := cookiejar.New(nil)
		if e != nil {
			t.Fatal(e)
		}
		return &http.Client{Transport: srv.Client().Transport, Jar: j}
	}
	call := func(c *http.Client, method, path string, body any, want int, out any) []byte {
		t.Helper()
		data, e := json.Marshal(body)
		if e != nil {
			t.Fatal(e)
		}
		req, e := http.NewRequest(method, srv.URL+path, bytes.NewReader(data))
		if e != nil {
			t.Fatal(e)
		}
		req.Header.Set("Content-Type", "application/json")
		res, e := c.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		raw, e := io.ReadAll(res.Body)
		if e != nil {
			t.Fatal(e)
		}
		if res.StatusCode != want {
			t.Fatalf("%s %s=%d want%d: %s", method, path, res.StatusCode, want, raw)
		}
		if out != nil {
			if e = json.Unmarshal(raw, out); e != nil {
				t.Fatal(e)
			}
		}
		return raw
	}
	register := func(role string) (*http.Client, domain.User) {
		c := client()
		var out struct {
			User domain.User `json:"user"`
		}
		call(c, "POST", "/api/v1/auth/register", map[string]any{"display_name": "Review " + role, "email": id() + "@example.test", "role": role, "password": "ReviewTesting123!", "accept_terms": true}, 201, &out)
		return c, out.User
	}
	a, author := register("tutor")
	b, tutor := register("tutor")
	sc, student := register("student")
	peer, _ := register("student")
	for _, tc := range []*http.Client{a, b} {
		var inv struct {
			Token string `json:"token"`
		}
		call(tc, "POST", "/api/v1/invitations", map[string]any{"student_email": student.Email}, 201, &inv)
		call(sc, "POST", "/api/v1/invitations/"+inv.Token+"/accept", map[string]any{}, 201, nil)
	}
	course := func(c *http.Client, visibility string) (domain.Course, domain.Topic) {
		t.Helper()
		var out domain.Course
		call(c, "POST", "/api/v1/catalog/courses", domain.CourseInput{Title: "Own review fixtures", Visibility: visibility, Track: "school", GradeMin: 5, GradeMax: 11, Difficulty: 1}, 201, &out)
		cp := "/api/v1/catalog/courses/" + out.ID
		var topic domain.Topic
		call(c, "POST", cp+"/topics", domain.TopicInput{Title: "Курс"}, 201, &topic)
		call(c, "POST", cp+"/publish", map[string]any{}, 200, nil)
		call(c, "POST", "/api/v1/catalog/topics/"+topic.ID+"/publish", map[string]any{}, 200, nil)
		return out, topic
	}
	c, tp := course(a, "organization")
	var lesson domain.LearningLesson
	call(a, "POST", "/api/v1/catalog/courses/"+c.ID+"/lessons", domain.LearningLessonInput{TopicID: tp.ID, Title: "Own source", Track: "school", GradeMin: 5, GradeMax: 11, Difficulty: 1}, 201, &lesson)
	lp := "/api/v1/catalog/lessons/" + lesson.ID
	original := make([]domain.Exercise, 3)
	bankIDs := []string{}
	for i := range original {
		kind, key := "numeric", "7"
		if i == 2 {
			kind = "long_text"
			key = "[manual]"
		}
		call(a, "POST", lp+"/exercises", domain.ExerciseInput{Kind: kind, Prompt: "Own review exercise", Difficulty: 1, Position: i, AnswerKey: domain.AnswerKeyInput{Answer: key, Explanation: "PRIVATE_REVIEW_KEY", Hints: []string{"Review hint"}}}, 201, &original[i])
	}
	call(a, "POST", lp+"/publish", map[string]any{}, 200, &lesson)
	for _, e := range original {
		num := 7
		var bank domain.BankExercise
		call(a, "POST", "/api/v1/exercise-bank", domain.BankExerciseInput{LessonID: lesson.ID, ExerciseID: e.ID, BankMetadata: domain.BankMetadata{Grade: 9, Direction: "oge", Topic: "Дроби", ExamNumber: &num, Source: "Own test", Attribution: author.DisplayName, RightsStatus: "own", License: "Own permission"}}, 201, &bank)
		bankIDs = append(bankIDs, bank.ID)
	}
	dest, topic := course(b, "linked_students")
	var collection domain.BankCollection
	call(b, "POST", "/api/v1/bank-collections", domain.BankCollectionInput{RequestID: id(), CourseID: dest.ID, TopicID: topic.ID, Title: "Private subset", BankIDs: bankIDs}, 201, &collection)
	assign := func(tc *http.Client, lessonID string, keys []string) domain.LearningState {
		t.Helper()
		var item domain.Assignment
		call(tc, "POST", "/api/v1/assignments/material", domain.MaterialAssignmentInput{RequestID: id(), StudentID: student.ID, LessonID: lessonID, ExerciseIDs: keys, DueAt: time.Now().Add(24 * time.Hour), Mode: "practice"}, 201, &item)
		var state domain.LearningState
		call(sc, "POST", "/api/v1/assignments/"+item.ID+"/learning-attempt", domain.LearningActionInput{RequestID: id()}, 201, &state)
		return state
	}
	answer := func(state domain.LearningState, key, value string) domain.ExerciseAttempt {
		t.Helper()
		var out domain.ExerciseAttempt
		call(sc, "POST", "/api/v1/catalog/lessons/"+state.Attempt.LessonID+"/exercises/"+key+"/attempts", domain.ExerciseAttemptInput{RequestID: id(), VersionID: state.Version.ID, LearningAttemptID: state.Attempt.ID, Answer: domain.ExerciseAnswer{Value: &value}}, 201, &out)
		return out
	}
	finish := func(state domain.LearningState) {
		t.Helper()
		call(sc, "POST", "/api/v1/catalog/lessons/"+state.Attempt.LessonID+"/learning-attempts/"+state.Attempt.ID+"/complete", domain.LearningActionInput{RequestID: id()}, 200, nil)
	}
	root := assign(b, collection.LessonID, []string{collection.ExerciseIDs[0], collection.ExerciseIDs[2]})
	answer(root, collection.ExerciseIDs[0], "0")
	manual := answer(root, collection.ExerciseIDs[2], "Доказательство")
	planPath := "/api/v1/learning-reviews/" + student.ID
	var plan domain.ReviewPlan
	if _, err = store.New(pool).LearningReviewPlan(t.Context(), student, student.ID, domain.ReviewFilter{Mode: "all"}); err != nil {
		t.Fatal("direct review plan", err)
	}
	call(sc, "GET", planPath+"?mode=all", nil, 200, &plan)
	if plan.Items.Total != 0 {
		t.Fatal("unfinished source revealed", plan)
	}
	finish(root)
	raw := call(sc, "GET", planPath+"?mode=errors", nil, 200, &plan)
	if plan.Items.Total != 1 || plan.Pending != 1 || plan.Unanswered != 0 || plan.Items.Items[0].Topic != "Дроби" || strings.Contains(string(raw), "PRIVATE_REVIEW_KEY") {
		t.Fatal("private source scope", plan)
	}
	call(peer, "GET", planPath, nil, 404, nil)
	call(sc, "GET", planPath+"?mode=secret", nil, 400, nil)
	for _, query := range []string{"active_offset=-1", "active_offset=100001", "mode=all&mode=due", "owner_id=1"} {
		call(sc, "GET", planPath+"?"+query, nil, 400, nil)
	}
	in := domain.ReviewStartInput{RequestID: id(), SourceRunID: root.Attempt.ID, Mode: "errors", Count: 10, TopicKey: plan.Items.Items[0].TopicKey}
	var review, replay domain.ReviewSession
	call(b, "POST", "/api/v1/learning-review-sessions", in, 403, nil)
	call(peer, "POST", "/api/v1/learning-review-sessions", in, 404, nil)
	singleConfig, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	singleConfig.MaxConns = 1
	single, e := pgxpool.NewWithConfig(t.Context(), singleConfig)
	if e != nil {
		t.Fatal(e)
	}
	limited, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	_, e = store.New(single).StartLearningReview(limited, student, in)
	cancel()
	single.Close()
	if e != nil {
		t.Fatal("review must work with a single DB connection", e)
	}
	call(sc, "POST", "/api/v1/learning-review-sessions", in, 201, &review)
	call(sc, "POST", "/api/v1/learning-review-sessions", in, 201, &replay)
	if replay.ID != review.ID || len(review.State.Version.Exercises) != 1 || review.State.Attempt.TotalExercises != 1 || review.State.Version.ID != root.Version.ID {
		t.Fatal("pinned partial review", review)
	}
	var page domain.ReviewPlan
	call(sc, "GET", planPath+"?limit=1&offset=1&active_offset=1", nil, 200, &page)
	if page.Items.Total != 1 || len(page.Items.Items) != 0 || page.Items.Offset != 1 || page.Active.Total != 1 || len(page.Active.Items) != 0 || page.Active.Offset != 1 || page.WeeklyCompleted != 0 {
		t.Fatal("independent pagination", page)
	}
	changed := in
	changed.Count = 1
	call(sc, "POST", "/api/v1/learning-review-sessions", changed, 409, nil)
	quizPath := "/api/v1/learning-review-sessions/" + review.ID
	call(peer, "GET", quizPath, nil, 404, nil)
	quizLesson := "/api/v1/catalog/lessons/" + collection.LessonID
	value := "7"
	for _, key := range collection.ExerciseIDs[1:] {
		call(sc, "POST", quizLesson+"/exercises/"+key+"/attempts", domain.ExerciseAttemptInput{RequestID: id(), VersionID: root.Version.ID, LearningAttemptID: review.ID, Answer: domain.ExerciseAnswer{Value: &value}}, 404, nil)
	}
	if _, err = pool.Exec(t.Context(), `insert into exercise_attempts(exercise_id,student_id,request_id,answer,status,learning_attempt_id,attempt_number)select e.id,$1,$2,'{"value":"7"}'::jsonb,'correct',$3,coalesce((select max(attempt_number) from exercise_attempts where student_id=$1 and exercise_id=e.id),0)+1 from exercises e where e.version_id=$4 and e.exercise_key=$5`, student.ID, id(), review.ID, root.Version.ID, collection.ExerciseIDs[1]); err == nil {
		t.Fatal("database accepted an exercise outside the review grant")
	}
	call(sc, "GET", quizPath, nil, 200, &replay)
	answer(review.State, collection.ExerciseIDs[0], "7")
	finish(review.State)
	call(sc, "GET", planPath+"?mode=errors", nil, 200, &plan)
	if plan.Items.Total != 0 || plan.WeeklyCompleted != 1 {
		t.Fatal("corrected logical task still wrong", plan)
	}
	call(b, "PUT", quizLesson+"/attempts/"+manual.ID+"/review", domain.ExerciseReviewInput{Status: "incorrect", Feedback: "Повторим"}, 200, nil)
	in.RequestID = id()
	call(sc, "POST", "/api/v1/learning-review-sessions", in, 201, &review)
	if len(review.State.Version.Exercises) != 1 || review.State.Version.Exercises[0].Kind != "long_text" {
		t.Fatal(review)
	}
	manual = answer(review.State, collection.ExerciseIDs[2], "Новое доказательство")
	var queue domain.CatalogPage[domain.StudyReview]
	call(b, "GET", "/api/v1/learning-review-checks", nil, 200, &queue)
	if queue.Total != 1 {
		t.Fatal(queue)
	}
	call(a, "GET", "/api/v1/learning-review-checks", nil, 200, &queue)
	if queue.Total != 0 {
		t.Fatal("author of shared source saw private review")
	}
	call(a, "PUT", quizLesson+"/attempts/"+manual.ID+"/review", domain.ExerciseReviewInput{Status: "correct"}, 404, nil)
	call(b, "PUT", quizLesson+"/attempts/"+manual.ID+"/review", domain.ExerciseReviewInput{Status: "correct"}, 200, nil)
	finish(review.State)
	call(sc, "GET", planPath+"?mode=all", nil, 200, &plan)
	if plan.Items.Total != 2 || plan.Pending != 0 || plan.DueTotal != 0 {
		t.Fatal("review interval or source scope", plan)
	}
	for _, item := range plan.Items.Items {
		if item.CleanDays != 1 || item.DueAt.Sub(plan.ServerTime) < 23*time.Hour {
			t.Fatal("first recall must be scheduled a day later", item)
		}
	}
	// Two different physical snapshots and the original source share one XP identity.
	old := assign(b, collection.LessonID, []string{collection.ExerciseIDs[0]})
	answer(old, collection.ExerciseIDs[0], "7")
	finish(old)
	if _, err = pool.Exec(t.Context(), `update exercise_attempts set created_at=$2 where learning_attempt_id=$1`, old.Attempt.ID, time.Now().Add(-48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	newRun := assign(b, lesson.ID, []string{original[0].ID})
	answer(newRun, original[0].ID, "7")
	finish(newRun)
	var rewards domain.RewardSummary
	call(sc, "GET", "/api/v1/learning-rewards/"+student.ID, nil, 200, &rewards)
	if rewards.XP != 10 || rewards.Awards.Total != 1 {
		t.Fatal("bank copy XP dedup", rewards)
	}
	call(sc, "GET", "/api/v1/learning-rewards/"+student.ID, nil, 200, &rewards)
	if rewards.XP != 10 {
		t.Fatal(rewards)
	}
	// Test real SQL schedule boundaries without waiting days or accepting client timestamps.
	if _, err = pool.Exec(t.Context(), `update exercise_attempts set created_at=$2 where learning_attempt_id=$1 and status='incorrect'`, root.Attempt.ID, time.Now().Add(-10*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	move := func(state domain.LearningState, days int) {
		t.Helper()
		at := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
		if _, e := pool.Exec(t.Context(), `update exercise_attempts set created_at=$2 where learning_attempt_id=$1`, state.Attempt.ID, at); e != nil {
			t.Fatal(e)
		}
		if _, e := pool.Exec(t.Context(), `update learning_attempts set started_at=$2::timestamptz,completed_at=$2::timestamptz+interval '1 second' where id=$1`, state.Attempt.ID, at); e != nil {
			t.Fatal(e)
		}
	}
	move(old, 2)
	for _, tc := range []struct{ days, want int }{{2, 3}, {3, 7}, {4, 14}, {5, 30}} {
		cleanDays, want := tc.days, tc.want
		if cleanDays > 2 {
			historical := assign(b, lesson.ID, []string{original[0].ID})
			answer(historical, original[0].ID, "7")
			finish(historical)
			move(historical, cleanDays)
		}
		call(sc, "GET", planPath+"?mode=all", nil, 200, &plan)
		found := false
		for _, item := range plan.Items.Items {
			if item.ExerciseID == original[0].ID {
				found = true
				delta := item.DueAt.Sub(plan.ServerTime)
				if item.CleanDays != cleanDays || delta < time.Duration(want)*24*time.Hour-time.Hour || delta > time.Duration(want)*24*time.Hour+time.Hour {
					t.Fatal("SQL schedule", cleanDays, want, item)
				}
			}
		}
		if !found {
			t.Fatal("missing original recall")
		}
	}
	var pref domain.RewardPreference
	call(b, "PUT", "/api/v1/learning-rewards/"+student.ID+"/preference", map[string]any{"enabled": false}, 200, &pref)
	call(a, "GET", "/api/v1/learning-rewards/"+student.ID+"/preference", nil, 200, &pref)
	if !pref.Enabled {
		t.Fatal("one tutor disabled the other's context")
	}
	call(sc, "GET", "/api/v1/learning-rewards/"+student.ID, nil, 200, &rewards)
	if !rewards.Enabled || rewards.XP != 0 {
		t.Fatal("enabled independent context hidden", rewards)
	}
	// An unfinished quiz loses its grant when the original tutor relationship ends.
	in.RequestID = id()
	in.Mode = "quiz"
	in.Count = 1
	call(sc, "POST", "/api/v1/learning-review-sessions", in, 201, &review)
	publicInput := domain.ReviewStartInput{RequestID: id(), SourceRunID: newRun.Attempt.ID, Mode: "quiz", Count: 1}
	var publicReview domain.ReviewSession
	call(sc, "POST", "/api/v1/learning-review-sessions", publicInput, 201, &publicReview)
	if _, err = pool.Exec(t.Context(), `update relations set status='ended' where student_id=$1 and tutor_id=$2`, student.ID, tutor.ID); err != nil {
		t.Fatal(err)
	}
	call(sc, "GET", "/api/v1/learning-review-sessions/"+review.ID, nil, 404, nil)
	call(sc, "GET", lp+"/learning-attempts/current", nil, 404, nil)
	call(sc, "POST", quizLesson+"/exercises/"+review.State.Version.Exercises[0].ID+"/attempts", domain.ExerciseAttemptInput{RequestID: id(), VersionID: review.State.Version.ID, LearningAttemptID: review.ID, Answer: domain.ExerciseAnswer{Value: &value}}, 404, nil)
	if _, err = pool.Exec(t.Context(), `update learning_review_sessions set exercise_keys=$2 where learning_attempt_id=$1`, review.ID, collection.ExerciseIDs); err == nil {
		t.Fatal("review grant mutable")
	}
	call(b, "GET", planPath, nil, 404, nil)
}
