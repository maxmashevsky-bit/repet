do $$ begin
 if exists(select 1 from tutor_plans) then
  raise exception 'tutor_plans contains saved plans; export them before rollback';
 end if;
end $$;
drop table tutor_plans;
