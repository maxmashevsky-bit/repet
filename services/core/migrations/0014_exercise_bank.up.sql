alter table learning_lessons add column assignment_only boolean not null default false;

create table bank_exercises (
 id uuid primary key default gen_random_uuid(),
 organization_id uuid not null references organizations(id),
 author_id uuid not null references users(id),
 exercise_id uuid not null references exercises(id),
 grade integer not null check (grade between 5 and 11),
 direction text not null check (direction in ('school','olympiad','oge','ege_basic','ege_profile')),
 subject text not null default 'math' check (subject='math'),
 topic text not null check (length(trim(topic)) between 1 and 200),
 subtopic text not null default '' check (length(subtopic)<=200),
 exam_number integer check (exam_number between 1 and 100),
 source text not null check (length(trim(source)) between 1 and 1000),
 attribution text not null check (length(trim(attribution)) between 1 and 1000),
 rights_status text not null check (rights_status in ('own','permission','open_license')),
 license text not null check (length(trim(license)) between 1 and 1000),
 search_text text not null,
 search_vector tsvector generated always as (to_tsvector('russian',topic||' '||subtopic||' '||source||' '||search_text)) stored,
 created_at timestamptz not null default now(),
 unique(author_id,exercise_id),
 check ((direction in ('school','olympiad') and exam_number is null) or (direction in ('oge','ege_basic','ege_profile') and exam_number is not null))
);
create index bank_filter_idx on bank_exercises(organization_id,direction,grade,exam_number,created_at desc,id);
create index bank_topic_idx on bank_exercises(organization_id,lower(topic),lower(subtopic));
create index bank_search_idx on bank_exercises using gin(search_vector);

create table bank_collections (
 id uuid primary key default gen_random_uuid(),
 organization_id uuid not null references organizations(id),
 tutor_id uuid not null references users(id),
 lesson_id uuid not null references learning_lessons(id),
 version_id uuid not null references lesson_versions(id),
 exercise_keys uuid[] not null check(cardinality(exercise_keys) between 1 and 100),
 provenance jsonb not null check(jsonb_typeof(provenance)='array'),
 request_id uuid not null,
 payload jsonb not null,
 created_at timestamptz not null default now(),
 unique(tutor_id,request_id)
);
create index bank_collections_owner_idx on bank_collections(organization_id,tutor_id,created_at desc,id);

create table bank_assignment_batches (
 tutor_id uuid not null references users(id),
 request_id uuid not null,
 collection_id uuid not null references bank_collections(id),
 payload jsonb not null,
 created_at timestamptz not null default now(),
 primary key(tutor_id,request_id)
);
create function protect_bank_collection() returns trigger language plpgsql as $$
begin
 raise exception 'bank snapshots and provenance are immutable' using errcode='23514';
end $$;
create trigger immutable_bank_collection before update or delete on bank_collections for each row execute function protect_bank_collection();
create trigger immutable_bank_entry before update or delete on bank_exercises for each row execute function protect_bank_collection();
create trigger immutable_bank_batch before update or delete on bank_assignment_batches for each row execute function protect_bank_collection();
