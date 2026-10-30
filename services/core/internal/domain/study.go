package domain

import "time"

type ExamSlot struct {
	Number int `json:"number"`
	Count  int `json:"count"`
	Points int `json:"points"`
}
type ExamSpecificationInput struct {
	RequestID       string     `json:"request_id"`
	ParentID        string     `json:"parent_id"`
	CollectionID    string     `json:"collection_id"`
	Title           string     `json:"title"`
	Direction       string     `json:"direction"`
	Year            int        `json:"year"`
	DurationSeconds int        `json:"duration_seconds"`
	Instructions    string     `json:"instructions"`
	Slots           []ExamSlot `json:"slots"`
}
type ExamSpecification struct {
	ID              string     `json:"id"`
	FamilyID        string     `json:"family_id"`
	Version         int        `json:"version"`
	AuthorID        string     `json:"author_id"`
	AuthorName      string     `json:"author_name"`
	CollectionID    string     `json:"collection_id"`
	Title           string     `json:"title"`
	Direction       string     `json:"direction"`
	Year            int        `json:"year"`
	DurationSeconds int        `json:"duration_seconds"`
	Instructions    string     `json:"instructions"`
	Slots           []ExamSlot `json:"slots"`
	Topics          []string   `json:"topics"`
	ExerciseCount   int        `json:"exercise_count"`
	Status          string     `json:"status"`
}
type StudyStartInput struct {
	RequestID       string `json:"request_id"`
	SpecificationID string `json:"specification_id"`
	Mode            string `json:"mode"`
	Seed            string `json:"seed"`
	Topic           string `json:"topic"`
	Count           int    `json:"count"`
	RepeatSessionID string `json:"repeat_session_id"`
}
type StudySession struct {
	ID            string            `json:"id"`
	Specification ExamSpecification `json:"specification"`
	Seed          string            `json:"seed"`
	DeadlineAt    *time.Time        `json:"deadline_at"`
	ServerTime    time.Time         `json:"server_time"`
	Points        int               `json:"points"`
	MaxPoints     int               `json:"max_points"`
	State         LearningState     `json:"state"`
}
type StudyReview struct {
	SessionID string          `json:"session_id"`
	LessonID  string          `json:"lesson_id"`
	Attempt   ExerciseAttempt `json:"attempt"`
}
type StudyHistoryEntry struct {
	ID         string          `json:"id"`
	Title      string          `json:"title"`
	Attempt    LearningAttempt `json:"attempt"`
	DeadlineAt *time.Time      `json:"deadline_at"`
	Seed       string          `json:"seed"`
}
