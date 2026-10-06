package database_test

import (
	"context"
	"os"
	"testing"

	"sourceink/backend/internal/database"
	"sourceink/backend/internal/model"
)

func TestIncrementArticleViewCount(t *testing.T) {
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
		ownerID   = "00000000-0000-4000-8000-000000000011"
		articleID = "00000000-0000-4000-8000-000000000012"
	)
	_, err = db.ExecContext(ctx, `
		insert into users (id, username) values ($1, 'view-owner');
		insert into articles (
			id, owner_id, source_path, slug, publish_mode, source_state,
			git_blob_sha, markdown, title
		) values ($2, $1, 'articles/views.md', 'views-store-test', 'manual',
			'available', 'views-blob-sha', '# Views', 'Views store test');
	`, ownerID, articleID)
	if err != nil {
		t.Fatalf("seed article: %v", err)
	}

	store := database.NewStore(db)
	if err := store.IncrementArticleViewCount(ctx, articleID); err != nil {
		t.Fatalf("IncrementArticleViewCount(): %v", err)
	}

	article, err := store.PublishedArticleBySlug(ctx, "views-store-test", "")
	if err != nil {
		t.Fatalf("PublishedArticleBySlug(): %v", err)
	}
	if article.ViewCount != 1 {
		t.Fatalf("view count = %d, want 1", article.ViewCount)
	}
	if err := store.IncrementArticleViewCount(ctx, "00000000-0000-4000-8000-000000000099"); err != model.ErrPublishedArticleNotFound {
		t.Fatalf("missing article error = %v, want ErrPublishedArticleNotFound", err)
	}
}
