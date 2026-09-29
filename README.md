# SourceInk

SourceInk publishes technical articles from GitHub repositories. Authors keep their Markdown and history in GitHub. SourceInk stores the drafts and published copies it needs to run the site.

The current build has a Next.js frontend, a Go API, PostgreSQL, account login, and GitHub App installation. SourceInk discovers Markdown with frontmatter and stores it as drafts. Publishing and public delivery are not built yet.

## Run locally

Copy `.env.example` to `.env`, choose a development database password, then run the API stack and Next.js development server in separate terminals:

```sh
cp .env.example .env
make dev
```

```sh
cd frontend
npm run dev
```

Open <http://localhost:3000>. Check the API at <http://localhost:8080/healthz> and <http://localhost:8080/readyz>.

`make dev` starts PostgreSQL and the backend. Keep it running while `npm run dev` serves the frontend with hot reload. Run `make up` to start the frontend in Compose too.

Set `DEV_ADMIN_EMAIL`, `DEV_ADMIN_USERNAME`, and `DEV_ADMIN_PASSWORD` in the gitignored `.env`. The development server creates that admin account on startup and keeps its password in sync with the file. The seed refuses to run outside `APP_ENV=development`.

Set `DEV_USER_EMAIL`, `DEV_USER_USERNAME`, and `DEV_USER_PASSWORD` for a regular development account. If `DEV_USER_PASSWORD` is empty, local development reuses `DEV_ADMIN_PASSWORD`; set it explicitly when the accounts should have different passwords.

GitHub App setup requires `GITHUB_PUBLISHER_CLIENT_ID` and `GITHUB_PUBLISHER_CLIENT_SECRET`. Set the callback URL to `http://localhost:3000/api/github/callback`. The backend keeps each OAuth attempt in memory for ten minutes. A restart cancels it.

Compose exposes the application on localhost. PostgreSQL stays on the Compose network.

## Validate

```sh
make test
make build
```

The backend routes requests with `chi`, queries PostgreSQL with `sqlx`, and runs migrations through `goose`. Local Compose startup applies pending migrations through `AUTO_MIGRATE=true`.

Migration commands:

```sh
make migrate
make migrate-status
make migration name=add_articles
```

Run `sourceink-migrate up` during a production release. Set `COOKIE_SECURE=true` behind HTTPS.
