create table lesson_shares (
  id uuid primary key default gen_random_uuid(),
  organization_id uuid not null references organizations(id),
  lesson_id uuid not null references learning_lessons(id),
  version_id uuid not null,
  grantor_id uuid not null references users(id),
  recipient_id uuid not null references users(id),
  author_name text not null,
  allow_copy boolean not null default false,
  license text not null check(license in ('view_only','personal_copy_permission')),
  created_at timestamptz not null default clock_timestamp(),
  revoked_at timestamptz,
  foreign key(lesson_id,version_id) references lesson_versions(lesson_id,id),
  check(grantor_id<>recipient_id),
  check(allow_copy=(license='personal_copy_permission'))
);
create unique index lesson_shares_active_idx on lesson_shares(lesson_id,recipient_id) where revoked_at is null;
create index lesson_shares_recipient_idx on lesson_shares(organization_id,recipient_id,created_at desc,id) where revoked_at is null;
create index lesson_shares_owner_idx on lesson_shares(organization_id,grantor_id,lesson_id,created_at desc,id);

create function protect_lesson_share() returns trigger language plpgsql as $$
begin
  if TG_OP='UPDATE' then
    if (to_jsonb(new)-'revoked_at')<>(to_jsonb(old)-'revoked_at') or old.revoked_at is not null then
      raise exception 'share grants are immutable except initial revocation' using errcode='23514';
    end if;
    return new;
  end if;
  if not exists(select 1 from learning_lessons l join courses c on c.id=l.course_id
    join lesson_versions v on v.id=new.version_id and v.lesson_id=l.id
    join users recipient on recipient.id=new.recipient_id
    join users grantor on grantor.id=new.grantor_id
    where l.id=new.lesson_id and l.organization_id=new.organization_id and l.author_id=new.grantor_id
    and grantor.organization_id=new.organization_id and grantor.role='tutor'
    and recipient.organization_id=new.organization_id and recipient.role='tutor'
    and l.status='published' and c.status='published' and not l.assignment_only and v.sealed and v.id=l.published_version_id) then
      raise exception 'invalid sharing scope' using errcode='23514';
  end if;
  return new;
end $$;
create trigger lesson_shares_scope before insert or update on lesson_shares for each row execute function protect_lesson_share();

create table learning_lesson_provenance (
  lesson_id uuid primary key references learning_lessons(id),
  source_lesson_id uuid not null references learning_lessons(id),
  source_version_id uuid not null references lesson_versions(id),
  source_version_number integer not null check(source_version_number>0),
  original_author_id uuid not null references users(id),
  original_author_name text not null,
  source_title text not null,
  license text not null check(license='personal_copy_permission'),
  created_at timestamptz not null default clock_timestamp(),
  check(lesson_id<>source_lesson_id)
);
create function protect_lesson_provenance() returns trigger language plpgsql as $$
begin
  if TG_OP<>'INSERT' then raise exception 'copy provenance is immutable' using errcode='23514'; end if;
  if not exists(select 1 from learning_lessons source join lesson_versions v on v.lesson_id=source.id
    join learning_lessons target on target.id=new.lesson_id and target.organization_id=source.organization_id
    where source.id=new.source_lesson_id and source.author_id=new.original_author_id
    and v.id=new.source_version_id and v.sealed and v.number=new.source_version_number) then
    raise exception 'invalid copy provenance' using errcode='23514';
  end if;
  return new;
end $$;
create trigger immutable_lesson_provenance before insert or update or delete on learning_lesson_provenance for each row execute function protect_lesson_provenance();

create table lesson_copy_requests (
  recipient_id uuid not null references users(id),
  request_id uuid not null,
  share_id uuid not null references lesson_shares(id),
  course_id uuid not null references courses(id),
  topic_id uuid not null references topics(id),
  title text not null,
  lesson_id uuid not null unique references learning_lessons(id),
  created_at timestamptz not null default clock_timestamp(),
  primary key(recipient_id,request_id)
);
