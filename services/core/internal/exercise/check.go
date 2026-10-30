package exercise

import (
	"encoding/json"
	"math/big"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
)

var decimal = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:[.,][0-9]+)?|[.,][0-9]+)$`)
var integer = regexp.MustCompile(`^[+-]?[0-9]+$`)

// math/big provides exact rational arithmetic; the grammar deliberately excludes expressions and exponents.
func ParseNumber(value string) (*big.Rat, error) {
	if len(value) > 128 {
		return nil, apperr.ErrBadRequest
	}
	value = strings.ReplaceAll(strings.TrimSpace(value), "−", "-")
	parts := strings.Split(value, "/")
	if len(parts) == 2 {
		a, b := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if !integer.MatchString(a) || !integer.MatchString(b) {
			return nil, apperr.ErrBadRequest
		}
		n, okN := new(big.Int).SetString(a, 10)
		d, okD := new(big.Int).SetString(b, 10)
		if !okN || !okD || d.Sign() == 0 {
			return nil, apperr.ErrBadRequest
		}
		return new(big.Rat).SetFrac(n, d), nil
	}
	if len(parts) != 1 || !decimal.MatchString(value) {
		return nil, apperr.ErrBadRequest
	}
	r, ok := new(big.Rat).SetString(strings.ReplaceAll(value, ",", "."))
	if !ok {
		return nil, apperr.ErrBadRequest
	}
	return r, nil
}

func validText(value string, max int) bool {
	return utf8.ValidString(value) && !strings.ContainsRune(value, 0) && len(value) <= max*4 && utf8.RuneCountInString(value) <= max && strings.TrimSpace(value) != ""
}

func NormalizeText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func ValidateAnswer(kind string, options []string, answer domain.ExerciseAnswer) error {
	if kind == "single_choice" || kind == "multiple_choice" {
		if answer.Value != nil || len(answer.Selected) == 0 || len(answer.Selected) > len(options) || (kind == "single_choice" && len(answer.Selected) != 1) {
			return apperr.ErrBadRequest
		}
		seen := map[string]bool{}
		for _, item := range answer.Selected {
			if seen[item] || !slices.Contains(options, item) {
				return apperr.ErrBadRequest
			}
			seen[item] = true
		}
		return nil
	}
	if answer.Value == nil || answer.Selected != nil {
		return apperr.ErrBadRequest
	}
	switch kind {
	case "numeric":
		_, err := ParseNumber(*answer.Value)
		return err
	case "text":
		if validText(*answer.Value, 1000) {
			return nil
		}
	case "long_text":
		if validText(*answer.Value, 20000) {
			return nil
		}
	}
	return apperr.ErrBadRequest
}

func ValidateKey(kind string, options []string, key string) error {
	if kind == "long_text" {
		return nil
	}
	if kind == "multiple_choice" {
		var selected []string
		if json.Unmarshal([]byte(key), &selected) != nil {
			return apperr.ErrBadRequest
		}
		return ValidateAnswer(kind, options, domain.ExerciseAnswer{Selected: selected})
	}
	if kind == "single_choice" {
		return ValidateAnswer(kind, options, domain.ExerciseAnswer{Selected: []string{key}})
	}
	return ValidateAnswer(kind, options, domain.ExerciseAnswer{Value: &key})
}

func Check(kind string, options []string, key string, answer domain.ExerciseAnswer) (string, error) {
	if err := ValidateAnswer(kind, options, answer); err != nil {
		return "", err
	}
	if kind == "long_text" {
		return "needs_review", nil
	}
	if ValidateKey(kind, options, key) != nil {
		return "", apperr.ErrInvalidState
	}
	correct := false
	switch kind {
	case "numeric":
		actual, _ := ParseNumber(*answer.Value)
		expected, _ := ParseNumber(key)
		correct = actual.Cmp(expected) == 0
	case "text":
		correct = NormalizeText(*answer.Value) == NormalizeText(key)
	case "single_choice":
		correct = answer.Selected[0] == key
	case "multiple_choice":
		var expected []string
		_ = json.Unmarshal([]byte(key), &expected)
		actual := slices.Clone(answer.Selected)
		slices.Sort(expected)
		slices.Sort(actual)
		correct = slices.Equal(actual, expected)
	}
	if correct {
		return "correct", nil
	}
	return "incorrect", nil
}
