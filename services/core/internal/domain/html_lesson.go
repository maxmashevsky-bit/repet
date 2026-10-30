package domain

import "time"

type HTMLMetadata struct {
	Title             string `json:"title"`
	Description       string `json:"description"`
	Language          string `json:"language"`
	Scripts           int    `json:"scripts"`
	ExternalResources int    `json:"external_resources"`
	Bytes             int    `json:"bytes"`
}
type HTMLImportInput struct {
	RequestID    string `json:"request_id"`
	CourseID     string `json:"course_id"`
	TopicID      string `json:"topic_id"`
	Title        string `json:"title"`
	Summary      string `json:"summary"`
	Track        string `json:"track"`
	GradeMin     int    `json:"grade_min"`
	GradeMax     int    `json:"grade_max"`
	Difficulty   int    `json:"difficulty"`
	Filename     string `json:"filename"`
	Source       string `json:"source"`
	RightsStatus string `json:"rights_status"`
	License      string `json:"license"`
	SourceLabel  string `json:"source_label"`
}
type HTMLVersion struct {
	ID           string       `json:"id"`
	Number       int          `json:"number"`
	TopicID      string       `json:"topic_id"`
	Title        string       `json:"title"`
	Summary      string       `json:"summary"`
	Track        string       `json:"track"`
	GradeMin     int          `json:"grade_min"`
	GradeMax     int          `json:"grade_max"`
	Difficulty   int          `json:"difficulty"`
	Filename     string       `json:"filename"`
	SHA256       string       `json:"sha256"`
	Metadata     HTMLMetadata `json:"metadata"`
	RightsStatus string       `json:"rights_status"`
	License      string       `json:"license"`
	SourceLabel  string       `json:"source_label"`
	CreatedAt    time.Time    `json:"created_at"`
	PublishedAt  *time.Time   `json:"published_at"`
}
type HTMLLesson struct {
	ID                    string      `json:"id"`
	CourseID              string      `json:"course_id"`
	AuthorID              string      `json:"author_id"`
	AuthorName            string      `json:"author_name"`
	Status                string      `json:"status"`
	HasUnpublishedChanges bool        `json:"has_unpublished_changes"`
	Version               HTMLVersion `json:"version"`
}
type HTMLAssignment struct {
	LessonID  string `json:"lesson_id"`
	VersionID string `json:"version_id"`
}
type HTMLAssignInput struct {
	RequestID string    `json:"request_id"`
	StudentID string    `json:"student_id"`
	DueAt     time.Time `json:"due_at"`
}
