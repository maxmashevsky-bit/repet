do $$ begin
  if exists(select 1 from lesson_shares) or exists(select 1 from learning_lesson_provenance) then
    raise exception 'sharing/copy records exist; export before rollback';
  end if;
end $$;
drop table lesson_copy_requests;
drop table learning_lesson_provenance;
drop function protect_lesson_provenance();
drop table lesson_shares;
drop function protect_lesson_share();
