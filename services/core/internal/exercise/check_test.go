package exercise

import (
	"strings"
	"testing"

	"tutor-platform/services/core/internal/domain"
)

func TestExactNumbers(t *testing.T) {
	for _, pair := range [][2]string{{"1/2", "0,5"}, {" -2 / -4 ", "+.5"}, {"−3 / 6", "-0.5"}, {"-0", "0/7"}, {"1 / -2", "-1/2"}, {"9007199254740993", "9007199254740993.0"}, {"  +12  ", "24/2"}, {"\u00a0-1/4\u00a0", "-0,25"}, {".000000000000000001", "1/1000000000000000000"}} {
		a, err := ParseNumber(pair[0])
		if err != nil {
			t.Fatalf("%q: %v", pair[0], err)
		}
		b, err := ParseNumber(pair[1])
		if err != nil || a.Cmp(b) != 0 {
			t.Errorf("%q != %q: %v", pair[0], pair[1], err)
		}
	}
	for _, value := range []string{"", " ", "1/0", "0/-0", "1//2", "1/2/3", "1 2", "1 1/2", "1.5/2", "--2", "- 2", "+", "1e3", "NaN", "Infinity", "1+2", "1,2.3", "1.", "½", "2%", "0x10", strings.Repeat("9", 129), "1\x00"} {
		if _, err := ParseNumber(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
	key := "9007199254740993"
	value := "9007199254740992"
	if status, err := Check("numeric", nil, key, domain.ExerciseAnswer{Value: &value}); err != nil || status != "incorrect" {
		t.Fatal(status, err)
	}
}

func TestAnswers(t *testing.T) {
	text := "\t РАВНЫЕ  \n углы \u00a0"
	if status, err := Check("text", nil, "равные углы", domain.ExerciseAnswer{Value: &text}); err != nil || status != "correct" {
		t.Fatal(status, err)
	}
	if status, err := Check("multiple_choice", []string{"a", "b", "c"}, `["a","c"]`, domain.ExerciseAnswer{Selected: []string{"c", "a"}}); err != nil || status != "correct" {
		t.Fatal(status, err)
	}
	if status, err := Check("multiple_choice", []string{"a", "b", "c"}, `["a","c"]`, domain.ExerciseAnswer{Selected: []string{"a"}}); err != nil || status != "incorrect" {
		t.Fatal(status, err)
	}
	for _, selected := range [][]string{nil, {}, {"a", "a"}, {"x"}} {
		if ValidateAnswer("multiple_choice", []string{"a", "b"}, domain.ExerciseAnswer{Selected: selected}) == nil {
			t.Error("accepted", selected)
		}
	}
	for _, value := range []string{"  ", "\x00", string([]byte{255}), strings.Repeat("я", 1001)} {
		if ValidateAnswer("text", nil, domain.ExerciseAnswer{Value: &value}) == nil {
			t.Error("accepted invalid text")
		}
	}
	if ValidateAnswer("text", nil, domain.ExerciseAnswer{Value: &text, Selected: []string{}}) == nil {
		t.Fatal("mixed answer accepted")
	}
	if ValidateKey("multiple_choice", []string{"a", "b"}, `["a","a"]`) == nil {
		t.Fatal("duplicate key accepted")
	}
	if ValidateKey("numeric", nil, "1/0") == nil {
		t.Fatal("invalid numeric key accepted")
	}
	if status, err := Check("long_text", nil, "", domain.ExerciseAnswer{Value: &text}); err != nil || status != "needs_review" {
		t.Fatal(status, err)
	}
	long := strings.Repeat("я", 20001)
	if ValidateAnswer("long_text", nil, domain.ExerciseAnswer{Value: &long}) == nil {
		t.Fatal("oversized solution accepted")
	}
}
