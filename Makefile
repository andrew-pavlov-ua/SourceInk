dev:
	docker compose up --build postgres backend

frontend-dev:
	cd frontend && npm run dev

up:
	docker compose up --build --detach

down:
	docker compose down

logs:
	docker compose logs --follow

test: test-backend test-frontend

test-backend:
	cd backend && go test ./...

test-frontend:
	cd frontend && npm test

build:
	cd backend && go build -buildvcs=false ./...
	cd frontend && npm run build

migrate:
	docker compose run --rm backend sourceink-migrate up

migrate-status:
	docker compose run --rm backend sourceink-migrate status

goose_lite_tags := no_azuresql no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb

migration:
	@test -n "$(name)" || (echo "usage: make migration name=add_articles" && exit 1)
	cd backend && go run -mod=readonly -tags="$(goose_lite_tags)" github.com/pressly/goose/v3/cmd/goose -dir internal/database/migrations create "$(name)" sql
