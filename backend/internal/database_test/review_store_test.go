package database_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"sourceink/backend/internal/database"
	"sourceink/backend/internal/model"
)

func TestReviewStoreReplaceAndDelete(t *testing.T) {
	databaseURL := os.Getenv("SOURCEINK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SOURCEINK_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	db, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.RunMigrations(ctx, db, "up"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	const (
		ownerID    = "00000000-0000-4000-8000-000000000001"
		reviewerID = "00000000-0000-4000-8000-000000000002"
		otherID    = "00000000-0000-4000-8000-000000000003"
		articleID  = "00000000-0000-4000-8000-000000000004"
		gitBlobSHA = "current-blob-sha"
	)

	_, err = db.ExecContext(ctx, `
		insert into users (id, username)
		values
			($1, 'review-owner'),
			($2, 'review-reader'),
			($3, 'other-reader')
	`, ownerID, reviewerID, otherID)
	if err != nil {
		t.Fatalf("seed users: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		insert into articles (
			id, owner_id, source_path, slug, publish_mode, source_state,
			git_blob_sha, markdown, title
		) values ($1, $2, 'articles/review.md', 'review-store-test', 'manual',
			'available', $3, '# Review', 'Review store test')
	`, articleID, ownerID, gitBlobSHA)
	if err != nil {
		t.Fatalf("seed article: %v", err)
	}

	store := database.NewStore(db)
	review := model.Review{
		ArticleID:  articleID,
		GitBlobSHA: gitBlobSHA,
		ReviewerID: reviewerID,
		Verdict:    model.VerdictApprove,
	}
	if err := store.CreateReview(ctx, &review); err != nil {
		t.Fatalf("CreateReview(approve): %v", err)
	}
	firstID := review.ID
	if firstID == "" || review.CreatedAt.IsZero() || review.UpdatedAt.IsZero() {
		t.Fatalf("created review metadata = %#v", review)
	}

	reason := model.ReasonOutdated
	review.Verdict = model.VerdictRequestChanges
	review.Reason = &reason
	if err := store.CreateReview(ctx, &review); err != nil {
		t.Fatalf("CreateReview(request changes): %v", err)
	}
	if review.ID == firstID {
		t.Fatalf("replacement review ID = %q, want a newly created review", review.ID)
	}

	reviews, err := store.ListReviewsForArticle(ctx, articleID, gitBlobSHA)
	if err != nil {
		t.Fatalf("ListReviewsForArticle(): %v", err)
	}
	if len(reviews) != 1 {
		t.Fatalf("reviews count = %d, want 1", len(reviews))
	}
	if reviews[0].Verdict != model.VerdictRequestChanges || reviews[0].Reason == nil || *reviews[0].Reason != reason {
		t.Fatalf("replacement review = %#v", reviews[0])
	}

	published, err := store.ListPublishedArticles(ctx, reviewerID)
	if err != nil {
		t.Fatalf("ListPublishedArticles(): %v", err)
	}
	if len(published) != 1 {
		t.Fatalf("published articles count = %d, want 1", len(published))
	}
	summary := published[0].Reviews
	if summary.RequestChangesCount != 1 || summary.RequestReasons.Outdated != 1 || summary.ApproveCount != 0 {
		t.Fatalf("published article review summary = %#v", summary)
	}
	if !summary.Authenticated || summary.ViewerReview == nil || summary.ViewerReview.Reason == nil || *summary.ViewerReview.Reason != reason {
		t.Fatalf("published article viewer review = %#v", summary.ViewerReview)
	}

	bySlug, err := store.PublishedArticleBySlug(ctx, "review-store-test", reviewerID)
	if err != nil {
		t.Fatalf("PublishedArticleBySlug(): %v", err)
	}
	if bySlug.ID != articleID || bySlug.Reviews.ViewerReview == nil || bySlug.Reviews.RequestChangesCount != 1 {
		t.Fatalf("PublishedArticleBySlug() = %#v", bySlug)
	}
	if _, err := store.PublishedArticleBySlug(ctx, "missing-article", reviewerID); !errors.Is(err, model.ErrPublishedArticleNotFound) {
		t.Fatalf("PublishedArticleBySlug(missing) error = %v", err)
	}

	staleReview := review
	staleReview.GitBlobSHA = "stale-blob-sha"
	if err := store.CreateReview(ctx, &staleReview); !errors.Is(err, model.ErrArticleReviewStale) {
		t.Fatalf("CreateReview(stale) error = %v, want ErrArticleReviewStale", err)
	}

	if err := store.DeleteReview(ctx, articleID, gitBlobSHA, otherID); !errors.Is(err, model.ErrArticleReviewNotFound) {
		t.Fatalf("DeleteReview(other reader) error = %v, want ErrArticleReviewNotFound", err)
	}
	if err := store.DeleteReview(ctx, articleID, gitBlobSHA, reviewerID); err != nil {
		t.Fatalf("DeleteReview(reviewer): %v", err)
	}

	reviews, err = store.ListReviewsForArticle(ctx, articleID, gitBlobSHA)
	if err != nil {
		t.Fatalf("ListReviewsForArticle(after delete): %v", err)
	}
	if len(reviews) != 0 {
		t.Fatalf("reviews after delete = %#v, want none", reviews)
	}
}
