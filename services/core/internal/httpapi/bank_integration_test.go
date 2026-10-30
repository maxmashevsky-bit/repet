package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

func TestExerciseBankPostgres(t *testing.T) {
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
	srv := httptest.NewServer(NewServer(config.Config{CORSAllowedOrigins: []string{"http://example.test"}, RequestBodyLimitByte: 1 << 20}, slog.New(slog.NewTextHandler(io.Discard, nil)), store.New(pool), auth.DevProvider{}))
	defer srv.Close()
	id := func() string {
		v, e := platform.NewUUID()
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	marker := strings.ReplaceAll(id(), "-", "")
	call := func(method, path string, body any, cookie *http.Cookie, want int, out any) []byte {
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
		if cookie != nil {
			req.AddCookie(cookie)
		}
		res, e := srv.Client().Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		raw, e := io.ReadAll(res.Body)
		if e != nil {
			t.Fatal(e)
		}
		if res.StatusCode != want {
			t.Fatalf("%s %s: %d want %d: %s", method, path, res.StatusCode, want, raw)
		}
		if out != nil {
			if e = json.Unmarshal(raw, out); e != nil {
				t.Fatal(e)
			}
		}
		return raw
	}
	register := func(role string) (domain.User, *http.Cookie) {
		body, _ := json.Marshal(map[string]any{"display_name": "Bank " + role, "email": id() + "@example.test", "password": "BankTesting123!", "role": role, "accept_terms": true})
		res, e := srv.Client().Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(body))
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		var u struct {
			User domain.User `json:"user"`
		}
		if res.StatusCode != 201 {
			t.Fatal("registration", res.StatusCode)
		}
		if e = json.NewDecoder(res.Body).Decode(&u); e != nil {
			t.Fatal(e)
		}
		for _, c := range res.Cookies() {
			if c.Name == store.SessionCookieName {
				return u.User, c
			}
		}
		t.Fatal("session")
		return domain.User{}, nil
	}
	tutor, tc := register("tutor")
	_, oc := register("tutor")
	student, sc := register("student")
	student2, s2c := register("student")
	_, pc := register("student")
	for _, pair := range []struct {
		u domain.User
		c *http.Cookie
	}{{student, sc}, {student2, s2c}} {
		var invite struct {
			Token string `json:"token"`
		}
		call("POST", "/api/v1/invitations", map[string]any{"student_email": pair.u.Email}, tc, 201, &invite)
		call("POST", "/api/v1/invitations/"+invite.Token+"/accept", map[string]any{}, pair.c, 201, nil)
	}
	call("GET", "/api/v1/exercise-bank", nil, nil, 401, nil)
	call("GET", "/api/v1/exercise-bank", nil, sc, 403, nil)
	var course domain.Course
	call("POST", "/api/v1/catalog/courses", domain.CourseInput{Title: "Own demo bank", Track: "school", GradeMin: 5, GradeMax: 11, Difficulty: 1, Visibility: "linked_students"}, tc, 201, &course)
	cp := "/api/v1/catalog/courses/" + course.ID
	var topic domain.Topic
	call("POST", cp+"/topics", domain.TopicInput{Title: "Числа"}, tc, 201, &topic)
	call("POST", cp+"/publish", map[string]any{}, tc, 200, nil)
	call("POST", "/api/v1/catalog/topics/"+topic.ID+"/publish", map[string]any{}, tc, 200, nil)
	entries := make([]domain.BankExercise, 0, 3)
	lessons := make([]domain.LearningLesson, 0, 3)
	number := 7
	for i := 0; i < 3; i++ {
		var lesson domain.LearningLesson
		call("POST", cp+"/lessons", domain.LearningLessonInput{TopicID: topic.ID, Title: "Собственный пример", Track: "school", GradeMin: 5, GradeMax: 11, Difficulty: 2}, tc, 201, &lesson)
		lp := "/api/v1/catalog/lessons/" + lesson.ID
		var ex domain.Exercise
		call("POST", lp+"/exercises", domain.ExerciseInput{Kind: "numeric", Prompt: "На числовой прямой найдите число: 3 + 4.", Difficulty: 2, AnswerKey: domain.AnswerKeyInput{Answer: "7", Explanation: "PRIVATE_BANK_SOLUTION", Hints: []string{"Прибавьте четыре."}}}, tc, 201, &ex)
		in := domain.BankExerciseInput{LessonID: lesson.ID, ExerciseID: ex.ID, BankMetadata: domain.BankMetadata{Grade: 9, Direction: "oge", Topic: "Числовая прямая", Subtopic: "Сложение", ExamNumber: &number, Source: "Собственный демонстрационный пример, не официальный экзамен " + marker, Attribution: tutor.DisplayName, RightsStatus: "own", License: "Разрешено назначать моим ученикам"}}
		call("POST", "/api/v1/exercise-bank", in, tc, 404, nil)
		call("POST", lp+"/publish", map[string]any{}, tc, 200, &lesson)
		var entry domain.BankExercise
		raw := call("POST", "/api/v1/exercise-bank", in, tc, 201, &entry)
		if strings.Contains(string(raw), "PRIVATE_BANK_SOLUTION") || entry.VersionID != lesson.Version.ID {
			t.Fatal("bank snapshot/key leak")
		}
		call("POST", "/api/v1/exercise-bank", in, tc, 409, nil)
		call("POST", "/api/v1/exercise-bank", in, sc, 403, nil)
		call("POST", "/api/v1/exercise-bank", in, oc, 404, nil)
		forged, _ := json.Marshal(in)
		var fields map[string]any
		json.Unmarshal(forged, &fields)
		fields["owner_id"] = tutor.ID
		call("POST", "/api/v1/exercise-bank", fields, tc, 400, nil)
		in.Grade = 4
		call("POST", "/api/v1/exercise-bank", in, tc, 400, nil)
		call("GET", "/api/v1/exercise-bank/"+entry.ID, nil, oc, 404, nil)
		call("GET", "/api/v1/exercise-bank/"+entry.ID+"/answer-key", nil, sc, 403, nil)
		raw = call("GET", "/api/v1/exercise-bank/"+entry.ID+"/answer-key", nil, tc, 200, nil)
		if !strings.Contains(string(raw), "PRIVATE_BANK_SOLUTION") {
			t.Fatal("teacher key")
		}
		entries = append(entries, entry)
		lessons = append(lessons, lesson)
	}
	var page domain.CatalogPage[domain.BankExercise]
	call("GET", "/api/v1/exercise-bank?direction=oge&grade=9&exam_number=7&difficulty=2&q=прямой%20"+marker+"&limit=1&offset=1", nil, tc, 200, &page)
	if page.Total != 3 || len(page.Items) != 1 {
		t.Fatal("search/page", page.Total, len(page.Items))
	}
	call("GET", "/api/v1/exercise-bank?direction=ege_basic&q="+marker, nil, tc, 200, &page)
	if page.Total != 0 || len(page.Items) != 0 {
		t.Fatal("filter")
	}
	for _, q := range []string{"grade=4", "limit=0", "offset=-1", "exam_number=0", "grade=9&grade=10", "owner_id=x", "direction=ege", "kind=invalid"} {
		call("GET", "/api/v1/exercise-bank?"+q, nil, tc, 400, nil)
	}
	var facets domain.BankFacets
	call("GET", "/api/v1/exercise-bank/facets?direction=oge&q="+marker, nil, tc, 200, &facets)
	if len(facets.Topics) != 1 || facets.Topics[0] != "Числовая прямая" {
		t.Fatal("facets", facets)
	}
	in := domain.BankCollectionInput{RequestID: id(), Title: "ОГЭ №7 — собственная подборка", CourseID: course.ID, TopicID: topic.ID, BankIDs: []string{entries[2].ID, entries[0].ID, entries[1].ID}}
	var collection, replay domain.BankCollection
	call("POST", "/api/v1/bank-collections", in, tc, 201, &collection)
	call("POST", "/api/v1/bank-collections", in, tc, 201, &replay)
	if collection.ID != replay.ID || len(collection.Version.Exercises) != 3 || collection.Provenance[0].ID != entries[2].ID {
		t.Fatal("collection snapshot/replay")
	}
	singleConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	singleConfig.MaxConns = 1
	single, err := pgxpool.NewWithConfig(t.Context(), singleConfig)
	if err != nil {
		t.Fatal(err)
	}
	limited, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	_, err = store.New(single).CreateBankCollection(limited, tutor, in)
	cancel()
	single.Close()
	if err != nil {
		t.Fatal("single-connection replay", err)
	}
	changed := in
	changed.Title = "Changed"
	call("POST", "/api/v1/bank-collections", changed, tc, 409, nil)
	changed.RequestID = id()
	changed.BankIDs = []string{entries[0].ID, strings.ToUpper(entries[0].ID)}
	call("POST", "/api/v1/bank-collections", changed, tc, 400, nil)
	call("GET", "/api/v1/bank-collections/"+collection.ID, nil, oc, 404, nil)
	call("GET", "/api/v1/bank-collections/"+collection.ID, nil, sc, 403, nil)
	call("GET", "/api/v1/catalog/lessons/"+collection.LessonID, nil, sc, 404, nil)
	call("GET", "/api/v1/catalog/lessons/"+collection.LessonID+"/versions/1", nil, sc, 404, nil)
	lp := "/api/v1/catalog/lessons/" + lessons[0].ID
	call("PUT", lp+"/exercises/"+entries[0].Exercise.ID, domain.ExerciseInput{Kind: "numeric", Prompt: "Changed question", Difficulty: 1, AnswerKey: domain.AnswerKeyInput{Answer: "99"}}, tc, 200, nil)
	call("POST", lp+"/publish", map[string]any{}, tc, 200, nil)
	call("GET", "/api/v1/bank-collections/"+collection.ID, nil, tc, 200, &replay)
	if replay.Version.Exercises[1].Prompt != entries[0].Exercise.Prompt {
		t.Fatal("retroactive edit")
	}
	assign := domain.BankAssignInput{RequestID: id(), StudentIDs: []string{student.ID, student2.ID}, DueAt: time.Now().Add(24 * time.Hour), Mode: "practice"}
	var results []domain.BankAssignResult
	call("POST", "/api/v1/bank-collections/"+collection.ID+"/assign", assign, tc, 200, &results)
	if len(results) != 2 {
		t.Fatal("batch size")
	}
	ids := map[string]string{}
	for _, r := range results {
		if r.Assignment == nil {
			t.Fatal("batch", r.Error)
		}
		ids[r.StudentID] = r.Assignment.ID
	}
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var again []domain.BankAssignResult
			call("POST", "/api/v1/bank-collections/"+collection.ID+"/assign", assign, tc, 200, &again)
			for _, r := range again {
				if r.Assignment == nil || ids[r.StudentID] != r.Assignment.ID {
					t.Error("concurrent replay")
				}
			}
		}()
	}
	wg.Wait()
	bad := assign
	bad.Mode = "exam"
	call("POST", "/api/v1/bank-collections/"+collection.ID+"/assign", bad, tc, 409, nil)
	bad.RequestID = id()
	bad.StudentIDs = []string{id()}
	call("POST", "/api/v1/bank-collections/"+collection.ID+"/assign", bad, tc, 404, nil)
	aid := ids[student.ID]
	call("GET", "/api/v1/assignments/"+aid+"/learning-results", nil, pc, 404, nil)
	var state domain.LearningState
	call("POST", "/api/v1/assignments/"+aid+"/learning-attempt", domain.LearningActionInput{RequestID: id()}, sc, 201, &state)
	if len(state.Version.Exercises) != 3 || state.Version.ID != collection.VersionID {
		t.Fatal("assigned snapshot")
	}
	raw := call("GET", "/api/v1/assignments/"+aid+"/learning-results", nil, sc, 200, nil)
	if strings.Contains(string(raw), "PRIVATE_BANK_SOLUTION") {
		t.Fatal("student key leak")
	}
	for _, ex := range state.Version.Exercises {
		var result domain.ExerciseAttempt
		call("POST", "/api/v1/catalog/lessons/"+collection.LessonID+"/exercises/"+ex.ID+"/attempts", domain.ExerciseAttemptInput{VersionID: state.Version.ID, LearningAttemptID: state.Attempt.ID, RequestID: id(), Answer: answerValue("7")}, sc, 201, &result)
		if result.Status != "correct" {
			t.Fatal("copied key grading", result.Status)
		}
	}
	call("POST", "/api/v1/catalog/lessons/"+collection.LessonID+"/learning-attempts/"+state.Attempt.ID+"/complete", domain.LearningActionInput{RequestID: id()}, sc, 200, &state)
	if state.Attempt.CorrectExercises != 3 {
		t.Fatal("bank completion")
	}
	call("GET", "/api/v1/catalog/lessons/"+collection.LessonID+"/learning-attempts/"+state.Attempt.ID, nil, s2c, 404, nil)
	call("POST", "/api/v1/catalog/lessons/"+collection.LessonID+"/learning-attempts", domain.LearningStartInput{RequestID: id()}, sc, 404, nil)
	// Publishing a source to the organization permits reuse, not editing or ownership changes.
	call("PUT", cp, domain.CourseInput{Title: course.Title, Track: "school", GradeMin: 5, GradeMax: 11, Difficulty: 1, Visibility: "organization"}, tc, 200, nil)
	call("GET", "/api/v1/exercise-bank/"+entries[0].ID, nil, oc, 200, nil)
	call("GET", "/api/v1/exercise-bank/"+entries[0].ID+"/answer-key", nil, oc, 200, nil)
	call("GET", "/api/v1/bank-collections/"+collection.ID, nil, oc, 404, nil)
	call("POST", cp+"/archive", map[string]any{}, tc, 200, nil)
	call("GET", "/api/v1/exercise-bank/"+entries[0].ID, nil, oc, 404, nil)
	call("GET", "/api/v1/assignments/"+aid+"/learning-results", nil, sc, 200, nil)
	for _, table := range []string{"bank_collections", "bank_exercises"} {
		_, e := pool.Exec(t.Context(), "update "+table+" set created_at=now() where id=$1", map[string]string{"bank_collections": collection.ID, "bank_exercises": entries[0].ID}[table])
		if e == nil {
			t.Fatal("mutable provenance", table)
		}
	}
	var indexes int
	if err = pool.QueryRow(t.Context(), `select count(*) from pg_indexes where indexname in ('bank_filter_idx','bank_topic_idx','bank_search_idx','bank_collections_owner_idx')`).Scan(&indexes); err != nil || indexes != 4 {
		t.Fatal("bank indexes", indexes, err)
	}
	tx, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(t.Context())
	if _, err = tx.Exec(t.Context(), `set local enable_seqscan=off`); err != nil {
		t.Fatal(err)
	}
	rows, err := tx.Query(t.Context(), `explain select id from bank_exercises where search_vector @@ plainto_tsquery('russian','прямая')`)
	if err != nil {
		t.Fatal(err)
	}
	var plan strings.Builder
	for rows.Next() {
		var line string
		if err = rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		plan.WriteString(line)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.String(), "bank_search_idx") {
		t.Fatal("FTS index not usable", plan.String())
	}
}
