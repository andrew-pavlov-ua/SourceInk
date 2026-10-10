# SourceInk

## Purpose

SourceInk is a Git-first publishing platform for technical articles. Authors keep Markdown, frontmatter, assets, history, and repository structure in repositories they own. SourceInk provides discovery, preview, publication, public delivery, and custom-domain support.

The GitHub repository owns article source. SourceInk stores the data it needs to publish and serve articles, but it must not replace the repository as the canonical source.

## Architecture

- Build the MVP as a modular monolith.
- Use Next.js with TypeScript for the frontend, Go with `chi` for the backend HTTP API, `sqlx` for queries, and PostgreSQL for persistence.
- Integrate through a GitHub App with webhooks and the GitHub API.
- Keep sync and background-job code inside the backend application until workload or operating constraints justify a separate service.
- Serve public articles from persisted SourceInk data. Reader requests must not depend on GitHub availability.
- Cache rendered HTML, metadata, search data, and published assets needed at request time. Avoid cloning complete repositories or retaining complete Git history without a demonstrated need.

The intended flow is:

1. A user installs the GitHub App for selected repositories.
2. SourceInk discovers candidate Markdown files and parses their YAML frontmatter.
3. The user chooses which candidates become articles.
4. A GitHub webhook schedules an idempotent sync after a push.
5. SourceInk records the latest revision and prepares a preview and diff.
6. Manual articles change their public revision only after an explicit publish action. Automatic articles publish the synced revision after validation succeeds.

## Domain invariants

- A Markdown file is a publication candidate, not a published article.
- Track the latest Git revision and the published public revision as separate values. Never derive public state from the newest commit alone.
- Give each article an immutable internal ID. Treat repository and source path as mutable source-location data.
- Preserve article identity and public URLs across source-file moves and renames when the available Git data supports a reliable match.
- Store an immutable content snapshot for each revision that SourceInk needs for preview or publication. A publication points to a specific stored revision.
- Keep manual publication explicit. Syncing a manual article must not change its published revision.
- Publish an automatic article only after sync, parsing, rendering, and validation succeed.
- Model push, sync, preview, and publish as separate operations. Code must tolerate delay, retries, reordering, and partial failure between them.
- Keep published content available when GitHub is slow, rate-limited, unavailable, or no longer grants access.
- Treat deleted files, branches, and repositories as source-state changes. Do not erase a published article as an implicit side effect of sync.
- Validate frontmatter and content limits. Report malformed frontmatter, missing assets, and oversized Markdown as actionable sync errors.

Names such as `Article`, `ArticleRevision`, `Publication`, `Repository`, `GitHubInstallation`, and `WebhookDelivery` describe domain concepts, not required type or table names. Follow names already established in the code.

## Directory conventions

Follow the checked-in layout and the nearest `AGENTS.md`. Do not reorganize the repository to match a generic template.

- `frontend/` contains the Next.js App Router application. Pages live in `frontend/app/`, reusable UI in `frontend/components/`, and server helpers in `frontend/lib/`.
- `backend/cmd/server/` starts the API. `backend/cmd/migrate/` provides the Goose migration runner.
- `backend/internal/auth/` owns credential, session, and auth HTTP logic.
- `backend/internal/database/` owns connection setup and embedded Goose migrations. SQL migrations live in `backend/internal/database/migrations/`.
- `backend/internal/httpserver/` owns `chi` routes and HTTP middleware.
- Keep GitHub integration, content parsing/rendering, publication, and public delivery as separate backend modules with explicit interfaces.
- Keep PostgreSQL access behind the repository's data-access conventions. Domain code must not scatter SQL or GitHub API calls across HTTP handlers.
- Store migrations in the migration directory selected by the backend tooling. Use one ordered migration sequence.
- Put tests beside the code or in the test directories already used by each application.
- Keep deployment, local-development, and CI configuration at their established paths.

## Development and test commands

Run project commands from the repository root:

- `make dev`: build and run the Compose stack in the foreground.
- `make up`: build and run the Compose stack in the background.
- `make down`: stop the Compose stack without deleting PostgreSQL data.
- `make logs`: follow service logs.
- `make test`: run backend tests plus frontend lint and type checks.
- `make build`: build both applications.
- `make migrate`: apply pending Goose migrations to the Compose database.
- `make migrate-status`: print migration status.
- `make migration name=add_articles`: create a timestamped SQL migration.

Run backend tests with `cd backend && go test ./...`. Run frontend checks with `cd frontend && npm test`; use `npm run build` for a production build.

Do not introduce a second package manager, task runner, test framework, or local environment path when the repository has already selected one. Do not claim validation from commands that are absent or that did not run.

## Database and migrations

- PostgreSQL stores platform metadata, source locations, revision snapshots needed by the product, publication state, rendered output, asset references, domains, and webhook-processing records.
- Represent latest and published revisions with separate columns or relations and enforce referential integrity for both.
- Keep article identity independent from path and slug. Treat public slug changes as URL-management decisions, not source-file identity changes.
- Record Git commit SHA and Git blob SHA when they help synchronization, deduplication, or rename detection.
- Give webhook deliveries a unique provider delivery ID and persist processing state so retries remain idempotent.
- Until the first production deployment, keep the bootstrap schema consolidated in `00001_users.sql`. After production has applied it, treat it as immutable and add ordered migrations for every schema change.
- Use Goose annotations in SQL migrations. Compose may apply migrations on startup through `AUTO_MIGRATE=true`; production deployments must run `sourceink-migrate up` as a controlled release step.
- Local Compose startup seeds or refreshes the configured admin only when `APP_ENV=development`. Keep `DEV_ADMIN_PASSWORD` in the gitignored `.env`; never hard-code or log it.
- Keep schema changes compatible with rolling application changes when deployment order requires it.
- Do not store every repository object, historical asset version, or Git commit by default.
- Do not run migrations against a shared or production database without explicit user approval.

## GitHub integration

- Use a GitHub App rather than user tokens or OAuth scopes intended for unrelated account access.
- Request read-only access and only the permissions needed for repositories the user selects.
- Validate every webhook signature against the raw request body before parsing or enqueueing work.
- Deduplicate deliveries and make webhook processing safe to retry. A successful duplicate must not create another revision or publication.
- Treat webhook payloads as event hints. Fetch required source data through authenticated GitHub APIs and validate repository, installation, branch, path, and object identity.
- Handle rate limits, redelivery, revoked installations, private repositories, repository or default-branch changes, force pushes, renames, and deletions.
- Limit sync to relevant changed files when reliable change data exists. Fall back to a bounded reconciliation job when events cannot prove current state.
- Never call GitHub from the public article request path.
- Never expose installation tokens, webhook secrets, private repository contents, or raw private asset URLs to the browser or logs.

## Agent constraints

- Inspect the current code, manifests, migrations, docs, and tests before changing architecture.
- Fit small changes into existing conventions. Avoid broad rewrites and generated boilerplate.
- Do not add microservices, distributed queues, event buses, or other infrastructure without an operating requirement that the modular monolith cannot meet.
- Do not equate discovery, sync, or a Git push with publication.
- Do not use a source path as the permanent article identity.
- Do not delete published content in response to a missing source file without an explicit product rule and user action.
- Do not grant broad GitHub permissions or place GitHub in the reader request path.
- Do not duplicate a full repository, full Git history, or all historical assets by default.
- Do not bypass webhook verification, idempotency, content validation, authorization, or publication-state checks.
- Do not add recommendation systems, social graphs, elaborate comments, excessive analytics, or premature discovery features to the MVP.
- Do not stage, commit, deploy, or change live data unless the user asks.
