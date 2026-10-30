do $$ begin
 if exists(select 1 from guardian_links) or exists(select 1 from guardian_notes) or exists(select 1 from guardian_events) then
  raise exception 'guardian data exists; export before downgrade';
 end if;
end $$;
drop trigger guardian_lesson_event on lessons;
drop trigger guardian_assignment_event on assignments;
drop trigger guardian_submission_event on assignment_submissions;
drop trigger guardian_completion_event on learning_attempts;
drop trigger guardian_review_event on exercise_attempts;
drop trigger guardian_note_event on guardian_notes;
drop function guardian_product_event();
drop function emit_guardian_event(uuid,uuid,text,uuid);
drop table guardian_events;
drop table guardian_notes;
drop table guardian_links;
drop function validate_guardian_link();
