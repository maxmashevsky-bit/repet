package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
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

func TestLearningRewardsPostgres(t *testing.T) {
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
	call := func(c *http.Client, method, path string, body any, want int, out any) {
		t.Helper()
		data, e := json.Marshal(body)
		if e != nil {
			t.Fatal(e)
		}
		r, e := http.NewRequest(method, srv.URL+path, bytes.NewReader(data))
		if e != nil {
			t.Fatal(e)
		}
		r.Header.Set("Content-Type", "application/json")
		res, e := c.Do(r)
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
	}
	register := func(role string) (*http.Client, domain.User) {
		c := client()
		var out struct {
			User domain.User `json:"user"`
		}
		call(c, "POST", "/api/v1/auth/register", map[string]any{"email": id() + "@example.test", "display_name": "Rewards " + role, "role": role, "password": "RewardsTesting123!", "accept_terms": true}, 201, &out)
		return c, out.User
	}
	tc, tutor := register("tutor")
	sc, student := register("student")
	foreign, _ := register("tutor")
	peer, _ := register("student")
	var inv struct {
		Token string `json:"token"`
	}
	call(tc, "POST", "/api/v1/invitations", map[string]any{"student_email": student.Email}, 201, &inv)
	call(sc, "POST", "/api/v1/invitations/"+inv.Token+"/accept", map[string]any{}, 201, nil)
	rp := "/api/v1/learning-rewards/" + student.ID
	call(peer, "GET", rp, nil, 404, nil)
	call(foreign, "GET", rp, nil, 404, nil)
	call(sc, "PUT", rp+"/preference", map[string]any{"enabled": false}, 403, nil)
	call(tc, "PUT", rp+"/preference", map[string]any{}, 400, nil)
	call(tc, "PUT", rp+"/preference", map[string]any{"enabled": true, "points": 1000}, 400, nil)
	var pref domain.RewardPreference
	call(tc, "GET", rp+"/preference", nil, 200, &pref)
	if !pref.Enabled || pref.UpdatedAt != nil {
		t.Fatal(pref)
	}
	var course domain.Course
	call(tc, "POST", "/api/v1/catalog/courses", domain.CourseInput{Title: "Own recall", Track: "school", GradeMin: 5, GradeMax: 11, Difficulty: 1, Visibility: "linked_students"}, 201, &course)
	cp := "/api/v1/catalog/courses/" + course.ID
	var topic domain.Topic
	call(tc, "POST", cp+"/topics", domain.TopicInput{Title: "Recall"}, 201, &topic)
	call(tc, "POST", cp+"/publish", map[string]any{}, 200, nil)
	call(tc, "POST", "/api/v1/catalog/topics/"+topic.ID+"/publish", map[string]any{}, 200, nil)
	var lesson domain.LearningLesson
	call(tc, "POST", cp+"/lessons", domain.LearningLessonInput{TopicID: topic.ID, Title: "Own recall", Track: "school", GradeMin: 5, GradeMax: 11, Difficulty: 1}, 201, &lesson)
	lp := "/api/v1/catalog/lessons/" + lesson.ID
	exercises := make([]domain.Exercise, 5)
	for i := range exercises {
		kind, key := "numeric", "7"
		options := []string{}
		if i == 2 {
			kind = "single_choice"
			key = "a"
			options = []string{"a", "b"}
		}
		if i == 4 {
			kind = "long_text"
			key = "[manual]"
		}
		call(tc, "POST", lp+"/exercises", domain.ExerciseInput{Kind: kind, Prompt: "Own exercise", Options: options, Position: i, Difficulty: 1, AnswerKey: domain.AnswerKeyInput{Answer: key, Hints: []string{"Hint"}}}, 201, &exercises[i])
	}
	call(tc, "POST", lp+"/publish", map[string]any{}, 200, &lesson)
	var run domain.LearningState
	var manual []domain.ExerciseAttempt
	var firstRun string
	for n := 0; n < 2; n++ {
		call(sc, "POST", lp+"/learning-attempts", domain.LearningStartInput{RequestID: id(), Restart: n > 0}, 201, &run)
		if n == 0 {
			firstRun = run.Attempt.ID
		}
		for i, e := range exercises {
			if i == 3 {
				call(sc, "POST", lp+"/learning-attempts/"+run.Attempt.ID+"/exercises/"+e.ID+"/hints", domain.LearningActionInput{RequestID: id()}, 200, nil)
			}
			value := "7"
			if i == 1 {
				value = "0"
			}
			if i == 4 {
				value = "Обоснование"
			}
			answer := domain.ExerciseAnswer{Value: &value}
			if i == 2 {
				answer = domain.ExerciseAnswer{Selected: []string{"a"}}
			}
			in := domain.ExerciseAttemptInput{RequestID: id(), VersionID: run.Version.ID, LearningAttemptID: run.Attempt.ID, Answer: answer}
			var attempt domain.ExerciseAttempt
			call(sc, "POST", lp+"/exercises/"+e.ID+"/attempts", in, 201, &attempt)
			call(sc, "POST", lp+"/exercises/"+e.ID+"/attempts", in, 201, nil)
			if i == 1 {
				value = "7"
				in.RequestID = id()
				in.Answer = domain.ExerciseAnswer{Value: &value}
				call(sc, "POST", lp+"/exercises/"+e.ID+"/attempts", in, 201, nil)
			}
			if i == 4 {
				manual = append(manual, attempt)
			}
		}
		call(sc, "POST", lp+"/learning-attempts/"+run.Attempt.ID+"/complete", domain.LearningActionInput{RequestID: id()}, 200, nil)
	}
	var rewards domain.RewardSummary
	if _, err = store.New(pool).LearningRewards(t.Context(), student, student.ID, domain.CatalogFilter{Limit: 20}); err != nil {
		t.Fatal("direct rewards", err)
	}
	call(sc, "GET", rp, nil, 200, &rewards)
	if rewards.XP != 0 {
		t.Fatal("same-day retries must not award XP", rewards)
	}
	// Only test fixtures move in time; APIs cannot supply these server-owned timestamps.
	past := time.Now().Add(-48 * time.Hour)
	if _, err = pool.Exec(t.Context(), `update exercise_attempts set created_at=$2 where learning_attempt_id=$1`, firstRun, past); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(t.Context(), `update learning_attempts set started_at=$2::timestamptz,completed_at=$2::timestamptz+interval '1 minute' where id=$1`, firstRun, past); err != nil {
		t.Fatal(err)
	}
	call(sc, "GET", rp, nil, 200, &rewards)
	if rewards.XP != 10 || rewards.Level != 1 || rewards.LevelProgress != 10 || rewards.Awards.Total != 1 {
		t.Fatal("only clean numeric recall should qualify", rewards)
	}
	call(sc, "GET", rp, nil, 200, &rewards)
	if rewards.XP != 10 || rewards.Awards.Total != 1 {
		t.Fatal("ledger replay", rewards)
	}
	for _, a := range manual {
		call(tc, "PUT", lp+"/attempts/"+a.ID+"/review", domain.ExerciseReviewInput{Status: "correct", Feedback: "Проверено"}, 200, nil)
	}
	call(tc, "GET", rp, nil, 200, &rewards)
	if rewards.XP != 20 || rewards.Awards.Total != 2 {
		t.Fatal("teacher-reviewed recall should qualify", rewards)
	}
	if _, err = pool.Exec(t.Context(), `update learning_reward_awards set points=10 where student_id=$1`, student.ID); err == nil {
		t.Fatal("award ledger mutable")
	}
	call(tc, "PUT", rp+"/preference", map[string]any{"enabled": false}, 200, &pref)
	updated := pref.UpdatedAt
	call(tc, "PUT", rp+"/preference", map[string]any{"enabled": false}, 200, &pref)
	if updated == nil || pref.UpdatedAt == nil || !updated.Equal(*pref.UpdatedAt) {
		t.Fatal("preference replay changed time")
	}
	call(sc, "GET", rp, nil, 200, &rewards)
	if rewards.Enabled || rewards.XP != 0 || rewards.Awards.Total != 0 {
		t.Fatal("disabled rewards exposed", rewards)
	}
	call(tc, "PUT", rp+"/preference", map[string]any{"enabled": true}, 200, &pref)
	if pref.EnabledFrom == nil {
		t.Fatal("reenable must not backfill disabled interval")
	}
	call(sc, "GET", rp, nil, 200, &rewards)
	if !rewards.Enabled || rewards.XP != 20 {
		t.Fatal("old achievements lost", rewards)
	}
	if _, err = pool.Exec(t.Context(), `update relations set status='ended' where student_id=$1 and tutor_id=$2`, student.ID, tutor.ID); err != nil {
		t.Fatal(err)
	}
	call(tc, "GET", rp, nil, 404, nil)
	call(tc, "PUT", rp+"/preference", map[string]any{"enabled": false}, 404, nil)
}
