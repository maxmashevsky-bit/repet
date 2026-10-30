alter table exercises drop constraint exercises_kind_check;
alter table exercises add constraint exercises_kind_check check (kind in ('text','numeric','single_choice','multiple_choice','long_text'));

create table exercise_attempts (
  id uuid primary key default gen_random_uuid(),
  exercise_id uuid not null references exercises(id),
  student_id uuid not null references users(id),
  request_id uuid not null,
  answer jsonb not null check (jsonb_typeof(answer) = 'object'),
  status text not null check (status in ('correct','incorrect','needs_review')),
  feedback text not null default '',
  reviewer_id uuid references users(id),
  reviewed_at timestamptz,
  created_at timestamptz not null default now(),
  unique (student_id,request_id),
  check ((reviewer_id is null and reviewed_at is null) or (reviewer_id is not null and reviewed_at is not null and status <> 'needs_review'))
);
create index exercise_attempts_student_idx on exercise_attempts(student_id,created_at desc,id);
create index exercise_attempts_exercise_idx on exercise_attempts(exercise_id,created_at desc,id);
