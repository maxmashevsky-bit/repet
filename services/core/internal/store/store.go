package store

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

const DefaultOrganizationID = "00000000-0000-4000-8000-000000000001"

type Store struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

func (s *Store) EnsureUser(ctx context.Context, p auth.Principal) (domain.User, error) {
	_, err := s.db.Exec(ctx, `
		insert into users (id, organization_id, email, display_name, role)
		values ($1, $2, $3, $4, $5)
		on conflict (id) do update set
		  email=excluded.email,
		  display_name=excluded.display_name,
		  role=excluded.role,
		  updated_at=now()
	`, p.UserID, DefaultOrganizationID, p.Email, p.DisplayName, string(p.Role))
	if err != nil {
		return domain.User{}, err
	}
	if p.Role == domain.RoleTutor || p.Role == domain.RoleAdmin {
		if _, err := s.db.Exec(ctx, `insert into tutor_profiles(user_id) values($1) on conflict do nothing`, p.UserID); err != nil {
			return domain.User{}, err
		}
	}
	if p.Role == domain.RoleStudent {
		if _, err := s.db.Exec(ctx, `insert into student_profiles(user_id) values($1) on conflict do nothing`, p.UserID); err != nil {
			return domain.User{}, err
		}
	}
	return s.GetUser(ctx, p.UserID)
}

func (s *Store) GetUser(ctx context.Context, userID string) (domain.User, error) {
	row := s.db.QueryRow(ctx, `
		select id::text, organization_id::text, email, display_name, role, timezone, locale, created_at
		from users where id=$1
	`, userID)
	var u domain.User
	if err := row.Scan(&u.ID, &u.OrganizationID, &u.Email, &u.DisplayName, &u.Role, &u.Timezone, &u.Locale, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, apperr.ErrNotFound
		}
		return domain.User{}, err
	}
	return u, nil
}

func (s *Store) CreateInvitation(ctx context.Context, actor domain.User, studentEmail, studentName string) (domain.Invitation, string, error) {
	if actor.Role != domain.RoleTutor && actor.Role != domain.RoleAdmin {
		return domain.Invitation{}, "", apperr.ErrForbidden
	}
	studentEmail = strings.ToLower(strings.TrimSpace(studentEmail))
	if !validEmail(studentEmail) || len(studentName) > 120 {
		return domain.Invitation{}, "", apperr.ErrBadRequest
	}
	id, err := platform.NewUUID()
	if err != nil {
		return domain.Invitation{}, "", err
	}
	token, err := platform.RandomToken(32)
	if err != nil {
		return domain.Invitation{}, "", err
	}
	inv := domain.Invitation{
		ID:           id,
		TutorID:      actor.ID,
		StudentEmail: studentEmail,
		StudentName:  strings.TrimSpace(studentName),
		Status:       "pending",
		ExpiresAt:    time.Now().UTC().Add(7 * 24 * time.Hour),
	}
	err = s.db.QueryRow(ctx, `
		insert into invitations (id, organization_id, tutor_id, student_email, student_name, token_hash, status, expires_at)
		values ($1, $2, $3, $4, $5, $6, 'pending', $7)
		returning created_at
	`, inv.ID, actor.OrganizationID, actor.ID, inv.StudentEmail, inv.StudentName, platform.TokenHash(token), inv.ExpiresAt).Scan(&inv.CreatedAt)
	if err != nil {
		return domain.Invitation{}, "", err
	}
	s.recordAudit(ctx, actor, "invitation.created", "invitation", inv.ID, map[string]any{"student_email": inv.StudentEmail})
	return inv, token, nil
}

