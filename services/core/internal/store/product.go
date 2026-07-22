package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

const SessionCookieName = "repet_session"

func (s *Store) RegisterUser(ctx context.Context, email, displayName, password string, role domain.Role, userAgent, ipAddress string) (domain.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)
	if email == "" || !strings.Contains(email, "@") || displayName == "" || len(password) < 8 || !role.Valid() || role == domain.RoleAdmin {
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
	_, err = s.db.Exec(ctx, `
		insert into users (id, organization_id, email, display_name, role, password_hash)
		values ($1, $2, $3, $4, $5, $6)
	`, id, DefaultOrganizationID, email, displayName, string(role), string(hash))
	if err != nil {
		return domain.User{}, "", apperr.ErrConflict
	}
	if role == domain.RoleTutor {
		if _, err := s.db.Exec(ctx, `insert into tutor_profiles(user_id) values($1) on conflict do nothing`, id); err != nil {
			return domain.User{}, "", err
		}
	} else {
		if _, err := s.db.Exec(ctx, `insert into student_profiles(user_id) values($1) on conflict do nothing`, id); err != nil {
			return domain.User{}, "", err
		}
	}
	user, err := s.GetUser(ctx, id)
	if err != nil {
		return domain.User{}, "", err
	}
	token, err := s.CreateSession(ctx, user, userAgent, ipAddress)
	if err != nil {
		return domain.User{}, "", err
	}
	_ = s.Audit(ctx, user, "identity.user_registered", "user", user.ID, nil)
	return user, token, nil
}

func (s *Store) LoginUser(ctx context.Context, email, password, userAgent, ipAddress string) (domain.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var user domain.User
	var passwordHash string
	err := s.db.QueryRow(ctx, `
		select id::text, organization_id::text, email, display_name, role, created_at, coalesce(password_hash, '')
		from users
		where organization_id=$1 and email=$2
	`, DefaultOrganizationID, email).Scan(&user.ID, &user.OrganizationID, &user.Email, &user.DisplayName, &user.Role, &user.CreatedAt, &passwordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, "", apperr.ErrUnauthorized
		}
		return domain.User{}, "", err
	}
	if passwordHash == "" || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		_ = s.Audit(ctx, user, "identity.login_failed", "user", user.ID, nil)
		return domain.User{}, "", apperr.ErrUnauthorized
	}
	token, err := s.CreateSession(ctx, user, userAgent, ipAddress)
	if err != nil {
		return domain.User{}, "", err
	}
	_ = s.Audit(ctx, user, "identity.login_succeeded", "session", "", nil)
	return user, token, nil
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
		select u.id::text, u.organization_id::text, u.email, u.display_name, u.role, u.created_at
		from sessions s
		join users u on u.id=s.user_id
		where s.token_hash=$1 and s.revoked_at is null and s.expires_at > now()
	`, platform.TokenHash(token))
	var user domain.User
	if err := row.Scan(&user.ID, &user.OrganizationID, &user.Email, &user.DisplayName, &user.Role, &user.CreatedAt); err != nil {
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
	if displayName == "" || timezone == "" || locale == "" {
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
	_ = s.Audit(ctx, user, "profile.updated", "user", user.ID, nil)
	return user, nil
}

func (s *Store) EnsureConversationsForRelations(ctx context.Context, actor domain.User) error {
	rows, err := s.db.Query(ctx, `
		select id::text, tutor_id::text, student_id::text
		from relations
		where organization_id=$1 and status='active' and ($2='admin' or tutor_id=$3 or student_id=$3)
	`, actor.OrganizationID, string(actor.Role), actor.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var relationID, tutorID, studentID string
		if err := rows.Scan(&relationID, &tutorID, &studentID); err != nil {
			return err
		}
		id, err := platform.NewUUID()
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(ctx, `
			insert into conversations (id, organization_id, relation_id, tutor_id, student_id)
			values ($1, $2, $3, $4, $5)
			on conflict (organization_id, tutor_id, student_id) do nothing
		`, id, actor.OrganizationID, relationID, tutorID, studentID); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *Store) ListConversations(ctx context.Context, actor domain.User) ([]domain.Conversation, error) {
	if err := s.EnsureConversationsForRelations(ctx, actor); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		select c.id::text, c.tutor_id::text, tutor.display_name, c.student_id::text, student.display_name,
		  coalesce((select body from messages m where m.conversation_id=c.id order by m.created_at desc limit 1), ''),
		  c.created_at
		from conversations c
		join users tutor on tutor.id=c.tutor_id
		join users student on student.id=c.student_id
		where c.organization_id=$1 and ($2='admin' or c.tutor_id=$3 or c.student_id=$3)
		order by c.created_at desc
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

func (s *Store) ListMessages(ctx context.Context, actor domain.User, conversationID string) ([]domain.Message, error) {
	if err := s.authorizeConversation(ctx, actor, conversationID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		select id::text, conversation_id::text, author_id::text, body, created_at
		from messages
		where organization_id=$1 and conversation_id=$2
		order by created_at asc, id asc
		limit 80
	`, actor.OrganizationID, conversationID)
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
	return out, rows.Err()
}

