do $$ begin
 if exists(select 1 from material_assignments) or exists(select 1 from exercise_sets) then
 raise exception 'material assignments and exercise sets must be migrated before rollback' using errcode='23514';
 end if;
end $$;
drop index learning_attempts_assignment_idx;
drop index learning_attempts_active_idx;
alter table learning_attempts drop column assignment_id;
create unique index learning_attempts_active_idx on learning_attempts(student_id,lesson_id) where completed_at is null;
drop table material_assignments;
drop table exercise_sets;
alter table assignments drop column kind;
