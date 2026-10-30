package domain

import "time"

type GuardianLink struct {
	ID            string     `json:"id"`
	StudentID     string     `json:"student_id"`
	StudentName   string     `json:"student_name"`
	GuardianEmail string     `json:"guardian_email"`
	GuardianID    *string    `json:"guardian_id"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	ExpiresAt     time.Time  `json:"expires_at"`
	AcceptedAt    *time.Time `json:"accepted_at"`
	RevokedAt     *time.Time `json:"revoked_at"`
}
type GuardianInviteInput struct {
	GuardianEmail string `json:"guardian_email"`
	ConfirmShare  bool   `json:"confirm_share"`
}
type GuardianNoteInput struct {
	StudentID string `json:"student_id"`
	Body      string `json:"body"`
	RequestID string `json:"request_id"`
}
type GuardianNote struct {
	ID        string    `json:"id"`
	TutorName string    `json:"tutor_name"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}
type FamilyLesson struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	TutorName string    `json:"tutor_name"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	Status    string    `json:"status"`
}
type FamilyAssignment struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	TutorName string    `json:"tutor_name"`
	Subject   string    `json:"subject"`
	Topic     string    `json:"topic"`
	Status    string    `json:"status"`
	DueAt     time.Time `json:"due_at"`
	Score     *int      `json:"score"`
	MaxScore  int       `json:"max_score"`
}
type FamilyEvent struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"created_at"`
	Unread    bool      `json:"unread"`
}
type FamilyResult struct {
	ID          string    `json:"id"`
	LessonTitle string    `json:"lesson_title"`
	Prompt      string    `json:"prompt"`
	Status      string    `json:"status"`
	Attempts    int       `json:"attempts"`
	CreatedAt   time.Time `json:"created_at"`
}
type FamilyFilter struct {
	AnalyticsFilter
	LessonOffset, AssignmentOffset, NoteOffset, EventOffset, ResultOffset int
}
type FamilyReport struct {
	Student     AnalyticsStudent              `json:"student"`
	From        string                        `json:"from"`
	To          string                        `json:"to"`
	GeneratedAt time.Time                     `json:"generated_at"`
	Summary     AnalyticsMetric               `json:"summary"`
	Topics      CatalogPage[AnalyticsTopic]   `json:"topics"`
	Timeline    []AnalyticsDay                `json:"timeline"`
	Exams       CatalogPage[AnalyticsExam]    `json:"exams"`
	Lessons     CatalogPage[FamilyLesson]     `json:"lessons"`
	Assignments CatalogPage[FamilyAssignment] `json:"assignments"`
	Notes       CatalogPage[GuardianNote]     `json:"notes"`
	Events      CatalogPage[FamilyEvent]      `json:"events"`
	Results     CatalogPage[FamilyResult]     `json:"results"`
	Unread      int                           `json:"unread"`
}
