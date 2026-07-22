alter table users add column if not exists password_hash text;
alter table users add column if not exists email_verified_at timestamptz;
alter table users add column if not exists timezone text not null default 'Europe/Moscow';
alter table users add column if not exists locale text not null default 'ru';

create table if not exists sessions (
  id uuid primary key,
  organization_id uuid not null references organizations(id),
  user_id uuid not null references users(id) on delete cascade,
  token_hash text not null unique,
  user_agent text not null default '',
  ip_address text not null default '',
  expires_at timestamptz not null,
  revoked_at timestamptz,
  created_at timestamptz not null default now()
);

create table if not exists conversations (
  id uuid primary key,
  organization_id uuid not null references organizations(id),
  relation_id uuid references relations(id),
  tutor_id uuid not null references users(id),
  student_id uuid not null references users(id),
  created_at timestamptz not null default now(),
  unique (organization_id, tutor_id, student_id)
);

create table if not exists messages (
  id uuid primary key,
  organization_id uuid not null references organizations(id),
  conversation_id uuid not null references conversations(id) on delete cascade,
  author_id uuid not null references users(id),
  body text not null,
  idempotency_key text not null,
  created_at timestamptz not null default now(),
  unique (conversation_id, author_id, idempotency_key)
);

create table if not exists assignments (
  id uuid primary key,
  organization_id uuid not null references organizations(id),
  tutor_id uuid not null references users(id),
  student_id uuid not null references users(id),
  title text not null,
  subject text not null,
  topic text not null,
  body text not null,
  status text not null check (status in ('draft', 'assigned', 'submitted', 'revision', 'done', 'overdue', 'archived')),
  due_at timestamptz not null,
  max_score integer not null default 100,
  created_at timestamptz not null default now()
);

create table if not exists assignment_submissions (
  id uuid primary key,
  organization_id uuid not null references organizations(id),
  assignment_id uuid not null references assignments(id) on delete cascade,
  student_id uuid not null references users(id),
  answer text not null,
  score integer,
  feedback text not null default '',
  submitted_at timestamptz not null default now(),
  graded_at timestamptz,
  unique (assignment_id, student_id)
);

create table if not exists notifications (
  id uuid primary key,
  organization_id uuid not null references organizations(id),
  user_id uuid not null references users(id) on delete cascade,
  title text not null,
  body text not null,
  href text not null,
  read_at timestamptz,
  idempotency_key text not null,
  created_at timestamptz not null default now(),
  unique (user_id, idempotency_key)
);

create index if not exists sessions_lookup_idx on sessions (token_hash) where revoked_at is null;
create index if not exists conversations_actor_idx on conversations (organization_id, tutor_id, student_id);
create index if not exists messages_page_idx on messages (conversation_id, created_at desc, id desc);
create index if not exists assignments_tutor_idx on assignments (organization_id, tutor_id, due_at);
create index if not exists assignments_student_idx on assignments (organization_id, student_id, due_at);
create index if not exists notifications_user_idx on notifications (organization_id, user_id, read_at, created_at desc);

