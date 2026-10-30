package httpapi

import (
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

func testLearningHTTP(t *testing.T, call func(string, string, any, *http.Cookie, int) []byte, decode func([]byte, any), tc, sc, peer, otherTutor *http.Cookie) {
	t.Helper()
	id := func() string {
		v, err := platform.NewUUID()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	var course domain.Course
	decode(call("POST", "/api/v1/catalog/courses", domain.CourseInput{Title: "Learning progress", Track: "school", GradeMin: 1, GradeMax: 11, Difficulty: 1}, tc, 201), &course)
	cp := "/api/v1/catalog/courses/" + course.ID
	var topic domain.Topic
	decode(call("POST", cp+"/topics", domain.TopicInput{Title: "Progress"}, tc, 201), &topic)
	var lesson domain.LearningLesson
	lessonInput := domain.LearningLessonInput{TopicID: topic.ID, Title: "Resume me", Content: "Pinned content", Track: "school", GradeMin: 1, GradeMax: 11, Difficulty: 1}
	decode(call("POST", cp+"/lessons", lessonInput, tc, 201), &lesson)
	lp := "/api/v1/catalog/lessons/" + lesson.ID
	rp := lp + "/learning-attempts"
	inputs := []domain.ExerciseInput{
		{Kind: "numeric", Prompt: "One", Difficulty: 1, AnswerKey: domain.AnswerKeyInput{Answer: "1", Hints: []string{"Private hint one", "Private hint two"}}},
		{Kind: "numeric", Prompt: "Two", Difficulty: 1, Position: 1, AnswerKey: domain.AnswerKeyInput{Answer: "2"}},
		{Kind: "text", Prompt: "Three", Difficulty: 1, Position: 2, AnswerKey: domain.AnswerKeyInput{Answer: "three"}},
		{Kind: "long_text", Prompt: "Explain", Difficulty: 1, Position: 3},
	}
	exercises := make([]domain.Exercise, len(inputs))
	for i, in := range inputs {
		decode(call("POST", lp+"/exercises", in, tc, 201), &exercises[i])
	}
	if exercises[0].HintCount != 2 {
		t.Fatal("create response hint count", exercises[0].HintCount)
	}
	call("POST", cp+"/publish", map[string]any{}, tc, 200)
	call("POST", "/api/v1/catalog/topics/"+topic.ID+"/publish", map[string]any{}, tc, 200)
	decode(call("POST", lp+"/publish", map[string]any{}, tc, 200), &lesson)
	call("GET", rp+"/current", nil, sc, 404)
	call("POST", rp, map[string]any{"request_id": id(), "student_id": id()}, sc, 400)
	start := domain.LearningStartInput{RequestID: id()}
	var state domain.LearningState
	decode(call("POST", rp, start, sc, 201), &state)
	run := state.Attempt.ID
	version := state.Version.ID
	runPath := rp + "/" + run
	if state.Attempt.TotalExercises != 4 || state.Attempt.Number != 1 || len(state.Answers) != 0 || len(state.Hints) != 0 {
		t.Fatalf("bad start %+v", state)
	}
	public := string(call("GET", lp, nil, sc, 200))
	if strings.Contains(public, "Private hint") || strings.Contains(public, "answer_key") {
		t.Fatal("hint/key leaked")
	}
	var resumed domain.LearningState
	decode(call("POST", rp, start, sc, 201), &resumed)
	if resumed.Attempt.ID != run {
		t.Fatal("start retry duplicated run")
	}
	call("POST", rp, domain.LearningStartInput{RequestID: start.RequestID, Restart: true}, sc, 409)
	var wg sync.WaitGroup
	results := make(chan string, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var r domain.LearningState
			decode(call("POST", rp, domain.LearningStartInput{RequestID: id()}, sc, 201), &r)
			results <- r.Attempt.ID
		}()
	}
	wg.Wait()
	close(results)
	for result := range results {
		if result != run {
			t.Fatal("concurrent start duplicated run")
		}
	}
	call("GET", runPath, nil, nil, 401)
	call("GET", runPath, nil, peer, 404)
	call("GET", runPath, nil, otherTutor, 403)
	call("GET", runPath+"/answers", nil, peer, 404)
	call("GET", rp+"/current/answers", nil, sc, 400)
	call("POST", lp+"/exercises/"+exercises[0].ID+"/attempts", map[string]any{"version_id": version, "request_id": id(), "learning_attempt_id": run, "answer": map[string]string{"value": "1"}, "elapsed_seconds": 0}, sc, 400)
	call("POST", runPath+"/complete", domain.LearningActionInput{RequestID: id()}, peer, 404)
	call("POST", runPath+"/complete", domain.LearningActionInput{RequestID: id()}, sc, 422)
	hintPath := runPath + "/exercises/" + exercises[0].ID + "/hints"
	hint := domain.LearningActionInput{RequestID: id()}
	decode(call("POST", hintPath, hint, sc, 200), &resumed)
	decode(call("POST", hintPath, hint, sc, 200), &resumed)
	if len(resumed.Hints) != 1 || resumed.Hints[0].Text != "Private hint one" {
		t.Fatal("hint retry", resumed.Hints)
	}
	call("POST", runPath+"/exercises/"+exercises[1].ID+"/hints", hint, sc, 409)
	call("POST", hintPath, domain.LearningActionInput{RequestID: id()}, sc, 200)
	call("POST", hintPath, domain.LearningActionInput{RequestID: id()}, sc, 422)
	time.Sleep(1100 * time.Millisecond)
	submit := func(i int, value string, requestID string, want int) domain.ExerciseAttempt {
		var a domain.ExerciseAttempt
		decode(call("POST", lp+"/exercises/"+exercises[i].ID+"/attempts", domain.ExerciseAttemptInput{VersionID: version, LearningAttemptID: run, RequestID: requestID, Answer: answerValue(value)}, sc, want), &a)
		return a
	}
	nonce := id()
	first := submit(0, "9", nonce, 201)
	again := submit(0, "9", nonce, 201)
	if first.ID != again.ID || first.AttemptNumber != 1 || first.HintsUsed != 2 || first.ElapsedSeconds < 1 || first.Status != "incorrect" {
		t.Fatalf("attempt metadata/retry %+v %+v", first, again)
	}
	submit(0, "1", nonce, 409)
	correct := submit(0, "1", id(), 201)
	if correct.AttemptNumber != 2 || correct.Status != "correct" {
		t.Fatal("attempt counter", correct)
	}
	submit(1, "2", id(), 201)
	submit(2, " THREE ", id(), 201)
	decode(call("GET", rp+"/current", nil, sc, 200), &resumed)
	if resumed.Attempt.AnsweredExercises != 3 || resumed.Attempt.IncorrectAttempts != 1 || len(resumed.Answers) != 3 {
		t.Fatal("resume missing three answers", resumed.Attempt)
	}
	var history domain.CatalogPage[domain.ExerciseAttempt]
	decode(call("GET", runPath+"/answers?limit=2&offset=0", nil, sc, 200), &history)
	if history.Total != 4 || len(history.Items) != 2 {
		t.Fatal("history pagination", history)
	}
	// Editing and republishing must not change the active run or its key/hints.
	inputs[0].AnswerKey.Answer = "100"
	inputs[0].AnswerKey.Hints = []string{"Replacement hint"}
	call("PUT", lp+"/exercises/"+exercises[0].ID, inputs[0], tc, 200)
	lessonInput.Content = "New content"
	call("PUT", lp, lessonInput, tc, 200)
	call("POST", lp+"/publish", map[string]any{}, tc, 200)
	decode(call("POST", rp, domain.LearningStartInput{RequestID: id()}, sc, 201), &resumed)
	if resumed.Version.ID != version || resumed.Version.Content != "Pinned content" {
		t.Fatal("published edit replaced active run")
	}
	if a := submit(0, "1", id(), 201); a.Status != "correct" {
		t.Fatal("old key changed")
	}
	manual := submit(3, "A complete explanation", id(), 201)
	if manual.Status != "needs_review" {
		t.Fatal("manual grading lost")
	}
	call("POST", runPath+"/complete", map[string]any{"request_id": id(), "elapsed_seconds": 0}, sc, 400)
	finish := domain.LearningActionInput{RequestID: id()}
	decode(call("POST", runPath+"/complete", finish, sc, 200), &resumed)
	completedAt := *resumed.Attempt.CompletedAt
	decode(call("POST", runPath+"/complete", finish, sc, 200), &resumed)
	if !resumed.Attempt.CompletedAt.Equal(completedAt) || resumed.Attempt.PendingReview != 1 {
		t.Fatal("completion retry/manual", resumed.Attempt)
	}
	submit(1, "2", id(), 422)
	submit(0, "9", nonce, 201)
	call("PUT", lp+"/attempts/"+manual.ID+"/review", domain.ExerciseReviewInput{Status: "correct", Feedback: "Reviewed"}, tc, 200)
	decode(call("GET", runPath, nil, sc, 200), &resumed)
	if resumed.Attempt.PendingReview != 0 || resumed.Attempt.CorrectExercises != 4 {
		t.Fatal("review not reflected", resumed.Attempt)
	}
	restart := domain.LearningStartInput{RequestID: id(), Restart: true}
	decode(call("POST", rp, restart, sc, 201), &resumed)
	newRun := resumed.Attempt.ID
	if newRun == run || resumed.Attempt.Number != 2 || resumed.Version.ID == version || len(resumed.Answers) != 0 || len(resumed.Hints) != 0 {
		t.Fatal("restart version/reset", resumed)
	}
	decode(call("POST", rp, restart, sc, 201), &resumed)
	if resumed.Attempt.ID != newRun {
		t.Fatal("restart retry duplicated")
	}
	var own domain.CatalogPage[domain.LearningAttempt]
	decode(call("GET", rp+"?limit=1&offset=1", nil, sc, 200), &own)
	if own.Total != 2 || len(own.Items) != 1 || own.Items[0].ID != run {
		t.Fatal("own results", own)
	}
	decode(call("GET", rp, nil, peer, 200), &own)
	if own.Total != 0 {
		t.Fatal("results leaked")
	}
	call("POST", "/api/v1/catalog/topics/"+topic.ID+"/archive", map[string]any{}, tc, 200)
	call("GET", runPath, nil, sc, 404)
	call("POST", runPath+"/complete", finish, sc, 404)
}
