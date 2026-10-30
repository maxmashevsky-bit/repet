package store

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

const guardianLinkJSON = `jsonb_build_object('id',g.id,'student_id',g.student_id,'student_name',u.display_name,'guardian_email',g.guardian_email,'guardian_id',g.guardian_id,'status',g.status,'created_at',g.created_at,'expires_at',g.expires_at,'accepted_at',g.accepted_at,'revoked_at',g.revoked_at)`

func (s *Store) GuardianLinks(ctx context.Context, actor domain.User, f domain.CatalogFilter) (domain.CatalogPage[domain.GuardianLink], error) {
	if err := ValidateCatalogFilter(&f); err != nil {
		return domain.CatalogPage[domain.GuardianLink]{}, err
	}
	return catalogJSON[domain.CatalogPage[domain.GuardianLink]](ctx, s.db, `with eligible as(select g.created_at,g.id,`+guardianLinkJSON+` item from guardian_links g join users u on u.id=g.student_id where g.organization_id=$1 and (g.student_id=$2 or g.guardian_id=$2 and g.status='active')) select jsonb_build_object('items',coalesce((select jsonb_agg(item order by created_at desc,id) from(select * from eligible order by created_at desc,id limit $3 offset $4)p),'[]'::jsonb),'total',(select count(*) from eligible),'limit',$3::int,'offset',$4::int)`, actor.OrganizationID, actor.ID, f.Limit, f.Offset)
}

func (s *Store) CreateGuardianInvitation(ctx context.Context, actor domain.User, in domain.GuardianInviteInput) (domain.GuardianLink, string, error) {
	var out domain.GuardianLink
	if actor.Role != domain.RoleStudent {
		return out, "", apperr.ErrForbidden
	}
	email := strings.ToLower(strings.TrimSpace(in.GuardianEmail))
	if !in.ConfirmShare || !validEmail(email) || strings.EqualFold(email, actor.Email) {
		return out, "", apperr.ErrBadRequest
	}
	token, err := platform.RandomToken(32)
	if err != nil {
		return out, "", err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return out, "", err
	}
	defer tx.Rollback(ctx)
	if err = lockLearning(ctx, tx, actor.ID); err != nil {
		return out, "", err
	}
	if _, err = tx.Exec(ctx, `update guardian_links set status='revoked',revoked_at=clock_timestamp() where student_id=$1 and organization_id=$2 and status='pending' and expires_at<=clock_timestamp()`, actor.ID, actor.OrganizationID); err != nil {
		return out, "", err
	}
	var count int
	if err = tx.QueryRow(ctx, `select count(*) from guardian_links where student_id=$1 and organization_id=$2 and created_at>clock_timestamp()-interval '1 day'`, actor.ID, actor.OrganizationID).Scan(&count); err != nil {
		return out, "", err
	}
	if count >= 10 {
		return out, "", apperr.ErrConflict
	}
	out, err = catalogJSON[domain.GuardianLink](ctx, tx, `with inserted as(insert into guardian_links(organization_id,student_id,guardian_email,token_hash)values($1,$2,$3,$4)returning *)select `+guardianLinkJSON+` from inserted g join users u on u.id=g.student_id`, actor.OrganizationID, actor.ID, email, platform.TokenHash(token))
	if err != nil {
		return out, "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return out, "", err
	}
	s.recordAudit(ctx, actor, "guardian.invited", "guardian_link", out.ID, nil)
	return out, token, nil
}

