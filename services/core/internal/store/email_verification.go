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

func (s *Store) IssueEmailVerification(ctx context.Context, actor domain.User) (string, string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback(ctx)
	var email string
	var verified bool
	if err := tx.QueryRow(ctx, `select email,email_verified_at is not null from users where id=$1 and organization_id=$2 for update`, actor.ID, actor.OrganizationID).Scan(&email, &verified); err != nil {
		return "", "", err
	}
	if verified {
		return "", email, nil
	}
	var recent bool
	if err := tx.QueryRow(ctx, `select exists(select 1 from email_verification_tokens where user_id=$1 and created_at>clock_timestamp()-interval '1 minute')`, actor.ID).Scan(&recent); err != nil {
		return "", "", err
	}
	if recent {
		return "", "", apperr.ErrConflict
	}
	token, err := platform.RandomToken(32)
	if err != nil {
		return "", "", err
	}
	_, err = tx.Exec(ctx, `insert into email_verification_tokens(user_id,email,token_hash) values($1,$2,$3) on conflict(user_id) do update set email=excluded.email,token_hash=excluded.token_hash,created_at=clock_timestamp(),expires_at=clock_timestamp()+interval '1 hour',used_at=null`, actor.ID, email, platform.TokenHash(token))
	if err != nil {
		return "", "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", "", err
	}
	return token, email, nil
}

func (s *Store) VerifyEmail(ctx context.Context, actor domain.User, token string) error {
	token = strings.TrimSpace(token)
	if len(token) < 32 || len(token) > 256 {
		return apperr.ErrBadRequest
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var email string
	if err = tx.QueryRow(ctx, `select email from users where id=$1 and organization_id=$2 for update`, actor.ID, actor.OrganizationID).Scan(&email); err != nil {
		return err
	}
	var valid bool
	err = tx.QueryRow(ctx, `select email=$3 and expires_at>clock_timestamp() from email_verification_tokens where user_id=$1 and token_hash=$2 for update`, actor.ID, platform.TokenHash(token), email).Scan(&valid)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && !valid {
		return apperr.ErrBadRequest
	}
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `update users set email_verified_at=coalesce(email_verified_at,clock_timestamp()) where id=$1`, actor.ID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `update email_verification_tokens set used_at=coalesce(used_at,clock_timestamp()) where user_id=$1`, actor.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
