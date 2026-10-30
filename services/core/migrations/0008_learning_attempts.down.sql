do $$ begin
 if exists(select 1 from learning_attempts) then
  raise exception 'learning progress must be migrated before rollback' using errcode='23514';
 end if;
end $$;
alter table exercise_attempts drop column learning_attempt_id, drop column attempt_number, drop column hints_used, drop column elapsed_seconds;
alter table exercise_answer_keys drop column hints;
drop table learning_hints;
drop table learning_events;
drop table learning_attempts;
