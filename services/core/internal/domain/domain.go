package domain

import "time"

type Role string

const (
	RoleTutor   Role = "tutor"
	RoleStudent Role = "student"
	RoleAdmin   Role = "admin"
)

func (r Role) Valid() bool {
	return r == RoleTutor || r == RoleStudent || r == RoleAdmin
}

type User struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Email          string    `json:"email"`
	DisplayName    string    `json:"display_name"`
	Role           Role      `json:"role"`
	Timezone       string    `json:"timezone"`
	Locale         string    `json:"locale"`
	CreatedAt      time.Time `json:"created_at"`
}

type Invitation struct {
	ID           string    `json:"id"`
	TutorID      string    `json:"tutor_id"`
	StudentEmail string    `json:"student_email"`
	StudentName  string    `json:"student_name"`
	Status       string    `json:"status"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type Relation struct {
	ID        string    `json:"id"`
	TutorID   string    `json:"tutor_id"`
	StudentID string    `json:"student_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Lesson struct {
	ID         string    `json:"id"`
	RelationID string    `json:"relation_id"`
	TutorID    string    `json:"tutor_id"`
	StudentID  string    `json:"student_id"`
	Title      string    `json:"title"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}
