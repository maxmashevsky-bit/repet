do $$ begin
 if exists(select 1 from teaching_preferences) then raise exception 'Export teaching preferences before rollback'; end if;
end $$;
drop table teaching_preferences;
