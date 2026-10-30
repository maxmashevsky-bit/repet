create index analytics_answers_period_idx on exercise_attempts(student_id,created_at,learning_attempt_id,exercise_id);
create index analytics_hints_period_idx on learning_hints(learning_attempt_id,exercise_id,revealed_at);
create index analytics_assignments_period_idx on assignments(organization_id,student_id,tutor_id,created_at) where status<>'archived';
