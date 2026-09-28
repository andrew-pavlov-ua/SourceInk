package database

import (
	"context"
	"embed"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
)

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

//go:embed migrations/*.sql
var migrationFiles embed.FS

func Open(ctx context.Context, databaseURL string) (*sqlx.DB, error) {
	db, err := sqlx.ConnectContext(ctx, "pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	return db, nil
}

func RunMigrations(ctx context.Context, db *sqlx.DB, command string, args ...string) error {
	goose.SetBaseFS(migrationFiles)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set migration dialect: %w", err)
	}
	if err := goose.RunContext(ctx, command, db.DB, "migrations", args...); err != nil {
		return fmt.Errorf("goose %s: %w", command, err)
	}
	return nil
}
