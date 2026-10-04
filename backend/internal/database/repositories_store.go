package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"sourceink/backend/internal/model"
)

func saveRepositories(
	ctx context.Context,
	tx *sqlx.Tx,
	installationID int64,
	repositories []model.Repository,
) error {
	for _, repository := range repositories {
		if repository.GitHubID <= 0 || repository.Owner == "" || repository.Name == "" ||
			repository.FullName == "" || repository.DefaultBranch == "" {
			return errors.New("GitHub returned invalid repository metadata")
		}

		_, err := tx.ExecContext(ctx, `
			insert into repositories (
				github_id,
				installation_id,
				owner,
				name,
				full_name,
				default_branch,
				private,
				archived
			)
			values ($1, $2, $3, $4, $5, $6, $7, $8)
			on conflict (github_id) do update set
				installation_id = excluded.installation_id,
				owner = excluded.owner,
				name = excluded.name,
				full_name = excluded.full_name,
				default_branch = excluded.default_branch,
				private = excluded.private,
				archived = excluded.archived,
				updated_at = now()
		`, repository.GitHubID, installationID, repository.Owner, repository.Name,
			repository.FullName, repository.DefaultBranch, repository.Private, repository.Archived)
		if err != nil {
			return fmt.Errorf("save GitHub repository %d: %w", repository.GitHubID, err)
		}
	}

	return nil
}

func syncInstallationRepositories(
	ctx context.Context,
	tx *sqlx.Tx,
	installationID int64,
	repositories []model.Repository,
) error {
	selectedRepositoryIDs := make(map[int64]struct{}, len(repositories))
	for _, repository := range repositories {
		selectedRepositoryIDs[repository.GitHubID] = struct{}{}
	}

	var savedRepositoryIDs []int64
	if err := tx.SelectContext(ctx, &savedRepositoryIDs, `
		select github_id
		from repositories
		where installation_id = $1
	`, installationID); err != nil {
		return fmt.Errorf("list stored GitHub repositories: %w", err)
	}

	for _, repositoryID := range savedRepositoryIDs {
		if _, isStillSelected := selectedRepositoryIDs[repositoryID]; isStillSelected {
			continue
		}

		if _, err := tx.ExecContext(ctx, `
			update articles as article
			set source_state = 'access_lost', updated_at = now()
			from repositories as repository
			where article.repository_id = repository.id
				and repository.installation_id = $1
				and repository.github_id = $2
		`, installationID, repositoryID); err != nil {
			return fmt.Errorf("mark articles from unselected GitHub repository %d as access lost: %w", repositoryID, err)
		}

		if _, err := tx.ExecContext(ctx, `
			delete from repositories
			where installation_id = $1 and github_id = $2
		`, installationID, repositoryID); err != nil {
			return fmt.Errorf("delete unselected GitHub repository %d: %w", repositoryID, err)
		}
	}

	return saveRepositories(ctx, tx, installationID, repositories)
}

func (s *Store) ListRepositoriesForUser(
	ctx context.Context,
	userID string,
) ([]model.Repository, error) {
	repositories := make([]model.Repository, 0)

	err := s.db.SelectContext(ctx, &repositories, `
		select
			r.id,
			r.installation_id,
			r.github_id,
			r.owner,
			r.name,
			r.full_name,
			r.default_branch,
			r.private,
			r.archived
		from repositories r
		join github_installations gi
			on gi.installation_id = r.installation_id
		where gi.user_id = $1
		order by lower(r.full_name)
	`, userID)

	return repositories, err
}

func (s *Store) ListRepositoriesForArticleSync(ctx context.Context) ([]model.Repository, error) {
	repositories := make([]model.Repository, 0)

	err := s.db.SelectContext(ctx, &repositories, `
		select
			r.id,
			r.installation_id,
			r.github_id,
			r.owner,
			r.name,
			r.full_name,
			r.default_branch,
			r.private,
			r.archived
		from repositories r
		join github_installations gi
			on gi.installation_id = r.installation_id
		where not gi.suspended and not r.archived
		order by gi.installation_id, lower(r.full_name)
	`)

	return repositories, err
}

func (s *Store) RepositoryForWebhookSync(ctx context.Context, installationID, githubRepositoryID int64) (model.Repository, error) {
	var repository model.Repository
	err := s.db.GetContext(ctx, &repository, `
		select r.id, r.installation_id, r.github_id, r.owner, r.name, r.full_name,
			r.default_branch, r.private, r.archived
		from repositories r
		join github_installations gi on gi.installation_id = r.installation_id
		where r.installation_id = $1
			and r.github_id = $2
			and not gi.suspended
			and not r.archived
	`, installationID, githubRepositoryID)
	return repository, err
}
