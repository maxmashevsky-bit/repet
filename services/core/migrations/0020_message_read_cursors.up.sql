alter table conversations add column last_message_sequence bigint not null default 0 check(last_message_sequence>=0);
alter table messages add column sequence bigint;
with ordered as(select id,row_number()over(partition by conversation_id order by created_at,id) n from messages)
 update messages m set sequence=o.n from ordered o where m.id=o.id;
update conversations c set last_message_sequence=coalesce((select max(sequence) from messages where conversation_id=c.id),0);
alter table messages alter column sequence set not null;
create unique index messages_sequence on messages(conversation_id,sequence);

create table message_read_cursors (
 conversation_id uuid not null references conversations(id) on delete cascade,
 user_id uuid not null references users(id),
 last_sequence bigint not null check(last_sequence>0),
 updated_at timestamptz not null default clock_timestamp(),
 primary key(conversation_id,user_id)
);

-- Serialize message order with the conversation row; transaction start times cannot lose late commits.
create function assign_message_sequence() returns trigger language plpgsql as $$ begin
 update conversations c set last_message_sequence=c.last_message_sequence+1
 where c.id=new.conversation_id and c.organization_id=new.organization_id
 and new.author_id in(c.tutor_id,c.student_id)
 and exists(select 1 from users u where u.id=new.author_id and u.organization_id=c.organization_id)
 and exists(select 1 from relations r where r.organization_id=c.organization_id and r.tutor_id=c.tutor_id and r.student_id=c.student_id and r.status='active')
 returning c.last_message_sequence into new.sequence;
 if not found then raise exception 'message participant required' using errcode='23514'; end if;
 new.created_at=clock_timestamp();
 return new;
end $$;
create trigger message_order before insert on messages for each row execute function assign_message_sequence();

create function validate_message_cursor() returns trigger language plpgsql as $$ begin
 if not exists(select 1 from conversations c join users u on u.id=new.user_id and u.organization_id=c.organization_id
 where c.id=new.conversation_id and new.user_id in(c.tutor_id,c.student_id) and new.last_sequence<=c.last_message_sequence
 and exists(select 1 from relations r where r.organization_id=c.organization_id and r.tutor_id=c.tutor_id and r.student_id=c.student_id and r.status='active'))
 or TG_OP='UPDATE' and (new.conversation_id,new.user_id) is distinct from(old.conversation_id,old.user_id)
 or TG_OP='UPDATE' and new.last_sequence<old.last_sequence then
  raise exception 'invalid message cursor' using errcode='23514';
 end if;
 return new;
end $$;
create trigger message_cursor_scope before insert or update on message_read_cursors for each row execute function validate_message_cursor();
