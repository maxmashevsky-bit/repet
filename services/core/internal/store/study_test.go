package store

import (
	"reflect"
	"testing"
	"tutor-platform/services/core/internal/domain"
)

func TestStudyVariant(t *testing.T) {
	choices := []studyChoice{{Key: "a", LogicalKey: "source-a", Direction: "oge", Number: 7, Topic: "Numbers"}, {Key: "b", LogicalKey: "source-b", Direction: "oge", Number: 7, Topic: "Numbers"}, {Key: "c", LogicalKey: "source-c", Direction: "oge", Number: 20, Topic: "Proof"}, {Key: "d", LogicalKey: "source-d", Direction: "ege_profile", Number: 7, Topic: "Numbers"}}
	before := append([]studyChoice(nil), choices...)
	spec := domain.ExamSpecification{ID: "version-one", Direction: "oge", Slots: []domain.ExamSlot{{Number: 7, Count: 2, Points: 1}, {Number: 20, Count: 1, Points: 3}}}
	a, points, err := studyVariant(choices, spec, "seed", "exam", "", 0, nil)
	if err != nil || len(a) != 3 || points["c"] != 3 {
		t.Fatal(a, points, err)
	}
	b, _, _ := studyVariant(choices, spec, "seed", "exam", "", 0, nil)
	if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(before, choices) {
		t.Fatal("unstable/mutated variant")
	}
	if _, _, err = studyVariant(choices, spec, "seed", "practice", "Proof", 2, nil); err == nil {
		t.Fatal("insufficient pool accepted")
	}
	a, _, err = studyVariant(choices, spec, "seed", "practice", "", 1, map[string]bool{"b": true})
	if err != nil || len(a) != 1 || a[0] != "b" {
		t.Fatal("repeat filter", a, err)
	}
	n := 7
	collection := domain.BankCollection{ExerciseIDs: []string{"one", "two"}, Provenance: []domain.BankExercise{{LessonID: "source", Exercise: domain.Exercise{ID: "same"}, BankMetadata: domain.BankMetadata{ExamNumber: &n}}, {LessonID: "source", Exercise: domain.Exercise{ID: "same"}, BankMetadata: domain.BankMetadata{ExamNumber: &n}}}}
	if len(studyChoices(collection)) != 1 {
		t.Fatal("logical duplicate not removed")
	}
}
