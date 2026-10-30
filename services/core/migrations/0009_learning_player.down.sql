do $$ begin
 if exists(select 1 from learning_attempts where mode='exam') or exists(select 1 from learning_events where kind='navigate') or exists(select 1 from lesson_versions where example is not null) then
 raise exception 'player content and progress must be migrated before rollback' using errcode='23514';
 end if;
end $$;
alter table learning_events drop constraint learning_events_kind_check;
alter table learning_events add constraint learning_events_kind_check check(kind in ('start','complete','hint'));
alter table learning_attempts drop column mode,drop column player_step,drop column theory_seen,drop column example_seen;
alter table lesson_versions drop column example;
