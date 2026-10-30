create table learning_attempts (
 id uuid primary key default gen_random_uuid(),
 lesson_id uuid not null references learning_lessons(id),
 version_id uuid not null references lesson_versions(id),
 student_id uuid not null references users(id),
 number integer not null check(number>0),
 started_at timestamptz not null default clock_timestamp(),
 completed_at timestamptz,
 unique(student_id,lesson_id,number),
 unique(id,student_id)
);
create unique index learning_attempts_active_idx on learning_attempts(student_id,lesson_id) where completed_at is null;
create index learning_attempts_results_idx on learning_attempts(student_id,started_at desc,id);
create table learning_events (
 student_id uuid not null,
 request_id uuid not null,
 learning_attempt_id uuid not null,
 kind text not null check(kind in ('start','complete','hint')),
 payload jsonb not null,
 created_at timestamptz not null default clock_timestamp(),
 primary key(student_id,request_id),
 foreign key(learning_attempt_id,student_id) references learning_attempts(id,student_id)
);
create table learning_hints (
 learning_attempt_id uuid not null references learning_attempts(id),
 exercise_id uuid not null references exercises(id),
 number integer not null check(number between 1 and 5),
 revealed_at timestamptz not null default clock_timestamp(),
 primary key(learning_attempt_id,exercise_id,number)
);
alter table exercise_answer_keys add column hints jsonb not null default '[]' check(jsonb_typeof(hints)='array' and jsonb_array_length(hints)<=5);
alter table exercise_attempts add column learning_attempt_id uuid;
alter table exercise_attempts add constraint exercise_attempts_learning_student_fk foreign key(learning_attempt_id,student_id) references learning_attempts(id,student_id);
alter table exercise_attempts add column attempt_number integer;
with numbered as(select id,row_number() over(partition by student_id,exercise_id order by created_at,id) n from exercise_attempts)
update exercise_attempts a set attempt_number=n.n from numbered n where a.id=n.id;
alter table exercise_attempts alter column attempt_number set not null;
alter table exercise_attempts add constraint exercise_attempts_number_check check(attempt_number>0);
create unique index exercise_attempts_number_idx on exercise_attempts(student_id,exercise_id,attempt_number);
alter table exercise_attempts add column hints_used integer not null default 0 check(hints_used between 0 and 5);
alter table exercise_attempts add column elapsed_seconds bigint not null default 0 check(elapsed_seconds>=0);
create index exercise_attempts_learning_idx on exercise_attempts(learning_attempt_id,created_at desc,id);
with groups as (
 select a.student_id,v.lesson_id,v.id version_id,min(a.created_at) started,max(a.created_at) ended
 from exercise_attempts a join exercises e on e.id=a.exercise_id join lesson_versions v on v.id=e.version_id
 group by a.student_id,v.lesson_id,v.id
), ranked as (
 select *,row_number() over(partition by student_id,lesson_id order by started,version_id) n,
 row_number() over(partition by student_id,lesson_id order by started desc,version_id desc) newest from groups
)
insert into learning_attempts(student_id,lesson_id,version_id,number,started_at,completed_at)
select student_id,lesson_id,version_id,n,started,case when newest=1 then null else ended end from ranked;
update exercise_attempts a set learning_attempt_id=la.id from exercises e,learning_attempts la
where e.id=a.exercise_id and la.version_id=e.version_id and la.student_id=a.student_id;
