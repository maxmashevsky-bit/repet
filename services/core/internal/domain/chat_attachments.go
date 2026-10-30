package domain

type ChatFile struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size_bytes"`
}

type ChatAttachment struct {
	Kind      string    `json:"kind"`
	ID        string    `json:"id,omitempty"`
	Title     string    `json:"title,omitempty"`
	Available bool      `json:"available"`
	File      *ChatFile `json:"file,omitempty"`
}

type MessageInput struct {
	Body         string `json:"body"`
	FileID       string `json:"file_id,omitempty"`
	AssignmentID string `json:"assignment_id,omitempty"`
	LessonID     string `json:"lesson_id,omitempty"`
}
