create table chat_files (
 id uuid primary key,
 organization_id uuid not null references organizations(id),
 conversation_id uuid not null references conversations(id),
 owner_id uuid not null references users(id),
 request_id uuid not null,
 filename text not null check(char_length(filename) between 1 and 120),
 content_type text not null check(content_type in('image/png','image/jpeg','text/plain; charset=utf-8')),
 size_bytes bigint not null check(size_bytes between 1 and 4194304),
 object_key text not null unique check(object_key ~ '^[a-f0-9]{64}$'),
 source_hash text not null check(source_hash ~ '^[a-f0-9]{64}$'),
 policy text not null default 'cdr-v1' check(policy='cdr-v1'),
 created_at timestamptz not null default clock_timestamp(),
 expires_at timestamptz not null default clock_timestamp()+interval '24 hours',
 attached_at timestamptz,
 unique(owner_id,request_id)
);
create index chat_files_owner_quota on chat_files(owner_id,created_at);
create index chat_files_pending on chat_files(expires_at) where attached_at is null;
alter table messages add column file_id uuid unique references chat_files(id);
alter table messages add column assignment_id uuid references assignments(id);
alter table messages add column learning_lesson_id uuid references learning_lessons(id);
alter table messages drop constraint messages_body_check;
alter table messages add constraint messages_body_check check(char_length(body)<=4000 and (char_length(btrim(body))>0 or num_nonnulls(file_id,assignment_id,learning_lesson_id)=1));
alter table messages add constraint message_one_attachment check(num_nonnulls(file_id,assignment_id,learning_lesson_id)<=1);

-- Both participants must already have access; a chat link never grants access.
create function chat_reference_available(conv uuid, assignment uuid, lesson uuid) returns boolean language sql stable as $$
 select exists(select 1 from conversations c where c.id=conv
 and exists(select 1 from relations r where r.organization_id=c.organization_id and r.tutor_id=c.tutor_id and r.student_id=c.student_id and r.status='active')
 and (assignment is null or exists(select 1 from assignments a where a.id=assignment and a.organization_id=c.organization_id and a.tutor_id=c.tutor_id and a.student_id=c.student_id and a.status not in('draft','archived')))
 and (lesson is null or exists(select 1 from learning_lessons l join courses cr on cr.id=l.course_id join lesson_versions v on v.id=l.published_version_id
 where l.id=lesson and cr.organization_id=c.organization_id and cr.status='published' and (cr.visibility='organization' or cr.author_id=c.tutor_id)
 and l.status='published' and not l.assignment_only and v.sealed
 and not exists(with recursive ancestors as(select id,parent_id,status from topics where id=v.topic_id union all select t.id,t.parent_id,t.status from topics t join ancestors a on a.parent_id=t.id) select 1 from ancestors where status<>'published'))))
$$;

create function validate_chat_attachment() returns trigger language plpgsql as $$ begin
 if TG_OP='UPDATE' then
  if (new.file_id,new.assignment_id,new.learning_lesson_id,new.body,new.author_id,new.conversation_id,new.organization_id,new.idempotency_key)
   is distinct from (old.file_id,old.assignment_id,old.learning_lesson_id,old.body,old.author_id,old.conversation_id,old.organization_id,old.idempotency_key) then
   raise exception 'messages are immutable' using errcode='23514';
  end if;
  return new;
 end if;
 if not chat_reference_available(new.conversation_id,new.assignment_id,new.learning_lesson_id) then
  raise exception 'chat reference unavailable' using errcode='23514';
 end if;
 if new.file_id is not null then
  update chat_files f set attached_at=clock_timestamp()
  where f.id=new.file_id and f.organization_id=new.organization_id and f.conversation_id=new.conversation_id and f.owner_id=new.author_id
   and f.attached_at is null and f.expires_at>clock_timestamp();
  if not found then raise exception 'chat file unavailable' using errcode='23514'; end if;
 end if;
 if btrim(new.body)='' and num_nonnulls(new.file_id,new.assignment_id,new.learning_lesson_id)=0 then
  raise exception 'empty message' using errcode='23514';
 end if;
 return new;
end $$;
create trigger message_attachment_scope before insert or update on messages for each row execute function validate_chat_attachment();
