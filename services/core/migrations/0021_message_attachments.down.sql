do $$ begin
 if exists(select 1 from chat_files) or exists(select 1 from messages where assignment_id is not null or learning_lesson_id is not null) then
  raise exception 'Export chat attachments and private objects before rollback';
 end if;
end $$;
drop trigger message_attachment_scope on messages;
drop function validate_chat_attachment();
alter table messages drop constraint messages_body_check;
alter table messages add constraint messages_body_check check(char_length(btrim(body)) between 1 and 4000);
alter table messages drop constraint message_one_attachment;
alter table messages drop column file_id,drop column assignment_id,drop column learning_lesson_id;
drop function chat_reference_available(uuid,uuid,uuid);
drop table chat_files;
