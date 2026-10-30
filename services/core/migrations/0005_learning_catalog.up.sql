create table courses (
  id uuid primary key default gen_random_uuid(),
  organization_id uuid not null references organizations(id),
  author_id uuid not null references users(id),
  title text not null check (length(title) between 1 and 200),
  description text not null default '',
  track text not null check (track in ('school','olympiad','oge','ege')),
  grade_min integer not null check (grade_min between 1 and 11),
  grade_max integer not null check (grade_max between grade_min and 11),
  difficulty integer not null check (difficulty between 1 and 5),
  visibility text not null default 'linked_students' check (visibility in ('linked_students','organization')),
  status text not null default 'draft' check (status in ('draft','published','archived')),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (organization_id,id),
  unique (organization_id,id,author_id)
);

create table topics (
  id uuid primary key default gen_random_uuid(),
  course_id uuid not null references courses(id),
  parent_id uuid,
  title text not null check (length(title) between 1 and 200),
  description text not null default '',
  position integer not null default 0 check (position >= 0),
  status text not null default 'draft' check (status in ('draft','published','archived')),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (course_id,id),
  foreign key (course_id,parent_id) references topics(course_id,id),
  check (parent_id is null or parent_id <> id)
);

create table learning_lessons (
  id uuid primary key default gen_random_uuid(),
  organization_id uuid not null,
  course_id uuid not null,
  author_id uuid not null references users(id),
  status text not null default 'draft' check (status in ('draft','published','archived')),
  draft_revision integer not null default 1,
  published_revision integer,
  published_version_id uuid,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  foreign key (organization_id,course_id,author_id) references courses(organization_id,id,author_id),
  unique (course_id,id)
);

create table lesson_versions (
  id uuid primary key default gen_random_uuid(),
  course_id uuid not null,
  lesson_id uuid not null,
  topic_id uuid not null,
  number integer not null check (number >= 0),
  title text not null check (length(title) between 1 and 200),
  summary text not null default '',
  content text not null,
  track text not null check (track in ('school','olympiad','oge','ege')),
  grade_min integer not null check (grade_min between 1 and 11),
  grade_max integer not null check (grade_max between grade_min and 11),
  difficulty integer not null check (difficulty between 1 and 5),
  created_at timestamptz not null default now(),
  published_at timestamptz,
  sealed boolean not null default false,
  search_vector tsvector generated always as (to_tsvector('russian',title || ' ' || summary || ' ' || content)) stored,
  foreign key (course_id,lesson_id) references learning_lessons(course_id,id),
  foreign key (course_id,topic_id) references topics(course_id,id),
  unique (lesson_id,number),
  unique (lesson_id,id),
  check ((number = 0 and published_at is null and not sealed) or (number > 0 and published_at is not null))
);
alter table learning_lessons add foreign key (id,published_version_id) references lesson_versions(lesson_id,id);

create table exercises (
  id uuid primary key default gen_random_uuid(),
  version_id uuid not null references lesson_versions(id),
  exercise_key uuid not null default gen_random_uuid(),
  kind text not null check (kind in ('text','numeric','single_choice')),
  prompt text not null check (length(prompt) > 0),
  options jsonb not null default '[]'::jsonb check (jsonb_typeof(options) = 'array'),
  difficulty integer not null check (difficulty between 1 and 5),
  position integer not null default 0 check (position >= 0),
  unique (version_id,exercise_key)
);

create table exercise_answer_keys (
  exercise_id uuid primary key references exercises(id) on delete cascade,
  answer text not null check (length(answer) > 0),
  explanation text not null default ''
);

-- Published snapshots, including private keys, are append-only at the database boundary.
create function protect_learning_snapshot() returns trigger language plpgsql as $$
declare version_number integer;
begin
  if TG_TABLE_NAME = 'lesson_versions' then
    if TG_OP = 'UPDATE' and OLD.number > 0 and not OLD.sealed and NEW.sealed and
      (to_jsonb(OLD)-'sealed'-'search_vector') = (to_jsonb(NEW)-'sealed'-'search_vector') then return NEW; end if;
    if OLD.number > 0 or (TG_OP = 'UPDATE' and (NEW.number <> OLD.number or NEW.id <> OLD.id or NEW.lesson_id <> OLD.lesson_id or NEW.course_id <> OLD.course_id)) then
      raise exception 'published version is immutable' using errcode='23514';
    end if;
  elsif TG_TABLE_NAME = 'exercises' then
    if TG_OP <> 'INSERT' then
      select number into version_number from lesson_versions where id=OLD.version_id;
      if version_number > 0 then raise exception 'published exercise is immutable' using errcode='23514'; end if;
    end if;
    if TG_OP <> 'DELETE' then
      select number into version_number from lesson_versions where id=NEW.version_id;
      if exists(select 1 from lesson_versions where id=NEW.version_id and sealed) then
        raise exception 'published exercise is immutable' using errcode='23514';
      end if;
    end if;
  else
    if TG_OP <> 'INSERT' then
      select v.number into version_number from exercises e join lesson_versions v on v.id=e.version_id where e.id=OLD.exercise_id;
      if version_number > 0 then raise exception 'published answer key is immutable' using errcode='23514'; end if;
    end if;
    if TG_OP <> 'DELETE' then
      if exists(select 1 from exercises e join lesson_versions v on v.id=e.version_id where e.id=NEW.exercise_id and v.sealed) then
        raise exception 'published answer key is immutable' using errcode='23514';
      end if;
    end if;
  end if;
  if TG_OP = 'DELETE' then return OLD; end if;
  return NEW;
end $$;
create trigger immutable_lesson_version before update or delete on lesson_versions for each row execute function protect_learning_snapshot();
create trigger immutable_exercise before insert or update or delete on exercises for each row execute function protect_learning_snapshot();
create trigger immutable_answer_key before insert or update or delete on exercise_answer_keys for each row execute function protect_learning_snapshot();

create index courses_catalog_idx on courses(organization_id,status,track,created_at desc,id);
create index topics_tree_idx on topics(course_id,parent_id,position,id);
create index learning_lessons_catalog_idx on learning_lessons(organization_id,course_id,status,created_at desc,id);
create index lesson_versions_search_idx on lesson_versions using gin(search_vector);
create index lesson_versions_topic_idx on lesson_versions(topic_id);
