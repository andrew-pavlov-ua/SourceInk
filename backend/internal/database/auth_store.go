package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"sourceink/backend/internal/model"
)

func (s *Store) CreateUserAndSession(
	ctx context.Context,
	email, username, passwordHash string,
	tokenHash []byte,
	expiresAt time.Time,
) (model.User, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return model.User{}, fmt.Errorf("begin registration transaction: %w", err)
	}
	defer tx.Rollback()

	var user model.User

	err = tx.GetContext(
		ctx,
		&user,
		`
			insert into users (email, username, password_hash)
			values ($1, $2, $3)
			on conflict do nothing
			returning id, email, username, role, created_at
		`,
		email,
		username,
		passwordHash,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, registrationConflict(ctx, tx, email, username)
		}

		return model.User{}, fmt.Errorf("create user: %w", err)
	}

	_, err = tx.ExecContext(
		ctx,
		`
			insert into sessions (user_id, token_hash, expires_at)
			values ($1, $2, $3)
		`,
		user.ID,
		tokenHash,
		expiresAt,
	)
	if err != nil {
		return model.User{}, fmt.Errorf("create session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return model.User{}, fmt.Errorf("commit registration transaction: %w", err)
	}

	return user, nil
}

func registrationConflict(
	ctx context.Context,
	tx *sqlx.Tx,
	email, username string,
) error {
	var conflict struct {
		EmailTaken    bool `db:"email_taken"`
		UsernameTaken bool `db:"username_taken"`
	}

	err := tx.GetContext(
		ctx,
		&conflict,
		`
			select
				exists(
					select 1
					from users
					where lower(email) = lower($1)
				) as email_taken,

				exists(
					select 1
					from users
					where lower(username) = lower($2)
				) as username_taken
		`,
		email,
		username,
	)
	if err != nil {
		return fmt.Errorf("check registration conflict: %w", err)
	}

	switch {
	case conflict.EmailTaken:
		return model.ErrEmailTaken

	case conflict.UsernameTaken:
		return model.ErrUsernameTaken

	default:
		return errors.New("registration conflict could not be identified")
	}
}

func (s *Store) FindUserByEmail(
	ctx context.Context,
	email string,
) (model.User, string, error) {
	var result struct {
		model.User
		PasswordHash string `db:"password_hash"`
	}

	err := s.db.GetContext(
		ctx,
		&result,
		`
			select
				id,
				email,
				username,
				github_user_id,
				coalesce(github_login, '') as github_login,
				coalesce(github_avatar_url, '') as github_avatar_url,
				role,
				created_at,
				password_hash
			from users
			where lower(email) = lower($1) and github_user_id is null
		`,
		email,
	)
	if err != nil {
		return model.User{}, "", err
	}

	return result.User, result.PasswordHash, nil
}

