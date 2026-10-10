# SourceInk roadmap

SourceInk should keep its current modular monolith while the product is small. Next.js serves the web interface, Go owns the API and sync logic, PostgreSQL stores product state, and the GitHub App supplies source content. Add another service only when a feature cannot fit cleanly inside that setup.

## Current foundation

The MVP already supports:

- email and GitHub-linked accounts;
- selected GitHub repositories;
- signed, deduplicated push webhooks;
- a periodic reconciliation fallback;
- Markdown discovery and validation;
- manual and automatic publishing;
- stored public copies that do not depend on GitHub at read time;
- article reviews and simple view counts.

Keep `backend/internal/database/migrations/00001_users.sql` as the single bootstrap migration until the first shared or production deployment. After that deployment, stop editing it and create a new ordered migration for each schema change.

## Upgrade order

### 1. Prove the production path

Do this before adding another product feature.

Add:

- a staging deployment with a real GitHub App;
- a real signed webhook delivery test;
- PostgreSQL backups and one restore drill;
- structured error logs for failed webhook deliveries and repository syncs;
- a release checklist for migrations, health checks, and rollback.

Implementation:

1. Set `APP_ENV=production`, `COOKIE_SECURE=true`, and `AUTO_MIGRATE=false` in staging.
2. Run `sourceink-migrate up` as a release step before starting the new backend.
3. Push a Markdown change to a connected private repository.
4. Confirm one webhook delivery, one sync, and one published revision.
5. Stop GitHub access temporarily and confirm that the published article still loads.
6. Restore a database backup into a disposable PostgreSQL instance and run the application against it.

Finish this milestone when a fresh deployment, webhook sync, database restore, and rollback have all been exercised outside local development.

### 2. Show sync health to authors

Authors need to know whether SourceInk received their push and why an article did not update.

Add:

- last successful sync time on each repository;
- current sync state: `idle`, `syncing`, or `failed`;
- the latest actionable error;
- a small **Sync now** action;
- recent webhook delivery status in the repository detail view.

Implementation:

1. Add sync status fields to `repositories`, or add one small `repository_syncs` table if you need attempt history.
2. Update those fields around the existing `SyncRepository` call.
3. Add `POST /api/repos/{repositoryID}/sync` with owner authorization.
4. Reuse the current article worker. Do not add a queue.
5. Display the status and retry action on the repositories dashboard.
6. Test success, GitHub timeout, revoked installation, and retry behavior.

### 3. Improve the reader surface

Make published work easier to find before building social features.

Add:

- tag pages;
- author pages;
- RSS feeds;
- `sitemap.xml` and canonical article URLs;
- PostgreSQL full-text search over title, description, tags, and Markdown.

Implementation:

1. Add public read routes for authors, tags, RSS, and search.
2. Start with PostgreSQL `tsvector` and a GIN index. Keep search inside PostgreSQL.
3. Generate RSS and sitemap responses from published articles only.
4. Add pagination before the article count becomes large enough to make full lists slow.
5. Measure query time before considering a separate search service.

### 4. Store article assets

Add this when authors need images or downloadable files in published articles.

Add:

- Markdown asset discovery during sync;
- persisted copies of assets required by a published article;
- stable public asset URLs;
- missing-file and size validation.

Implementation:

1. Parse relative image and file references from validated Markdown.
2. Fetch only referenced files through the GitHub installation token.
3. Store asset metadata in PostgreSQL.
4. Store file bytes in S3-compatible object storage. A local MinIO container is enough for development.
5. Rewrite rendered asset URLs to SourceInk-owned public URLs.
6. Keep old published assets while a published article still references them.

Do not proxy private GitHub asset URLs to readers and do not copy the complete repository.

### 5. Add publication history and rollback

Build this only when authors need to compare or restore older published versions.

Add:

- immutable article revision snapshots;
- separate latest and published revision references;
- a publication history page;
- rollback to a stored revision.

Implementation:

1. Add `article_revisions` with the article ID, Git commit SHA, Git blob SHA, Markdown snapshot, frontmatter, validation result, and creation time.
2. Add `latest_revision_id` and `published_revision_id` references to the article record.
3. Make sync create or reuse an immutable revision.
4. Keep manual publish as the only operation that changes the published revision for manual articles.
5. Point reviews at the immutable revision ID instead of copying revision identity into new features.
6. Add rollback as another explicit publish operation.

Do not add this model only to preserve history that nobody uses. GitHub already owns the complete source history.

### 6. Add custom domains

Add custom domains after the standard SourceInk article URL works reliably in production.

Add:

- domain ownership verification;
- one primary domain per publication or author;
- automatic TLS through the reverse proxy;
- canonical redirects from old domains.

Implementation:

1. Add a `domains` table with owner, hostname, verification token, status, and timestamps.
2. Ask the author to create a DNS TXT record for verification.
3. Route verified hostnames to the correct published content.
4. Let Caddy or another existing reverse proxy manage certificates.
5. Reject duplicate hostnames and reserve SourceInk system domains.

### 7. Expand analytics only when authors ask for it

The current `view_count` answers the first question: how many article opens came from the feed.

Possible later additions:

- daily view totals per article;
- referrer category such as direct, search, or external;
- a 7-day and 30-day dashboard chart.

Use daily aggregate rows. Avoid raw IP storage, browser fingerprinting, session replay, and a separate analytics platform unless the product requirements change.

### 8. Add background-job infrastructure after measurement

The existing in-process worker should remain until sync work causes slow startup, missed retries, or multiple backend replicas competing for the same work.

First upgrade:

- a PostgreSQL `jobs` table;
- `FOR UPDATE SKIP LOCKED` claiming;
- attempt count, next-attempt time, and last error;
- one worker loop inside the existing Go process.

Consider Redis, a separate worker service, or a message broker only after PostgreSQL jobs become a measured bottleneck.

## Standard implementation path

Use the same order for each upgrade:

1. Write the product rule and failure behavior.
2. Add the smallest schema change.
3. Add store methods in `backend/internal/database/`.
4. Add service logic and authorization.
5. Add the `chi` handler and route.
6. Allow the route through `frontend/app/api/[...path]/route.ts` when browser access is required.
7. Add frontend response types and validation.
8. Build the smallest useful interface.
9. Test the store, service, handler, and user-visible failure state.
10. Run `make test`, `make build`, and one live request through the running stack.

After the first shared deployment, create schema changes with:

```sh
make migration name=describe_the_change
make migrate-status
make migrate
```

Back up PostgreSQL before applying a production migration. Deploy additive schema changes before code that requires them. Remove old columns only in a later release after the running code no longer reads them.

## Keep out of the near-term stack

Do not add these without a measured requirement:

- microservices;
- Kubernetes;
- Kafka or another event bus;
- Redis for data already handled by PostgreSQL;
- Elasticsearch for the first version of search;
- a full repository clone or complete Git history;
- recommendations, follower graphs, comments, or reputation scores.

The next concrete milestone is production-path verification, followed by author-visible sync health.
