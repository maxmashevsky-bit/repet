package exercise

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
)

type NumberLineConfig struct {
	Min  int `json:"min"`
	Max  int `json:"max"`
	Step int `json:"step"`
}
type GridConfig struct {
	Rows    int `json:"rows"`
	Columns int `json:"columns"`
}
type VisualItem struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}
type MatchingConfig struct {
	Left  []VisualItem `json:"left"`
	Right []VisualItem `json:"right"`
}
type OrderingConfig struct {
	Items []VisualItem `json:"items"`
}
type GeometryPoint struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
}
type GeometryObject struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Kind     string   `json:"kind"`
	PointIDs []string `json:"point_ids"`
}
type GeometryConfig struct {
	Mode    string           `json:"mode"`
	Rows    int              `json:"rows"`
	Columns int              `json:"columns"`
	Points  []GeometryPoint  `json:"points,omitempty"`
	Objects []GeometryObject `json:"objects,omitempty"`
}

var visualID = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,15}$`)

func (p *GeometryPoint) UnmarshalJSON(raw []byte) error {
	type plain GeometryPoint
	var value plain
	if err := configDecode(raw, &value, "id", "label", "x", "y"); err != nil {
		return err
	}
	*p = GeometryPoint(value)
	return nil
}
func IsVisual(kind string) bool {
	return slices.Contains([]string{"number_line", "grid_select", "matching", "ordering", "geometry"}, kind)
}

func configDecode(raw json.RawMessage, out any, required ...string) error {
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	if len(raw) > 32000 {
		return apperr.ErrBadRequest
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return apperr.ErrBadRequest
	}
	for _, key := range required {
		if value, ok := fields[key]; !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return apperr.ErrBadRequest
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(out) != nil {
		return apperr.ErrBadRequest
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return apperr.ErrBadRequest
	}
	return nil
}
func validItems(items []VisualItem, min, max int) bool {
	if len(items) < min || len(items) > max {
		return false
	}
	ids, labels := map[string]bool{}, map[string]bool{}
	for _, item := range items {
		if !visualID.MatchString(item.ID) || !validText(item.Label, 80) || ids[item.ID] || labels[item.Label] {
			return false
		}
		ids[item.ID] = true
		labels[item.Label] = true
	}
	return true
}
func hasItem(items []VisualItem, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}
func ValidateConfig(kind string, raw json.RawMessage) error {
	bad := apperr.ErrBadRequest
	switch kind {
	case "number_line":
		var c NumberLineConfig
		if configDecode(raw, &c, "min", "max", "step") != nil || c.Min < -1000 || c.Max > 1000 || c.Max <= c.Min || c.Step < 1 || c.Step > 100 || (c.Max-c.Min)%c.Step != 0 || (c.Max-c.Min)/c.Step > 100 {
			return bad
		}
	case "grid_select":
		var c GridConfig
		if configDecode(raw, &c, "rows", "columns") != nil || c.Rows < 1 || c.Rows > 8 || c.Columns < 1 || c.Columns > 8 {
			return bad
		}
	case "matching":
		var c MatchingConfig
		if configDecode(raw, &c, "left", "right") != nil || !validItems(c.Left, 2, 8) || !validItems(c.Right, 2, 8) || len(c.Left) != len(c.Right) {
			return bad
		}
	case "ordering":
		var c OrderingConfig
		if configDecode(raw, &c, "items") != nil || !validItems(c.Items, 2, 12) {
			return bad
		}
	case "geometry":
		var c GeometryConfig
		if configDecode(raw, &c, "mode", "rows", "columns") != nil || c.Rows < 2 || c.Rows > 8 || c.Columns < 2 || c.Columns > 8 {
			return bad
		}
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(raw, &fields)
		if c.Mode == "point" {
			if len(fields) != 3 {
				return bad
			}
			return nil
		}
		if c.Mode != "connect" && c.Mode != "select" {
			return bad
		}
		if c.Mode == "connect" && len(fields) != 4 || c.Mode == "select" && len(fields) != 5 {
			return bad
		}
		if len(c.Points) < 2 || len(c.Points) > 12 {
			return bad
		}
		ids, positions, labels := map[string]bool{}, map[string]bool{}, map[string]bool{}
		for _, p := range c.Points {
			pos := strconv.Itoa(p.X) + ":" + strconv.Itoa(p.Y)
			if !visualID.MatchString(p.ID) || !validText(p.Label, 80) || p.X < 0 || p.X >= c.Columns || p.Y < 0 || p.Y >= c.Rows || ids[p.ID] || positions[pos] || labels[p.Label] {
				return bad
			}
			ids[p.ID] = true
			positions[pos] = true
			labels[p.Label] = true
		}
		if c.Mode == "select" {
			if len(c.Objects) < 2 || len(c.Objects) > 12 {
				return bad
			}
			seen, objectLabels := map[string]bool{}, map[string]bool{}
			for _, object := range c.Objects {
				if !visualID.MatchString(object.ID) || !validText(object.Label, 80) || seen[object.ID] || objectLabels[object.Label] {
					return bad
				}
				seen[object.ID] = true
				objectLabels[object.Label] = true
				if object.Kind == "segment" {
					if len(object.PointIDs) != 2 {
						return bad
					}
				} else if object.Kind == "polygon" {
					if len(object.PointIDs) < 3 || len(object.PointIDs) > 12 {
						return bad
					}
				} else {
					return bad
				}
				used := map[string]bool{}
				for _, id := range object.PointIDs {
					if !ids[id] || used[id] {
						return bad
					}
					used[id] = true
				}
			}
		}
	default:
		var empty struct{}
		return configDecode(raw, &empty)
	}
	return nil
}

func cellWithin(value string, rows, columns int) bool {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return false
	}
	r, e1 := strconv.Atoi(parts[0])
	c, e2 := strconv.Atoi(parts[1])
	return e1 == nil && e2 == nil && r >= 0 && r < rows && c >= 0 && c < columns && value == strconv.Itoa(r)+":"+strconv.Itoa(c)
}
func ValidateExerciseAnswer(spec domain.Exercise, answer domain.ExerciseAnswer) error {
	if !IsVisual(spec.Kind) {
		return ValidateAnswer(spec.Kind, spec.Options, answer)
	}
	if ValidateConfig(spec.Kind, spec.Config) != nil {
		return apperr.ErrBadRequest
	}
	bad := apperr.ErrBadRequest
	if spec.Kind == "number_line" {
		if answer.Value == nil || answer.Selected != nil {
			return bad
		}
		if !integer.MatchString(strings.ReplaceAll(strings.TrimSpace(*answer.Value), "−", "-")) {
			return bad
		}
		value, err := ParseNumber(*answer.Value)
		if err != nil || !value.IsInt() || !value.Num().IsInt64() {
			return bad
		}
		var c NumberLineConfig
		_ = json.Unmarshal(spec.Config, &c)
		n := value.Num().Int64()
		if n < int64(c.Min) || n > int64(c.Max) || (n-int64(c.Min))%int64(c.Step) != 0 {
			return bad
		}
		return nil
	}
	if answer.Value != nil || len(answer.Selected) == 0 || len(answer.Selected) > 66 {
		return bad
	}
	seen := map[string]bool{}
	for _, id := range answer.Selected {
		if seen[id] || len(id) > 40 {
			return bad
		}
		seen[id] = true
	}
	switch spec.Kind {
	case "grid_select":
		var c GridConfig
		_ = json.Unmarshal(spec.Config, &c)
		for _, id := range answer.Selected {
			if !cellWithin(id, c.Rows, c.Columns) {
				return bad
			}
		}
	case "matching":
		var c MatchingConfig
		_ = json.Unmarshal(spec.Config, &c)
		if len(answer.Selected) != len(c.Left) {
			return bad
		}
		left, right := map[string]bool{}, map[string]bool{}
		for _, pair := range answer.Selected {
			parts := strings.Split(pair, ":")
			if len(parts) != 2 || !hasItem(c.Left, parts[0]) || !hasItem(c.Right, parts[1]) || left[parts[0]] || right[parts[1]] {
				return bad
			}
			left[parts[0]] = true
			right[parts[1]] = true
		}
	case "ordering":
		var c OrderingConfig
		_ = json.Unmarshal(spec.Config, &c)
		if len(answer.Selected) != len(c.Items) {
			return bad
		}
		for _, id := range answer.Selected {
			if !hasItem(c.Items, id) {
				return bad
			}
		}
	case "geometry":
		var c GeometryConfig
		_ = json.Unmarshal(spec.Config, &c)
		switch c.Mode {
		case "point":
			if len(answer.Selected) != 1 || !cellWithin(answer.Selected[0], c.Columns, c.Rows) {
				return bad
			}
		case "select":
			if len(answer.Selected) != 1 {
				return bad
			}
			found := false
			for _, object := range c.Objects {
				found = found || object.ID == answer.Selected[0]
			}
			if !found {
				return bad
			}
		case "connect":
			points := map[string]bool{}
			for _, p := range c.Points {
				points[p.ID] = true
			}
			for _, edge := range answer.Selected {
				pair := strings.Split(edge, ":")
				if len(pair) != 2 || !points[pair[0]] || !points[pair[1]] || pair[0] >= pair[1] {
					return bad
				}
			}
		}
	}
	return nil
}

func ValidateExerciseKey(spec domain.Exercise, key string) error {
	if ValidateConfig(spec.Kind, spec.Config) != nil {
		return apperr.ErrBadRequest
	}
	if !IsVisual(spec.Kind) {
		return ValidateKey(spec.Kind, spec.Options, key)
	}
	answer := domain.ExerciseAnswer{}
	if spec.Kind == "number_line" {
		answer.Value = &key
	} else if json.Unmarshal([]byte(key), &answer.Selected) != nil {
		return apperr.ErrBadRequest
	}
	return ValidateExerciseAnswer(spec, answer)
}
func CheckExercise(spec domain.Exercise, key string, answer domain.ExerciseAnswer) (string, error) {
	if !IsVisual(spec.Kind) {
		return Check(spec.Kind, spec.Options, key, answer)
	}
	if err := ValidateExerciseAnswer(spec, answer); err != nil {
		return "", err
	}
	if ValidateExerciseKey(spec, key) != nil {
		return "", apperr.ErrInvalidState
	}
	if spec.Kind == "number_line" {
		return Check("numeric", nil, key, answer)
	}
	var expected []string
	_ = json.Unmarshal([]byte(key), &expected)
	actual := slices.Clone(answer.Selected)
	if spec.Kind != "ordering" {
		slices.Sort(expected)
		slices.Sort(actual)
	}
	if slices.Equal(actual, expected) {
		return "correct", nil
	}
	return "incorrect", nil
}
