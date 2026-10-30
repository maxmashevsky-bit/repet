package domain

import "time"

type BankMetadata struct {
	Grade        int    `json:"grade"`
	Direction    string `json:"direction"`
	Topic        string `json:"topic"`
	Subtopic     string `json:"subtopic"`
	ExamNumber   *int   `json:"exam_number"`
	Source       string `json:"source"`
	Attribution  string `json:"attribution"`
	RightsStatus string `json:"rights_status"`
	License      string `json:"license"`
}
type BankExerciseInput struct {
	LessonID   string `json:"lesson_id"`
	ExerciseID string `json:"exercise_id"`
	BankMetadata
}
type BankExercise struct {
	ID         string   `json:"id"`
	AuthorID   string   `json:"author_id"`
	AuthorName string   `json:"author_name"`
	LessonID   string   `json:"lesson_id"`
	VersionID  string   `json:"version_id"`
	Subject    string   `json:"subject"`
	Exercise   Exercise `json:"exercise"`
	BankMetadata
	CreatedAt time.Time `json:"created_at"`
}
type BankFilter struct {
	Query, Direction, Topic, Subtopic, Kind      string
	Grade, ExamNumber, Difficulty, Limit, Offset int
}
type BankFacets struct {
	Topics    []string `json:"topics"`
	Subtopics []string `json:"subtopics"`
}
type BankCollectionInput struct {
	RequestID string   `json:"request_id"`
	Title     string   `json:"title"`
	CourseID  string   `json:"course_id"`
	TopicID   string   `json:"topic_id"`
	BankIDs   []string `json:"bank_ids"`
}
type BankCollection struct {
	ID          string         `json:"id"`
	LessonID    string         `json:"lesson_id"`
	VersionID   string         `json:"version_id"`
	Title       string         `json:"title"`
	ExerciseIDs []string       `json:"exercise_ids"`
	Provenance  []BankExercise `json:"provenance"`
	Version     LessonVersion  `json:"version"`
	CreatedAt   time.Time      `json:"created_at"`
}
type BankAssignInput struct {
	RequestID  string    `json:"request_id"`
	StudentIDs []string  `json:"student_ids"`
	DueAt      time.Time `json:"due_at"`
	Mode       string    `json:"mode"`
}
type BankAssignResult struct {
	StudentID  string      `json:"student_id"`
	Assignment *Assignment `json:"assignment"`
	Error      string      `json:"error,omitempty"`
}
