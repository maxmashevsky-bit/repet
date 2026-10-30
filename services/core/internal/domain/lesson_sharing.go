package domain

import "time"

type LessonShareInput struct {
	RecipientEmail  string `json:"recipient_email"`
	AllowCopy       bool   `json:"allow_copy"`
	RightsConfirmed bool   `json:"rights_confirmed"`
}

type LessonShare struct {
	ID             string     `json:"id"`
	LessonID       string     `json:"lesson_id"`
	VersionID      string     `json:"version_id"`
	VersionNumber  int        `json:"version_number"`
	Title          string     `json:"title"`
	AuthorName     string     `json:"author_name"`
	RecipientEmail string     `json:"recipient_email"`
	AllowCopy      bool       `json:"allow_copy"`
	License        string     `json:"license"`
	CreatedAt      time.Time  `json:"created_at"`
	RevokedAt      *time.Time `json:"revoked_at"`
}

type SharedLesson struct {
	Share  LessonShare    `json:"share"`
	Lesson LearningLesson `json:"lesson"`
}

type LessonCopyInput struct {
	RequestID string `json:"request_id"`
	CourseID  string `json:"course_id"`
	TopicID   string `json:"topic_id"`
	Title     string `json:"title"`
}

type LessonProvenance struct {
	SourceLessonID      string    `json:"source_lesson_id"`
	SourceVersionID     string    `json:"source_version_id"`
	SourceVersionNumber int       `json:"source_version_number"`
	OriginalAuthorID    string    `json:"original_author_id"`
	OriginalAuthorName  string    `json:"original_author_name"`
	SourceTitle         string    `json:"source_title"`
	License             string    `json:"license"`
	CreatedAt           time.Time `json:"created_at"`
}
