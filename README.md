# SourceInk

SourceInk publishes technical articles from GitHub repositories. Authors keep their Markdown and history in GitHub. SourceInk stores the drafts and published copies it needs to run the site.

The current build has a Next.js frontend, a Go API, PostgreSQL, account login, GitHub App installation, draft discovery, and manual or automatic publishing. Published articles are served from SourceInk's stored copy.

## Start on a personal server

Use this section to run SourceInk from your own server. You need Docker, a domain name, and an HTTPS reverse proxy. Point the domain at the server and send HTTPS traffic to `127.0.0.1:3000` before you create the GitHub App.

This guide uses `https://sourceink.example.com`. Replace it with your domain.

### 1. Create the configuration files

Run these commands in the SourceInk project folder:

```sh
cp .env.example .env
mkdir -p .secrets
chmod 700 .secrets
```

`.env` stores configuration and passwords. `.secrets` stores the GitHub private key. Git ignores both paths.

### 2. Create a GitHub App

Open GitHub and click **your profile picture > Settings > Developer settings > GitHub Apps > New GitHub App**.

Give the App a name, such as `My SourceInk`. Fill in these fields:

| GitHub field | Enter this value |
| --- | --- |
| Homepage URL | `https://sourceink.example.com` |
| Callback URL | `https://sourceink.example.com/api/github/callback` |
| Setup URL | `https://sourceink.example.com/api/github/setup` |
| Webhook URL | `https://sourceink.example.com/api/github/webhook` |
| Webhook secret | A long random secret that you will copy into `.env` |

Set the remaining GitHub options:

- Leave **Request user authorization (OAuth) during installation** off.
- Turn **Active** on for webhooks.
- Turn **Redirect on update** on.
- Under **Repository permissions**, set **Contents** to **Read-only**.
- Under **Subscribe to events**, select **Push**.
- Keep the App private if only you will use this server.

Click **Create GitHub App**.

### 3. Save the GitHub credentials

On the GitHub App settings page:

1. Copy the **Client ID**.
2. Create a **Client secret** and copy it. GitHub only shows the full secret once.
3. Click **Generate a private key**. GitHub downloads a `.pem` file.
4. Read the App slug from the App URL. For `https://github.com/apps/my-sourceink`, the slug is `my-sourceink`.

Move the downloaded key into `.secrets`:

```sh
mv /path/to/downloaded-key.pem .secrets/sourceink-publisher.pem
chmod 600 .secrets/sourceink-publisher.pem
```

Do not commit the `.pem` file, GitHub client secret, webhook secret, or passwords.

### 4. Fill in `.env`

Open `.env` and set these values:

```dotenv
APP_ENV=production
APP_ORIGIN=https://sourceink.example.com
COOKIE_SECURE=true

POSTGRES_PASSWORD=choose-a-strong-database-password
DEV_ADMIN_PASSWORD=choose-an-unused-strong-password

GITHUB_PUBLISHER_APP_SLUG=my-sourceink
GITHUB_PUBLISHER_CLIENT_ID=your-client-id
GITHUB_PUBLISHER_CLIENT_SECRET=your-client-secret
GITHUB_WEBHOOK_SECRET=the-same-secret-entered-on-github
GITHUB_PUBLISHER_PRIVATE_KEY_PATH=.secrets/sourceink-publisher.pem
```

Compose requires `DEV_ADMIN_PASSWORD` even in production. SourceInk only creates development users when `APP_ENV=development`.

### 5. Start SourceInk

Run:

```sh
make up
docker compose ps
curl --fail http://127.0.0.1:8080/readyz
```

The last command should return a successful response. Open `https://sourceink.example.com` in your browser.

### 6. Connect your repositories

1. Create an account in SourceInk and sign in.
2. Open the dashboard and choose **Manage repositories**.
3. Choose **Connect GitHub**.
4. GitHub opens the App installation page. Choose your account or organization.
5. Choose the repositories that SourceInk may read.
6. Finish the GitHub flow and return to SourceInk.

### 7. Add an article

Create a Markdown file in a connected repository:

```md
---
title: My first article
slug: my-first-article
description: A short summary of the article.
tags:
  - go
  - postgres
publish_mode: manual
---

# My first article

Write the article here.
```

Push the file to GitHub. SourceInk receives the push and shows the draft in the Articles dashboard. Use `publish_mode: manual` to publish from SourceInk. Use `publish_mode: auto` to publish after SourceInk validates a synced file.

Article slugs use lowercase letters, numbers, and single hyphens. Markdown source files may be up to 512 KiB.

GitHub webhooks normally trigger the sync immediately. SourceInk also reconciles connected repositories every 15 minutes so a missed delivery or server restart does not leave drafts stale indefinitely.

## Run locally

Use local mode when you work on SourceInk code. Complete the GitHub App setup in sections 1 through 4 first. The current Compose configuration always mounts the GitHub App private key, and the backend rejects incomplete GitHub credentials. A copied `.env.example` with its placeholder values cannot start the backend.

Copy the example configuration, set `APP_ENV=development` and `COOKIE_SECURE=false`, then replace every `GITHUB_PUBLISHER_*` value and `GITHUB_WEBHOOK_SECRET` with your GitHub App values. Keep the private key at `.secrets/sourceink-publisher.pem`.

Run the backend and database in one terminal:

```sh
cp .env.example .env
make dev
```

Run the frontend in another terminal:

```sh
cd frontend
npm ci
npm run dev
```

Open <http://localhost:3000>. `npm ci` installs the exact packages from `package-lock.json`. Run it after a fresh clone, after a lockfile update, or if `npm run dev` cannot find `next` or another package.

`make dev` starts PostgreSQL and the backend. In development mode, SourceInk also seeds the configured development admin and user accounts. `npm run dev` starts the frontend with hot reload. Run `make up` when you want Docker Compose to start the frontend too.

GitHub cannot send webhooks to `localhost`. Use a public HTTPS forwarding URL ending in `/api/github/webhook` to test GitHub push events on your computer.

## Validate

```sh
make test
make build
```

The backend routes requests with `chi`, queries PostgreSQL with `sqlx`, and runs migrations through `goose`. The current Compose configuration sets `AUTO_MIGRATE=true`, so `make dev` and `make up` apply pending migrations when the backend starts.

Migration commands:

```sh
make migrate
make migrate-status
make migration name=add_articles
```

The current MVP is still in its bootstrap stage, so its complete schema lives in `backend/internal/database/migrations/00001_users.sql`. Do not edit that file after the first shared or production deployment; create a new ordered migration with `make migration` for every later schema change.

`make migrate` runs `sourceink-migrate up` manually, but it does not disable the startup migration. For a controlled production migration step, set `AUTO_MIGRATE=false` in the deployment configuration and run `sourceink-migrate up` before starting the backend. Set `COOKIE_SECURE=true` behind HTTPS.

## License

Copyright (C) 2026 Andrew Pavlov.

SourceInk is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License as published by the Free Software Foundation, version 3. See [LICENSE](LICENSE) for the full text.

If you run a modified version of SourceInk as a network service, the AGPL requires you to offer your users the corresponding source code.

Articles that authors publish through SourceInk stay under the license each author chooses. The AGPL covers only the SourceInk software.
