package articles_test

import (
	"context"
	"errors"
	"testing"

	articles "sourceink/backend/internal/articles"
	"sourceink/backend/internal/model"
)

func TestArticlesWorkerStartStopsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	worker := articles.NewArticlesWorker(articles.NewService(&serviceTestStore{}, &serviceTestLoader{}))

	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
}

func TestSyncOnceDiscoversAndSavesArticles(t *testing.T) {
	repository := model.Repository{ID: "repository-id", FullName: "octocat/docs"}
	store := &serviceTestStore{syncRepositories: []model.Repository{repository}}
	loader := &serviceTestLoader{articlesByRepoID: map[string][]model.UnpublishedArticle{
		repository.ID: {{RepositoryID: repository.ID, SourcePath: "README.md"}},
	}}
	worker := articles.NewArticlesWorker(articles.NewService(store, loader))

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
	worker := articles.NewArticlesWorker(articles.NewService(store, loader))

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

func TestSyncRepositoryUpdatesOnlyTheWebhookRepository(t *testing.T) {
	repository := model.Repository{ID: "repository-id", InstallationID: 44, GitHubID: 99, FullName: "octocat/docs"}
	store := &serviceTestStore{webhookRepository: repository}
	loader := &serviceTestLoader{articlesByRepoID: map[string][]model.UnpublishedArticle{
		repository.ID: {{RepositoryID: repository.ID, SourcePath: "README.md", Frontmatter: model.Frontmatter{Title: "Read me", Slug: "read-me", PublishMode: "manual"}, GitBlobSHA: "blob", Present: true}},
	}}
	worker := articles.NewArticlesWorker(articles.NewService(store, loader))

	if err := worker.SyncRepository(context.Background(), 44, 99); err != nil {
		t.Fatalf("SyncRepository() error = %v", err)
	}
	if len(loader.loadedRepositories) != 1 || loader.loadedRepositories[0].ID != repository.ID {
		t.Fatalf("loaded repositories = %#v", loader.loadedRepositories)
	}
	if store.missingRepositoryID != repository.ID || len(store.missingPaths) != 1 || store.missingPaths[0] != "README.md" {
		t.Fatalf("missing reconciliation = repository:%q paths:%#v", store.missingRepositoryID, store.missingPaths)
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
	worker := articles.NewArticlesWorker(articles.NewService(store, loader))

	_, err := worker.SyncOnce(context.Background())
	if err == nil || err.Error() != "validate article slugs: duplicate slug validation failed" {
		t.Fatalf("SyncOnce() error = %v", err)
	}
	if store.autoPublishCalls != 0 {
		t.Fatalf("auto-publish calls = %d, want 0", store.autoPublishCalls)
	}
}
