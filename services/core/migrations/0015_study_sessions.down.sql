do $$ begin
 if exists(select 1 from study_sessions) or exists(select 1 from exam_specifications) then
  raise exception 'study data exists; export before downgrade';
 end if;
end $$;
drop trigger study_answer_window on exercise_attempts;
drop function enforce_study_answer();
drop table study_sessions;
drop table exam_specifications;
drop function protect_exam_specification();
