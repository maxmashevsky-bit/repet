alter table assignments add column kind text not null default 'text' check(kind in ('text','material'));
create table exercise_sets (
 id uuid primary key default gen_random_uuid(),
 organization_id uuid not null references organizations(id),
 tutor_id uuid not null references users(id),
 title text not null check(length(title) between 1 and 180),
 lesson_id uuid not null references learning_lessons(id),
 version_id uuid not null references lesson_versions(id),
 exercise_keys uuid[] not null check(cardinality(exercise_keys) between 1 and 200),
 created_at timestamptz not null default now()
);
create index exercise_sets_owner_idx on exercise_sets(tutor_id,created_at desc,id);
create table material_assignments (
 assignment_id uuid primary key references assignments(id),
 lesson_id uuid not null references learning_lessons(id),
 version_id uuid not null references lesson_versions(id),
 exercise_keys uuid[] not null check(cardinality(exercise_keys) between 1 and 200),
 calendar_lesson_id uuid references lessons(id),
 repeat_of uuid references material_assignments(assignment_id),
 mode text not null check(mode in ('practice','exam')),
 request_id uuid not null,
 request_owner uuid not null references users(id),
 payload jsonb not null,
 unique(request_owner,request_id)
);
alter table learning_attempts add column assignment_id uuid references material_assignments(assignment_id);
drop index learning_attempts_active_idx;
create unique index learning_attempts_active_idx on learning_attempts(student_id,lesson_id) where completed_at is null and assignment_id is null;
create unique index learning_attempts_assignment_idx on learning_attempts(assignment_id) where assignment_id is not null;
