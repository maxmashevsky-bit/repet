create extension if not exists pgcrypto;

create table if not exists organizations (
  id uuid primary key,
  name text not null,
  created_at timestamptz not null default now()
);

insert into organizations (id, name)
values ('00000000-0000-4000-8000-000000000001', 'Development Organization')
on conflict (id) do nothing;

create table users (
  id uuid primary key,
  organization_id uuid not null references organizations(id),
  email text not null,
  display_name text not null,
  role text not null check (role in ('tutor', 'student', 'admin')),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (organization_id, email)
);

create table tutor_profiles (
  user_id uuid primary key references users(id) on delete cascade,
  headline text not null default '',
  created_at timestamptz not null default now()
);

create table student_profiles (
  user_id uuid primary key references users(id) on delete cascade,
  guardian_email text not null default '',
  created_at timestamptz not null default now()
);

create table invitations (
  id uuid primary key,
  organization_id uuid not null references organizations(id),
  tutor_id uuid not null references users(id),
  student_email text not null,
  student_name text not null default '',
  token_hash text not null unique,
  status text not null check (status in ('pending', 'accepted', 'revoked', 'expired')),
  expires_at timestamptz not null,
  accepted_by uuid references users(id),
  accepted_at timestamptz,
  created_at timestamptz not null default now()
);

create table relations (
  id uuid primary key,
  organization_id uuid not null references organizations(id),
  tutor_id uuid not null references users(id),
  student_id uuid not null references users(id),
  invitation_id uuid references invitations(id),
  status text not null check (status in ('active', 'blocked', 'ended')),
  created_at timestamptz not null default now(),
  unique (organization_id, tutor_id, student_id)
);

create table lessons (
  id uuid primary key,
  organization_id uuid not null references organizations(id),
  relation_id uuid not null references relations(id),
  tutor_id uuid not null references users(id),
  student_id uuid not null references users(id),
  title text not null,
  starts_at timestamptz not null,
  ends_at timestamptz not null,
  status text not null check (status in ('scheduled', 'cancelled', 'completed')),
  created_by uuid not null references users(id),
  created_at timestamptz not null default now(),
  check (ends_at > starts_at)
);

create table audit_events (
  id uuid primary key,
  organization_id uuid not null references organizations(id),
  actor_id uuid references users(id),
  action text not null,
  resource_type text not null,
  resource_id uuid,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create index invitations_tutor_idx on invitations (organization_id, tutor_id, created_at desc);
create index relations_tutor_idx on relations (organization_id, tutor_id);
create index relations_student_idx on relations (organization_id, student_id);
create index lessons_relation_idx on lessons (organization_id, relation_id, starts_at desc);
create index audit_events_actor_idx on audit_events (organization_id, actor_id, created_at desc);

