create table plan_calendar_exports (
 tutor_id uuid not null references users(id),
 request_id uuid not null,
 organization_id uuid not null references organizations(id),
 payload_hash text not null,
 result jsonb not null,
 created_at timestamptz not null default now(),
 primary key(tutor_id,request_id)
);
create table plan_calendar_occurrences (
 tutor_id uuid not null references users(id),
 plan_lesson_key uuid not null,
 local_date date not null,
 lesson_id uuid not null unique references lessons(id),
 primary key(tutor_id,plan_lesson_key,local_date)
);
