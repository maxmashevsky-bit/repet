package domain

// Local IDs remain stable; an optional relation links a card to an existing account.
type PlanStudent struct {
	RelationID   string `json:"relation_id,omitempty"`
	ID           string `json:"id"`
	Name         string `json:"name"`
	Subject      string `json:"subject"`
	PriceKopecks int64  `json:"price_kopecks"`
	Duration     int    `json:"duration"`
	Contact      string `json:"contact"`
}

type PlanCalendarInput struct {
	Revision  int64  `json:"revision"`
	From      string `json:"from"`
	To        string `json:"to"`
	RequestID string `json:"request_id"`
}
type PlanCalendarResult struct {
	Created  int    `json:"created"`
	Existing int    `json:"existing"`
	Unlinked int    `json:"unlinked"`
	Paused   int    `json:"paused"`
	From     string `json:"from"`
	To       string `json:"to"`
}

type PlanLesson struct {
	ID          string `json:"id"`
	StudentID   string `json:"student_id"`
	Weekday     int    `json:"weekday"`
	StartMinute int    `json:"start_minute"`
	Duration    int    `json:"duration"`
	Paused      bool   `json:"paused"`
}

type PlanWindow struct {
	Weekday     int `json:"weekday"`
	StartMinute int `json:"start_minute"`
}

type PlanCalculator struct {
	GoalKopecks       int64   `json:"goal_kopecks"`
	PriceKopecks      int64   `json:"price_kopecks"`
	LessonsPerStudent int     `json:"lessons_per_student"`
	Duration          int     `json:"duration"`
	Weeks             float64 `json:"weeks"`
	People            int     `json:"people"`
	PreparationHours  float64 `json:"preparation_hours"`
}

type TutorPlan struct {
	Revision   int64          `json:"revision"`
	Students   []PlanStudent  `json:"students"`
	Lessons    []PlanLesson   `json:"lessons"`
	Windows    []PlanWindow   `json:"windows"`
	Calculator PlanCalculator `json:"calculator"`
}

func EmptyTutorPlan() TutorPlan {
	return TutorPlan{
		Students: []PlanStudent{}, Lessons: []PlanLesson{}, Windows: []PlanWindow{},
		Calculator: PlanCalculator{GoalKopecks: 10000000, PriceKopecks: 120000, LessonsPerStudent: 2, Duration: 60, Weeks: 4, People: 1},
	}
}
