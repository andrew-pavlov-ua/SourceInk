-- +goose Up
create table if not exists article_reviews (
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

create index if not exists article_reviews_reviewer_id_idx
    on article_reviews (reviewer_id);

-- +goose Down
drop table if exists article_reviews;
