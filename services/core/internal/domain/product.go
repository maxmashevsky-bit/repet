package domain

import "time"

type Conversation struct {
	ID          string    `json:"id"`
	TutorID     string    `json:"tutor_id"`
	TutorName   string    `json:"tutor_name"`
	StudentID   string    `json:"student_id"`
	StudentName string    `json:"student_name"`
	LastMessage string    `json:"last_message"`
	CreatedAt   time.Time `json:"created_at"`
}

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	AuthorID       string    `json:"author_id"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"created_at"`
}

type Assignment struct {
	ID          string    `json:"id"`
	TutorID     string    `json:"tutor_id"`
	StudentID   string    `json:"student_id"`
	StudentName string    `json:"student_name"`
	Title       string    `json:"title"`
	Subject     string    `json:"subject"`
	Topic       string    `json:"topic"`
	Body        string    `json:"body"`
	Status      string    `json:"status"`
	DueAt       time.Time `json:"due_at"`
	MaxScore    int       `json:"max_score"`
	Score       *int      `json:"score,omitempty"`
	Feedback    string    `json:"feedback"`
	Answer      string    `json:"answer"`
	CreatedAt   time.Time `json:"created_at"`
}

type Notification struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Href      string     `json:"href"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}
