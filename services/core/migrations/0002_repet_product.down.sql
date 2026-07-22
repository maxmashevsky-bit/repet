drop table if exists notifications;
drop table if exists assignment_submissions;
drop table if exists assignments;
drop table if exists messages;
drop table if exists conversations;
drop table if exists sessions;
alter table users drop column if exists locale;
alter table users drop column if exists timezone;
alter table users drop column if exists email_verified_at;
alter table users drop column if exists password_hash;

