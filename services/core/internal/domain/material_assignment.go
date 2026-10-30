package domain

import "time"

type AssignmentMaterial struct {
	LessonID         string           `json:"lesson_id"`
	VersionID        string           `json:"version_id"`
	ExerciseIDs      []string         `json:"exercise_ids"`
	CalendarLessonID *string          `json:"calendar_lesson_id"`
	RepeatOf         *string          `json:"repeat_of"`
	Mode             string           `json:"mode"`
	Attempt          *LearningAttempt `json:"attempt"`
}
type MaterialAssignmentInput struct {
	RequestID        string    `json:"request_id"`
	StudentID        string    `json:"student_id"`
	LessonID         string    `json:"lesson_id"`
	SetID            string    `json:"set_id"`
	ExerciseIDs      []string  `json:"exercise_ids"`
	DueAt            time.Time `json:"due_at"`
	CalendarLessonID string    `json:"calendar_lesson_id"`
	Mode             string    `json:"mode"`
}
type MaterialRepeatInput struct {
	RequestID        string    `json:"request_id"`
	DueAt            time.Time `json:"due_at"`
	CalendarLessonID string    `json:"calendar_lesson_id"`
}
type ExerciseSetInput struct {
	Title       string   `json:"title"`
	LessonID    string   `json:"lesson_id"`
	ExerciseIDs []string `json:"exercise_ids"`
}
type ExerciseSet struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	LessonID    string    `json:"lesson_id"`
	VersionID   string    `json:"version_id"`
	ExerciseIDs []string  `json:"exercise_ids"`
	CreatedAt   time.Time `json:"created_at"`
}
type MaterialResults struct {
	Stats      []AssignmentExerciseStat     `json:"exercise_stats"`
	Assignment Assignment                   `json:"assignment"`
	Version    LessonVersion                `json:"version"`
	State      *LearningState               `json:"state"`
	History    CatalogPage[ExerciseAttempt] `json:"history"`
}
type AssignmentExerciseStat struct {
	ExerciseID string `json:"exercise_id"`
	Attempts   int    `json:"attempts"`
	Mistakes   int    `json:"mistakes"`
}
