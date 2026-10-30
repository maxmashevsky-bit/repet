create table exam_specifications (
 id uuid primary key default gen_random_uuid(),
 family_id uuid not null,
 version integer not null check(version>0),
 organization_id uuid not null references organizations(id),
 author_id uuid not null references users(id),
 collection_id uuid not null references bank_collections(id),
 title text not null check(length(trim(title)) between 1 and 200),
 direction text not null check(direction in ('oge','ege_basic','ege_profile')),
 exam_year integer not null check(exam_year between 1 and 9999),
 duration_seconds integer not null check(duration_seconds between 60 and 28800),
 instructions text not null default '' check(length(instructions)<=10000),
 slots jsonb not null check(jsonb_typeof(slots)='array' and jsonb_array_length(slots) between 1 and 100),
 status text not null default 'draft' check(status in ('draft','published','archived')),
 request_id uuid not null,
 payload jsonb not null,
 created_at timestamptz not null default now(),
 unique(family_id,version),unique(author_id,request_id)
);
create index exam_specifications_catalog_idx on exam_specifications(organization_id,status,direction,created_at desc,id);
create function protect_exam_specification() returns trigger language plpgsql as $$
begin
 if TG_OP='DELETE' or (to_jsonb(NEW)-'status')<>(to_jsonb(OLD)-'status') then
  raise exception 'create a new specification version instead' using errcode='23514';
 end if;
 return NEW;
end $$;
create trigger immutable_exam_specification before update or delete on exam_specifications for each row execute function protect_exam_specification();

create table study_sessions (
 id uuid primary key default gen_random_uuid(),
 specification_id uuid not null references exam_specifications(id),
 learning_attempt_id uuid not null unique references learning_attempts(id),
 student_id uuid not null references users(id),
 exercise_keys uuid[] not null check(cardinality(exercise_keys) between 1 and 100),
 exercise_points jsonb not null check(jsonb_typeof(exercise_points)='object'),
 seed text not null check(length(seed) between 1 and 64),
 deadline_at timestamptz,
 request_id uuid not null,
 payload jsonb not null,
 created_at timestamptz not null default now(),
 unique(student_id,request_id)
);
create index study_sessions_history_idx on study_sessions(student_id,created_at desc,id);
create index study_sessions_due_idx on study_sessions(deadline_at,learning_attempt_id) where deadline_at is not null;
create trigger immutable_study_session before update or delete on study_sessions for each row execute function protect_bank_collection();

create function enforce_study_answer() returns trigger language plpgsql as $$
begin
 if exists(select 1 from study_sessions ss join learning_attempts la on la.id=ss.learning_attempt_id
  join exercises e on e.id=NEW.exercise_id where la.id=NEW.learning_attempt_id and
  (la.completed_at is not null or ss.student_id<>NEW.student_id or e.version_id<>la.version_id
   or not(e.exercise_key=any(ss.exercise_keys)) or (ss.deadline_at is not null and clock_timestamp()>=ss.deadline_at))) then
  raise exception 'study answer window closed or exercise outside variant' using errcode='23514',constraint='study_answer_window';
 end if;
 return NEW;
end $$;
create trigger study_answer_window before insert on exercise_attempts for each row execute function enforce_study_answer();
