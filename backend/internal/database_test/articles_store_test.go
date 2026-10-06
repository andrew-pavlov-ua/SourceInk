package database_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"sourceink/backend/internal/database"
	"sourceink/backend/internal/model"
)

func TestUnchangedDraftAndPublicationKeepTheirTimestamps(t *testing.T) {
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

	suffix := time.Now().UnixNano()
	username := fmt.Sprintf("sync-%d", suffix)
	slug := fmt.Sprintf("sync-%d", suffix)
	installationID := suffix
	var userID, repositoryID string
	if err := db.GetContext(ctx, &userID, `insert into users (username) values ($1) returning id`, username); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `delete from articles where owner_id = $1`, userID)
		_, _ = db.ExecContext(context.Background(), `delete from users where id = $1`, userID)
	})
	if _, err := db.ExecContext(ctx, `
		insert into github_installations (
			user_id, installation_id, github_account_id, github_account_login,
			github_account_type, repository_selection
		) values ($1, $2, $2, $3, 'User', 'selected')
	`, userID, installationID, username); err != nil {
		t.Fatalf("seed installation: %v", err)
	}
	if err := db.GetContext(ctx, &repositoryID, `
		insert into repositories (
			github_id, installation_id, owner, name, full_name, default_branch, private
		) values ($1, $2, $3, 'docs', $3 || '/docs', 'main', false)
		returning id
	`, suffix, installationID, username); err != nil {
		t.Fatalf("seed repository: %v", err)
	}

	store := database.NewStore(db)
	draft := model.UnpublishedArticle{
		RepositoryID: repositoryID,
		SourcePath:   "articles/idempotent.md",
		GitBlobSHA:   "same-blob-sha",
		Content:      "# Idempotent sync\n",
		Frontmatter: model.Frontmatter{
			Title:       "Idempotent sync",
			Slug:        slug,
			Description: "No timestamp churn",
			Tags:        []string{"sync"},
			PublishMode: string(model.ArticlePublishModeAuto),
		},
	}
	if err := store.UpsertUnpublishedArticle(ctx, &draft); err != nil {
		t.Fatalf("first UpsertUnpublishedArticle(): %v", err)
	}
	firstDraftUpdatedAt := draft.UpdatedAt
	if err := store.UpsertUnpublishedArticle(ctx, &draft); err != nil {
		t.Fatalf("second UpsertUnpublishedArticle(): %v", err)
	}
	if !draft.UpdatedAt.Equal(firstDraftUpdatedAt) {
		t.Fatalf("unchanged draft updated_at = %v, want %v", draft.UpdatedAt, firstDraftUpdatedAt)
	}

	article, err := store.AutoPublishArticle(ctx, draft.ID, draft.GitBlobSHA)
	if err != nil {
		t.Fatalf("first AutoPublishArticle(): %v", err)
	}
	republished, err := store.AutoPublishArticle(ctx, draft.ID, draft.GitBlobSHA)
	if err != nil {
		t.Fatalf("second AutoPublishArticle(): %v", err)
	}
	if !republished.PublishedAt.Equal(article.PublishedAt) || !republished.UpdatedAt.Equal(article.UpdatedAt) {
		t.Fatalf("unchanged publication timestamps = (%v, %v), want (%v, %v)",
			republished.PublishedAt, republished.UpdatedAt, article.PublishedAt, article.UpdatedAt)
	}

	if _, err := db.ExecContext(ctx, `update unpublished_articles set present = false where id = $1`, draft.ID); err != nil {
		t.Fatalf("mark draft missing: %v", err)
	}
	if _, err := store.AutoPublishArticle(ctx, draft.ID, draft.GitBlobSHA); !errors.Is(err, model.ErrArticleNotPublishable) {
		t.Fatalf("AutoPublishArticle(missing draft) error = %v, want ErrArticleNotPublishable", err)
	}
}
