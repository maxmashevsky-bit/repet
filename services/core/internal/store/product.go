package store

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

const SessionCookieName = "repet_session"

func (s *Store) RegisterUser(ctx context.Context, email, displayName, password string, role domain.Role, userAgent, ipAddress string) (domain.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)
	if !validEmail(email) || displayName == "" || len(displayName) > 120 || len(password) < 8 || len(password) > 72 || !role.Valid() || role == domain.RoleAdmin {
		return domain.User{}, "", apperr.ErrBadRequest
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, "", err
	}
	id, err := platform.NewUUID()
	if err != nil {
		return domain.User{}, "", err
	}
	token, err := platform.RandomToken(32)
	if err != nil {
		return domain.User{}, "", err
	}
	sessionID, err := platform.NewUUID()
	if err != nil {
		return domain.User{}, "", err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.User{}, "", err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		insert into users (id, organization_id, email, display_name, role, password_hash)
		values ($1, $2, $3, $4, $5, $6)
	`, id, DefaultOrganizationID, email, displayName, string(role), string(hash))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.User{}, "", apperr.ErrConflict
		}
		return domain.User{}, "", err
	}
	if role == domain.RoleTutor {
		if _, err := tx.Exec(ctx, `insert into tutor_profiles(user_id) values($1)`, id); err != nil {
			return domain.User{}, "", err
		}
	} else {
		if _, err := tx.Exec(ctx, `insert into student_profiles(user_id) values($1)`, id); err != nil {
			return domain.User{}, "", err
		}
	}
	if _, err := tx.Exec(ctx, `insert into sessions (id, organization_id, user_id, token_hash, user_agent, ip_address, expires_at) values ($1,$2,$3,$4,$5,$6,$7)`, sessionID, DefaultOrganizationID, id, platform.TokenHash(token), limit(userAgent, 240), limit(ipAddress, 80), time.Now().UTC().Add(14*24*time.Hour)); err != nil {
		return domain.User{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, "", err
	}
	user, err := s.GetUser(ctx, id)
	if err != nil {
		return domain.User{}, "", err
	}
	s.recordAudit(ctx, user, "identity.user_registered", "user", user.ID, nil)
	return user, token, nil
}

func (s *Store) LoginUser(ctx context.Context, email, password, userAgent, ipAddress string) (domain.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var user domain.User
	var passwordHash string
	err := s.db.QueryRow(ctx, `
		select id::text, organization_id::text, email, display_name, role, timezone, locale, created_at, coalesce(password_hash, '')
		from users
		where organization_id=$1 and email=$2
	`, DefaultOrganizationID, email).Scan(&user.ID, &user.OrganizationID, &user.Email, &user.DisplayName, &user.Role, &user.Timezone, &user.Locale, &user.CreatedAt, &passwordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, "", apperr.ErrUnauthorized
		}
		return domain.User{}, "", err
	}
	if passwordHash == "" || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		s.recordAudit(ctx, user, "identity.login_failed", "user", user.ID, nil)
		return domain.User{}, "", apperr.ErrUnauthorized
	}
	token, err := s.CreateSession(ctx, user, userAgent, ipAddress)
	if err != nil {
		return domain.User{}, "", err
	}
	s.recordAudit(ctx, user, "identity.login_succeeded", "session", "", nil)
	return user, token, nil
}

func (s *Store) IssuePasswordReset(ctx context.Context, email string) (string, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !validEmail(email) {
		return "", "", apperr.ErrBadRequest
	}
	var userID string
	err := s.db.QueryRow(ctx, `select id::text from users where organization_id=$1 and email=$2 and password_hash is not null`, DefaultOrganizationID, email).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	token, err := platform.RandomToken(32)
	if err != nil {
		return "", "", err
	}
	id, err := platform.NewUUID()
	if err != nil {
		return "", "", err
	}
	if _, err := s.db.Exec(ctx, `insert into password_reset_tokens (id, user_id, token_hash, expires_at) values ($1,$2,$3,$4) on conflict (user_id) do update set token_hash=excluded.token_hash, expires_at=excluded.expires_at, used_at=null, created_at=now()`, id, userID, platform.TokenHash(token), time.Now().UTC().Add(30*time.Minute)); err != nil {
		return "", "", err
	}
	return token, email, nil
}

func (s *Store) ResetPassword(ctx context.Context, token, password string) error {
	if len(token) < 32 || len(token) > 128 || len(password) < 8 || len(password) > 72 {
		return apperr.ErrBadRequest
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var userID string
	err = tx.QueryRow(ctx, `select user_id::text from password_reset_tokens where token_hash=$1 and used_at is null and expires_at > now() for update`, platform.TokenHash(token)).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.ErrInvalidState
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `update users set password_hash=$1, updated_at=now() where id=$2`, string(hash), userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `update sessions set revoked_at=now() where user_id=$1 and revoked_at is null`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `update password_reset_tokens set used_at=now() where user_id=$1 and used_at is null`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) CreateSession(ctx context.Context, user domain.User, userAgent, ipAddress string) (string, error) {
	id, err := platform.NewUUID()
	if err != nil {
		return "", err
	}
	token, err := platform.RandomToken(32)
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec(ctx, `
		insert into sessions (id, organization_id, user_id, token_hash, user_agent, ip_address, expires_at)
		values ($1, $2, $3, $4, $5, $6, $7)
	`, id, user.OrganizationID, user.ID, platform.TokenHash(token), limit(userAgent, 240), limit(ipAddress, 80), time.Now().UTC().Add(14*24*time.Hour))
	return token, err
}

func (s *Store) UserBySessionToken(ctx context.Context, token string) (domain.User, error) {
	if token == "" {
		return domain.User{}, apperr.ErrUnauthorized
	}
	row := s.db.QueryRow(ctx, `
		select u.id::text, u.organization_id::text, u.email, u.display_name, u.role, u.timezone, u.locale, u.created_at
		from sessions s
		join users u on u.id=s.user_id
		where s.token_hash=$1 and s.revoked_at is null and s.expires_at > now()
	`, platform.TokenHash(token))
	var user domain.User
	if err := row.Scan(&user.ID, &user.OrganizationID, &user.Email, &user.DisplayName, &user.Role, &user.Timezone, &user.Locale, &user.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, apperr.ErrUnauthorized
		}
		return domain.User{}, err
	}
	return user, nil
}

func (s *Store) RevokeSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	_, err := s.db.Exec(ctx, `update sessions set revoked_at=now() where token_hash=$1 and revoked_at is null`, platform.TokenHash(token))
	return err
}

func (s *Store) UpdateProfile(ctx context.Context, actor domain.User, displayName, timezone, locale string) (domain.User, error) {
	displayName = strings.TrimSpace(displayName)
	timezone = strings.TrimSpace(timezone)
	locale = strings.TrimSpace(locale)
	if displayName == "" || len(displayName) > 120 || timezone == "" || len(timezone) > 100 || (locale != "ru" && locale != "en") {
		return domain.User{}, apperr.ErrBadRequest
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return domain.User{}, apperr.ErrBadRequest
	}
	if _, err := s.db.Exec(ctx, `
		update users set display_name=$1, timezone=$2, locale=$3, updated_at=now()
		where id=$4 and organization_id=$5
	`, displayName, timezone, locale, actor.ID, actor.OrganizationID); err != nil {
		return domain.User{}, err
	}
	user, err := s.GetUser(ctx, actor.ID)
	if err != nil {
		return domain.User{}, err
	}
	s.recordAudit(ctx, user, "profile.updated", "user", user.ID, nil)
	return user, nil
}

func (s *Store) ListConversations(ctx context.Context, actor domain.User) ([]domain.Conversation, error) {
	rows, err := s.db.Query(ctx, `
		select c.id::text, c.tutor_id::text, tutor.display_name, c.student_id::text, student.display_name,
		  coalesce((select body from messages m where m.conversation_id=c.id order by m.created_at desc, m.id desc limit 1), ''),
		  c.created_at
		from conversations c
		join users tutor on tutor.id=c.tutor_id
		join users student on student.id=c.student_id
		where c.organization_id=$1 and ($2='admin' or c.tutor_id=$3 or c.student_id=$3)
		order by c.created_at desc
		limit 100
	`, actor.OrganizationID, string(actor.Role), actor.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Conversation
	for rows.Next() {
		var item domain.Conversation
		if err := rows.Scan(&item.ID, &item.TutorID, &item.TutorName, &item.StudentID, &item.StudentName, &item.LastMessage, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) ListMessages(ctx context.Context, actor domain.User, conversationID string, offset int) ([]domain.Message, error) {
	if !platform.ValidUUID(conversationID) || offset < 0 || offset > 100000 {
		return nil, apperr.ErrBadRequest
	}
	if err := s.authorizeConversation(ctx, actor, conversationID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		select id::text, conversation_id::text, author_id::text, body, created_at
		from messages
		where organization_id=$1 and conversation_id=$2
		order by created_at desc, id desc
		limit 80
		offset $3
	`, actor.OrganizationID, conversationID, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Message
	for rows.Next() {
		var msg domain.Message
		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.AuthorID, &msg.Body, &msg.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (s *Store) SendMessage(ctx context.Context, actor domain.User, conversationID, body, idempotencyKey string) (domain.Message, error) {
	body = strings.TrimSpace(body)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if !platform.ValidUUID(conversationID) || body == "" || len(body) > 4000 || idempotencyKey == "" || len(idempotencyKey) > 120 {
		return domain.Message{}, apperr.ErrBadRequest
	}
	if err := s.authorizeConversation(ctx, actor, conversationID); err != nil {
		return domain.Message{}, err
	}
	id, err := platform.NewUUID()
	if err != nil {
		return domain.Message{}, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.Message{}, err
	}
	defer tx.Rollback(ctx)
	var msg domain.Message
	err = tx.QueryRow(ctx, `
		insert into messages (id, organization_id, conversation_id, author_id, body, idempotency_key)
		values ($1, $2, $3, $4, $5, $6)
		on conflict (conversation_id, author_id, idempotency_key) do update set body=messages.body
		returning id::text, conversation_id::text, author_id::text, body, created_at
	`, id, actor.OrganizationID, conversationID, actor.ID, body, idempotencyKey).Scan(&msg.ID, &msg.ConversationID, &msg.AuthorID, &msg.Body, &msg.CreatedAt)
	if err != nil {
		return domain.Message{}, err
	}
	var recipientID string
	if err := tx.QueryRow(ctx, `select case when tutor_id=$2 then student_id::text else tutor_id::text end from conversations where id=$1 and organization_id=$3`, conversationID, actor.ID, actor.OrganizationID).Scan(&recipientID); err != nil {
		return domain.Message{}, err
	}
	if err := insertNotification(ctx, tx, actor.OrganizationID, recipientID, "Новое сообщение", actor.DisplayName, "/messages?conversation="+conversationID, "message:"+msg.ID); err != nil {
		return domain.Message{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Message{}, err
	}
	s.recordAudit(ctx, actor, "message.sent", "conversation", conversationID, nil)
	return msg, nil
}

func (s *Store) ListAssignments(ctx context.Context, actor domain.User) ([]domain.Assignment, error) {
	return s.ListAssignmentsPage(ctx, actor, 0)
}

func (s *Store) ListAssignmentsPage(ctx context.Context, actor domain.User, offset int) ([]domain.Assignment, error) {
	if offset < 0 || offset > 100000 {
		return nil, apperr.ErrBadRequest
	}
	rows, err := s.db.Query(ctx, `
		select a.id::text, a.tutor_id::text, a.student_id::text, student.display_name, a.title, a.subject, a.topic, a.body,
		  a.status, a.due_at, a.max_score, sub.score, coalesce(sub.feedback, ''), coalesce(sub.answer, ''), a.created_at
		from assignments a
		join users student on student.id=a.student_id
		left join assignment_submissions sub on sub.assignment_id=a.id and sub.student_id=a.student_id
		where a.organization_id=$1 and ($2='admin' or a.tutor_id=$3 or a.student_id=$3)
		order by a.due_at asc, a.id asc
		limit 100
		offset $4
	`, actor.OrganizationID, string(actor.Role), actor.ID, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Assignment
	for rows.Next() {
		var item domain.Assignment
		if err := rows.Scan(&item.ID, &item.TutorID, &item.StudentID, &item.StudentName, &item.Title, &item.Subject, &item.Topic, &item.Body, &item.Status, &item.DueAt, &item.MaxScore, &item.Score, &item.Feedback, &item.Answer, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) CreateAssignment(ctx context.Context, actor domain.User, studentID, title, subject, topic, body string, dueAt time.Time, maxScore int) (domain.Assignment, error) {
	if actor.Role != domain.RoleTutor && actor.Role != domain.RoleAdmin {
		return domain.Assignment{}, apperr.ErrForbidden
	}
	if !platform.ValidUUID(studentID) || strings.TrimSpace(title) == "" || len(title) > 180 || strings.TrimSpace(body) == "" || len(body) > 4000 || len(subject) > 80 || len(topic) > 120 || !dueAt.After(time.Now()) || maxScore < 1 || maxScore > 1000 {
		return domain.Assignment{}, apperr.ErrBadRequest
	}
	if _, err := s.GetRelationByParticipants(ctx, actor, actor.ID, studentID); err != nil {
		return domain.Assignment{}, err
	}
	id, err := platform.NewUUID()
	if err != nil {
		return domain.Assignment{}, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.Assignment{}, err
	}
	defer tx.Rollback(ctx)
	var item domain.Assignment
	err = tx.QueryRow(ctx, `
		insert into assignments (id, organization_id, tutor_id, student_id, title, subject, topic, body, status, due_at, max_score)
		values ($1, $2, $3, $4, $5, $6, $7, $8, 'assigned', $9, $10)
		returning id::text, tutor_id::text, student_id::text, title, subject, topic, body, status, due_at, max_score, created_at
	`, id, actor.OrganizationID, actor.ID, studentID, limit(title, 180), limit(subject, 80), limit(topic, 120), limit(body, 4000), dueAt, maxScore).Scan(&item.ID, &item.TutorID, &item.StudentID, &item.Title, &item.Subject, &item.Topic, &item.Body, &item.Status, &item.DueAt, &item.MaxScore, &item.CreatedAt)
	if err != nil {
		return domain.Assignment{}, err
	}
	if err := insertNotification(ctx, tx, actor.OrganizationID, studentID, "Новое задание", item.Title, "/tasks?task="+item.ID, "assignment:"+item.ID+":published"); err != nil {
		return domain.Assignment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Assignment{}, err
	}
	s.recordAudit(ctx, actor, "assignment.published", "assignment", item.ID, nil)
	return s.getAssignmentForActor(ctx, actor, item.ID)
}

func (s *Store) SubmitAssignment(ctx context.Context, actor domain.User, assignmentID, answer string) (domain.Assignment, error) {
	if actor.Role != domain.RoleStudent {
		return domain.Assignment{}, apperr.ErrForbidden
	}
	answer = strings.TrimSpace(answer)
	if !platform.ValidUUID(assignmentID) || answer == "" || len(answer) > 8000 {
		return domain.Assignment{}, apperr.ErrBadRequest
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.Assignment{}, err
	}
	defer tx.Rollback(ctx)
	var tutorID, title, status string
	err = tx.QueryRow(ctx, `select tutor_id::text, title, status from assignments where id=$1 and organization_id=$2 and student_id=$3 for update`, assignmentID, actor.OrganizationID, actor.ID).Scan(&tutorID, &title, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Assignment{}, apperr.ErrNotFound
	}
	if err != nil {
		return domain.Assignment{}, err
	}
	if status != "assigned" && status != "revision" {
		return domain.Assignment{}, apperr.ErrInvalidState
	}
	subID, err := platform.NewUUID()
	if err != nil {
		return domain.Assignment{}, err
	}
	var submittedAt time.Time
	err = tx.QueryRow(ctx, `
		insert into assignment_submissions (id, organization_id, assignment_id, student_id, answer)
		values ($1, $2, $3, $4, $5)
		on conflict (assignment_id, student_id) do update set answer=excluded.answer, submitted_at=now(), score=null, feedback='', graded_at=null
		returning submitted_at
	`, subID, actor.OrganizationID, assignmentID, actor.ID, answer).Scan(&submittedAt)
	if err != nil {
		return domain.Assignment{}, err
	}
	_, err = tx.Exec(ctx, `update assignments set status='submitted' where id=$1 and organization_id=$2`, assignmentID, actor.OrganizationID)
	if err != nil {
		return domain.Assignment{}, err
	}
	if err := insertNotification(ctx, tx, actor.OrganizationID, tutorID, "Работа отправлена", title, "/tasks?task="+assignmentID, "assignment:"+assignmentID+":submitted:"+submittedAt.Format(time.RFC3339Nano)); err != nil {
		return domain.Assignment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Assignment{}, err
	}
	s.recordAudit(ctx, actor, "assignment.submitted", "assignment", assignmentID, nil)
	return s.getAssignmentForActor(ctx, actor, assignmentID)
}

func (s *Store) GradeAssignment(ctx context.Context, actor domain.User, assignmentID string, score int, feedback string, revision bool) (domain.Assignment, error) {
	if actor.Role != domain.RoleTutor && actor.Role != domain.RoleAdmin {
		return domain.Assignment{}, apperr.ErrForbidden
	}
	if !platform.ValidUUID(assignmentID) || len(feedback) > 4000 {
		return domain.Assignment{}, apperr.ErrBadRequest
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.Assignment{}, err
	}
	defer tx.Rollback(ctx)
	var studentID, title, currentStatus string
	var maxScore int
	err = tx.QueryRow(ctx, `select student_id::text, title, status, max_score from assignments where id=$1 and organization_id=$2 and ($3='admin' or tutor_id=$4) for update`, assignmentID, actor.OrganizationID, string(actor.Role), actor.ID).Scan(&studentID, &title, &currentStatus, &maxScore)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Assignment{}, apperr.ErrNotFound
	}
	if err != nil {
		return domain.Assignment{}, err
	}
	if currentStatus != "submitted" {
		return domain.Assignment{}, apperr.ErrInvalidState
	}
	status := "done"
	action := "assignment.graded"
	if revision {
		status = "revision"
		action = "assignment.revision_requested"
	}
	if score < 0 || score > maxScore {
		return domain.Assignment{}, apperr.ErrBadRequest
	}
	tag, err := tx.Exec(ctx, `
		update assignment_submissions set score=$1, feedback=$2, graded_at=now()
		where assignment_id=$3 and organization_id=$4 and student_id=$5
	`, score, strings.TrimSpace(feedback), assignmentID, actor.OrganizationID, studentID)
	if err != nil {
		return domain.Assignment{}, err
	}
	if tag.RowsAffected() == 0 {
		return domain.Assignment{}, apperr.ErrInvalidState
	}
	_, err = tx.Exec(ctx, `update assignments set status=$1 where id=$2 and organization_id=$3`, status, assignmentID, actor.OrganizationID)
	if err != nil {
		return domain.Assignment{}, err
	}
	if err := insertNotification(ctx, tx, actor.OrganizationID, studentID, "Проверка задания", title, "/tasks?task="+assignmentID, "assignment:"+assignmentID+":"+status+":"+time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return domain.Assignment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Assignment{}, err
	}
	s.recordAudit(ctx, actor, action, "assignment", assignmentID, nil)
	return s.getAssignmentForActor(ctx, actor, assignmentID)
}

func (s *Store) ListNotifications(ctx context.Context, actor domain.User) ([]domain.Notification, error) {
	return s.ListNotificationsPage(ctx, actor, 0)
}

func (s *Store) ListNotificationsPage(ctx context.Context, actor domain.User, offset int) ([]domain.Notification, error) {
	if offset < 0 || offset > 100000 {
		return nil, apperr.ErrBadRequest
	}
	rows, err := s.db.Query(ctx, `
		select id::text, title, body, href, read_at, created_at
		from notifications
		where organization_id=$1 and user_id=$2
		order by created_at desc, id desc
		limit 30
		offset $3
	`, actor.OrganizationID, actor.ID, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Notification
	for rows.Next() {
		var item domain.Notification
		if err := rows.Scan(&item.ID, &item.Title, &item.Body, &item.Href, &item.ReadAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func insertNotification(ctx context.Context, tx pgx.Tx, organizationID, userID, title, body, href, idempotencyKey string) error {
	id, err := platform.NewUUID()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		insert into notifications (id, organization_id, user_id, title, body, href, idempotency_key)
		values ($1, $2, $3, $4, $5, $6, $7)
		on conflict (user_id, idempotency_key) do nothing
	`, id, organizationID, userID, limit(title, 120), limit(body, 300), limit(href, 300), idempotencyKey)
	return err
}

func (s *Store) MarkNotificationRead(ctx context.Context, actor domain.User, notificationID string) error {
	if !platform.ValidUUID(notificationID) {
		return apperr.ErrBadRequest
	}
	tag, err := s.db.Exec(ctx, `update notifications set read_at=coalesce(read_at, now()) where id=$1 and organization_id=$2 and user_id=$3`, notificationID, actor.OrganizationID, actor.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (s *Store) MarkAllNotificationsRead(ctx context.Context, actor domain.User) error {
	_, err := s.db.Exec(ctx, `update notifications set read_at=now() where organization_id=$1 and user_id=$2 and read_at is null`, actor.OrganizationID, actor.ID)
	return err
}

func (s *Store) GetRelationByParticipants(ctx context.Context, actor domain.User, tutorID, studentID string) (domain.Relation, error) {
	if !platform.ValidUUID(tutorID) || !platform.ValidUUID(studentID) {
		return domain.Relation{}, apperr.ErrBadRequest
	}
	row := s.db.QueryRow(ctx, `
		select id::text, tutor_id::text, student_id::text, status, created_at
		from relations
		where organization_id=$1 and tutor_id=$2 and student_id=$3 and status='active' and ($4='admin' or tutor_id=$5 or student_id=$5)
	`, actor.OrganizationID, tutorID, studentID, string(actor.Role), actor.ID)
	var rel domain.Relation
	if err := row.Scan(&rel.ID, &rel.TutorID, &rel.StudentID, &rel.Status, &rel.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Relation{}, apperr.ErrNotFound
		}
		return domain.Relation{}, err
	}
	return rel, nil
}

func (s *Store) authorizeConversation(ctx context.Context, actor domain.User, conversationID string) error {
	if !platform.ValidUUID(conversationID) {
		return apperr.ErrBadRequest
	}
	var ok bool
	err := s.db.QueryRow(ctx, `
		select exists(
		  select 1 from conversations
		  where id=$1 and organization_id=$2 and ($3='admin' or tutor_id=$4 or student_id=$4)
		)
	`, conversationID, actor.OrganizationID, string(actor.Role), actor.ID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return apperr.ErrNotFound
	}
	return nil
}

func (s *Store) getAssignmentForActor(ctx context.Context, actor domain.User, assignmentID string) (domain.Assignment, error) {
	if !platform.ValidUUID(assignmentID) {
		return domain.Assignment{}, apperr.ErrBadRequest
	}
	row := s.db.QueryRow(ctx, `
		select a.id::text, a.tutor_id::text, a.student_id::text, student.display_name, a.title, a.subject, a.topic, a.body,
		  a.status, a.due_at, a.max_score, sub.score, coalesce(sub.feedback, ''), coalesce(sub.answer, ''), a.created_at
		from assignments a
		join users student on student.id=a.student_id
		left join assignment_submissions sub on sub.assignment_id=a.id and sub.student_id=a.student_id
		where a.id=$1 and a.organization_id=$2 and ($3='admin' or a.tutor_id=$4 or a.student_id=$4)
	`, assignmentID, actor.OrganizationID, string(actor.Role), actor.ID)
	var item domain.Assignment
	if err := row.Scan(&item.ID, &item.TutorID, &item.StudentID, &item.StudentName, &item.Title, &item.Subject, &item.Topic, &item.Body, &item.Status, &item.DueAt, &item.MaxScore, &item.Score, &item.Feedback, &item.Answer, &item.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Assignment{}, apperr.ErrNotFound
		}
		return domain.Assignment{}, err
	}
	return item, nil
}

func limit(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func validEmail(email string) bool {
	if email == "" || len(email) > 254 || strings.ContainsAny(email, "\r\n") {
		return false
	}
	parsed, err := mail.ParseAddress(email)
	return err == nil && parsed.Address == email
}
