package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

func testPlayerHTTP(t *testing.T, call func(string, string, any, *http.Cookie, int) []byte, decode func([]byte, any), tc, sc, peer *http.Cookie) {
	t.Helper()
	newID := func() string {
		id, err := platform.NewUUID()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	var c domain.Course
	decode(call("POST", "/api/v1/catalog/courses", domain.CourseInput{Title: "Player", Track: "school", GradeMin: 1, GradeMax: 11, Difficulty: 1}, tc, 201), &c)
	cp := "/api/v1/catalog/courses/" + c.ID
	var topic domain.Topic
	decode(call("POST", cp+"/topics", domain.TopicInput{Title: "Player"}, tc, 201), &topic)
	input := domain.LearningLessonInput{TopicID: topic.ID, Title: "Steps", Content: "Explanation", Track: "school", GradeMin: 1, GradeMax: 11, Difficulty: 1, Example: &domain.LearningExample{Title: "Two plus one", Explanation: "Move along the line", Min: 0, Max: 10, Steps: []domain.LearningExampleStep{{Text: "Start", Value: 2}, {Text: "One step", Value: 3}}}}
	bad := input
	badExample := *input.Example
	badExample.Steps = []domain.LearningExampleStep{{Text: "Outside", Value: 999}, {Text: "Next", Value: 2}}
	bad.Example = &badExample
	call("POST", cp+"/lessons", bad, tc, 400)
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, defect := range []string{"missing", "null", "unknown"} {
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatal(err)
		}
		example := body["example"].(map[string]any)
		switch defect {
		case "missing":
			delete(example, "min")
		case "null":
			example["min"] = nil
		case "unknown":
			example["answer_key"] = "forged"
		}
		call("POST", cp+"/lessons", body, tc, 400)
	}
	var lesson domain.LearningLesson
	decode(call("POST", cp+"/lessons", input, tc, 201), &lesson)
	lp := "/api/v1/catalog/lessons/" + lesson.ID
	rp := lp + "/learning-attempts"
	var exercises [2]domain.Exercise
	for i := range exercises {
		decode(call("POST", lp+"/exercises", domain.ExerciseInput{Kind: "numeric", Prompt: "Seven", Difficulty: 1, Position: i, AnswerKey: domain.AnswerKeyInput{Answer: "7", Explanation: "PRIVATE_SOLUTION", Hints: []string{"A hint"}}}, tc, 201), &exercises[i])
	}
	call("POST", cp+"/publish", map[string]any{}, tc, 200)
	call("POST", "/api/v1/catalog/topics/"+topic.ID+"/publish", map[string]any{}, tc, 200)
	call("POST", lp+"/publish", map[string]any{}, tc, 200)
	edited := input
	editedExample := *input.Example
	editedExample.Title = "Unpublished change"
	edited.Example = &editedExample
	call("PUT", lp, edited, tc, 200)
	var public domain.LearningLesson
	decode(call("GET", lp, nil, sc, 200), &public)
	if public.Version.Example == nil || public.Version.Example.Title != input.Example.Title {
		t.Fatal("draft edit changed the published example")
	}
	for _, mode := range []string{"practice", "exam"} {
		t.Run(mode, func(t *testing.T) {
			var state domain.LearningState
			decode(call("POST", rp, domain.LearningStartInput{RequestID: newID(), Mode: mode, Restart: mode == "exam"}, sc, 201), &state)
			run := rp + "/" + state.Attempt.ID
			if state.Attempt.Mode != mode || state.Attempt.PlayerStep != "theory" || state.Version.Example == nil {
				t.Fatal("start", state.Attempt)
			}
			nav := func(step string, want int) {
				decode(call("POST", run+"/navigate", domain.LearningNavigateInput{RequestID: newID(), Step: step}, sc, want), &state)
			}
			nav("exercise:"+exercises[1].ID, 422)
			call("POST", run+"/navigate", domain.LearningNavigateInput{RequestID: newID(), Step: "example"}, peer, 404)
			nav("example", 200)
			nav("exercise:"+exercises[0].ID, 200)
			nonce := newID()
			n := domain.LearningNavigateInput{RequestID: nonce, Step: "exercise:" + exercises[0].ID}
			call("POST", run+"/navigate", n, sc, 200)
			call("POST", run+"/navigate", n, sc, 200)
			n.Step = "theory"
			call("POST", run+"/navigate", n, sc, 409)
			decode(call("GET", run, nil, sc, 200), &state)
			if state.Attempt.PlayerStep != "exercise:"+exercises[0].ID {
				t.Fatal("cursor missing")
			}
			solution := run + "/exercises/" + exercises[0].ID + "/solution"
			call("GET", solution, nil, sc, 422)
			call("GET", solution, nil, peer, 404)
			for i, e := range exercises {
				if i > 0 {
					nav("exercise:"+e.ID, 200)
					n.Step = "exercise:" + exercises[0].ID
					var replay domain.LearningState
					decode(call("POST", run+"/navigate", n, sc, 200), &replay)
					if replay.Attempt.PlayerStep != "exercise:"+e.ID {
						t.Fatal("navigation replay moved the cursor backwards")
					}
				}
				in := domain.ExerciseAttemptInput{VersionID: state.Version.ID, LearningAttemptID: state.Attempt.ID, RequestID: newID(), Answer: answerValue("7")}
				var a domain.ExerciseAttempt
				decode(call("POST", lp+"/exercises/"+e.ID+"/attempts", in, sc, 201), &a)
				if mode == "exam" {
					if a.Status != "submitted" || a.Feedback != "" || a.ReviewedAt != nil {
						t.Fatal("exam result leaked", a)
					}
					call("POST", lp+"/exercises/"+e.ID+"/attempts", in, sc, 201)
					in.RequestID = newID()
					call("POST", lp+"/exercises/"+e.ID+"/attempts", in, sc, 422)
					in.LearningAttemptID = ""
					call("POST", lp+"/exercises/"+e.ID+"/attempts", in, sc, 422)
					call("GET", solution, nil, sc, 422)
					call("POST", run+"/exercises/"+e.ID+"/hints", domain.LearningActionInput{RequestID: newID()}, sc, 422)
					for _, path := range []string{run, run + "/answers", lp + "/attempts", rp} {
						payload := string(call("GET", path, nil, sc, 200))
						if strings.Contains(payload, "PRIVATE_SOLUTION") {
							t.Fatal("exam projection leaked", path, payload)
						}
					}
					var hidden domain.LearningState
					decode(call("GET", run, nil, sc, 200), &hidden)
					if hidden.Attempt.CorrectExercises != 0 || hidden.Attempt.IncorrectAttempts != 0 {
						t.Fatal("exam counts leaked")
					}
					for _, answer := range hidden.Answers {
						if answer.Status != "submitted" {
							t.Fatal("exam answer status leaked")
						}
					}
				} else {
					call("GET", solution, nil, sc, 200)
				}
			}
			nav("results", 422)
			decode(call("POST", run+"/complete", domain.LearningActionInput{RequestID: newID()}, sc, 200), &state)
			if state.Attempt.PlayerStep != "results" || state.Attempt.CorrectExercises != 2 {
				t.Fatal("results missing", state.Attempt)
			}
			var resolved domain.LearningSolution
			decode(call("GET", solution, nil, sc, 200), &resolved)
			if !resolved.Available || resolved.Text != "PRIVATE_SOLUTION" {
				t.Fatal("missing explanation", resolved)
			}
		})
	}
}
