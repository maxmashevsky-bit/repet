package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

func testVisualHTTP(t *testing.T, call func(string, string, any, *http.Cookie, int) []byte, decode func([]byte, any), tc, sc *http.Cookie) {
	t.Helper()
	newID := func() string {
		id, err := platform.NewUUID()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	base := "/api/v1/catalog"
	var course domain.Course
	decode(call("POST", base+"/courses", domain.CourseInput{Title: "Visual math", Track: "school", GradeMin: 1, GradeMax: 11, Difficulty: 2}, tc, 201), &course)
	var topic domain.Topic
	decode(call("POST", base+"/courses/"+course.ID+"/topics", domain.TopicInput{Title: "Objects"}, tc, 201), &topic)
	var lesson domain.LearningLesson
	decode(call("POST", base+"/courses/"+course.ID+"/lessons", domain.LearningLessonInput{TopicID: topic.ID, Title: "Visual types", Content: "Interact", Track: "school", GradeMin: 1, GradeMax: 11, Difficulty: 2}, tc, 201), &lesson)
	lp := base + "/lessons/" + lesson.ID
	points := `[{"id":"a","label":"A","x":0,"y":0},{"id":"b","label":"B","x":4,"y":0},{"id":"c","label":"C","x":2,"y":4}]`
	cases := []struct {
		kind, config, key string
		good, bad         domain.ExerciseAnswer
	}{
		{"number_line", `{"min":-5,"max":5,"step":1}`, "2", answerValue("2"), answerValue("-2")},
		{"grid_select", `{"rows":3,"columns":3}`, `["0:0","2:2"]`, answerSet("2:2", "0:0"), answerSet("0:1")},
		{"matching", `{"left":[{"id":"a","label":"A"},{"id":"b","label":"B"}],"right":[{"id":"x","label":"X"},{"id":"y","label":"Y"}]}`, `["a:x","b:y"]`, answerSet("b:y", "a:x"), answerSet("a:y", "b:x")},
		{"ordering", `{"items":[{"id":"a","label":"3"},{"id":"b","label":"1"},{"id":"c","label":"2"}]}`, `["b","c","a"]`, answerSet("b", "c", "a"), answerSet("a", "b", "c")},
		{"geometry", `{"mode":"point","rows":5,"columns":5}`, `["2:3"]`, answerSet("2:3"), answerSet("3:2")},
		{"geometry", `{"mode":"connect","rows":5,"columns":5,"points":` + points + `}`, `["a:b","b:c"]`, answerSet("b:c", "a:b"), answerSet("a:c")},
		{"geometry", `{"mode":"select","rows":5,"columns":5,"points":` + points + `,"objects":[{"id":"seg","label":"Segment","kind":"segment","point_ids":["a","b"]},{"id":"tri","label":"Triangle","kind":"polygon","point_ids":["a","b","c"]}]}`, `["tri"]`, answerSet("tri"), answerSet("seg")},
	}
	exercises := make([]domain.Exercise, len(cases))
	for i, tcx := range cases {
		in := domain.ExerciseInput{Kind: tcx.kind, Prompt: "Visual " + tcx.kind, Difficulty: 2, Position: i, Config: json.RawMessage(tcx.config), AnswerKey: domain.AnswerKeyInput{Answer: tcx.key, Explanation: "VISUAL_PRIVATE_KEY"}}
		noConfig := in
		noConfig.Config = nil
		call("POST", lp+"/exercises", noConfig, tc, 400)
		forged := in
		var config map[string]any
		decode(in.Config, &config)
		config["answer"] = "forged"
		forged.Config, _ = json.Marshal(config)
		call("POST", lp+"/exercises", forged, tc, 400)
		decode(call("POST", lp+"/exercises", in, tc, 201), &exercises[i])
	}
	call("POST", base+"/courses/"+course.ID+"/publish", map[string]any{}, tc, 200)
	call("POST", base+"/topics/"+topic.ID+"/publish", map[string]any{}, tc, 200)
	decode(call("POST", lp+"/publish", map[string]any{}, tc, 200), &lesson)
	public := call("GET", lp, nil, sc, 200)
	if strings.Contains(string(public), "VISUAL_PRIVATE_KEY") || strings.Contains(string(public), "answer_key") {
		t.Fatal("visual keys leaked")
	}
	for i, tcx := range cases {
		path := lp + "/exercises/" + exercises[i].ID + "/attempts"
		for _, test := range []struct {
			answer domain.ExerciseAnswer
			want   string
		}{{tcx.good, "correct"}, {tcx.bad, "incorrect"}} {
			var attempt domain.ExerciseAttempt
			decode(call("POST", path, domain.ExerciseAttemptInput{VersionID: lesson.Version.ID, RequestID: newID(), Answer: test.answer}, sc, 201), &attempt)
			if attempt.Status != test.want {
				t.Fatalf("%s got %s want %s", tcx.kind, attempt.Status, test.want)
			}
			call("PUT", lp+"/attempts/"+attempt.ID+"/review", domain.ExerciseReviewInput{Status: "correct"}, tc, 422)
		}
		call("POST", path, domain.ExerciseAttemptInput{VersionID: lesson.Version.ID, RequestID: newID(), Answer: answerSet("unknown")}, sc, 400)
	}
	// Configuration and keys are copied together; draft editing cannot affect a running publication.
	call("PUT", lp+"/exercises/"+exercises[0].ID, domain.ExerciseInput{Kind: "number_line", Prompt: "New line", Difficulty: 2, Config: json.RawMessage(`{"min":-10,"max":10,"step":1}`), AnswerKey: domain.AnswerKeyInput{Answer: "7"}}, tc, 200)
	var old domain.ExerciseAttempt
	requestID := newID()
	path := lp + "/exercises/" + exercises[0].ID + "/attempts"
	decode(call("POST", path, domain.ExerciseAttemptInput{VersionID: lesson.Version.ID, RequestID: requestID, Answer: answerValue("2")}, sc, 201), &old)
	if old.Status != "correct" {
		t.Fatal("draft changed published rules")
	}
	call("POST", path, domain.ExerciseAttemptInput{VersionID: lesson.Version.ID, RequestID: newID(), Answer: answerValue("7")}, sc, 400)
	var next domain.LearningLesson
	decode(call("POST", lp+"/publish", map[string]any{}, tc, 200), &next)
	call("POST", path, domain.ExerciseAttemptInput{VersionID: lesson.Version.ID, RequestID: newID(), Answer: answerValue("2")}, sc, 409)
	var updated domain.ExerciseAttempt
	decode(call("POST", path, domain.ExerciseAttemptInput{VersionID: next.Version.ID, RequestID: newID(), Answer: answerValue("7")}, sc, 201), &updated)
	if updated.Status != "correct" {
		t.Fatal("new config not applied")
	}
	var replay domain.ExerciseAttempt
	decode(call("POST", path, domain.ExerciseAttemptInput{VersionID: lesson.Version.ID, RequestID: requestID, Answer: answerValue("2")}, sc, 201), &replay)
	if replay.ID != old.ID || replay.Status != "correct" {
		t.Fatal("old attempt changed")
	}
}
func answerValue(value string) domain.ExerciseAnswer { return domain.ExerciseAnswer{Value: &value} }
func answerSet(values ...string) domain.ExerciseAnswer {
	return domain.ExerciseAnswer{Selected: values}
}