func (s *Store) SendMessage(ctx context.Context, actor domain.User, conversationID, body, idempotencyKey string) (domain.Message, error) {
	body = strings.TrimSpace(body)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if body == "" || len(body) > 4000 || idempotencyKey == "" {
		return domain.Message{}, apperr.ErrBadRequest
	}
	if err := s.authorizeConversation(ctx, actor, conversationID); err != nil {
		return domain.Message{}, err
	}
	id, err := platform.NewUUID()
	if err != nil {
		return domain.Message{}, err
	}
	var msg domain.Message
	err = s.db.QueryRow(ctx, `
		insert into messages (id, organization_id, conversation_id, author_id, body, idempotency_key)
		values ($1, $2, $3, $4, $5, $6)
		on conflict (conversation_id, author_id, idempotency_key) do update set body=messages.body
		returning id::text, conversation_id::text, author_id::text, body, created_at
	`, id, actor.OrganizationID, conversationID, actor.ID, body, idempotencyKey).Scan(&msg.ID, &msg.ConversationID, &msg.AuthorID, &msg.Body, &msg.CreatedAt)
	if err != nil {
		return domain.Message{}, err
	}
	_ = s.Audit(ctx, actor, "message.sent", "conversation", conversationID, nil)
	return msg, nil
}

func (s *Store) ListAssignments(ctx context.Context, actor domain.User) ([]domain.Assignment, error) {
	rows, err := s.db.Query(ctx, `
		select a.id::text, a.tutor_id::text, a.student_id::text, student.display_name, a.title, a.subject, a.topic, a.body,
		  a.status, a.due_at, a.max_score, sub.score, coalesce(sub.feedback, ''), coalesce(sub.answer, ''), a.created_at
		from assignments a
		join users student on student.id=a.student_id
		left join assignment_submissions sub on sub.assignment_id=a.id and sub.student_id=a.student_id
		where a.organization_id=$1 and ($2='admin' or a.tutor_id=$3 or a.student_id=$3)
		order by a.due_at asc
	`, actor.OrganizationID, string(actor.Role), actor.ID)
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
	if strings.TrimSpace(studentID) == "" || strings.TrimSpace(title) == "" || strings.TrimSpace(body) == "" || !dueAt.After(time.Now().Add(-time.Hour)) {
		return domain.Assignment{}, apperr.ErrBadRequest
	}
	if actor.Role == domain.RoleTutor {
		if _, err := s.GetRelationByParticipants(ctx, actor, actor.ID, studentID); err != nil {
			return domain.Assignment{}, err
		}
	}
	id, err := platform.NewUUID()
	if err != nil {
		return domain.Assignment{}, err
	}
	var item domain.Assignment
	err = s.db.QueryRow(ctx, `
		insert into assignments (id, organization_id, tutor_id, student_id, title, subject, topic, body, status, due_at, max_score)
		values ($1, $2, $3, $4, $5, $6, $7, $8, 'assigned', $9, $10)
		returning id::text, tutor_id::text, student_id::text, title, subject, topic, body, status, due_at, max_score, created_at
	`, id, actor.OrganizationID, actor.ID, studentID, limit(title, 180), limit(subject, 80), limit(topic, 120), limit(body, 4000), dueAt, maxScore).Scan(&item.ID, &item.TutorID, &item.StudentID, &item.Title, &item.Subject, &item.Topic, &item.Body, &item.Status, &item.DueAt, &item.MaxScore, &item.CreatedAt)
	if err != nil {
		return domain.Assignment{}, err
	}
	_ = s.CreateNotification(ctx, actor.OrganizationID, studentID, "Новое задание", item.Title, "/tasks?task="+item.ID, "assignment:"+item.ID+":published")
	_ = s.Audit(ctx, actor, "assignment.published", "assignment", item.ID, nil)
	return item, nil
}

