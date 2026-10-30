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

func TestLearningAnalyticsPostgres(t *testing.T) {
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
		jar, e := cookiejar.New(nil)
		if e != nil {
			t.Fatal(e)
		}
		c := srv.Client()
		return &http.Client{Transport: c.Transport, Jar: jar}
	}
	call := func(c *http.Client, method, path string, body any, want int, out any) []byte {
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
		return raw
	}
	register := func(role string) (*http.Client, domain.User) {
		t.Helper()
		c := client()
		var out struct {
			User domain.User `json:"user"`
		}
		call(c, "POST", "/api/v1/auth/register", map[string]any{"email": id() + "@example.test", "display_name": "Analytics " + role, "password": "AnalyticsTesting123!", "role": role, "accept_terms": true}, 201, &out)
		return c, out.User
	}
	tc, tutor := register("tutor")
	other, otherTutor := register("tutor")
	sc, student := register("student")
	peer, peerUser := register("student")
	link := func(c *http.Client) {
		var inv struct {
			Token string `json:"token"`
		}
		call(c, "POST", "/api/v1/invitations", map[string]any{"student_email": student.Email}, 201, &inv)
		call(sc, "POST", "/api/v1/invitations/"+inv.Token+"/accept", map[string]any{}, 201, nil)
	}
	link(tc)
	path := "/api/v1/learning-analytics/" + student.ID
	call(client(), "GET", path, nil, 401, nil)
	call(peer, "GET", path, nil, 404, nil)
	call(tc, "GET", "/api/v1/learning-analytics/"+peerUser.ID, nil, 404, nil)
	call(other, "GET", path, nil, 404, nil)
	var empty domain.AnalyticsReport
	now := time.Now().UTC()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if _, err = store.New(pool).LearningAnalytics(t.Context(), tutor, student.ID, domain.AnalyticsFilter{From: day.AddDate(0, 0, -29), To: day}); err != nil {
		t.Fatal("direct report", err)
	}
	call(tc, "GET", path, nil, 200, &empty)
	if empty.Summary.Accuracy != nil || len(empty.Topics.Items) != 0 || len(empty.Recommendations) != 0 {
		t.Fatal("empty report", empty)
	}
	makeLesson := func(c *http.Client, title string, count int) (domain.LearningLesson, []domain.Exercise) {
		t.Helper()
		var course domain.Course
		call(c, "POST", "/api/v1/catalog/courses", domain.CourseInput{Title: title, Track: "school", GradeMin: 5, GradeMax: 11, Difficulty: 1, Visibility: "linked_students"}, 201, &course)
		cp := "/api/v1/catalog/courses/" + course.ID
		var topic domain.Topic
		call(c, "POST", cp+"/topics", domain.TopicInput{Title: title}, 201, &topic)
		call(c, "POST", cp+"/publish", map[string]any{}, 200, nil)
		call(c, "POST", "/api/v1/catalog/topics/"+topic.ID+"/publish", map[string]any{}, 200, nil)
		var lesson domain.LearningLesson
		call(c, "POST", cp+"/lessons", domain.LearningLessonInput{Title: title, TopicID: topic.ID, Track: "school", GradeMin: 5, GradeMax: 11, Difficulty: 1}, 201, &lesson)
		lp := "/api/v1/catalog/lessons/" + lesson.ID
		ex := make([]domain.Exercise, 0, count)
		for i := 0; i < count; i++ {
			var e domain.Exercise
			kind := "numeric"
			key := "7"
			if i == 10 {
				kind = "long_text"
				key = "[manual]"
			}
			call(c, "POST", lp+"/exercises", domain.ExerciseInput{Kind: kind, Prompt: "Собственная задача", Position: i, Difficulty: 1, AnswerKey: domain.AnswerKeyInput{Answer: key, Explanation: "PRIVATE_ANALYTICS_KEY", Hints: []string{"Сложите числа"}}}, 201, &e)
			ex = append(ex, e)
		}
		call(c, "POST", lp+"/publish", map[string]any{}, 200, &lesson)
		return lesson, ex
	}
	var runs []domain.LearningState
	for topicIndex, title := range []string{"Дроби", "Уравнения"} {
		count := 10
		if topicIndex == 1 {
			count = 11
		}
		lesson, ex := makeLesson(tc, title, count)
		var assignment domain.Assignment
		call(tc, "POST", "/api/v1/assignments/material", domain.MaterialAssignmentInput{RequestID: id(), StudentID: student.ID, LessonID: lesson.ID, Mode: "practice", DueAt: time.Now().Add(24 * time.Hour)}, 201, &assignment)
		var state domain.LearningState
		call(sc, "POST", "/api/v1/assignments/"+assignment.ID+"/learning-attempt", domain.LearningActionInput{RequestID: id()}, 201, &state)
		runs = append(runs, state)
		lp := "/api/v1/catalog/lessons/" + lesson.ID
		rp := lp + "/learning-attempts/" + state.Attempt.ID
		if topicIndex == 0 {
			call(sc, "POST", rp+"/navigate", domain.LearningNavigateInput{RequestID: id(), Step: "exercise:" + ex[0].ID}, 200, nil)
			call(sc, "POST", rp+"/exercises/"+ex[0].ID+"/hints", domain.LearningActionInput{RequestID: id()}, 200, nil)
		}
		for i, e := range ex {
			value := "7"
			if topicIndex == 0 && i < 5 {
				value = "0"
			}
			if e.Kind == "long_text" {
				value = "Моё обоснование"
			}
			in := domain.ExerciseAttemptInput{VersionID: state.Version.ID, LearningAttemptID: state.Attempt.ID, RequestID: id(), Answer: domain.ExerciseAnswer{Value: &value}}
			call(sc, "POST", lp+"/exercises/"+e.ID+"/attempts", in, 201, nil)
			if i == 0 {
				call(sc, "POST", lp+"/exercises/"+e.ID+"/attempts", in, 201, nil)
			}
		}
	}
	var report domain.AnalyticsReport
	raw := call(tc, "GET", path, nil, 200, &report)
	if strings.Contains(string(raw), "PRIVATE_ANALYTICS") || strings.Contains(string(raw), "answer_key") || report.Summary.Attempts != 21 || report.Summary.Graded != 20 || report.Summary.Pending != 1 || report.Summary.Correct != 15 || report.Summary.Incorrect != 5 || report.Summary.Accuracy == nil || *report.Summary.Accuracy != 75 || report.Summary.Hints != 1 || report.Homework.Total != 2 {
		t.Fatalf("wrong aggregate: %+v", report)
	}
	if len(report.Topics.Items) != 2 || report.Topics.Items[0].Level != "practice" || report.Topics.Items[1].Level != "secure" || len(report.Recommendations) != 1 || report.Recommendations[0].Reason != "weak_topic" || len(report.Recommendations[0].Exercises) != 5 {
		t.Fatal("topics/recommendations", report.Topics, report.Recommendations)
	}
	// Synthetic timestamps exercise both halves without accepting browser-owned time.
	if _, err = pool.Exec(t.Context(), `update exercise_attempts set created_at=case when status='incorrect' then $2::timestamptz else $3::timestamptz end where learning_attempt_id=$1`, runs[0].Attempt.ID, day.AddDate(0, 0, -20).Add(time.Hour), day.AddDate(0, 0, -1).Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	call(tc, "GET", path, nil, 200, &report)
	if report.Topics.Items[0].Change == nil || *report.Topics.Items[0].Change != 100 || report.Topics.Items[0].PreviousGraded != 5 || report.Topics.Items[0].RecentGraded != 5 || len(report.Timeline) != 3 {
		t.Fatal("period trend", report.Topics, report.Timeline)
	}
	var self domain.AnalyticsReport
	call(sc, "GET", path, nil, 200, &self)
	if self.Summary.Graded != 20 {
		t.Fatal(self.Summary)
	}
	var students domain.CatalogPage[domain.AnalyticsStudent]
	call(tc, "GET", "/api/v1/learning-analytics/students?limit=1", nil, 200, &students)
	if students.Total != 1 || students.Items[0].ID != student.ID {
		t.Fatal(students)
	}
	call(sc, "GET", "/api/v1/learning-analytics/students", nil, 403, nil)
	for _, query := range []string{"?from=bad", "?from=2020-01-01&to=2026-01-01", "?topic_offset=-1", "?owner_id=" + tutor.ID, "?from=2026-01-01&from=2026-01-02"} {
		call(sc, "GET", path+query, nil, 400, nil)
	}
	var absent domain.AnalyticsReport
	call(tc, "GET", path+"?from=2000-01-01&to=2000-01-01&topic_offset=20", nil, 200, &absent)
	if absent.Summary.Attempts != 0 || absent.Homework.Total != 0 || len(absent.Topics.Items) != 0 {
		t.Fatal(absent)
	}
	rec := report.Recommendations[0]
	repeat := domain.AnalyticsRepeatInput{RequestID: id(), RunID: rec.RunID, TopicKey: rec.TopicKey, DueAt: time.Now().Add(48 * time.Hour)}
	call(sc, "POST", path+"/repeat", repeat, 403, nil)
	call(other, "POST", path+"/repeat", repeat, 404, nil)
	var assigned, replayed domain.Assignment
	call(tc, "POST", path+"/repeat", repeat, 201, &assigned)
	call(tc, "POST", path+"/repeat", repeat, 201, &replayed)
	if assigned.ID != replayed.ID || assigned.Material == nil || len(assigned.Material.ExerciseIDs) != 5 || assigned.Material.VersionID != runs[0].Version.ID {
		t.Fatal("repeat must pin only five mistakes", assigned)
	}
	changed := repeat
	changed.TopicKey = report.Topics.Items[1].Key
	call(tc, "POST", path+"/repeat", changed, 409, nil)
	// Same student can have two tutors, without disclosing either tutor's private context.
	link(other)
	foreign, ex := makeLesson(other, "Private other tutor", 1)
	var fs domain.LearningState
	call(sc, "POST", "/api/v1/catalog/lessons/"+foreign.ID+"/learning-attempts", domain.LearningStartInput{RequestID: id()}, 201, &fs)
	value := "7"
	call(sc, "POST", "/api/v1/catalog/lessons/"+foreign.ID+"/exercises/"+ex[0].ID+"/attempts", domain.ExerciseAttemptInput{RequestID: id(), VersionID: fs.Version.ID, LearningAttemptID: fs.Attempt.ID, Answer: domain.ExerciseAnswer{Value: &value}}, 201, nil)
	call(tc, "GET", path, nil, 200, &report)
	if report.Summary.Graded != 20 {
		t.Fatal("foreign context leaked", report.Summary)
	}
	call(other, "GET", path, nil, 200, &self)
	if self.Summary.Graded != 1 {
		t.Fatal("other tutor scope", self.Summary)
	}
	call(sc, "GET", path, nil, 200, &self)
	if self.Summary.Graded != 21 {
		t.Fatal("student own all results", self.Summary)
	}
	call(other, "POST", path+"/repeat", domain.AnalyticsRepeatInput{RequestID: id(), RunID: rec.RunID, TopicKey: rec.TopicKey, DueAt: repeat.DueAt}, 404, nil)
	if _, err = pool.Exec(t.Context(), `update relations set status='ended' where tutor_id=$1 and student_id=$2`, otherTutor.ID, student.ID); err != nil {
		t.Fatal(err)
	}
	call(other, "GET", path, nil, 404, nil)
}