func (s *Store) FindOrCreateUserByGitHub(
	ctx context.Context,
	githubUser model.GitHubUser,
	tokenHash []byte,
	expiresAt time.Time,
) (model.User, error) {
	if githubUser.ID <= 0 || githubUser.Login == "" {
		return model.User{}, errors.New("invalid GitHub user")
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return model.User{}, fmt.Errorf("begin GitHub login transaction: %w", err)
	}
	defer tx.Rollback()

	var user model.User
	err = tx.GetContext(ctx, &user, `
		select u.id, coalesce(u.email, '') as email, u.username, u.github_user_id,
			coalesce(u.github_login, '') as github_login,
			coalesce(u.github_avatar_url, '') as github_avatar_url, u.role, u.created_at
		from github_installations gi
		join users u on u.id = gi.user_id
		where gi.github_account_type = 'User' and gi.github_account_id = $1
		order by gi.created_at
		limit 1
		for update of u
	`, githubUser.ID)
	if err == nil {
		if user.GitHubUserID != nil && *user.GitHubUserID != githubUser.ID {
			return model.User{}, model.ErrGitHubUsernameTaken
		}
		user, err = refreshGitHubIdentity(ctx, tx, user, githubUser)
		if err != nil {
			return model.User{}, err
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return model.User{}, fmt.Errorf("find user from personal GitHub installation: %w", err)
	} else {
		err = tx.GetContext(ctx, &user, `
			select id, coalesce(email, '') as email, username, github_user_id, coalesce(github_login, '') as github_login,
				coalesce(github_avatar_url, '') as github_avatar_url, role, created_at
			from users
			where github_user_id = $1
			for update
		`, githubUser.ID)
		if errors.Is(err, sql.ErrNoRows) {
			err = tx.GetContext(ctx, &user, `
				insert into users (username, github_user_id, github_login, github_avatar_url)
				values ($1, $2, $3, $4)
				on conflict do nothing
				returning id, coalesce(email, '') as email, username, github_user_id, github_login, github_avatar_url, role, created_at
			`, githubUsername(githubUser.Login, githubUser.ID), githubUser.ID, githubUser.Login, githubUser.AvatarURL)
			if errors.Is(err, sql.ErrNoRows) {
				return model.User{}, model.ErrGitHubUsernameTaken
			}
			if err != nil {
				return model.User{}, fmt.Errorf("create GitHub user: %w", err)
			}
		} else if err != nil {
			return model.User{}, fmt.Errorf("find GitHub user: %w", err)
		} else {
			user, err = refreshGitHubIdentity(ctx, tx, user, githubUser)
			if err != nil {
				return model.User{}, err
			}
		}
	}

	_, err = tx.ExecContext(ctx, `insert into sessions (user_id, token_hash, expires_at) values ($1, $2, $3)`, user.ID, tokenHash, expiresAt)
	if err != nil {
		return model.User{}, fmt.Errorf("create GitHub login session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return model.User{}, fmt.Errorf("commit GitHub login transaction: %w", err)
	}
	return user, nil
}

func refreshGitHubIdentity(
	ctx context.Context,
	tx *sqlx.Tx,
	user model.User,
	githubUser model.GitHubUser,
) (model.User, error) {
	var refreshed model.User
	err := tx.GetContext(ctx, &refreshed, `
		update users
		set
			github_user_id = case
				when github_user_id is not null then github_user_id
				when not exists (
					select 1 from users other
					where other.github_user_id = $2 and other.id <> $1
				) then $2
				else null
			end,
			github_login = case
				when github_login is not null then github_login
				when not exists (
					select 1 from users other
					where other.github_login = $3 and other.id <> $1
				) then $3
				else null
			end,
			github_avatar_url = $4,
			updated_at = now()
		where id = $1
		returning id, coalesce(email, '') as email, username, github_user_id,
			coalesce(github_login, '') as github_login,
			coalesce(github_avatar_url, '') as github_avatar_url, role, created_at
	`, user.ID, githubUser.ID, githubUser.Login, githubUser.AvatarURL)
	if err != nil {
		return model.User{}, fmt.Errorf("refresh GitHub user: %w", err)
	}
	return refreshed, nil
}

func githubUsername(login string, id int64) string {
	// GitHub logins are valid SourceInk handles. The suffix is only needed if a
	// login exceeds the product's 32-character local handle limit.
	if len(login) <= 32 {
		return login
	}
	return fmt.Sprintf("github-%d", id)
}

func (s *Store) CreateSession(
	ctx context.Context,
	userID string,
	tokenHash []byte,
	expiresAt time.Time,
) error {
	_, err := s.db.ExecContext(
		ctx,
		`
			insert into sessions (user_id, token_hash, expires_at)
			values ($1, $2, $3)
		`,
		userID,
		tokenHash,
		expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

func (s *Store) UserBySession(
	ctx context.Context,
	tokenHash []byte,
) (model.User, error) {
	var user model.User

	err := s.db.GetContext(
		ctx,
		&user,
		`
			select
				u.id,
				coalesce(u.email, '') as email,
				u.username,
				u.github_user_id,
				coalesce(u.github_login, '') as github_login,
				coalesce(u.github_avatar_url, '') as github_avatar_url,
				u.role,
				u.created_at
			from sessions s
			join users u on u.id = s.user_id
			where
				s.token_hash = $1
				and s.expires_at > now()
		`,
		tokenHash,
	)

	return user, err
}

func (s *Store) UserIDBySession(
	ctx context.Context,
	tokenHash []byte,
) (int, error) {
	var userID int

	err := s.db.GetContext(
		ctx,
		&userID,
		`
			select user_id
			from sessions
			where
				token_hash = $1
				and expires_at > now()
		`,
		tokenHash,
	)

	return userID, err
}

func (s *Store) DeleteSession(
	ctx context.Context,
	tokenHash []byte,
) error {
	_, err := s.db.ExecContext(
		ctx,
		`
			delete from sessions
			where token_hash = $1
		`,
		tokenHash,
	)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

var _ model.AccountStore = (*Store)(nil)
