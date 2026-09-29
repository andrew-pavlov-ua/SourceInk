-- +goose Up
create extension if not exists pgcrypto;

create table users (
    id uuid primary key default gen_random_uuid(),
    email text,
    username text not null,
    password_hash text,
    github_user_id bigint unique,
    github_login text unique,
    github_avatar_url text,
    role text not null default 'user',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),

    constraint users_email_length check (char_length(email) between 3 and 254),
    constraint users_username_length check (char_length(username) between 3 and 32),
    constraint users_role_valid check (role in ('user', 'admin'))
);

create unique index users_email_unique on users (lower(email));
create unique index users_username_unique on users (lower(username));

create table sessions (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    token_hash bytea not null unique,
    expires_at timestamptz not null,
    created_at timestamptz not null default now()
);

create index sessions_user_id_idx on sessions (user_id);
create index sessions_expires_at_idx on sessions (expires_at);

-- +goose Down
drop table if exists sessions;
drop table if exists users;
