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

create table github_installations (
    user_id uuid not null references users(id) on delete cascade,
    installation_id bigint primary key,
    github_account_id bigint not null,
    github_account_login text not null,
    github_account_type text not null,
    repository_selection text not null,
    suspended boolean not null default false,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),

    constraint github_installations_repository_selection_valid
        check (repository_selection in ('all', 'selected'))
);

create index github_installations_user_id_idx on github_installations (user_id);

create table repositories (
    id uuid primary key default gen_random_uuid(),
    github_id bigint not null unique,
    installation_id bigint not null references github_installations(installation_id) on delete cascade,
    owner text not null,
    name text not null,
    full_name text not null,
    default_branch text not null,
    private boolean not null,
    archived boolean not null default false,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index repositories_installation_id_idx on repositories (installation_id);

create table unpublished_articles (
    id uuid primary key default gen_random_uuid(),
    repository_id uuid not null references repositories(id) on delete cascade,
    source_path text not null,
    git_blob_sha text not null,
    markdown text not null,
    title text,
    slug text,
    description text,
    tags text[],
    publish_mode text,
    validation_error text,
    present boolean not null default true,
    discovered_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (repository_id, source_path)
);

create table articles (
    id uuid primary key default gen_random_uuid(),
    unpublished_article_id uuid unique references unpublished_articles(id) on delete set null,
    owner_id uuid not null references users(id),
    repository_id uuid references repositories(id) on delete set null,
    source_path text not null,
    slug text not null unique,
    publish_mode text not null default 'manual'
        check (publish_mode in ('manual', 'auto')),
    source_state text not null default 'available'
        check (source_state in ('available', 'missing', 'access_lost')),
    git_blob_sha text not null,
    markdown text not null,
    title text not null,
    frontmatter jsonb,
    view_count bigint not null default 0 check (view_count >= 0),
    published_at timestamptz not null default now(),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create unique index articles_current_source_unique
    on articles (repository_id, source_path)
    where repository_id is not null and source_state = 'available';

create table article_reviews (
    id uuid primary key default gen_random_uuid(),
    article_id uuid not null references articles(id) on delete cascade,
    git_blob_sha text not null,
    reviewer_id uuid not null references users(id) on delete cascade,
    verdict text not null,
    reason text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),

    constraint article_reviews_revision_reviewer_unique
        unique (article_id, git_blob_sha, reviewer_id),
    constraint article_reviews_verdict_valid
        check (verdict in ('approve', 'request_changes')),
    constraint article_reviews_reason_valid
        check (reason is null or reason in ('incorrect', 'outdated', 'unclear')),
    constraint article_reviews_verdict_reason_valid
        check (
            (verdict = 'approve' and reason is null)
            or
            (verdict = 'request_changes' and reason is not null)
        )
);

create index article_reviews_reviewer_id_idx on article_reviews (reviewer_id);

create table github_webhook_deliveries (
    delivery_id text primary key,
    event text not null,
    installation_id bigint not null,
    repository_github_id bigint not null,
    status text not null check (status in ('received', 'processed', 'failed')),
    error text,
    received_at timestamptz not null default now(),
    processed_at timestamptz
);

-- +goose Down
drop table if exists github_webhook_deliveries;
drop table if exists article_reviews;
drop table if exists articles;
drop table if exists unpublished_articles;
drop table if exists repositories;
drop table if exists github_installations;
drop table if exists sessions;
drop table if exists users;
