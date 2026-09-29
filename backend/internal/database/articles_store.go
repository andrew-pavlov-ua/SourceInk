package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"sourceink/backend/internal/model"
)

const (
	draftColumns = `
		ua.id, ua.repository_id, ua.source_path, ua.git_blob_sha,
		ua.title, ua.slug, ua.description, ua.tags, ua.publish_mode,
		ua.markdown, ua.validation_error, ua.present, ua.discovered_at, ua.updated_at`

	draftFrom = `
		from unpublished_articles as ua
		join repositories as r on r.id = ua.repository_id
		join github_installations as gi on gi.installation_id = r.installation_id`

	articleColumns = `
		a.id, a.unpublished_article_id, a.owner_id, u.username as author_username,
		a.repository_id, a.source_path, a.slug, a.publish_mode, a.source_state,
		a.git_blob_sha, a.markdown, a.title, a.frontmatter, a.published_at,
		a.created_at, a.updated_at`

	articleReturningColumns = `
		id, unpublished_article_id, owner_id, repository_id, source_path, slug,
		publish_mode, source_state, git_blob_sha, markdown, title, frontmatter,
		published_at, created_at, updated_at`

	articleFrom = `
		from articles as a
		join users as u on u.id = a.owner_id`

	upsertDraftSQL = `
		insert into unpublished_articles (
			repository_id, source_path, git_blob_sha, markdown, title, slug,
			description, tags, publish_mode, validation_error, present
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, true)
		on conflict (repository_id, source_path) do update set
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
		returning id, discovered_at, updated_at`

	publishDraftSQL = `
		insert into articles (
			unpublished_article_id, owner_id, repository_id, source_path, slug,
			publish_mode, source_state, git_blob_sha, markdown, title, frontmatter,
			published_at
		)
		select
			ua.id, gi.user_id, ua.repository_id, ua.source_path, ua.slug,
			ua.publish_mode, 'available', ua.git_blob_sha, ua.markdown, ua.title,
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
			and ua.git_blob_sha = $2
			and ($3::uuid is null or gi.user_id = $3)
			and ua.present = true
			and ua.validation_error is null
			and ua.title is not null and ua.title <> ''
			and ua.slug is not null and ua.slug <> ''
			and ua.publish_mode = $4
			and not exists (
				select 1
				from unpublished_articles as other
				join repositories as other_repository on other_repository.id = other.repository_id
				join github_installations as other_installation on other_installation.installation_id = other_repository.installation_id
				where other.id <> ua.id
					and other.present = true
					and other.slug = ua.slug
					and other_installation.user_id = gi.user_id
			)
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
		returning ` + articleReturningColumns
)

const duplicateSlugErrorPrefix = "duplicate slug: "

func (s *Store) UpsertUnpublishedArticle(ctx context.Context, article *model.UnpublishedArticle) error {
	err := s.db.QueryRowContext(ctx, upsertDraftSQL,
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
	).Scan(&article.ID, &article.DiscoveredAt, &article.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert unpublished article %s: %w", article.SourcePath, err)
	}

	article.Present = true
	return nil
}

