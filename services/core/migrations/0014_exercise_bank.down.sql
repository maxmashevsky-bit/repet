do $$ begin
 if exists(select 1 from bank_collections) or exists(select 1 from bank_exercises) then
  raise exception 'bank contains saved materials; export before downgrade';
 end if;
end $$;
drop table bank_assignment_batches;
drop table bank_collections;
drop table bank_exercises;
drop function protect_bank_collection();
alter table learning_lessons drop column assignment_only;
