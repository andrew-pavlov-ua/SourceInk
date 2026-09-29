package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

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
			u.username as author_username,
			a.repository_id,
			a.source_path,
			a.slug,
			a.publish_mode,
			a.source_state,
			a.git_blob_sha,
			a.markdown,
			a.title,
			a.frontmatter,
			a.published_at,
			a.created_at,
			a.updated_at
		from articles as a
		join users as u on u.id = a.owner_id
		where a.owner_id = $1
		order by a.published_at desc, a.id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list articles for user %s: %w", userID, err)
	}

	return articles, nil
}

func (s *Store) ListPublishedArticles(ctx context.Context) ([]model.Article, error) {
	articles := make([]model.Article, 0)
	err := s.db.SelectContext(ctx, &articles, `
		select
			a.id,
			a.unpublished_article_id,
			a.owner_id,
			u.username as author_username,
			a.repository_id,
			a.source_path,
			a.slug,
			a.publish_mode,
			a.source_state,
			a.git_blob_sha,
			a.markdown,
			a.title,
			a.frontmatter,
			a.published_at,
			a.created_at,
			a.updated_at
		from articles as a
		join users as u on u.id = a.owner_id
		order by a.published_at desc, a.id
	`)
	if err != nil {
		return nil, fmt.Errorf("list published articles: %w", err)
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

func (s *Store) FindUnpublishedArticleForPublish(ctx context.Context, userID, draftID string) (model.UnpublishedArticle, error) {
	if userID == "" || draftID == "" {
		return model.UnpublishedArticle{}, model.ErrArticleNotFound
	}

	var draft model.UnpublishedArticle
	err := s.db.GetContext(ctx, &draft, `
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
		where ua.id = $1 and gi.user_id = $2
	`, draftID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.UnpublishedArticle{}, fmt.Errorf("find article draft %s: %w", draftID, model.ErrArticleNotFound)
	}
	if err != nil {
		return model.UnpublishedArticle{}, fmt.Errorf("find article draft %s for user %s: %w", draftID, userID, err)
	}

	return draft, nil
}

func (s *Store) PublishArticle(
	ctx context.Context,
	userID, draftID, gitBlobSHA string,
) (model.Article, error) {
	if userID == "" || draftID == "" || gitBlobSHA == "" {
		return model.Article{}, model.ErrArticleNotPublishable
	}

	var article model.Article
	err := s.db.GetContext(ctx, &article, `
		insert into articles (
			unpublished_article_id,
			owner_id,
			repository_id,
			source_path,
			slug,
			publish_mode,
			source_state,
			git_blob_sha,
			markdown,
			title,
			frontmatter,
			published_at
		)
		select
			ua.id,
			gi.user_id,
			ua.repository_id,
			ua.source_path,
			ua.slug,
			ua.publish_mode,
			'available',
			ua.git_blob_sha,
			ua.markdown,
			ua.title,
			jsonb_build_object(
				'title', ua.title,
				'slug', ua.slug,
				'description', coalesce(ua.description, ''),
				'tags', coalesce(to_jsonb(ua.tags), '[]'::jsonb),
				'publish_mode', ua.publish_mode
			),
			now()
		from unpublished_articles as ua
		join repositories as r on r.id = ua.repository_id
		join github_installations as gi on gi.installation_id = r.installation_id
		where ua.id = $1
			and gi.user_id = $2
			and ua.git_blob_sha = $3
			and ua.present = true
			and ua.validation_error is null
			and ua.title is not null and ua.title <> ''
			and ua.slug is not null and ua.slug <> ''
			and ua.publish_mode in ('manual', 'automatic')
			and not exists (
				select 1
				from articles as existing
				where existing.slug = ua.slug
					and existing.unpublished_article_id <> ua.id
			)
		on conflict (unpublished_article_id) do update set
			owner_id = excluded.owner_id,
			repository_id = excluded.repository_id,
			source_path = excluded.source_path,
			slug = excluded.slug,
			publish_mode = excluded.publish_mode,
			source_state = excluded.source_state,
			git_blob_sha = excluded.git_blob_sha,
			markdown = excluded.markdown,
			title = excluded.title,
			frontmatter = excluded.frontmatter,
			published_at = excluded.published_at,
			updated_at = now()
		returning
			id,
			unpublished_article_id,
			owner_id,
			repository_id,
			source_path,
			slug,
			publish_mode,
			source_state,
			git_blob_sha,
			markdown,
			title,
			frontmatter,
			published_at,
			created_at,
			updated_at
	`, draftID, userID, gitBlobSHA)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Article{}, fmt.Errorf("publish article draft %s: %w", draftID, model.ErrArticleNotPublishable)
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.Article{}, fmt.Errorf("publish article draft %s conflicts with an existing publication: %w", draftID, model.ErrArticleNotPublishable)
		}
		return model.Article{}, fmt.Errorf("publish article %s for user %s: %w", draftID, userID, err)
	}

	return article, nil
}
