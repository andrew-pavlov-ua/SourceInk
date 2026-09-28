package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

func EnsureDevelopmentAdmin(ctx context.Context, db *sqlx.DB, environment, email, username, password string) error {
	if environment != "development" {
		return nil
	}
	if email == "" || username == "" || password == "" {
		return errors.New("DEV_ADMIN_EMAIL, DEV_ADMIN_USERNAME, and DEV_ADMIN_PASSWORD are required in development")
	}

	email, username, err := NormalizeAndValidateRegistration(email, username, password)
	if err != nil {
		return fmt.Errorf("validate development admin: %w", err)
	}
	passwordHash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash development admin password: %w", err)
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin development admin seed: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var matches []struct {
		ID       string `db:"id"`
		Email    string `db:"email"`
		Username string `db:"username"`
	}
	err = tx.SelectContext(ctx, &matches, `
		select id, email, username
		from users
		where lower(email) = lower($1) or lower(username) = lower($2)
		for update
	`, email, username)
	if err != nil {
		return fmt.Errorf("find development admin: %w", err)
	}

	switch len(matches) {
	case 0:
		_, err = tx.ExecContext(ctx, `
			insert into users (email, username, password_hash, role)
			values ($1, $2, $3, 'admin')
		`, email, username, passwordHash)
	case 1:
		existing := matches[0]
		if !strings.EqualFold(existing.Email, email) || !strings.EqualFold(existing.Username, username) {
			return errors.New("development admin email or username conflicts with another account")
		}
		_, err = tx.ExecContext(ctx, `
			update users
			set password_hash = $2, role = 'admin', updated_at = now()
			where id = $1
		`, existing.ID, passwordHash)
	default:
		return errors.New("development admin email and username conflict with separate accounts")
	}
	if err != nil {
		return fmt.Errorf("write development admin: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit development admin seed: %w", err)
	}
	return nil
}
