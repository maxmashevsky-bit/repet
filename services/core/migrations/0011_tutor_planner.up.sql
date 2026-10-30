create table tutor_plans (
 tutor_id uuid primary key references users(id),
 organization_id uuid not null references organizations(id),
 revision bigint not null check(revision > 0),
 document jsonb not null check(jsonb_typeof(document) = 'object'),
 updated_at timestamptz not null default now()
);
