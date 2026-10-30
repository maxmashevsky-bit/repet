package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"time"

	"github.com/jackc/pgx/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/chatfiles"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

const chatFileColumns = `id::text,filename,content_type,size_bytes`
const chatFileQuota int64 = 50 << 20
const chatFileCount = 100

func (s *Store) UploadChatFile(ctx context.Context, actor domain.User, conv, name string, source []byte, requestID string) (domain.ChatFile, error) {
	var out domain.ChatFile
	if !platform.ValidUUID(requestID) {
		return out, apperr.ErrBadRequest
	}
	if err := s.authorizeConversation(ctx, actor, conv); err != nil {
		return out, err
	}
	if s.objects == nil {
		return out, apperr.ErrInvalidState
	}
	content, err := chatfiles.Normalize(name, source)
	if err != nil {
		return out, apperr.ErrBadRequest
	}
	hash := sha256.Sum256(source)
	sourceHash := hex.EncodeToString(hash[:])
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	// Serialize quota, retry and cleanup for this owner across all conversations.
	if _, err = tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1,1521))`, actor.ID); err != nil {
		return out, err
	}
	if err = s.cleanupChatFiles(ctx, tx, actor); err != nil {
		return out, err
	}
	var previousHash, previousConv string
	err = tx.QueryRow(ctx, `select `+chatFileColumns+`,source_hash,conversation_id::text from chat_files where owner_id=$1 and request_id=$2`, actor.ID, requestID).Scan(&out.ID, &out.Filename, &out.ContentType, &out.Size, &previousHash, &previousConv)
	if err == nil {
		if previousHash != sourceHash || previousConv != conv || out.Filename != content.Filename {
			return domain.ChatFile{}, apperr.ErrConflict
		}
		return out, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	var used int64
	var count int
	if err = tx.QueryRow(ctx, `select coalesce(sum(size_bytes),0),count(*) from chat_files where owner_id=$1`, actor.ID).Scan(&used, &count); err != nil {
		return out, err
	}
	if count >= chatFileCount || used+int64(len(content.Bytes)) > chatFileQuota {
		return out, apperr.ErrInvalidState
	}
	object, err := s.objects.Put(ctx, bytes.NewReader(content.Bytes), chatfiles.MaxBytes)
	if err != nil {
		return out, err
	}
	committed := false
	defer func() {
		if !committed {
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			_ = tx.Rollback(cleanup)
			// A lost commit acknowledgement must not delete a successfully recorded object.
			var recorded bool
			if e := s.db.QueryRow(cleanup, `select exists(select 1 from chat_files where object_key=$1)`, object.Key).Scan(&recorded); e == nil && !recorded {
				_ = s.objects.Delete(cleanup, object.Key)
			}
		}
	}()
	id, err := platform.NewUUID()
	if err != nil {
		return out, err
	}
	// Recheck the active relationship in the writing transaction.
	err = tx.QueryRow(ctx, `insert into chat_files(id,organization_id,conversation_id,owner_id,request_id,filename,content_type,size_bytes,object_key,source_hash)
 select $1,$2,c.id,$4,$5,$6,$7,$8,$9,$10 from conversations c where c.id=$3 and c.organization_id=$2 and $4 in(c.tutor_id,c.student_id) and chat_reference_available(c.id,null,null)
 returning `+chatFileColumns, id, actor.OrganizationID, conv, actor.ID, requestID, content.Filename, content.Type, object.Size, object.Key, sourceHash).Scan(&out.ID, &out.Filename, &out.ContentType, &out.Size)
	if err != nil {
		return domain.ChatFile{}, catalogError(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.ChatFile{}, err
	}
	committed = true
	s.recordAudit(ctx, actor, "chat_file.uploaded", "chat_file", out.ID, map[string]any{"size": out.Size, "policy": "cdr-v1"})
	return out, nil
}

func (s *Store) cleanupChatFiles(ctx context.Context, tx pgx.Tx, actor domain.User) error {
	rows, err := tx.Query(ctx, `select id::text,object_key from chat_files where owner_id=$1 and attached_at is null and expires_at<=clock_timestamp() order by expires_at limit 100 for update`, actor.ID)
	if err != nil {
		return err
	}
	type item struct{ id, key string }
	var items []item
	for rows.Next() {
		var v item
		if err = rows.Scan(&v.id, &v.key); err != nil {
			rows.Close()
			return err
		}
		items = append(items, v)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	for _, v := range items {
		if err = s.objects.Delete(ctx, v.key); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if _, err = tx.Exec(ctx, `delete from chat_files where id=$1`, v.id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteChatFile(ctx context.Context, actor domain.User, conv, id string) error {
	if !platform.ValidUUID(id) {
		return apperr.ErrBadRequest
	}
	if err := s.authorizeConversation(ctx, actor, conv); err != nil {
		return err
	}
	if s.objects == nil {
		return apperr.ErrInvalidState
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var key string
	if err = tx.QueryRow(ctx, `select object_key from chat_files where id=$1 and conversation_id=$2 and owner_id=$3 and attached_at is null for update`, id, conv, actor.ID).Scan(&key); err != nil {
		return catalogError(err)
	}
	if err = s.objects.Delete(ctx, key); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if _, err = tx.Exec(ctx, `delete from chat_files where id=$1`, id); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	s.recordAudit(ctx, actor, "chat_file.deleted", "chat_file", id, nil)
	return nil
}

func (s *Store) OpenChatFile(ctx context.Context, actor domain.User, conv, id string) (domain.ChatFile, io.ReadCloser, error) {
	var out domain.ChatFile
	if !platform.ValidUUID(id) {
		return out, nil, apperr.ErrBadRequest
	}
	if err := s.authorizeConversation(ctx, actor, conv); err != nil {
		return out, nil, err
	}
	if s.objects == nil {
		return out, nil, apperr.ErrInvalidState
	}
	var key string
	err := s.db.QueryRow(ctx, `select `+chatFileColumns+`,object_key from chat_files where id=$1 and conversation_id=$2 and organization_id=$3 and (attached_at is not null or (owner_id=$4 and expires_at>clock_timestamp()))`, id, conv, actor.OrganizationID, actor.ID).Scan(&out.ID, &out.Filename, &out.ContentType, &out.Size, &key)
	if err != nil {
		return out, nil, catalogError(err)
	}
	reader, err := s.objects.Open(ctx, key)
	if errors.Is(err, fs.ErrNotExist) {
		err = apperr.ErrNotFound
	}
	if err == nil {
		s.recordAudit(ctx, actor, "chat_file.downloaded", "chat_file", id, nil)
	}
	return out, reader, err
}
