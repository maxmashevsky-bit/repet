package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"time"
)

type CourseInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Track       string `json:"track"`
	GradeMin    int    `json:"grade_min"`
	GradeMax    int    `json:"grade_max"`
	Difficulty  int    `json:"difficulty"`
	Visibility  string `json:"visibility"`
}

type Course struct {
	ID         string `json:"id"`
	AuthorID   string `json:"author_id"`
	AuthorName string `json:"author_name"`
	CourseInput
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TopicInput struct {
	ParentID    *string `json:"parent_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Position    int     `json:"position"`
}

type Topic struct {
	ID       string `json:"id"`
	CourseID string `json:"course_id"`
	TopicInput
	Status string `json:"status"`
}

type LearningLessonInput struct {
	Example    *LearningExample `json:"example,omitempty"`
	TopicID    string           `json:"topic_id"`
	Title      string           `json:"title"`
	Summary    string           `json:"summary"`
	Content    string           `json:"content"`
	Track      string           `json:"track"`
	GradeMin   int              `json:"grade_min"`
	GradeMax   int              `json:"grade_max"`
	Difficulty int              `json:"difficulty"`
}
type LearningExample struct {
	Title       string                `json:"title"`
	Explanation string                `json:"explanation"`
	Min         int                   `json:"min"`
	Max         int                   `json:"max"`
	Steps       []LearningExampleStep `json:"steps"`
}
type LearningExampleStep struct {
	Text  string `json:"text"`
	Value int    `json:"value"`
}

func exampleFields(raw []byte, keys ...string) error {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || len(fields) != len(keys) {
		return errors.New("invalid example fields")
	}
	for _, key := range keys {
		value, ok := fields[key]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return errors.New("missing example field")
		}
	}
	return nil
}
func (e *LearningExample) UnmarshalJSON(raw []byte) error {
	if err := exampleFields(raw, "title", "explanation", "min", "max", "steps"); err != nil {
		return err
	}
	type plain LearningExample
	var out plain
	if err := json.Unmarshal(raw, &out); err != nil {
		return err
	}
	*e = LearningExample(out)
	return nil
}
func (e *LearningExampleStep) UnmarshalJSON(raw []byte) error {
	if err := exampleFields(raw, "text", "value"); err != nil {
		return err
	}
	type plain LearningExampleStep
	var out plain
	if err := json.Unmarshal(raw, &out); err != nil {
		return err
	}
	*e = LearningExampleStep(out)
	return nil
}

type Exercise struct {
	Media      []ExerciseMedia `json:"media,omitempty"`
	ID         string          `json:"id"`
	Kind       string          `json:"kind"`
	Prompt     string          `json:"prompt"`
	Options    []string        `json:"options"`
	Difficulty int             `json:"difficulty"`
	Position   int             `json:"position"`
	Config     json.RawMessage `json:"config,omitempty"`
	HintCount  int             `json:"hint_count,omitempty"`
}

type AnswerKeyInput struct {
	Answer      string   `json:"answer"`
	Explanation string   `json:"explanation"`
	Hints       []string `json:"hints,omitempty"`
}

// Keys are deliberately not embedded in any public exercise or version DTO.
type ExerciseInput struct {
	Media      []ExerciseMedia `json:"media,omitempty"`
	Kind       string          `json:"kind"`
	Prompt     string          `json:"prompt"`
	Options    []string        `json:"options"`
	Difficulty int             `json:"difficulty"`
	Position   int             `json:"position"`
	AnswerKey  AnswerKeyInput  `json:"answer_key"`
	Config     json.RawMessage `json:"config,omitempty"`
}

type ExerciseAnswerKey struct {
	ExerciseID string `json:"exercise_id"`
	AnswerKeyInput
}

type ExerciseMedia struct {
	AssetID string `json:"asset_id"`
	Alt     string `json:"alt"`
}

type LessonVersion struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	LearningLessonInput
	Exercises   []Exercise `json:"exercises"`
	CreatedAt   time.Time  `json:"created_at"`
	PublishedAt *time.Time `json:"published_at"`
}

type LearningLesson struct {
	Provenance            *LessonProvenance `json:"provenance,omitempty"`
	ID                    string            `json:"id"`
	CourseID              string            `json:"course_id"`
	AuthorID              string            `json:"author_id"`
	AuthorName            string            `json:"author_name"`
	Status                string            `json:"status"`
	HasUnpublishedChanges bool              `json:"has_unpublished_changes"`
	Version               LessonVersion     `json:"version"`
}

type CatalogFilter struct {
	Query      string
	Track      string
	Grade      int
	Difficulty int
	AuthorID   string
	CourseID   string
	TopicID    string
	Status     string
	Limit      int
	Offset     int
}

type CatalogPage[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
