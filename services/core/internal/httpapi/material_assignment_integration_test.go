package httpapi

import (
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"testing"
	"time"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

func testMaterialAssignmentsHTTP(t *testing.T, call func(string, string, any, *http.Cookie, int) []byte, decode func([]byte, any), tc, sc, peer, otherTutor *http.Cookie, tutor, student domain.User, pool *pgxpool.Pool) {
	t.Helper()
	id := func() string {
		v, err := platform.NewUUID()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	var course domain.Course
	decode(call("POST", "/api/v1/catalog/courses", domain.CourseInput{Title: "Assigned course", Track: "school", GradeMin: 1, GradeMax: 11, Difficulty: 1, Visibility: "linked_students"}, tc, 201), &course)
	cp := "/api/v1/catalog/courses/" + course.ID
	var topic domain.Topic
	decode(call("POST", cp+"/topics", domain.TopicInput{Title: "Homework"}, tc, 201), &topic)
	input := domain.LearningLessonInput{TopicID: topic.ID, Title: "Material homework", Content: "Theory", Track: "school", GradeMin: 1, GradeMax: 11, Difficulty: 1}
	var lesson domain.LearningLesson
	decode(call("POST", cp+"/lessons", input, tc, 201), &lesson)
	lp := "/api/v1/catalog/lessons/" + lesson.ID
	es := make([]domain.Exercise, 3)
	for i := range es {
		kind := "numeric"
		key := "7"
		if i == 2 {
			kind = "long_text"
			key = ""
		}
		decode(call("POST", lp+"/exercises", domain.ExerciseInput{Kind: kind, Prompt: "Solve", Position: i, Difficulty: 1, AnswerKey: domain.AnswerKeyInput{Answer: key, Explanation: "PRIVATE_MATERIAL_KEY", Hints: []string{"Hint"}}}, tc, 201), &es[i])
	}
	call("POST", cp+"/publish", map[string]any{}, tc, 200)
	call("POST", "/api/v1/catalog/topics/"+topic.ID+"/publish", map[string]any{}, tc, 200)
	decode(call("POST", lp+"/publish", map[string]any{}, tc, 200), &lesson)
	call("GET", lp, nil, peer, 404)
	var rel domain.Relation
	if err := pool.QueryRow(t.Context(), `select id::text from relations where tutor_id=$1 and student_id=$2`, tutor.ID, student.ID).Scan(&rel.ID); err != nil {
		t.Fatal(err)
	}
	var calendar domain.Lesson
	decode(call("POST", "/api/v1/lessons", map[string]any{"relation_id": rel.ID, "title": "Homework calendar", "starts_at": time.Now().Add(time.Hour), "ends_at": time.Now().Add(2 * time.Hour)}, tc, 201), &calendar)
	var set domain.ExerciseSet
	setInput := domain.ExerciseSetInput{Title: "My selection", LessonID: lesson.ID, ExerciseIDs: []string{es[0].ID, es[2].ID}}
	decode(call("POST", "/api/v1/exercise-sets", setInput, tc, 201), &set)
	call("POST", "/api/v1/exercise-sets", setInput, sc, 403)
	call("PUT", "/api/v1/exercise-sets/"+set.ID, setInput, otherTutor, 404)
	call("DELETE", "/api/v1/exercise-sets/"+set.ID, nil, otherTutor, 404)
	var sets domain.CatalogPage[domain.ExerciseSet]
	decode(call("GET", "/api/v1/exercise-sets?limit=1", nil, tc, 200), &sets)
	if sets.Total < 1 || len(sets.Items) != 1 {
		t.Fatal("set pagination")
	}
	due := time.Now().Add(24 * time.Hour)
	in := domain.MaterialAssignmentInput{RequestID: id(), StudentID: student.ID, SetID: set.ID, DueAt: due, CalendarLessonID: calendar.ID, Mode: "practice"}
	var item domain.Assignment
	decode(call("POST", "/api/v1/assignments/material", in, tc, 201), &item)
	if item.Kind != "material" || item.Material == nil || len(item.Material.ExerciseIDs) != 2 || item.Material.CalendarLessonID == nil {
		t.Fatal("material metadata", item)
	}
	var replay domain.Assignment
	decode(call("POST", "/api/v1/assignments/material", in, tc, 201), &replay)
	if replay.ID != item.ID {
		t.Fatal("assignment replay duplicated")
	}
	changed := in
	changed.DueAt = due.Add(time.Hour)
	call("POST", "/api/v1/assignments/material", changed, tc, 409)
	call("POST", "/api/v1/assignments/material", in, sc, 403)
	call("POST", "/api/v1/assignments/material", map[string]any{"request_id": id(), "student_id": student.ID, "lesson_id": lesson.ID, "due_at": due, "owner_id": tutor.ID}, tc, 400)
	bad := in
	bad.RequestID = id()
	bad.StudentID = id()
	call("POST", "/api/v1/assignments/material", bad, tc, 404)
	bad = in
	bad.RequestID = id()
	bad.CalendarLessonID = id()
	call("POST", "/api/v1/assignments/material", bad, tc, 404)
	bad = in
	bad.RequestID = id()
	bad.SetID = ""
	bad.LessonID = lesson.ID
	bad.ExerciseIDs = []string{es[0].ID, es[0].ID}
	call("POST", "/api/v1/assignments/material", bad, tc, 400)
	base := "/api/v1/assignments/" + item.ID
	for _, cookie := range []*http.Cookie{peer, otherTutor} {
		call("GET", base+"/learning-results", nil, cookie, 404)
		call("POST", base+"/repeat", domain.MaterialRepeatInput{RequestID: id(), DueAt: due}, cookie, map[bool]int{true: 403, false: 404}[cookie == peer])
	}
	call("GET", base+"/learning-results", nil, nil, 401)
	var dashboard struct {
		Assignments []domain.Assignment `json:"assignments"`
	}
	decode(call("GET", "/api/v1/dashboard", nil, sc, 200), &dashboard)
	found := false
	for _, a := range dashboard.Assignments {
		if a.ID == item.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("assignment missing from dashboard")
	}
	input.Title = "Edited after assignment"
	call("PUT", lp, input, tc, 200)
	call("POST", lp+"/publish", map[string]any{}, tc, 200)
	call("POST", cp+"/archive", map[string]any{}, tc, 200)
	call("GET", lp, nil, sc, 404)
	var results domain.MaterialResults
	decode(call("GET", base+"/learning-results", nil, sc, 200), &results)
	if results.State != nil || results.Version.Title != "Material homework" || len(results.Version.Exercises) != 2 {
		t.Fatal("assignment snapshot mutated")
	}
	var run domain.LearningState
	action := domain.LearningActionInput{RequestID: id()}
	decode(call("POST", base+"/learning-attempt", action, sc, 201), &run)
	if run.Attempt.AssignmentID == nil || *run.Attempt.AssignmentID != item.ID || run.Attempt.TotalExercises != 2 {
		t.Fatal("assignment run missing")
	}
	var again domain.LearningState
	decode(call("POST", base+"/learning-attempt", action, sc, 201), &again)
	if again.Attempt.ID != run.Attempt.ID {
		t.Fatal("run duplicate")
	}
	rp := lp + "/learning-attempts/" + run.Attempt.ID
	call("GET", rp, nil, peer, 404)
	answer := func(e string, value string, want int) {
		t.Helper()
		call("POST", lp+"/exercises/"+e+"/attempts", domain.ExerciseAttemptInput{VersionID: run.Version.ID, LearningAttemptID: run.Attempt.ID, RequestID: id(), Answer: answerValue(value)}, sc, want)
	}
	answer(es[1].ID, "7", 404)
	call("POST", rp+"/exercises/"+es[1].ID+"/hints", domain.LearningActionInput{RequestID: id()}, sc, 404)
	call("POST", rp+"/complete", domain.LearningActionInput{RequestID: id()}, sc, 422)
	answer(es[0].ID, "9", 201)
	answer(es[0].ID, "7", 201)
	answer(es[2].ID, "My reasoning", 201)
	decode(call("POST", rp+"/complete", domain.LearningActionInput{RequestID: id()}, sc, 200), &run)
	decode(call("GET", base+"/learning-results?limit=1&offset=1", nil, tc, 200), &results)
	if results.Assignment.Status != "submitted" || results.State.Attempt.CorrectExercises != 1 || results.State.Attempt.IncorrectAttempts != 1 || results.History.Total != 3 || len(results.History.Items) != 1 {
		t.Fatal("results/history missing", results)
	}
	var manual domain.ExerciseAttempt
	for _, a := range run.Answers {
		if a.Kind == "long_text" {
			manual = a
		}
	}
	call("PUT", lp+"/attempts/"+manual.ID+"/review", domain.ExerciseReviewInput{Status: "correct", Feedback: "Good"}, tc, 200)
	decode(call("GET", base+"/learning-results", nil, tc, 200), &results)
	if results.Assignment.Status != "done" || results.State.Attempt.CorrectExercises != 2 {
		t.Fatal("manual result not synchronized")
	}
	call("POST", base+"/submit", map[string]string{"answer": "forged"}, sc, 404)
	call("POST", base+"/grade", map[string]any{"score": 100, "feedback": "fake"}, tc, 404)
	var repeated domain.Assignment
	rep := domain.MaterialRepeatInput{RequestID: id(), DueAt: due.Add(time.Hour)}
	decode(call("POST", base+"/repeat", rep, tc, 201), &repeated)
	if repeated.ID == item.ID || repeated.Material.RepeatOf == nil || *repeated.Material.RepeatOf != item.ID || repeated.Material.VersionID != item.Material.VersionID {
		t.Fatal("repeat lost source")
	}
	decode(call("POST", base+"/repeat", rep, tc, 201), &replay)
	if replay.ID != repeated.ID {
		t.Fatal("repeat duplicate")
	}
	// A dedicated exam cannot be used as a grading oracle through another assignment.
	examInput := domain.MaterialAssignmentInput{RequestID: id(), StudentID: student.ID, LessonID: lesson.ID, ExerciseIDs: []string{es[0].ID}, DueAt: due, Mode: "exam"}
	var exam domain.Assignment
	decode(call("POST", "/api/v1/assignments/material", examInput, tc, 201), &exam)
	examBase := "/api/v1/assignments/" + exam.ID
	var examRun domain.LearningState
	decode(call("POST", examBase+"/learning-attempt", domain.LearningActionInput{RequestID: id()}, sc, 201), &examRun)
	call("POST", "/api/v1/assignments/"+repeated.ID+"/learning-attempt", domain.LearningActionInput{RequestID: id()}, sc, 422)
	answer(es[0].ID, "7", 422)
	examRP := lp + "/learning-attempts/" + examRun.Attempt.ID
	call("POST", examRP+"/navigate", domain.LearningNavigateInput{RequestID: id(), Step: "exercise:" + es[0].ID}, sc, 200)
	call("POST", lp+"/exercises/"+es[0].ID+"/attempts", domain.ExerciseAttemptInput{VersionID: examRun.Version.ID, LearningAttemptID: examRun.Attempt.ID, RequestID: id(), Answer: answerValue("9")}, sc, 201)
	var hidden domain.MaterialResults
	decode(call("GET", examBase+"/learning-results", nil, tc, 200), &hidden)
	if hidden.State.Attempt.IncorrectAttempts != 0 || hidden.History.Items[0].Status != "submitted" || hidden.Stats[0].Mistakes != 0 {
		t.Fatal("exam assignment leaked grading")
	}
	call("POST", examRP+"/complete", domain.LearningActionInput{RequestID: id()}, sc, 200)
	decode(call("GET", examBase+"/learning-results", nil, tc, 200), &hidden)
	if hidden.State.Attempt.IncorrectAttempts != 1 || hidden.Stats[0].Mistakes != 1 {
		t.Fatal("exam final grading missing")
	}
	// The assigning tutor, not an unrelated material author, owns student results.
	var sharedCourse domain.Course
	decode(call("POST", "/api/v1/catalog/courses", domain.CourseInput{Title: "Shared author", Track: "school", GradeMin: 1, GradeMax: 11, Difficulty: 1, Visibility: "organization"}, otherTutor, 201), &sharedCourse)
	sharedCP := "/api/v1/catalog/courses/" + sharedCourse.ID
	var sharedTopic domain.Topic
	decode(call("POST", sharedCP+"/topics", domain.TopicInput{Title: "Shared"}, otherTutor, 201), &sharedTopic)
	sharedInput := input
	sharedInput.TopicID = sharedTopic.ID
	var sharedLesson domain.LearningLesson
	decode(call("POST", sharedCP+"/lessons", sharedInput, otherTutor, 201), &sharedLesson)
	sharedLP := "/api/v1/catalog/lessons/" + sharedLesson.ID
	var sharedExercise domain.Exercise
	decode(call("POST", sharedLP+"/exercises", domain.ExerciseInput{Kind: "long_text", Prompt: "Explain", Difficulty: 1, AnswerKey: domain.AnswerKeyInput{Answer: ""}}, otherTutor, 201), &sharedExercise)
	call("POST", sharedCP+"/publish", map[string]any{}, otherTutor, 200)
	call("POST", "/api/v1/catalog/topics/"+sharedTopic.ID+"/publish", map[string]any{}, otherTutor, 200)
	call("POST", sharedLP+"/publish", map[string]any{}, otherTutor, 200)
	var sharedAssignment domain.Assignment
	decode(call("POST", "/api/v1/assignments/material", domain.MaterialAssignmentInput{RequestID: id(), StudentID: student.ID, LessonID: sharedLesson.ID, DueAt: due}, tc, 201), &sharedAssignment)
	sharedBase := "/api/v1/assignments/" + sharedAssignment.ID
	var sharedRun domain.LearningState
	decode(call("POST", sharedBase+"/learning-attempt", domain.LearningActionInput{RequestID: id()}, sc, 201), &sharedRun)
	var sharedAnswer domain.ExerciseAttempt
	decode(call("POST", sharedLP+"/exercises/"+sharedExercise.ID+"/attempts", domain.ExerciseAttemptInput{VersionID: sharedRun.Version.ID, LearningAttemptID: sharedRun.Attempt.ID, RequestID: id(), Answer: answerValue("Reasoning")}, sc, 201), &sharedAnswer)
	call("POST", sharedLP+"/learning-attempts/"+sharedRun.Attempt.ID+"/complete", domain.LearningActionInput{RequestID: id()}, sc, 200)
	call("GET", sharedBase+"/learning-results", nil, otherTutor, 404)
	var authorHistory domain.CatalogPage[domain.ExerciseAttempt]
	decode(call("GET", sharedLP+"/attempts", nil, otherTutor, 200), &authorHistory)
	if authorHistory.Total != 0 {
		t.Fatal("unrelated author saw assignment results")
	}
	call("PUT", sharedLP+"/attempts/"+sharedAnswer.ID+"/review", domain.ExerciseReviewInput{Status: "correct"}, otherTutor, 404)
	call("PUT", sharedLP+"/attempts/"+sharedAnswer.ID+"/review", domain.ExerciseReviewInput{Status: "correct"}, tc, 200)
	var freeHistory domain.CatalogPage[domain.LearningAttempt]
	decode(call("GET", sharedLP+"/learning-attempts", nil, sc, 200), &freeHistory)
	if freeHistory.Total != 0 {
		t.Fatal("assigned run appeared in free practice history")
	}
	setInput.Title = "Edited reusable set"
	setInput.ExerciseIDs = []string{es[1].ID}
	call("PUT", "/api/v1/exercise-sets/"+set.ID, setInput, tc, 200)
	decode(call("GET", base+"/learning-results", nil, sc, 200), &results)
	if len(results.Version.Exercises) != 2 || results.Version.ID != item.Material.VersionID {
		t.Fatal("editing a set mutated an issued assignment")
	}
	call("DELETE", "/api/v1/exercise-sets/"+set.ID, nil, tc, 204)
	call("GET", base+"/learning-results", nil, sc, 200)
	foreign := student
	foreign.OrganizationID = id()
	if _, err := store.New(pool).MaterialResults(t.Context(), foreign, item.ID, domain.CatalogFilter{}); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatal("cross-organization results", err)
	}
	if _, err := store.New(pool).GetLearningState(t.Context(), foreign, lesson.ID, run.Attempt.ID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatal("cross-organization run", err)
	}
	// Active relationship is required even for previously granted snapshots.
	if _, err := pool.Exec(t.Context(), `update relations set status='blocked' where id=$1`, rel.ID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := pool.Exec(t.Context(), `update relations set status='active' where id=$1`, rel.ID); err != nil {
			t.Error(err)
		}
	}()
	call("GET", base+"/learning-results", nil, sc, 404)
	call("GET", rp, nil, sc, 404)
	answer(es[0].ID, "7", 404)
	decode(call("GET", "/api/v1/dashboard", nil, sc, 200), &dashboard)
	for _, a := range dashboard.Assignments {
		if a.ID == item.ID {
			t.Fatal("revoked assignment leaked through dashboard")
		}
	}
}
