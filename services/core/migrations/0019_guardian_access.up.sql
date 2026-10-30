create table guardian_links (
 id uuid primary key default gen_random_uuid(),
 organization_id uuid not null references organizations(id),
 student_id uuid not null references users(id),
 guardian_email text not null check (guardian_email=lower(trim(guardian_email))),
 guardian_id uuid references users(id),
 token_hash text not null unique,
 status text not null default 'pending' check (status in ('pending','active','revoked')),
 created_at timestamptz not null default clock_timestamp(),
 expires_at timestamptz not null default clock_timestamp()+interval '7 days',
 accepted_at timestamptz,
 revoked_at timestamptz,
 seen_at timestamptz,
 check (student_id is distinct from guardian_id),
 check (expires_at>created_at),
 check (status<>'active' or guardian_id is not null and accepted_at is not null),
 check (status<>'revoked' or revoked_at is not null)
);
create unique index guardian_live_email on guardian_links(organization_id,student_id,guardian_email) where status in ('pending','active');
create index guardian_children on guardian_links(organization_id,guardian_id,student_id) where status='active';
create index guardian_student on guardian_links(organization_id,student_id,created_at desc);

create function validate_guardian_link() returns trigger language plpgsql as $$ begin
 if not exists(select 1 from users where id=new.student_id and organization_id=new.organization_id and role='student')
 or new.guardian_id is not null and not exists(select 1 from users where id=new.guardian_id and organization_id=new.organization_id and lower(email)=new.guardian_email) then
  raise exception 'invalid guardian participants' using errcode='23514';
 end if;
 if TG_OP='UPDATE' and (new.id,new.organization_id,new.student_id,new.guardian_email,new.token_hash,new.created_at,new.expires_at)
 is distinct from (old.id,old.organization_id,old.student_id,old.guardian_email,old.token_hash,old.created_at,old.expires_at) then
  raise exception 'immutable guardian invitation' using errcode='23514';
 end if;
 if TG_OP='UPDATE' and (old.status='revoked' and new is distinct from old or old.guardian_id is not null and new.guardian_id is distinct from old.guardian_id
 or old.status='active' and new.status='pending') then
  raise exception 'invalid guardian transition' using errcode='23514';
 end if;
 return new;
end $$;
create trigger guardian_link_scope before insert or update on guardian_links for each row execute function validate_guardian_link();

create table guardian_notes (
 id uuid primary key default gen_random_uuid(),
 organization_id uuid not null references organizations(id),
 tutor_id uuid not null references users(id),
 student_id uuid not null references users(id),
 body text not null check (length(body) between 1 and 4000),
 request_id uuid not null,
 created_at timestamptz not null default clock_timestamp(),
 withdrawn_at timestamptz,
 unique(tutor_id,request_id)
);
create index guardian_notes_student on guardian_notes(organization_id,student_id,created_at desc,id);

create table guardian_events (
 id uuid primary key default gen_random_uuid(),
 organization_id uuid not null references organizations(id),
 student_id uuid not null references users(id),
 kind text not null check (kind in ('lesson','assignment','submitted','graded','completed','reviewed','note')),
 source_id uuid not null,
 created_at timestamptz not null default clock_timestamp()
);
create index guardian_events_student on guardian_events(organization_id,student_id,created_at desc,id);

-- Academic events are deliberately separate from the child's private notifications/messages.
create function emit_guardian_event(org uuid,student uuid,event_kind text,source uuid) returns void language sql as $$
 insert into guardian_events(organization_id,student_id,kind,source_id)
 select org,student,event_kind,source where exists(select 1 from guardian_links where organization_id=org and student_id=student and status='active');
$$;
create function guardian_product_event() returns trigger language plpgsql as $$ begin
 if TG_TABLE_NAME='lessons' then
  if TG_OP='INSERT' or (new.title,new.starts_at,new.ends_at,new.status) is distinct from (old.title,old.starts_at,old.ends_at,old.status) then
   perform emit_guardian_event(new.organization_id,new.student_id,'lesson',new.id);
  end if;
 elsif TG_TABLE_NAME='assignments' then
  if new.status not in ('draft','archived') and (TG_OP='INSERT' or (new.title,new.due_at) is distinct from(old.title,old.due_at)
   or new.status in ('assigned','revision','overdue') and new.status is distinct from old.status) then
   perform emit_guardian_event(new.organization_id,new.student_id,'assignment',new.id);
  end if;
 elsif TG_TABLE_NAME='assignment_submissions' then
  if TG_OP='INSERT' then
   perform emit_guardian_event(new.organization_id,new.student_id,'submitted',new.assignment_id);
  elsif (new.score,new.graded_at) is distinct from (old.score,old.graded_at) and new.graded_at is not null then
   perform emit_guardian_event(new.organization_id,new.student_id,'graded',new.assignment_id);
  end if;
 elsif TG_TABLE_NAME='learning_attempts' then
  if new.completed_at is not null and old.completed_at is null then
   perform emit_guardian_event((select organization_id from users where id=new.student_id),new.student_id,'completed',new.id);
  end if;
 elsif TG_TABLE_NAME='exercise_attempts' then
  if new.reviewed_at is not null and (new.status,new.reviewed_at) is distinct from(old.status,old.reviewed_at) then
   perform emit_guardian_event((select organization_id from users where id=new.student_id),new.student_id,'reviewed',new.id);
  end if;
 elsif TG_TABLE_NAME='guardian_notes' then
  perform emit_guardian_event(new.organization_id,new.student_id,'note',new.id);
 end if;
 return new;
end $$;
create trigger guardian_lesson_event after insert or update on lessons for each row execute function guardian_product_event();
create trigger guardian_assignment_event after insert or update on assignments for each row execute function guardian_product_event();
create trigger guardian_submission_event after insert or update on assignment_submissions for each row execute function guardian_product_event();
create trigger guardian_completion_event after update on learning_attempts for each row execute function guardian_product_event();
create trigger guardian_review_event after update on exercise_attempts for each row execute function guardian_product_event();
create trigger guardian_note_event after insert on guardian_notes for each row execute function guardian_product_event();
