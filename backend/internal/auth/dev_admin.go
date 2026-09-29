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
	return ensureDevelopmentAccount(ctx, db, environment, email, username, password, "admin", true)
}

// EnsureDevelopmentUser seeds the regular account used to test the app locally.
func EnsureDevelopmentUser(ctx context.Context, db *sqlx.DB, environment, email, username, password string) error {
	return ensureDevelopmentAccount(ctx, db, environment, email, username, password, "user", false)
}

func ensureDevelopmentAccount(ctx context.Context, db *sqlx.DB, environment, email, username, password, role string, refreshRole bool) error {
	if environment != "development" {
		return nil
	}
	if email == "" || username == "" || password == "" {
		if role == "admin" {
			return errors.New("DEV_ADMIN_EMAIL, DEV_ADMIN_USERNAME, and DEV_ADMIN_PASSWORD are required in development")
		}
		return errors.New("DEV_USER_EMAIL, DEV_USER_USERNAME, and DEV_USER_PASSWORD are required in development")
	}

	email, username, err := NormalizeAndValidateRegistration(email, username, password)
	if err != nil {
		return fmt.Errorf("validate development %s: %w", role, err)
	}
	passwordHash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash development %s password: %w", role, err)
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin development %s seed: %w", role, err)
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
		return fmt.Errorf("find development %s: %w", role, err)
	}

	switch len(matches) {
	case 0:
		if role == "admin" {
			_, err = tx.ExecContext(ctx, `
				insert into users (email, username, password_hash, role)
				values ($1, $2, $3, 'admin')
			`, email, username, passwordHash)
		} else {
			_, err = tx.ExecContext(ctx, `
				insert into users (email, username, password_hash)
				values ($1, $2, $3)
			`, email, username, passwordHash)
		}
	case 1:
		existing := matches[0]
		if !strings.EqualFold(existing.Email, email) || !strings.EqualFold(existing.Username, username) {
			return fmt.Errorf("development %s email or username conflicts with another account", role)
		}
		if refreshRole {
			_, err = tx.ExecContext(ctx, `
				update users
				set password_hash = $2, role = 'admin', updated_at = now()
				where id = $1
			`, existing.ID, passwordHash)
		} else {
			_, err = tx.ExecContext(ctx, `
				update users
				set password_hash = $2, updated_at = now()
				where id = $1
			`, existing.ID, passwordHash)
		}
	default:
		return fmt.Errorf("development %s email and username conflict with separate accounts", role)
	}
	if err != nil {
		return fmt.Errorf("write development %s: %w", role, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit development %s seed: %w", role, err)
	}
	return nil
}
