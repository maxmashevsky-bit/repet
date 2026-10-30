-- Refuse rollback rather than discard authored exercises of the new types.
do $$ begin
  if exists(select 1 from exercises where kind in ('multiple_choice','long_text')) then
    raise exception 'remove new exercise types before rollback' using errcode='23514';
  end if;
end $$;
drop table exercise_attempts;
alter table exercises drop constraint exercises_kind_check;
alter table exercises add constraint exercises_kind_check check (kind in ('text','numeric','single_choice'));
