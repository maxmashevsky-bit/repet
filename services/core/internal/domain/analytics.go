package domain

import "time"

type AnalyticsFilter struct {
	From, To                time.Time
	TopicOffset, ExamOffset int
}
type AnalyticsStudent struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}
type AnalyticsMetric struct {
	Attempts  int      `json:"attempts"`
	Graded    int      `json:"graded"`
	Correct   int      `json:"correct"`
	Incorrect int      `json:"incorrect"`
	Pending   int      `json:"pending"`
	Mistakes  int      `json:"mistakes"`
	Hints     int      `json:"hints"`
	Hinted    int      `json:"hinted"`
	Accuracy  *float64 `json:"accuracy"`
	HintRate  *float64 `json:"hint_rate"`
}
type AnalyticsTopic struct {
	Key      string `json:"key"`
	Topic    string `json:"topic"`
	Subtopic string `json:"subtopic"`
	AnalyticsMetric
	Level          string   `json:"level"`
	PreviousGraded int      `json:"previous_graded"`
	RecentGraded   int      `json:"recent_graded"`
	Change         *float64 `json:"change"`
}
type AnalyticsDay struct {
	Day      string   `json:"day"`
	Attempts int      `json:"attempts"`
	Graded   int      `json:"graded"`
	Correct  int      `json:"correct"`
	Accuracy *float64 `json:"accuracy"`
}
type AnalyticsHomework struct {
	Total     int `json:"total"`
	Done      int `json:"done"`
	Submitted int `json:"submitted"`
	Overdue   int `json:"overdue"`
}
type AnalyticsExam struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Points      *int       `json:"points"`
	MaxPoints   int        `json:"max_points"`
	Pending     int        `json:"pending"`
}
type AnalyticsRecommendation struct {
	Exercises []AnalyticsError `json:"exercises"`
	TopicKey  string           `json:"topic_key"`
	Topic     string           `json:"topic"`
	Subtopic  string           `json:"subtopic"`
	RunID     string           `json:"run_id"`
	LessonID  string           `json:"lesson_id"`
	Href      string           `json:"href"`
	Sample    int              `json:"sample"`
	Errors    int              `json:"errors"`
	Reason    string           `json:"reason"`
}
type AnalyticsError struct {
	ExerciseID string `json:"exercise_id"`
	Prompt     string `json:"prompt"`
	Attempts   int    `json:"attempts"`
	Mistakes   int    `json:"mistakes"`
}
type AnalyticsReport struct {
	Student         AnalyticsStudent            `json:"student"`
	From            string                      `json:"from"`
	To              string                      `json:"to"`
	GeneratedAt     time.Time                   `json:"generated_at"`
	Summary         AnalyticsMetric             `json:"summary"`
	Homework        AnalyticsHomework           `json:"homework"`
	Topics          CatalogPage[AnalyticsTopic] `json:"topics"`
	Timeline        []AnalyticsDay              `json:"timeline"`
	Exams           CatalogPage[AnalyticsExam]  `json:"exams"`
	Recommendations []AnalyticsRecommendation   `json:"recommendations"`
}
type AnalyticsRepeatInput struct {
	RequestID string    `json:"request_id"`
	RunID     string    `json:"run_id"`
	TopicKey  string    `json:"topic_key"`
	DueAt     time.Time `json:"due_at"`
}
