create table learning_review_sessions (
 learning_attempt_id uuid primary key,
 source_run_id uuid not null,
 student_id uuid not null references users(id),
 tutor_id uuid not null references users(id),
 exercise_keys uuid[] not null check(cardinality(exercise_keys) between 1 and 10),
 request_id uuid not null,
 payload jsonb not null,
 created_at timestamptz not null default clock_timestamp(),
 unique(student_id,request_id),
 foreign key(learning_attempt_id,student_id) references learning_attempts(id,student_id),
 foreign key(source_run_id,student_id) references learning_attempts(id,student_id),
 check(learning_attempt_id<>source_run_id)
);
create index learning_reviews_source_idx on learning_review_sessions(source_run_id,learning_attempt_id);
create index learning_reviews_history_idx on learning_review_sessions(student_id,created_at desc,learning_attempt_id);
create function validate_learning_review() returns trigger language plpgsql as $$
begin
 if exists(select 1 from learning_review_sessions where learning_attempt_id=NEW.source_run_id)
 or not exists(select 1 from learning_attempts source join learning_attempts target on target.id=NEW.learning_attempt_id
   where source.id=NEW.source_run_id and source.completed_at is not null and source.version_id=target.version_id
   and target.mode='practice' and target.assignment_id is null and source.student_id=target.student_id
   and not exists(select 1 from study_sessions where learning_attempt_id=target.id)
   and cardinality(NEW.exercise_keys)=(select count(distinct key) from unnest(NEW.exercise_keys) key)
   and not exists(select 1 from unnest(NEW.exercise_keys) key where not exists(select 1 from exercises e where e.version_id=source.version_id and e.exercise_key=key
    and (source.assignment_id is null or key in(select unnest(exercise_keys) from material_assignments where assignment_id=source.assignment_id))
    and (not exists(select 1 from study_sessions where learning_attempt_id=source.id) or key in(select unnest(exercise_keys) from study_sessions where learning_attempt_id=source.id))))) then
  raise exception 'invalid review source or selected exercises' using errcode='23514';
 end if;
 return NEW;
end $$;
create trigger validate_learning_review before insert on learning_review_sessions for each row execute function validate_learning_review();
create trigger immutable_learning_review before update or delete on learning_review_sessions for each row execute function protect_bank_collection();
create function enforce_review_answer() returns trigger language plpgsql as $$
begin
 if exists(select 1 from learning_review_sessions rs join learning_attempts la on la.id=rs.learning_attempt_id
  join exercises e on e.id=NEW.exercise_id where rs.learning_attempt_id=NEW.learning_attempt_id
   and (la.completed_at is not null or la.mode<>'practice' or e.version_id<>la.version_id or not(e.exercise_key=any(rs.exercise_keys)))) then
  raise exception 'review answer outside allowed exercises' using errcode='23514';
 end if;
 return NEW;
end $$;
create trigger learning_review_answer_scope before insert on exercise_attempts for each row execute function enforce_review_answer();
