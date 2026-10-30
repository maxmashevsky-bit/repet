package store

import (
	"context"
	"strings"

	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

func (s *Store) chatAttachment(ctx context.Context, q catalogQuerier, conv, fileID, assignmentID, lessonID string) (*domain.ChatAttachment, error) {
	if fileID != "" {
		var f domain.ChatFile
		err := q.QueryRow(ctx, `select `+chatFileColumns+` from chat_files where id=$1 and conversation_id=$2 and attached_at is not null`, fileID, conv).Scan(&f.ID, &f.Filename, &f.ContentType, &f.Size)
		if err != nil {
			return nil, catalogError(err)
		}
		return &domain.ChatAttachment{Kind: "file", ID: f.ID, Title: f.Filename, Available: true, File: &f}, nil
	}
	if assignmentID == "" && lessonID == "" {
		return nil, nil
	}
	a := &domain.ChatAttachment{Kind: "assignment"}
	if lessonID != "" {
		a.Kind = "lesson"
	}
	var available bool
	if err := q.QueryRow(ctx, `select chat_reference_available($1,$2,$3)`, conv, nullableUUID(assignmentID), nullableUUID(lessonID)).Scan(&available); err != nil {
		return nil, err
	}
	if !available {
		return a, nil
	}
	a.ID = assignmentID
	var err error
	if lessonID != "" {
		a.ID = lessonID
		err = q.QueryRow(ctx, `select v.title from learning_lessons l join lesson_versions v on v.id=l.published_version_id where l.id=$1`, lessonID).Scan(&a.Title)
	} else {
		err = q.QueryRow(ctx, `select title from assignments where id=$1`, assignmentID).Scan(&a.Title)
	}
	a.Available = err == nil
	return a, catalogError(err)
}

func (s *Store) ListChatReferences(ctx context.Context, actor domain.User, conv, kind, search string, offset int) ([]domain.ChatAttachment, error) {
	if err := s.authorizeConversation(ctx, actor, conv); err != nil {
		return nil, err
	}
	search = strings.TrimSpace(search)
	if (kind != "assignment" && kind != "lesson") || len([]rune(search)) > 120 || offset < 0 || offset > 100000 {
		return nil, apperr.ErrBadRequest
	}
	query := `select a.id::text,a.title from assignments a where chat_reference_available($1,a.id,null) and strpos(lower(a.title),lower($2))>0 order by a.created_at desc,a.id limit 20 offset $3`
	if kind == "lesson" {
		query = `select l.id::text,v.title from learning_lessons l join lesson_versions v on v.id=l.published_version_id where chat_reference_available($1,null,l.id) and strpos(lower(v.title),lower($2))>0 order by l.created_at desc,l.id limit 20 offset $3`
	}
	rows, err := s.db.Query(ctx, query, conv, search, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.ChatAttachment, 0)
	for rows.Next() {
		a := domain.ChatAttachment{Kind: kind, Available: true}
		if err = rows.Scan(&a.ID, &a.Title); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

type ChatAssignmentPreview struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
	DueAt  string `json:"due_at"`
}

func (s *Store) ChatAssignment(ctx context.Context, actor domain.User, conv, id string) (ChatAssignmentPreview, error) {
	var out ChatAssignmentPreview
	if !platform.ValidUUID(id) {
		return out, apperr.ErrBadRequest
	}
	if err := s.authorizeConversation(ctx, actor, conv); err != nil {
		return out, err
	}
	err := s.db.QueryRow(ctx, `select id::text,title,body,kind,status,to_char(due_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') from assignments where id=$1 and chat_reference_available($2,id,null)`, id, conv).Scan(&out.ID, &out.Title, &out.Body, &out.Kind, &out.Status, &out.DueAt)
	return out, catalogError(err)
}
