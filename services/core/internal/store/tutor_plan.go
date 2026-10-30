package store

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

var telegramPlanContact = regexp.MustCompile(`^@[A-Za-z0-9_]{5,32}$`)

func validPlanText(value string, max int) bool {
	return utf8.ValidString(value) && utf8.RuneCountInString(value) <= max && !strings.ContainsFunc(value, unicode.IsControl)
}

func validateTutorPlan(p *domain.TutorPlan) error {
	if p.Revision < 0 || p.Revision == math.MaxInt64 || len(p.Students) > 1000 || len(p.Lessons) > 3000 || len(p.Windows) > 182 {
		return apperr.ErrBadRequest
	}
	ids, names, relations := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for i := range p.Students {
		s := &p.Students[i]
		s.Name, s.Subject, s.Contact = strings.TrimSpace(s.Name), strings.TrimSpace(s.Subject), strings.TrimSpace(s.Contact)
		name := strings.ToLower(s.Name)
		if !platform.ValidUUID(s.ID) || ids[s.ID] || names[name] || s.Name == "" || !validPlanText(s.Name, 120) || !validPlanText(s.Subject, 180) || !validPlanText(s.Contact, 1000) || s.PriceKopecks < 0 || s.PriceKopecks > 1000000000 || s.Duration < 1 || s.Duration > 780 {
			return apperr.ErrBadRequest
		}
		if s.Contact != "" && !telegramPlanContact.MatchString(s.Contact) {
			u, err := url.Parse(s.Contact)
			if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil || strings.ContainsAny(s.Contact, " \\") {
				return apperr.ErrBadRequest
			}
		}
		ids[s.ID], names[name] = true, true
		if s.RelationID != "" {
			if !platform.ValidUUID(s.RelationID) || relations[s.RelationID] {
				return apperr.ErrBadRequest
			}
			relations[s.RelationID] = true
		}
	}
	lessonIDs := map[string]bool{}
	for _, l := range p.Lessons {
		if !platform.ValidUUID(l.ID) || lessonIDs[l.ID] || !ids[l.StudentID] || l.Weekday < 1 || l.Weekday > 7 || l.StartMinute < 540 || l.Duration < 1 || l.Duration > 780 || l.StartMinute+l.Duration > 1320 {
			return apperr.ErrBadRequest
		}
		lessonIDs[l.ID] = true
	}
	windows := map[domain.PlanWindow]bool{}
	for _, w := range p.Windows {
		if w.Weekday < 1 || w.Weekday > 7 || w.StartMinute < 540 || w.StartMinute > 1290 || w.StartMinute%30 != 0 || windows[w] {
			return apperr.ErrBadRequest
		}
		windows[w] = true
	}
	c := p.Calculator
	if c.GoalKopecks < 0 || c.GoalKopecks > 100000000000 || c.PriceKopecks < 1 || c.PriceKopecks > 1000000000 || c.LessonsPerStudent < 1 || c.LessonsPerStudent > 21 || c.Duration < 1 || c.Duration > 780 || math.IsNaN(c.Weeks) || math.IsInf(c.Weeks, 0) || c.Weeks < 1 || c.Weeks > 6 || c.People < 1 || c.People > 100 || math.IsNaN(c.PreparationHours) || math.IsInf(c.PreparationHours, 0) || c.PreparationHours < 0 || c.PreparationHours > 168 {
		return apperr.ErrBadRequest
	}
	if p.Students == nil {
		p.Students = []domain.PlanStudent{}
	}
	if p.Lessons == nil {
		p.Lessons = []domain.PlanLesson{}
	}
	if p.Windows == nil {
		p.Windows = []domain.PlanWindow{}
	}
	return nil
}

func (s *Store) GetTutorPlan(ctx context.Context, actor domain.User) (domain.TutorPlan, error) {
	if actor.Role != domain.RoleTutor {
		return domain.TutorPlan{}, apperr.ErrForbidden
	}
	var data []byte
	var revision int64
	err := s.db.QueryRow(ctx, `select document, revision from tutor_plans where tutor_id=$1 and organization_id=$2`, actor.ID, actor.OrganizationID).Scan(&data, &revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.EmptyTutorPlan(), nil
	}
	if err != nil {
		return domain.TutorPlan{}, err
	}
	var p domain.TutorPlan
	if err = json.Unmarshal(data, &p); err != nil {
		return p, err
	}
	p.Revision = revision
	return p, nil
}

func (s *Store) SaveTutorPlan(ctx context.Context, actor domain.User, p domain.TutorPlan) (domain.TutorPlan, error) {
	if actor.Role != domain.RoleTutor {
		return domain.TutorPlan{}, apperr.ErrForbidden
	}
	if err := validateTutorPlan(&p); err != nil {
		return domain.TutorPlan{}, err
	}
	for _, student := range p.Students {
		if student.RelationID != "" {
			var found bool
			if err := s.db.QueryRow(ctx, `select exists(select 1 from relations where id=$1 and organization_id=$2 and tutor_id=$3 and status='active')`, student.RelationID, actor.OrganizationID, actor.ID).Scan(&found); err != nil {
				return domain.TutorPlan{}, err
			}
			if !found {
				return domain.TutorPlan{}, apperr.ErrBadRequest
			}
		}
	}
	data, err := json.Marshal(p)
	if err != nil {
		return domain.TutorPlan{}, err
	}
	// Compare-and-swap prevents one tab from overwriting another tab's changes.
	if p.Revision == 0 {
		err = s.db.QueryRow(ctx, `insert into tutor_plans(tutor_id,organization_id,revision,document)
		 values($1,$2,1,$3) on conflict(tutor_id) do nothing returning revision`, actor.ID, actor.OrganizationID, data).Scan(&p.Revision)
	} else {
		err = s.db.QueryRow(ctx, `update tutor_plans set revision=revision+1,document=$4::jsonb,updated_at=now()
		 where tutor_id=$1 and organization_id=$2 and revision=$3 returning revision`, actor.ID, actor.OrganizationID, p.Revision, data).Scan(&p.Revision)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.TutorPlan{}, apperr.ErrConflict
	}
	if err != nil {
		return domain.TutorPlan{}, err
	}
	return p, nil
}
