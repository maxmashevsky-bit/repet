package store

import (
	"context"
	"errors"
	"math"
	"slices"

	"github.com/jackc/pgx/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
)

var teachingSubjects = []string{"math", "physics", "informatics", "russian", "english", "chemistry", "biology", "history", "geography", "social_studies"}

func (s *Store) GetTeachingPreferences(ctx context.Context, actor domain.User) (domain.TeachingPreferences, error) {
	if actor.Role != domain.RoleTutor {
		return domain.TeachingPreferences{}, apperr.ErrForbidden
	}
	out := domain.DefaultTeachingPreferences()
	err := s.db.QueryRow(ctx, `select revision,subjects,grades,default_duration_minutes from teaching_preferences where tutor_id=$1 and organization_id=$2`, actor.ID, actor.OrganizationID).Scan(&out.Revision, &out.Subjects, &out.Grades, &out.Duration)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.DefaultTeachingPreferences(), nil
	}
	return out, err
}

func (s *Store) SaveTeachingPreferences(ctx context.Context, actor domain.User, in domain.TeachingPreferences) (domain.TeachingPreferences, error) {
	var out domain.TeachingPreferences
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	if in.Revision < 0 || in.Revision == math.MaxInt64 || len(in.Subjects) < 1 || len(in.Subjects) > len(teachingSubjects) || len(in.Grades) < 1 || len(in.Grades) > 11 || in.Duration < 1 || in.Duration > 240 {
		return out, apperr.ErrBadRequest
	}
	seen := map[string]bool{}
	for _, subject := range in.Subjects {
		if !slices.Contains(teachingSubjects, subject) || seen[subject] {
			return out, apperr.ErrBadRequest
		}
		seen[subject] = true
	}
	grades := map[int]bool{}
	for _, grade := range in.Grades {
		if grade < 1 || grade > 11 || grades[grade] {
			return out, apperr.ErrBadRequest
		}
		grades[grade] = true
	}
	query := `update teaching_preferences set revision=revision+1,subjects=$4,grades=$5,default_duration_minutes=$6,updated_at=clock_timestamp() where tutor_id=$1 and organization_id=$2 and revision=$3 returning revision,subjects,grades,default_duration_minutes`
	if in.Revision == 0 {
		query = `insert into teaching_preferences(tutor_id,organization_id,revision,subjects,grades,default_duration_minutes)
 select id,organization_id,1,$4,$5,$6 from users where id=$1 and organization_id=$2 and role='tutor' and $3::bigint=0 on conflict(tutor_id) do nothing returning revision,subjects,grades,default_duration_minutes`
	}
	err := s.db.QueryRow(ctx, query, actor.ID, actor.OrganizationID, in.Revision, in.Subjects, in.Grades, in.Duration).Scan(&out.Revision, &out.Subjects, &out.Grades, &out.Duration)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, apperr.ErrConflict
	}
	if err == nil {
		s.recordAudit(ctx, actor, "teaching_preferences.updated", "user", actor.ID, nil)
	}
	return out, err
}