func (s *Store) GuardianInvitation(ctx context.Context, actor domain.User, token string, accept bool) (domain.GuardianLink, error) {
	var out domain.GuardianLink
	if len(token) < 32 || len(token) > 128 {
		return out, apperr.ErrBadRequest
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	var id, status, studentID, email string
	var guardian *string
	var valid bool
	err = tx.QueryRow(ctx, `select id::text,status,student_id::text,guardian_email,guardian_id::text,expires_at>clock_timestamp() from guardian_links where token_hash=$1 and organization_id=$2 for update`, platform.TokenHash(token), actor.OrganizationID).Scan(&id, &status, &studentID, &email, &guardian, &valid)
	if err != nil {
		return out, catalogError(err)
	}
	if !strings.EqualFold(email, actor.Email) || actor.ID == studentID {
		return out, apperr.ErrNotFound
	}
	if status == "revoked" || status == "pending" && !valid || status == "active" && (guardian == nil || *guardian != actor.ID) {
		return out, apperr.ErrInvalidState
	}
	if accept && status == "pending" {
		if _, err = tx.Exec(ctx, `update guardian_links set status='active',guardian_id=$2,accepted_at=clock_timestamp() where id=$1`, id, actor.ID); err != nil {
			return out, catalogError(err)
		}
	}
	out, err = catalogJSON[domain.GuardianLink](ctx, tx, `select `+guardianLinkJSON+` from guardian_links g join users u on u.id=g.student_id where g.id=$1`, id)
	if err != nil {
		return out, err
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	if accept {
		s.recordAudit(ctx, actor, "guardian.accepted", "guardian_link", id, nil)
	}
	return out, nil
}

func (s *Store) RevokeGuardianLink(ctx context.Context, actor domain.User, id string) error {
	if !platform.ValidUUID(id) {
		return apperr.ErrBadRequest
	}
	tag, err := s.db.Exec(ctx, `update guardian_links set status='revoked',revoked_at=coalesce(revoked_at,clock_timestamp()) where id=$1 and organization_id=$2 and (student_id=$3 or guardian_id=$3) and status<>'revoked'`, id, actor.OrganizationID, actor.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var found bool
		if err = s.db.QueryRow(ctx, `select exists(select 1 from guardian_links where id=$1 and organization_id=$2 and (student_id=$3 or guardian_id=$3) and status='revoked')`, id, actor.OrganizationID, actor.ID).Scan(&found); err != nil {
			return err
		}
		if !found {
			return apperr.ErrNotFound
		}
	}
	s.recordAudit(ctx, actor, "guardian.revoked", "guardian_link", id, nil)
	return nil
}

// This grant is not a tutor/student role and never changes the authority of other product APIs.
func guardianAccess(ctx context.Context, q catalogQuerier, actor domain.User, studentID string) (domain.AnalyticsStudent, error) {
	if !platform.ValidUUID(studentID) {
		return domain.AnalyticsStudent{}, apperr.ErrBadRequest
	}
	return catalogJSON[domain.AnalyticsStudent](ctx, q, `select jsonb_build_object('id',u.id,'display_name',u.display_name) from users u where u.id=$1 and u.organization_id=$2 and u.role='student' and exists(select 1 from guardian_links g where g.student_id=u.id and g.organization_id=$2 and g.guardian_id=$3 and g.status='active')`, studentID, actor.OrganizationID, actor.ID)
}

func (s *Store) MarkFamilyRead(ctx context.Context, actor domain.User, studentID string) error {
	if !platform.ValidUUID(studentID) {
		return apperr.ErrBadRequest
	}
	tag, err := s.db.Exec(ctx, `update guardian_links set seen_at=clock_timestamp() where organization_id=$1 and guardian_id=$2 and student_id=$3 and status='active'`, actor.OrganizationID, actor.ID, studentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (s *Store) CreateGuardianNote(ctx context.Context, actor domain.User, in domain.GuardianNoteInput) (domain.GuardianNote, error) {
	var out domain.GuardianNote
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	body := strings.TrimSpace(in.Body)
	if !platform.ValidUUID(in.StudentID) || !platform.ValidUUID(in.RequestID) || len([]rune(body)) < 1 || len([]rune(body)) > 4000 {
		return out, apperr.ErrBadRequest
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if _, err = analyticsAccess(ctx, tx, actor, in.StudentID); err != nil {
		return out, err
	}
	if err = lockLearning(ctx, tx, actor.ID); err != nil {
		return out, err
	}
	var match bool
	err = tx.QueryRow(ctx, `select id::text,student_id=$3 and body=$4 from guardian_notes where tutor_id=$1 and request_id=$2`, actor.ID, in.RequestID, in.StudentID, body).Scan(&out.ID, &match)
	if err == nil && !match {
		return out, apperr.ErrConflict
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `insert into guardian_notes(organization_id,tutor_id,student_id,body,request_id)values($1,$2,$3,$4,$5)returning id::text`, actor.OrganizationID, actor.ID, in.StudentID, body, in.RequestID).Scan(&out.ID)
		if err != nil {
			return out, err
		}
	}
	out, err = catalogJSON[domain.GuardianNote](ctx, tx, `select jsonb_build_object('id',n.id,'tutor_name',u.display_name,'body',n.body,'created_at',n.created_at) from guardian_notes n join users u on u.id=n.tutor_id where n.id=$1`, out.ID)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

func (s *Store) WithdrawGuardianNote(ctx context.Context, actor domain.User, id string) error {
	if actor.Role != domain.RoleTutor {
		return apperr.ErrForbidden
	}
	if !platform.ValidUUID(id) {
		return apperr.ErrBadRequest
	}
	tag, err := s.db.Exec(ctx, `update guardian_notes set withdrawn_at=coalesce(withdrawn_at,clock_timestamp()) where id=$1 and organization_id=$2 and tutor_id=$3`, id, actor.OrganizationID, actor.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (s *Store) GuardianNotes(ctx context.Context, actor domain.User, studentID string, f domain.CatalogFilter) (domain.CatalogPage[domain.GuardianNote], error) {
	var out domain.CatalogPage[domain.GuardianNote]
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	if err := ValidateCatalogFilter(&f); err != nil {
		return out, err
	}
	if _, err := s.analyticsAccess(ctx, actor, studentID); err != nil {
		return out, err
	}
	return catalogJSON[domain.CatalogPage[domain.GuardianNote]](ctx, s.db, `with eligible as(select n.id,n.created_at,jsonb_build_object('id',n.id,'tutor_name',u.display_name,'body',n.body,'created_at',n.created_at) item from guardian_notes n join users u on u.id=n.tutor_id where n.organization_id=$1 and n.tutor_id=$2 and n.student_id=$3 and n.withdrawn_at is null and exists(select 1 from relations r where r.organization_id=$1 and r.tutor_id=$2 and r.student_id=$3 and r.status='active'))select jsonb_build_object('items',coalesce((select jsonb_agg(item order by created_at desc,id) from(select * from eligible order by created_at desc,id limit $4 offset $5)p),'[]'::jsonb),'total',(select count(*) from eligible),'limit',$4::int,'offset',$5::int)`, actor.OrganizationID, actor.ID, studentID, f.Limit, f.Offset)
}