func (s *Store) SubmitAssignment(ctx context.Context, actor domain.User, assignmentID, answer string) (domain.Assignment, error) {
	if actor.Role != domain.RoleStudent {
		return domain.Assignment{}, apperr.ErrForbidden
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return domain.Assignment{}, apperr.ErrBadRequest
	}
	assignment, err := s.getAssignmentForActor(ctx, actor, assignmentID)
	if err != nil {
		return domain.Assignment{}, err
	}
	if assignment.StudentID != actor.ID {
		return domain.Assignment{}, apperr.ErrForbidden
	}
	if assignment.Status == "done" || assignment.Status == "archived" {
		return domain.Assignment{}, apperr.ErrInvalidState
	}
	subID, err := platform.NewUUID()
	if err != nil {
		return domain.Assignment{}, err
	}
	_, err = s.db.Exec(ctx, `
		insert into assignment_submissions (id, organization_id, assignment_id, student_id, answer)
		values ($1, $2, $3, $4, $5)
		on conflict (assignment_id, student_id) do update set answer=excluded.answer, submitted_at=now(), score=null, feedback='', graded_at=null
	`, subID, actor.OrganizationID, assignmentID, actor.ID, limit(answer, 8000))
	if err != nil {
		return domain.Assignment{}, err
	}
	_, err = s.db.Exec(ctx, `update assignments set status='submitted' where id=$1 and organization_id=$2`, assignmentID, actor.OrganizationID)
	if err != nil {
		return domain.Assignment{}, err
	}
	_ = s.CreateNotification(ctx, actor.OrganizationID, assignment.TutorID, "Работа отправлена", assignment.Title, "/tasks?task="+assignment.ID, "assignment:"+assignment.ID+":submitted")
	_ = s.Audit(ctx, actor, "assignment.submitted", "assignment", assignmentID, nil)
	return s.getAssignmentForActor(ctx, actor, assignmentID)
}

func (s *Store) GradeAssignment(ctx context.Context, actor domain.User, assignmentID string, score int, feedback string, revision bool) (domain.Assignment, error) {
	if actor.Role != domain.RoleTutor && actor.Role != domain.RoleAdmin {
		return domain.Assignment{}, apperr.ErrForbidden
	}
	assignment, err := s.getAssignmentForActor(ctx, actor, assignmentID)
	if err != nil {
		return domain.Assignment{}, err
	}
	if actor.Role == domain.RoleTutor && assignment.TutorID != actor.ID {
		return domain.Assignment{}, apperr.ErrForbidden
	}
	status := "done"
	action := "assignment.graded"
	if revision {
		status = "revision"
		action = "assignment.revision_requested"
	}
	if score < 0 || score > assignment.MaxScore {
		return domain.Assignment{}, apperr.ErrBadRequest
	}
	_, err = s.db.Exec(ctx, `
		update assignment_submissions set score=$1, feedback=$2, graded_at=now()
		where assignment_id=$3 and organization_id=$4
	`, score, limit(feedback, 4000), assignmentID, actor.OrganizationID)
	if err != nil {
		return domain.Assignment{}, err
	}
	_, err = s.db.Exec(ctx, `update assignments set status=$1 where id=$2 and organization_id=$3`, status, assignmentID, actor.OrganizationID)
	if err != nil {
		return domain.Assignment{}, err
	}
	_ = s.CreateNotification(ctx, actor.OrganizationID, assignment.StudentID, "Проверка задания", assignment.Title, "/tasks?task="+assignment.ID, "assignment:"+assignment.ID+":"+status)
	_ = s.Audit(ctx, actor, action, "assignment", assignmentID, nil)
	return s.getAssignmentForActor(ctx, actor, assignmentID)
}

func (s *Store) ListNotifications(ctx context.Context, actor domain.User) ([]domain.Notification, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, title, body, href, read_at, created_at
		from notifications
		where organization_id=$1 and user_id=$2
		order by created_at desc
		limit 30
	`, actor.OrganizationID, actor.ID)
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

func (s *Store) CreateNotification(ctx context.Context, organizationID, userID, title, body, href, idempotencyKey string) error {
	id, err := platform.NewUUID()
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `
		insert into notifications (id, organization_id, user_id, title, body, href, idempotency_key)
		values ($1, $2, $3, $4, $5, $6, $7)
		on conflict (user_id, idempotency_key) do nothing
	`, id, organizationID, userID, limit(title, 120), limit(body, 300), limit(href, 300), idempotencyKey)
	return err
}

func (s *Store) MarkNotificationRead(ctx context.Context, actor domain.User, notificationID string) error {
	tag, err := s.db.Exec(ctx, `update notifications set read_at=now() where id=$1 and organization_id=$2 and user_id=$3`, notificationID, actor.OrganizationID, actor.ID)
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
	row := s.db.QueryRow(ctx, `
		select id::text, tutor_id::text, student_id::text, status, created_at
		from relations
		where organization_id=$1 and tutor_id=$2 and student_id=$3 and ($4='admin' or tutor_id=$5 or student_id=$5)
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
