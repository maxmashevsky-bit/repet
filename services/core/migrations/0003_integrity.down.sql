drop index if exists messages_author_idx;
drop index if exists lessons_student_page_idx;
drop index if exists lessons_tutor_page_idx;
drop index if exists sessions_user_idx;
alter table invitations drop constraint if exists invitations_email_check;
alter table lessons drop constraint if exists lessons_title_check;
alter table messages drop constraint if exists messages_body_check;
alter table assignment_submissions drop constraint if exists submissions_score_check;
alter table assignments drop constraint if exists assignments_max_score_check;
