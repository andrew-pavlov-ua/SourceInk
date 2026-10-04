-- +goose Up
create table if not exists github_webhook_deliveries (
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
