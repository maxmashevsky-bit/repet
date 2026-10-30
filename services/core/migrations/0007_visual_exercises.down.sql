do $$ begin
  if exists(select 1 from exercises where kind in ('number_line','grid_select','matching','ordering','geometry')) then
    raise exception 'visual exercises must be migrated before rollback' using errcode='23514';
  end if;
end $$;
alter table exercises drop constraint exercises_kind_check;
alter table exercises add constraint exercises_kind_check check (kind in ('text','numeric','single_choice','multiple_choice','long_text'));
alter table exercises drop column config;
