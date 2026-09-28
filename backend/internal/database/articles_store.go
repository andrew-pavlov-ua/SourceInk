package database

import (
	"context"
	"fmt"

	"sourceink/backend/internal/model"
)

func (s *Store) UpsertUnpublishedArticle(
	ctx context.Context,
	article *model.UnpublishedArticle,
) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin unpublished article upsert transaction: %w", err)
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx, `
        insert into unpublished_articles (
            repository_id,
            source_path,
            git_blob_sha,
            markdown,
            title,
            slug,
            description,
            tags,
            publish_mode,
            validation_error,
            present
        )
        values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, true)
        on conflict (repository_id, source_path)
        do update set
            git_blob_sha = excluded.git_blob_sha,
            markdown = excluded.markdown,
            title = excluded.title,
            slug = excluded.slug,
            description = excluded.description,
            tags = excluded.tags,
            publish_mode = excluded.publish_mode,
            validation_error = excluded.validation_error,
            present = true,
            updated_at = now()
        returning id, discovered_at, updated_at
    `,
		article.RepositoryID,
		article.SourcePath,
		article.GitBlobSHA,
		article.Content,
		article.Title,
		article.Slug,
		article.Description,
		article.Tags,
		article.PublishMode,
		article.ValidationError,
	).Scan(
		&article.ID,
		&article.DiscoveredAt,
		&article.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"upsert unpublished article %s: %w",
			article.SourcePath,
			err,
		)
	}

	article.Present = true
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit unpublished article upsert transaction: %w", err)
	}
	return nil
}

func (s *Store) ListUserArticles(ctx context.Context, userID string) ([]model.Article, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	articles := make([]model.Article, 0)
	err := s.db.SelectContext(ctx, &articles, `
		select
			a.id,
			a.unpublished_article_id,
			a.owner_id,
			a.repository_id,
			a.source_path,
			a.slug,
			a.publish_mode,
			a.source_state,
			a.git_blob_sha,
			a.markdown,
			a.title,
			a.frontmatter,
			a.rendered_html,
			a.published_at,
			a.created_at,
			a.updated_at
		from articles as a
		where a.owner_id = $1
		order by a.published_at desc, a.id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list articles for user %s: %w", userID, err)
	}

	return articles, nil
}

func (s *Store) ListUserUnpublishedArticles(ctx context.Context, userID string) ([]model.UnpublishedArticle, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	articles := make([]model.UnpublishedArticle, 0)
	err := s.db.SelectContext(ctx, &articles, `
		select
			ua.id,
			ua.repository_id,
			ua.source_path,
			ua.git_blob_sha,
			ua.title,
			ua.slug,
			ua.description,
			ua.tags,
			ua.publish_mode,
			ua.markdown,
			ua.validation_error,
			ua.present,
			ua.discovered_at,
			ua.updated_at
		from unpublished_articles as ua
		join repositories as r on r.id = ua.repository_id
		join github_installations as gi on gi.installation_id = r.installation_id
		where gi.user_id = $1
		order by ua.updated_at desc, ua.id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list unpublished articles for user %s: %w", userID, err)
	}

	return articles, nil
}
