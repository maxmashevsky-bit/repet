package domain

import "time"

type ReviewFilter struct {
	Mode                        string
	Limit, Offset, ActiveOffset int
}
type ReviewItem struct {
	SourceRunID string    `json:"source_run_id"`
	LessonID    string    `json:"lesson_id"`
	ExerciseID  string    `json:"exercise_id"`
	TopicKey    string    `json:"topic_key"`
	Topic       string    `json:"topic"`
	Subtopic    string    `json:"subtopic"`
	Prompt      string    `json:"prompt"`
	Status      string    `json:"status"`
	CleanDays   int       `json:"clean_days"`
	DueAt       time.Time `json:"due_at"`
}
type ReviewEntry struct {
	ID      string          `json:"id"`
	Title   string          `json:"title"`
	Attempt LearningAttempt `json:"attempt"`
}
type ReviewPlan struct {
	ServerTime      time.Time                `json:"server_time"`
	DueTotal        int                      `json:"due_total"`
	MistakeTotal    int                      `json:"mistake_total"`
	Unanswered      int                      `json:"unanswered"`
	Pending         int                      `json:"pending"`
	WeeklyCompleted int                      `json:"weekly_completed"`
	WeeklyTarget    int                      `json:"weekly_target"`
	Items           CatalogPage[ReviewItem]  `json:"items"`
	Active          CatalogPage[ReviewEntry] `json:"active"`
}
type ReviewStartInput struct {
	RequestID   string `json:"request_id"`
	SourceRunID string `json:"source_run_id"`
	TopicKey    string `json:"topic_key"`
	Mode        string `json:"mode"`
	Count       int    `json:"count"`
}
type ReviewSession struct {
	ID    string        `json:"id"`
	State LearningState `json:"state"`
}
