package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"sourceink/backend/internal/model"
)

func (s *Store) ListReviewsForArticle(
	ctx context.Context,
	articleID string,
	gitBlobSHA string,
) ([]model.Review, error) {
	if err := s.requireCurrentArticleRevision(ctx, articleID, gitBlobSHA); err != nil {
		return nil, err
	}

	reviews := make([]model.Review, 0)
	if err := s.db.SelectContext(ctx, &reviews, `
		select
			id,
			article_id,
			git_blob_sha,
			reviewer_id,
			verdict,
			reason,
			created_at,
			updated_at
		from article_reviews
		where article_id = $1
			and git_blob_sha = $2
		order by created_at desc
	`, articleID, gitBlobSHA); err != nil {
		return nil, fmt.Errorf("list reviews for article %s at revision %s: %w", articleID, gitBlobSHA, err)
	}
	return reviews, nil
}

// CreateReview atomically replaces the review left by the same reader for the
// same published revision. A failed replacement preserves the previous review.
func (s *Store) CreateReview(ctx context.Context, review *model.Review) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin article review transaction: %w", err)
	}
	defer tx.Rollback()

	var currentGitBlobSHA string
	err = tx.GetContext(ctx, &currentGitBlobSHA, `
		select git_blob_sha
		from articles
		where id = $1
		for update
	`, review.ArticleID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.ErrPublishedArticleNotFound
	}
	if err != nil {
		return fmt.Errorf("load published article %s for review: %w", review.ArticleID, err)
	}
	if currentGitBlobSHA != review.GitBlobSHA {
		return model.ErrArticleReviewStale
	}

	if _, err := tx.ExecContext(ctx, `
		delete from article_reviews
		where article_id = $1
			and git_blob_sha = $2
			and reviewer_id = $3
	`, review.ArticleID, review.GitBlobSHA, review.ReviewerID); err != nil {
		return fmt.Errorf("remove previous review for article %s: %w", review.ArticleID, err)
	}

	var reason any
	if review.Reason != nil {
		reason = string(*review.Reason)
	}
	err = tx.QueryRowContext(ctx, `
		insert into article_reviews (
			article_id,
			git_blob_sha,
			reviewer_id,
			verdict,
			reason
		) values ($1, $2, $3, $4, $5)
		returning id, created_at, updated_at
	`, review.ArticleID, review.GitBlobSHA, review.ReviewerID, string(review.Verdict), reason).Scan(
		&review.ID,
		&review.CreatedAt,
		&review.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create review for article %s: %w", review.ArticleID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit article review transaction: %w", err)
	}
	return nil
}

func (s *Store) DeleteReview(
	ctx context.Context,
	articleID string,
	gitBlobSHA string,
	reviewerID string,
) error {
	if err := s.requireCurrentArticleRevision(ctx, articleID, gitBlobSHA); err != nil {
		return err
	}

	result, err := s.db.ExecContext(ctx, `
		delete from article_reviews
		where article_id = $1
			and git_blob_sha = $2
			and reviewer_id = $3
	`, articleID, gitBlobSHA, reviewerID)
	if err != nil {
		return fmt.Errorf("delete review for article %s: %w", articleID, err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted review count for article %s: %w", articleID, err)
	}
	if deleted == 0 {
		return model.ErrArticleReviewNotFound
	}
	return nil
}

func (s *Store) requireCurrentArticleRevision(ctx context.Context, articleID, gitBlobSHA string) error {
	var currentGitBlobSHA string
	err := s.db.GetContext(ctx, &currentGitBlobSHA, `
		select git_blob_sha
		from articles
		where id = $1
	`, articleID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.ErrPublishedArticleNotFound
	}
	if err != nil {
		return fmt.Errorf("load published article %s for review: %w", articleID, err)
	}
	if currentGitBlobSHA != gitBlobSHA {
		return model.ErrArticleReviewStale
	}
	return nil
}
