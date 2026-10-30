do $$ begin
 if exists(select 1 from learning_review_sessions) then
  raise exception 'learning review data exists; export before downgrade';
 end if;
end $$;
drop trigger learning_review_answer_scope on exercise_attempts;
drop function enforce_review_answer();
drop table learning_review_sessions;
drop function validate_learning_review();
