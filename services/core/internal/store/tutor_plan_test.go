package store

import (
	"math"
	"testing"
	"tutor-platform/services/core/internal/domain"
)

func sampleTutorPlan() domain.TutorPlan {
	p := domain.EmptyTutorPlan()
	p.Students = []domain.PlanStudent{{ID: "00000000-0000-4000-8000-000000000002", Name: "Student", PriceKopecks: 120000, Duration: 60, Contact: "@student_test"}}
	p.Lessons = []domain.PlanLesson{{ID: "00000000-0000-4000-8000-000000000003", StudentID: p.Students[0].ID, Weekday: 7, StartMinute: 1260, Duration: 60}}
	p.Windows = []domain.PlanWindow{{Weekday: 7, StartMinute: 1290}}
	return p
}

func TestTutorPlanValidation(t *testing.T) {
	if err := validateTutorPlan(ptrPlan(domain.EmptyTutorPlan())); err != nil {
		t.Fatal(err)
	}
	for _, contact := range []string{"", "@student_test", "https://example.test/student", "http://example.test"} {
		p := sampleTutorPlan()
		p.Students[0].Contact = contact
		if err := validateTutorPlan(&p); err != nil {
			t.Errorf("contact %q: %v", contact, err)
		}
	}
	cases := map[string]func(*domain.TutorPlan){
		"duplicate name": func(p *domain.TutorPlan) {
			s := p.Students[0]
			s.ID = "00000000-0000-4000-8000-000000000004"
			s.Name = " STUDENT "
			p.Students = append(p.Students, s)
		},
		"duplicate id":      func(p *domain.TutorPlan) { s := p.Students[0]; s.Name = "Other"; p.Students = append(p.Students, s) },
		"blank name":        func(p *domain.TutorPlan) { p.Students[0].Name = "  " },
		"control":           func(p *domain.TutorPlan) { p.Students[0].Name = "A\nB" },
		"negative price":    func(p *domain.TutorPlan) { p.Students[0].PriceKopecks = -1 },
		"javascript":        func(p *domain.TutorPlan) { p.Students[0].Contact = "javascript:alert(1)" },
		"userinfo":          func(p *domain.TutorPlan) { p.Students[0].Contact = "https://a@evil.test/" },
		"hostless":          func(p *domain.TutorPlan) { p.Students[0].Contact = "https:///student" },
		"short contact":     func(p *domain.TutorPlan) { p.Students[0].Contact = "@x" },
		"unknown student":   func(p *domain.TutorPlan) { p.Lessons[0].StudentID = "00000000-0000-4000-8000-000000000004" },
		"duplicate lesson":  func(p *domain.TutorPlan) { p.Lessons = append(p.Lessons, p.Lessons[0]) },
		"early":             func(p *domain.TutorPlan) { p.Lessons[0].StartMinute = 539 },
		"late":              func(p *domain.TutorPlan) { p.Lessons[0].Duration = 61 },
		"zero duration":     func(p *domain.TutorPlan) { p.Lessons[0].Duration = 0 },
		"weekday":           func(p *domain.TutorPlan) { p.Lessons[0].Weekday = 8 },
		"window duplicate":  func(p *domain.TutorPlan) { p.Windows = append(p.Windows, p.Windows[0]) },
		"window off grid":   func(p *domain.TutorPlan) { p.Windows[0].StartMinute = 1289 },
		"window late":       func(p *domain.TutorPlan) { p.Windows[0].StartMinute = 1320 },
		"zero divisor":      func(p *domain.TutorPlan) { p.Calculator.People = 0 },
		"nan":               func(p *domain.TutorPlan) { p.Calculator.Weeks = math.NaN() },
		"inf":               func(p *domain.TutorPlan) { p.Calculator.PreparationHours = math.Inf(1) },
		"negative revision": func(p *domain.TutorPlan) { p.Revision = -1 },
		"overflow revision": func(p *domain.TutorPlan) { p.Revision = math.MaxInt64 },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			p := sampleTutorPlan()
			change(&p)
			if validateTutorPlan(&p) == nil {
				t.Fatal("accepted invalid plan")
			}
		})
	}
}

func ptrPlan(p domain.TutorPlan) *domain.TutorPlan { return &p }
