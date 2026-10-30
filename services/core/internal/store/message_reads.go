package store

import (
	"context"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

func (s *Store) MarkMessagesRead(ctx context.Context, actor domain.User, conversationID, throughID string) (int, error) {
	if !platform.ValidUUID(conversationID) || !platform.ValidUUID(throughID) {
		return 0, apperr.ErrBadRequest
	}
	if err := s.authorizeConversation(ctx, actor, conversationID); err != nil {
		return 0, err
	}
	var unread int
	err := s.db.QueryRow(ctx, `with target as(select m.sequence from messages m join conversations c on c.id=m.conversation_id
 where m.id=$1 and m.conversation_id=$2 and m.organization_id=$3 and (c.tutor_id=$4 or c.student_id=$4)
 and exists(select 1 from relations r where r.organization_id=c.organization_id and r.tutor_id=c.tutor_id and r.student_id=c.student_id and r.status='active')),
 saved as(insert into message_read_cursors(conversation_id,user_id,last_sequence)select $2,$4,sequence from target
 on conflict(conversation_id,user_id)do update set last_sequence=greatest(message_read_cursors.last_sequence,excluded.last_sequence),updated_at=case when excluded.last_sequence>message_read_cursors.last_sequence then clock_timestamp()else message_read_cursors.updated_at end returning last_sequence)
 select (select count(*) from messages where conversation_id=$2 and organization_id=$3 and author_id<>$4 and sequence>saved.last_sequence)from saved`, throughID, conversationID, actor.OrganizationID, actor.ID).Scan(&unread)
	return unread, catalogError(err)
}
