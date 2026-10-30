create table learning_reward_preferences (
 tutor_id uuid not null references users(id),
 student_id uuid not null references users(id),
 organization_id uuid not null references organizations(id),
 enabled boolean not null,
 enabled_from timestamptz,
 updated_at timestamptz not null default clock_timestamp(),
 primary key(tutor_id,student_id)
);
create table learning_reward_awards (
 id uuid primary key default gen_random_uuid(),
 organization_id uuid not null references organizations(id),
 student_id uuid not null references users(id),
 tutor_id uuid not null references users(id),
 source_lesson_id uuid not null references learning_lessons(id),
 source_exercise_key uuid not null,
 evidence_attempt_id uuid not null references exercise_attempts(id),
 points integer not null default 10 check(points=10),
 earned_at timestamptz not null,
 awarded_at timestamptz not null default clock_timestamp(),
 unique(student_id,source_lesson_id,source_exercise_key)
);
create index learning_reward_awards_student_idx on learning_reward_awards(student_id,tutor_id,earned_at desc,id);
create trigger immutable_learning_reward_award before update or delete on learning_reward_awards for each row execute function protect_bank_collection();
