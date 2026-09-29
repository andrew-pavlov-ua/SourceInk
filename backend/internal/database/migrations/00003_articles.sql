-- +goose Up
create table if not exists unpublished_articles (
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

create table if not exists articles (
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
    published_at timestamptz not null default now(),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create unique index if not exists articles_current_source_unique
    on articles (repository_id, source_path)
    where repository_id is not null and source_state = 'available';

-- +goose Down
drop table if exists articles;
drop table if exists unpublished_articles;
