-- +goose Up
create table if not exists github_installations (
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

create index if not exists github_installations_user_id_idx on github_installations (user_id);

create table if not exists repositories (
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

create index if not exists repositories_installation_id_idx on repositories (installation_id);

-- +goose Down
drop table if exists repositories;
drop table if exists github_installations;
