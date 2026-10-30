package store

import (
	"testing"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

func TestBankFiltersAndNonces(t *testing.T) {
	for _, direction := range []string{"school", "olympiad", "oge", "ege_basic", "ege_profile"} {
		for _, grade := range []int{5, 11} {
			f := domain.BankFilter{Direction: direction, Grade: grade, ExamNumber: 100, Limit: 100}
			if err := ValidateBankFilter(&f); err != nil {
				t.Fatal(direction, grade, err)
			}
		}
	}
	for _, f := range []domain.BankFilter{{Grade: 4}, {Grade: 12}, {Limit: 101}, {Limit: -1}, {Offset: -1}, {ExamNumber: 101}, {Direction: "ege"}, {Kind: "unknown"}} {
		if ValidateBankFilter(&f) == nil {
			t.Fatal("accepted invalid filter", f)
		}
	}
	f := domain.BankFilter{Query: "  число  "}
	if err := ValidateBankFilter(&f); err != nil || f.Limit != 20 || f.Query != "число" {
		t.Fatal("filter defaults", f, err)
	}
	a := bankChildRequest("batch", "student-a")
	if !platform.ValidUUID(a) || a != bankChildRequest("batch", "student-a") || a == bankChildRequest("batch", "student-b") || a == bankChildRequest("other", "student-a") {
		t.Fatal("child nonce")
	}
}
