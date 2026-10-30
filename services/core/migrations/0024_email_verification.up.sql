create table email_verification_tokens (
    user_id uuid primary key references users(id) on delete cascade,
    email text not null,
    token_hash text not null unique,
    created_at timestamptz not null default clock_timestamp(),
    expires_at timestamptz not null default clock_timestamp() + interval '1 hour',
    used_at timestamptz
);
