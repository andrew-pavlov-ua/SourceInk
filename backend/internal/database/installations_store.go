package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"sourceink/backend/internal/model"
)

func (s *Store) InstallationForUser(ctx context.Context, userID string) (model.GitHubInstallation, error) {
	var installation model.GitHubInstallation
	err := s.db.GetContext(ctx, &installation, `
		select
			installation_id,
			github_account_id,
			github_account_login,
			github_account_type,
			repository_selection,
			suspended
		from github_installations
		where user_id = $1
		order by created_at desc
		limit 1
	`, userID)
	return installation, err
}

func (s *Store) SaveInstallation(
	ctx context.Context,
	userID string,
	installation model.GitHubInstallation,
) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin save github installation transaction: %w", err)
	}
	defer tx.Rollback()

	var savedInstallationID int64
	err = tx.GetContext(ctx, &savedInstallationID, `
		insert into github_installations (
			user_id,
			installation_id,
			github_account_id,
			github_account_login,
			github_account_type,
			repository_selection,
			suspended
		)
		values ($1, $2, $3, $4, $5, $6, $7)
		on conflict (installation_id) do update set
			github_account_id = excluded.github_account_id,
			github_account_login = excluded.github_account_login,
			github_account_type = excluded.github_account_type,
			repository_selection = excluded.repository_selection,
			suspended = excluded.suspended,
			updated_at = now()
		where github_installations.user_id = excluded.user_id
		returning installation_id
	`, userID, installation.ID, installation.AccountID, installation.AccountLogin,
		installation.AccountType, installation.RepositorySelection, installation.Suspended)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ErrGitHubInstallationConflict
		}
		return fmt.Errorf("save github installation: %w", err)
	}

	if err := syncInstallationRepositories(ctx, tx, savedInstallationID, installation.Repositories); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit GitHub installation transaction: %w", err)
	}
	return nil
}
