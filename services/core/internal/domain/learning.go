package domain

import "time"

type LearningStartInput struct {
	RequestID string `json:"request_id"`
	Restart   bool   `json:"restart"`
	Mode      string `json:"mode,omitempty"`
}
type LearningActionInput struct {
	RequestID string `json:"request_id"`
}
type LearningAttempt struct {
	AssignmentID      *string    `json:"assignment_id,omitempty"`
	Mode              string     `json:"mode"`
	PlayerStep        string     `json:"player_step"`
	TheorySeen        bool       `json:"theory_seen"`
	ExampleSeen       bool       `json:"example_seen"`
	ID                string     `json:"id"`
	LessonID          string     `json:"lesson_id"`
	VersionID         string     `json:"version_id"`
	Number            int        `json:"number"`
	Status            string     `json:"status"`
	StartedAt         time.Time  `json:"started_at"`
	CompletedAt       *time.Time `json:"completed_at"`
	ElapsedSeconds    int64      `json:"elapsed_seconds"`
	TotalExercises    int        `json:"total_exercises"`
	AnsweredExercises int        `json:"answered_exercises"`
	CorrectExercises  int        `json:"correct_exercises"`
	PendingReview     int        `json:"pending_review"`
	IncorrectAttempts int        `json:"incorrect_attempts"`
}
type LearningNavigateInput struct {
	RequestID string `json:"request_id"`
	Step      string `json:"step"`
}
type LearningSolution struct {
	ExerciseID string `json:"exercise_id"`
	Available  bool   `json:"available"`
	Text       string `json:"text"`
}
type LearningHint struct {
	ExerciseID string    `json:"exercise_id"`
	Number     int       `json:"number"`
	Text       string    `json:"text"`
	RevealedAt time.Time `json:"revealed_at"`
}
type LearningState struct {
	Attempt LearningAttempt   `json:"attempt"`
	Version LessonVersion     `json:"version"`
	Answers []ExerciseAttempt `json:"answers"`
	Hints   []LearningHint    `json:"hints"`
}
