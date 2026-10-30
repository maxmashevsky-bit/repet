package exercise

import (
	"encoding/json"
	"testing"
	"tutor-platform/services/core/internal/domain"
)

func TestVisualExercises(t *testing.T) {
	cases := []struct {
		kind, config, key string
		correct, wrong    domain.ExerciseAnswer
	}{
		{"number_line", `{"min":-5,"max":5,"step":1}`, "-2", scalar(" −2 "), scalar("3")},
		{"grid_select", `{"rows":3,"columns":4}`, `["0:0","2:3"]`, selection("2:3", "0:0"), selection("1:2")},
		{"matching", `{"left":[{"id":"a","label":"A"},{"id":"b","label":"B"}],"right":[{"id":"x","label":"X"},{"id":"y","label":"Y"}]}`, `["a:x","b:y"]`, selection("b:y", "a:x"), selection("a:y", "b:x")},
		{"ordering", `{"items":[{"id":"a","label":"3"},{"id":"b","label":"1"},{"id":"c","label":"2"}]}`, `["b","c","a"]`, selection("b", "c", "a"), selection("a", "b", "c")},
		{"geometry", `{"mode":"point","rows":5,"columns":5}`, `["2:3"]`, selection("2:3"), selection("3:2")},
		{"geometry", `{"mode":"connect","rows":5,"columns":5,"points":[{"id":"a","label":"A","x":0,"y":0},{"id":"b","label":"B","x":4,"y":0},{"id":"c","label":"C","x":2,"y":4}]}`, `["a:b","b:c"]`, selection("b:c", "a:b"), selection("a:c")},
		{"geometry", `{"mode":"select","rows":5,"columns":5,"points":[{"id":"a","label":"A","x":0,"y":0},{"id":"b","label":"B","x":4,"y":0},{"id":"c","label":"C","x":2,"y":4}],"objects":[{"id":"seg","label":"Отрезок","kind":"segment","point_ids":["a","b"]},{"id":"tri","label":"Треугольник","kind":"polygon","point_ids":["a","b","c"]}]}`, `["tri"]`, selection("tri"), selection("seg")},
	}
	for _, tc := range cases {
		t.Run(tc.kind+tc.config, func(t *testing.T) {
			spec := domain.Exercise{Kind: tc.kind, Config: json.RawMessage(tc.config)}
			if err := ValidateExerciseKey(spec, tc.key); err != nil {
				t.Fatal(err)
			}
			status, err := CheckExercise(spec, tc.key, tc.correct)
			if err != nil || status != "correct" {
				t.Fatal(status, err)
			}
			status, err = CheckExercise(spec, tc.key, tc.wrong)
			if err != nil || status != "incorrect" {
				t.Fatal(status, err)
			}
			for _, bad := range []domain.ExerciseAnswer{{}, selection("unknown"), selection("0:0", "0:0"), {Value: tc.correct.Value, Selected: []string{}}} {
				if ValidateExerciseAnswer(spec, bad) == nil {
					t.Fatal("invalid answer accepted", bad)
				}
			}
		})
	}
}
func scalar(value string) domain.ExerciseAnswer { return domain.ExerciseAnswer{Value: &value} }
func selection(values ...string) domain.ExerciseAnswer {
	return domain.ExerciseAnswer{Selected: values}
}
func TestVisualConfigAndBoundaries(t *testing.T) {
	for _, tc := range [][2]string{
		{"number_line", `{"min":0,"max":10,"step":3}`}, {"number_line", `{"min":0,"max":10,"step":0}`}, {"number_line", `{"min":0,"max":1000,"step":1}`}, {"number_line", `{"min":null,"max":10,"step":1}`},
		{"grid_select", `{"rows":9,"columns":2}`}, {"grid_select", `{"rows":2,"columns":2,"answer":["0:0"]}`}, {"grid_select", `{"rows":2}`},
		{"ordering", `{"items":[{"id":"a","label":"1"},{"id":"a","label":"2"}]}`},
		{"matching", `{"left":[{"id":"a","label":"A"},{"id":"b","label":"B"}],"right":[{"id":"x","label":"X"}]}`},
		{"geometry", `{"mode":"point","rows":5,"columns":5,"objects":[]}`},
		{"geometry", `{"mode":"connect","rows":5,"columns":5,"points":[{"id":"a","label":"A","y":0},{"id":"b","label":"B","x":2,"y":2}]}`},
		{"geometry", `{"mode":"connect","rows":5,"columns":5,"points":[{"id":"a","label":"A","x":null,"y":0},{"id":"b","label":"B","x":2,"y":2}]}`},
		{"geometry", `{"mode":"connect","rows":5,"columns":5,"points":[{"id":"a","label":"A","x":0,"y":0},{"id":"b","label":"B","x":0,"y":0}]}`},
		{"text", `{"answer":"secret"}`},
	} {
		if ValidateConfig(tc[0], json.RawMessage(tc[1])) == nil {
			t.Error("accepted config", tc)
		}
	}
	spec := domain.Exercise{Kind: "number_line", Config: json.RawMessage(`{"min":-4,"max":4,"step":2}`)}
	for _, answer := range []string{"-3", "5", "1/2", "2.0", "NaN", "1e2", "4/2"} {
		if ValidateExerciseAnswer(spec, scalar(answer)) == nil {
			t.Error("accepted coordinate", answer)
		}
	}
	grid := domain.Exercise{Kind: "grid_select", Config: json.RawMessage(`{"rows":2,"columns":3}`)}
	for _, answer := range []string{"2:0", "0:3", "-1:0", "01:0", "1:1:0"} {
		if ValidateExerciseAnswer(grid, selection(answer)) == nil {
			t.Error("accepted cell", answer)
		}
	}
}
