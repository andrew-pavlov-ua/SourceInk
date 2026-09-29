package articles

import (
	"context"
	"errors"
	"testing"
	"time"

	"sourceink/backend/internal/model"
)

func TestRunOnIntervalWaitsForIntervalAndRepeatsUntilCancelled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	runs := 0
	err := runOnInterval(ctx, time.Millisecond, func() error {
		runs++
		if runs == 2 {
			cancel()
		}
		return nil
	})
	if err != nil {
		t.Fatalf("runOnInterval() error = %v", err)
	}
	if runs < 2 {
		t.Fatalf("runs = %d, want at least 2", runs)
	}
}

func TestSyncOnceDiscoversAndSavesArticles(t *testing.T) {
	repository := model.Repository{ID: "repository-id", FullName: "octocat/docs"}
	store := &serviceTestStore{syncRepositories: []model.Repository{repository}}
	loader := &serviceTestLoader{articlesByRepoID: map[string][]model.UnpublishedArticle{
		repository.ID: {{RepositoryID: repository.ID, SourcePath: "README.md"}},
	}}
	worker := NewArticlesWorker(NewService(store, loader))

	articles, err := worker.SyncOnce(context.Background())
	if err != nil {
		t.Fatalf("SyncOnce() error = %v", err)
	}
	if len(articles) != 1 || len(store.upsertedSourcePath) != 1 || store.upsertedSourcePath[0] != "README.md" {
		t.Fatalf("SyncOnce() articles = %#v, saved paths = %#v", articles, store.upsertedSourcePath)
	}
}

func TestSyncOnceAutomaticallyPublishesOnlyAutomaticDrafts(t *testing.T) {
	repository := model.Repository{ID: "repository-id", FullName: "octocat/docs"}
	store := &serviceTestStore{
		syncRepositories:  []model.Repository{repository},
		autoPublishDraft:  publishableDraft("auto"),
		autoPublishResult: model.Article{ID: "article-id"},
	}
	loader := &serviceTestLoader{articlesByRepoID: map[string][]model.UnpublishedArticle{
		repository.ID: {
			{RepositoryID: repository.ID, SourcePath: "automatic.md", Frontmatter: model.Frontmatter{Title: "Automatic", Slug: "automatic", PublishMode: "auto"}, GitBlobSHA: "blob-sha", Present: true},
			{RepositoryID: repository.ID, SourcePath: "manual.md", Frontmatter: model.Frontmatter{Title: "Manual", Slug: "manual", PublishMode: "manual"}, GitBlobSHA: "other-blob", Present: true},
		},
	}}
	worker := NewArticlesWorker(NewService(store, loader))

	if _, err := worker.SyncOnce(context.Background()); err != nil {
		t.Fatalf("SyncOnce() error = %v", err)
	}
	if store.autoPublishCalls != 1 || store.autoPublishDraftID != "saved-automatic.md" {
		t.Fatalf("auto-publish call = count:%d draft:%q", store.autoPublishCalls, store.autoPublishDraftID)
	}
	if store.reconcileCalls != 1 {
		t.Fatalf("duplicate slug checks = %d, want 1", store.reconcileCalls)
	}
}

func TestSyncOnceStopsBeforeAutoPublishWhenSlugValidationFails(t *testing.T) {
	repository := model.Repository{ID: "repository-id", FullName: "octocat/docs"}
	store := &serviceTestStore{
		syncRepositories: []model.Repository{repository},
		reconcileErr:     errors.New("duplicate slug validation failed"),
	}
	loader := &serviceTestLoader{articlesByRepoID: map[string][]model.UnpublishedArticle{
		repository.ID: {{RepositoryID: repository.ID, SourcePath: "automatic.md", Frontmatter: model.Frontmatter{Title: "Automatic", Slug: "automatic", PublishMode: "auto"}}},
	}}
	worker := NewArticlesWorker(NewService(store, loader))

	_, err := worker.SyncOnce(context.Background())
	if err == nil || err.Error() != "validate article slugs: duplicate slug validation failed" {
		t.Fatalf("SyncOnce() error = %v", err)
	}
	if store.autoPublishCalls != 0 {
		t.Fatalf("auto-publish calls = %d, want 0", store.autoPublishCalls)
	}
}

func TestRunOnIntervalReturnsRunError(t *testing.T) {
	wantErr := errors.New("sync failed")
	runs := 0

	err := runOnInterval(context.Background(), time.Millisecond, func() error {
		runs++
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("runOnInterval() error = %v, want %v", err, wantErr)
	}
	if runs != 1 {
		t.Fatalf("runs = %d, want 1", runs)
	}
}

func TestRunOnIntervalDoesNotRunAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	runs := 0
	err := runOnInterval(ctx, time.Millisecond, func() error {
		runs++
		return nil
	})
	if err != nil {
		t.Fatalf("runOnInterval() error = %v", err)
	}
	if runs != 0 {
		t.Fatalf("runs = %d, want 0", runs)
	}
}
