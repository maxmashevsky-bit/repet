insert into conversations (id, organization_id, relation_id, tutor_id, student_id)
select gen_random_uuid(), r.organization_id, r.id, r.tutor_id, r.student_id
from relations r
where r.status = 'active'
on conflict (organization_id, tutor_id, student_id) do nothing;

alter table assignments add constraint assignments_max_score_check check (max_score between 1 and 1000);
alter table assignment_submissions add constraint submissions_score_check check (score is null or score >= 0);
alter table messages add constraint messages_body_check check (length(trim(body)) between 1 and 4000);
alter table lessons add constraint lessons_title_check check (length(trim(title)) between 1 and 180);
alter table invitations add constraint invitations_email_check check (length(student_email) between 3 and 254);

create index sessions_user_idx on sessions (user_id, expires_at desc);
create index lessons_tutor_page_idx on lessons (organization_id, tutor_id, starts_at desc);
create index lessons_student_page_idx on lessons (organization_id, student_id, starts_at desc);
create index messages_author_idx on messages (author_id, created_at desc);
