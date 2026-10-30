package domain

import "time"

type ExerciseAnswer struct {
	Value    *string  `json:"value,omitempty"`
	Selected []string `json:"selected,omitempty"`
}

type ExerciseAttemptInput struct {
	VersionID         string         `json:"version_id"`
	RequestID         string         `json:"request_id"`
	Answer            ExerciseAnswer `json:"answer"`
	LearningAttemptID string         `json:"learning_attempt_id,omitempty"`
}

type ExerciseReviewInput struct {
	Status   string `json:"status"`
	Feedback string `json:"feedback"`
}

type ExerciseAttempt struct {
	ID                string         `json:"id"`
	ExerciseID        string         `json:"exercise_id"`
	VersionID         string         `json:"version_id"`
	Version           int            `json:"version"`
	StudentID         string         `json:"student_id"`
	StudentName       string         `json:"student_name"`
	Kind              string         `json:"kind"`
	Prompt            string         `json:"prompt"`
	Answer            ExerciseAnswer `json:"answer"`
	Status            string         `json:"status"`
	Feedback          string         `json:"feedback"`
	CreatedAt         time.Time      `json:"created_at"`
	ReviewedAt        *time.Time     `json:"reviewed_at"`
	LearningAttemptID *string        `json:"learning_attempt_id"`
	AttemptNumber     int            `json:"attempt_number"`
	HintsUsed         int            `json:"hints_used"`
	ElapsedSeconds    int64          `json:"elapsed_seconds"`
}
