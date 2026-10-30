do $$ begin
 lock table html_lessons in access exclusive mode;
 if exists(select 1 from html_lessons) then raise exception 'Export HTML lessons before rollback'; end if;
 drop table html_conversions,html_preview_grants,html_assignment_links;
 alter table html_lessons drop constraint html_lessons_id_current_version_id_fkey;
 alter table html_lessons drop constraint html_lessons_id_published_version_id_fkey;
 drop table html_lesson_versions,html_lessons;
 drop function protect_html_version();
end $$;