// ReconcileDuplicateSlugs marks valid drafts that share an owner's slug.
func (s *Store) ReconcileDuplicateSlugs(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		with duplicate_slugs as (
			select gi.user_id, ua.slug
			`+draftFrom+`
			where ua.present = true
				and ua.validation_error is null
				and nullif(btrim(ua.slug), '') is not null
			group by gi.user_id, ua.slug
			having count(*) > 1
		)
		update unpublished_articles as ua
		set validation_error = case
			when exists (
				select 1 from duplicate_slugs ds
				where ds.user_id = gi.user_id and ds.slug = ua.slug
			) then $1 || quote_literal(ua.slug) || ' is used by multiple source files; choose a unique slug.'
			when ua.validation_error like $1 || '%' then null
			else ua.validation_error
		end,
		updated_at = now()
		from repositories as r
		join github_installations as gi on gi.installation_id = r.installation_id
		where ua.repository_id = r.id
			and (
				exists (
					select 1 from duplicate_slugs ds
					where ds.user_id = gi.user_id and ds.slug = ua.slug
				)
				or ua.validation_error like $1 || '%'
			)
	`, duplicateSlugErrorPrefix)
	if err != nil {
		return fmt.Errorf("reconcile duplicate article slugs: %w", err)
	}
	return nil
}

func (s *Store) ListUserArticles(ctx context.Context, userID string) ([]model.Article, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}
	return s.listArticles(ctx, "where a.owner_id = $1\norder by a.published_at desc, a.id", userID)
}

func (s *Store) ListPublishedArticles(ctx context.Context) ([]model.Article, error) {
	return s.listArticles(ctx, "order by a.published_at desc, a.id")
}

func (s *Store) listArticles(ctx context.Context, filter string, args ...any) ([]model.Article, error) {
	articles := make([]model.Article, 0)
	if err := s.db.SelectContext(ctx, &articles, "select "+articleColumns+articleFrom+"\n"+filter, args...); err != nil {
		return nil, fmt.Errorf("list articles: %w", err)
	}
	return articles, nil
}

func (s *Store) ListUserUnpublishedArticles(ctx context.Context, userID string) ([]model.UnpublishedArticle, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	articles := make([]model.UnpublishedArticle, 0)
	err := s.db.SelectContext(ctx, &articles, "select "+draftColumns+draftFrom+`
		where gi.user_id = $1
		order by ua.updated_at desc, ua.id`, userID)
	if err != nil {
		return nil, fmt.Errorf("list unpublished articles for user %s: %w", userID, err)
	}
	return articles, nil
}

func (s *Store) FindUnpublishedArticleForPublish(ctx context.Context, userID, draftID string) (model.UnpublishedArticle, error) {
	if userID == "" || draftID == "" {
		return model.UnpublishedArticle{}, model.ErrArticleNotFound
	}
	return s.findDraft(ctx, draftID, userID, "", "find")
}

func (s *Store) AutoFindUnpublishedArticleForPublish(ctx context.Context, draftID string) (model.UnpublishedArticle, error) {
	if draftID == "" {
		return model.UnpublishedArticle{}, model.ErrArticleNotFound
	}
	return s.findDraft(ctx, draftID, "", string(model.ArticlePublishModeAuto), "auto-find")
}

func (s *Store) findDraft(ctx context.Context, draftID, userID, publishMode, operation string) (model.UnpublishedArticle, error) {
	query := "select " + draftColumns + draftFrom + "\nwhere ua.id = $1"
	args := []any{draftID}
	if userID != "" {
		query += fmt.Sprintf(" and gi.user_id = $%d", len(args)+1)
		args = append(args, userID)
	}
	if publishMode != "" {
		query += fmt.Sprintf(" and ua.publish_mode = $%d", len(args)+1)
		args = append(args, publishMode)
	}

	var draft model.UnpublishedArticle
	err := s.db.GetContext(ctx, &draft, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return model.UnpublishedArticle{}, fmt.Errorf("%s article draft %s: %w", operation, draftID, model.ErrArticleNotFound)
	}
	if err != nil {
		return model.UnpublishedArticle{}, fmt.Errorf("%s article draft %s: %w", operation, draftID, err)
	}
	return draft, nil
}

func (s *Store) PublishArticle(ctx context.Context, userID, draftID, gitBlobSHA string) (model.Article, error) {
	if userID == "" || draftID == "" || gitBlobSHA == "" {
		return model.Article{}, model.ErrArticleNotPublishable
	}
	return s.publishDraft(ctx, "publish", draftID, gitBlobSHA, userID, model.ArticlePublishModeManual)
}

func (s *Store) AutoPublishArticle(ctx context.Context, draftID, gitBlobSHA string) (model.Article, error) {
	if draftID == "" || gitBlobSHA == "" {
		return model.Article{}, model.ErrArticleNotPublishable
	}
	return s.publishDraft(ctx, "auto-publish", draftID, gitBlobSHA, nil, model.ArticlePublishModeAuto)
}

func (s *Store) publishDraft(
	ctx context.Context,
	operation, draftID, gitBlobSHA string,
	userID any,
	publishMode model.ArticlePublishMode,
) (model.Article, error) {
	var article model.Article
	err := s.db.GetContext(ctx, &article, publishDraftSQL, draftID, gitBlobSHA, userID, publishMode)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Article{}, fmt.Errorf("%s article draft %s: %w", operation, draftID, model.ErrArticleNotPublishable)
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.Article{}, fmt.Errorf("%s article draft %s conflicts with an existing publication: %w", operation, draftID, model.ErrArticleNotPublishable)
		}
		return model.Article{}, fmt.Errorf("%s article %s: %w", operation, draftID, err)
	}
	return article, nil
}
