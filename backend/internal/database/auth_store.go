package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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
			where lower(email) = lower($1) and password_hash is not null
		`,
		email,
	)
	if err != nil {
		return model.User{}, "", err
	}

	return result.User, result.PasswordHash, nil
}

// LinkGitHubUser attaches a verified GitHub identity to an existing account.
// When that identity belongs to a passwordless account, move its installations
// and published articles into this account before removing the duplicate.
func (s *Store) LinkGitHubUser(
	ctx context.Context,
	userID string,
	githubUser model.GitHubUser,
) (model.User, error) {
	if userID == "" || githubUser.ID <= 0 || githubUser.Login == "" {
		return model.User{}, errors.New("invalid GitHub account link")
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return model.User{}, fmt.Errorf("begin GitHub account link transaction: %w", err)
	}
	defer tx.Rollback()

	var user model.User
	err = tx.GetContext(ctx, &user, `
		select id, coalesce(email, '') as email, username, github_user_id,
			coalesce(github_login, '') as github_login,
			coalesce(github_avatar_url, '') as github_avatar_url, role, created_at
		from users
		where id = $1
		for update
	`, userID)
	if err != nil {
		return model.User{}, fmt.Errorf("find account for GitHub link: %w", err)
	}
	if user.GitHubUserID != nil && *user.GitHubUserID != githubUser.ID {
		return model.User{}, model.ErrGitHubAccountConflict
	}

	var githubOnlyUser struct {
		ID          string `db:"id"`
		HasEmail    bool   `db:"has_email"`
		HasPassword bool   `db:"has_password"`
	}
	err = tx.GetContext(ctx, &githubOnlyUser, `
		select id, email is not null as has_email, password_hash is not null as has_password
		from users
		where github_user_id = $1 and id <> $2
		for update
	`, githubUser.ID, userID)
	if err == nil {
		if githubOnlyUser.HasEmail || githubOnlyUser.HasPassword {
			return model.User{}, model.ErrGitHubAccountConflict
		}
		if _, err := tx.ExecContext(ctx, `
			update github_installations set user_id = $1, updated_at = now() where user_id = $2
		`, userID, githubOnlyUser.ID); err != nil {
			return model.User{}, fmt.Errorf("move GitHub installations to linked account: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			update articles set owner_id = $1, updated_at = now() where owner_id = $2
		`, userID, githubOnlyUser.ID); err != nil {
			return model.User{}, fmt.Errorf("move articles to linked account: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `delete from users where id = $1`, githubOnlyUser.ID); err != nil {
			return model.User{}, fmt.Errorf("remove merged GitHub-only account: %w", err)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return model.User{}, fmt.Errorf("find existing GitHub account: %w", err)
	}

	err = tx.GetContext(ctx, &user, `
		update users
		set github_user_id = $2,
			github_login = case
				when not exists (
					select 1 from users other
					where lower(other.github_login) = lower($3) and other.id <> $1
				) then $3
				else null
			end,
			github_avatar_url = $4,
			updated_at = now()
		where id = $1
		returning id, coalesce(email, '') as email, username, github_user_id,
			coalesce(github_login, '') as github_login,
			coalesce(github_avatar_url, '') as github_avatar_url, role, created_at
	`, userID, githubUser.ID, githubUser.Login, githubUser.AvatarURL)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.User{}, model.ErrGitHubAccountConflict
		}
		return model.User{}, fmt.Errorf("link GitHub account: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return model.User{}, fmt.Errorf("commit GitHub account link: %w", err)
	}
	return user, nil
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

func (s *Store) SetEmailPasswordForGitHubUser(
	ctx context.Context,
	userID, email, passwordHash string,
) (model.User, error) {
	if userID == "" || email == "" || passwordHash == "" {
		return model.User{}, errors.New("invalid email sign-in connection")
	}

	var user model.User
	err := s.db.GetContext(ctx, &user, `
		update users
		set email = $2, password_hash = $3, updated_at = now()
		where id = $1
			and github_user_id is not null
			and email is null
			and password_hash is null
		returning id, email, username, github_user_id,
			coalesce(github_login, '') as github_login,
			coalesce(github_avatar_url, '') as github_avatar_url, role, created_at
	`, userID, email, passwordHash)
	if err == nil {
		return user, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return model.User{}, model.ErrEmailTaken
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return model.User{}, fmt.Errorf("connect email sign-in: %w", err)
	}

	var state struct {
		HasGitHub   bool `db:"has_github"`
		HasEmail    bool `db:"has_email"`
		HasPassword bool `db:"has_password"`
	}
	if err := s.db.GetContext(ctx, &state, `
		select github_user_id is not null as has_github,
			email is not null as has_email,
			password_hash is not null as has_password
		from users
		where id = $1
	`, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, model.ErrEmailSignInNotAllowed
		}
		return model.User{}, fmt.Errorf("check email sign-in state: %w", err)
	}
	if state.HasEmail || state.HasPassword {
		return model.User{}, model.ErrEmailSignInAlreadySet
	}
	return model.User{}, model.ErrEmailSignInNotAllowed
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
	// Keep the GitHub login unless it exceeds SourceInk's 32-character handle limit.
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