func (s *Store) AcceptInvitation(ctx context.Context, actor domain.User, token string) (domain.Relation, error) {
	if actor.Role != domain.RoleStudent {
		return domain.Relation{}, apperr.ErrForbidden
	}
	if len(token) < 32 || len(token) > 128 {
		return domain.Relation{}, apperr.ErrBadRequest
	}
	hash := platform.TokenHash(token)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.Relation{}, err
	}
	defer tx.Rollback(ctx)

	var invitationID, orgID, tutorID, studentEmail, status string
	var expiresAt time.Time
	err = tx.QueryRow(ctx, `
		select id::text, organization_id::text, tutor_id::text, student_email, status, expires_at
		from invitations
		where token_hash=$1
		for update
	`, hash).Scan(&invitationID, &orgID, &tutorID, &studentEmail, &status, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Relation{}, apperr.ErrNotFound
		}
		return domain.Relation{}, err
	}
	if orgID != actor.OrganizationID || !strings.EqualFold(studentEmail, actor.Email) {
		return domain.Relation{}, apperr.ErrForbidden
	}
	if status != "pending" || time.Now().UTC().After(expiresAt) {
		return domain.Relation{}, apperr.ErrInvalidState
	}
	relationID, err := platform.NewUUID()
	if err != nil {
		return domain.Relation{}, err
	}
	rel := domain.Relation{ID: relationID, TutorID: tutorID, StudentID: actor.ID, Status: "active"}
	err = tx.QueryRow(ctx, `
		insert into relations (id, organization_id, tutor_id, student_id, invitation_id, status)
		values ($1, $2, $3, $4, $5, 'active')
		on conflict (organization_id, tutor_id, student_id) do update set status='active'
		returning id::text, created_at
	`, rel.ID, actor.OrganizationID, tutorID, actor.ID, invitationID).Scan(&rel.ID, &rel.CreatedAt)
	if err != nil {
		return domain.Relation{}, err
	}
	if _, err := tx.Exec(ctx, `
		update invitations
		set status='accepted', accepted_by=$1, accepted_at=now()
		where id=$2
	`, actor.ID, invitationID); err != nil {
		return domain.Relation{}, err
	}
	conversationID, err := platform.NewUUID()
	if err != nil {
		return domain.Relation{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into conversations (id, organization_id, relation_id, tutor_id, student_id)
		values ($1, $2, $3, $4, $5)
		on conflict (organization_id, tutor_id, student_id) do nothing
	`, conversationID, actor.OrganizationID, rel.ID, tutorID, actor.ID); err != nil {
		return domain.Relation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Relation{}, err
	}
	s.recordAudit(ctx, actor, "invitation.accepted", "relation", rel.ID, map[string]any{"invitation_id": invitationID})
	return rel, nil
}

func (s *Store) ListRelations(ctx context.Context, actor domain.User) ([]domain.Relation, error) {
	query := `
		select id::text, tutor_id::text, student_id::text, status, created_at
		from relations
		where organization_id=$1 and ($2='admin' or tutor_id=$3 or student_id=$3)
		order by created_at desc
	`
	rows, err := s.db.Query(ctx, query, actor.OrganizationID, string(actor.Role), actor.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Relation
	for rows.Next() {
		var rel domain.Relation
		if err := rows.Scan(&rel.ID, &rel.TutorID, &rel.StudentID, &rel.Status, &rel.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, rel)
	}
	return out, rows.Err()
}

func (s *Store) CreateLesson(ctx context.Context, actor domain.User, relationID, title string, startsAt, endsAt time.Time) (domain.Lesson, error) {
	title = strings.TrimSpace(title)
	if !platform.ValidUUID(relationID) || title == "" || len(title) > 180 || startsAt.IsZero() || !endsAt.After(startsAt) {
		return domain.Lesson{}, apperr.ErrBadRequest
	}
	rel, err := s.GetRelationForActor(ctx, actor, relationID)
	if err != nil {
		return domain.Lesson{}, err
	}
	if rel.Status != "active" {
		return domain.Lesson{}, apperr.ErrInvalidState
	}
	if actor.Role != domain.RoleTutor && actor.Role != domain.RoleAdmin {
		return domain.Lesson{}, apperr.ErrForbidden
	}
	if actor.Role == domain.RoleTutor && rel.TutorID != actor.ID {
		return domain.Lesson{}, apperr.ErrForbidden
	}
	id, err := platform.NewUUID()
	if err != nil {
		return domain.Lesson{}, err
	}
	lesson := domain.Lesson{
		ID:         id,
		RelationID: rel.ID,
		TutorID:    rel.TutorID,
		StudentID:  rel.StudentID,
		Title:      title,
		StartsAt:   startsAt,
		EndsAt:     endsAt,
		Status:     "scheduled",
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.Lesson{}, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `
		insert into lessons (id, organization_id, relation_id, tutor_id, student_id, title, starts_at, ends_at, status, created_by)
		values ($1, $2, $3, $4, $5, $6, $7, $8, 'scheduled', $9)
		returning created_at
	`, lesson.ID, actor.OrganizationID, lesson.RelationID, lesson.TutorID, lesson.StudentID, lesson.Title, lesson.StartsAt, lesson.EndsAt, actor.ID).Scan(&lesson.CreatedAt)
	if err != nil {
		return domain.Lesson{}, err
	}
	if err := insertNotification(ctx, tx, actor.OrganizationID, lesson.StudentID, "Новое занятие", lesson.Title, "/calendar?date="+lesson.StartsAt.Format("2006-01-02"), "lesson:"+lesson.ID+":created"); err != nil {
		return domain.Lesson{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Lesson{}, err
	}
	s.recordAudit(ctx, actor, "lesson.created", "lesson", lesson.ID, map[string]any{"relation_id": lesson.RelationID})
	return lesson, nil
}

func (s *Store) ListLessons(ctx context.Context, actor domain.User) ([]domain.Lesson, error) {
	return s.ListLessonsPage(ctx, actor, 0)
}

func (s *Store) ListLessonsPage(ctx context.Context, actor domain.User, offset int) ([]domain.Lesson, error) {
	if offset < 0 || offset > 100000 {
		return nil, apperr.ErrBadRequest
	}
	rows, err := s.db.Query(ctx, `
		select id::text, relation_id::text, tutor_id::text, student_id::text, title, starts_at, ends_at, status, created_at
		from lessons
		where organization_id=$1 and ($2='admin' or tutor_id=$3 or student_id=$3)
		order by starts_at desc, id desc
		limit 100
		offset $4
	`, actor.OrganizationID, string(actor.Role), actor.ID, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Lesson
	for rows.Next() {
		var lesson domain.Lesson
		if err := rows.Scan(&lesson.ID, &lesson.RelationID, &lesson.TutorID, &lesson.StudentID, &lesson.Title, &lesson.StartsAt, &lesson.EndsAt, &lesson.Status, &lesson.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, lesson)
	}
	return out, rows.Err()
}

func (s *Store) GetLessonForActor(ctx context.Context, actor domain.User, lessonID string) (domain.Lesson, error) {
	if !platform.ValidUUID(lessonID) {
		return domain.Lesson{}, apperr.ErrBadRequest
	}
	row := s.db.QueryRow(ctx, `
		select id::text, relation_id::text, tutor_id::text, student_id::text, title, starts_at, ends_at, status, created_at
		from lessons
		where id=$1 and organization_id=$2 and ($3='admin' or tutor_id=$4 or student_id=$4)
	`, lessonID, actor.OrganizationID, string(actor.Role), actor.ID)
	var lesson domain.Lesson
	if err := row.Scan(&lesson.ID, &lesson.RelationID, &lesson.TutorID, &lesson.StudentID, &lesson.Title, &lesson.StartsAt, &lesson.EndsAt, &lesson.Status, &lesson.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Lesson{}, apperr.ErrNotFound
		}
		return domain.Lesson{}, err
	}
	return lesson, nil
}

func (s *Store) GetRelationForActor(ctx context.Context, actor domain.User, relationID string) (domain.Relation, error) {
	if !platform.ValidUUID(relationID) {
		return domain.Relation{}, apperr.ErrBadRequest
	}
	row := s.db.QueryRow(ctx, `
		select id::text, tutor_id::text, student_id::text, status, created_at
		from relations
		where id=$1 and organization_id=$2 and ($3='admin' or tutor_id=$4 or student_id=$4)
	`, relationID, actor.OrganizationID, string(actor.Role), actor.ID)
	var rel domain.Relation
	if err := row.Scan(&rel.ID, &rel.TutorID, &rel.StudentID, &rel.Status, &rel.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Relation{}, apperr.ErrNotFound
		}
		return domain.Relation{}, err
	}
	return rel, nil
}

func (s *Store) Audit(ctx context.Context, actor domain.User, action, resourceType, resourceID string, metadata map[string]any) error {
	id, err := platform.NewUUID()
	if err != nil {
		return err
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `
		insert into audit_events (id, organization_id, actor_id, action, resource_type, resource_id, metadata)
		values ($1, $2, $3, $4, $5, $6, $7)
	`, id, actor.OrganizationID, actor.ID, action, resourceType, nullableUUID(resourceID), data)
	return err
}

func (s *Store) recordAudit(ctx context.Context, actor domain.User, action, resourceType, resourceID string, metadata map[string]any) {
	if err := s.Audit(ctx, actor, action, resourceType, resourceID, metadata); err != nil {
		slog.Error("audit write failed", "action", action, "error", err)
	}
}

func nullableUUID(id string) any {
	if id == "" {
		return nil
	}
	return id
}
