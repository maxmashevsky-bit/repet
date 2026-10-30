do $$ begin
 if exists(select 1 from plan_calendar_occurrences) then raise exception 'Export planner calendar links before rollback'; end if;
end $$;
drop table plan_calendar_occurrences,plan_calendar_exports;
