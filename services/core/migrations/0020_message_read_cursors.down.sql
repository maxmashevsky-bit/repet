do $$ begin
 if exists(select 1 from message_read_cursors) or exists(select 1 from messages) then
  raise exception 'message order/read data exists; export before downgrade';
 end if;
end $$;
drop table message_read_cursors;
drop function validate_message_cursor();
drop trigger message_order on messages;
drop function assign_message_sequence();
drop index messages_sequence;
alter table messages drop column sequence;
alter table conversations drop column last_message_sequence;
