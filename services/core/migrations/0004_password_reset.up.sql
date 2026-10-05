create table password_reset_tokens (
  id uuid primary key,
  user_id uuid not null unique references users(id) on delete cascade,
  token_hash text not null unique,
  expires_at timestamptz not null,
  used_at timestamptz,
  created_at timestamptz not null default now()
);
