package articles_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	articles "sourceink/backend/internal/articles"
	"sourceink/backend/internal/model"
)

type serviceTestStore struct {
	published          []model.Article
	unpublished        []model.UnpublishedArticle
	userRepositories   []model.Repository
	syncRepositories   []model.Repository
	upsertedSourcePath []string
	reconcileCalls     int
	reconcileErr       error
	publishDraft       model.UnpublishedArticle
	publishResult      model.Article
	findPublishErr     error
	findPublishUserID  string
	findPublishDraftID string
	publishErr         error
	publishCalls       int
	autoPublishDraft   model.UnpublishedArticle
	autoPublishResult  model.Article
	autoFindErr        error
	autoPublishErr     error
	autoPublishCalls   int
	autoPublishDraftID string
	autoPublishBlobSHA string
}

func (s *serviceTestStore) ListUserArticles(context.Context, string) ([]model.Article, error) {
	return s.published, nil
}

func (s *serviceTestStore) ListUserUnpublishedArticles(context.Context, string) ([]model.UnpublishedArticle, error) {
	return s.unpublished, nil
}

func (s *serviceTestStore) ListRepositoriesForUser(context.Context, string) ([]model.Repository, error) {
	return s.userRepositories, nil
}

func (s *serviceTestStore) ListRepositoriesForArticleSync(context.Context) ([]model.Repository, error) {
	return s.syncRepositories, nil
}

func (s *serviceTestStore) UpsertUnpublishedArticle(_ context.Context, article *model.UnpublishedArticle) error {
	s.upsertedSourcePath = append(s.upsertedSourcePath, article.SourcePath)
	article.ID = "saved-" + article.SourcePath
	s.unpublished = append(s.unpublished, *article)
	return nil
}

func (s *serviceTestStore) ReconcileDuplicateSlugs(context.Context) error {
	s.reconcileCalls++
	return s.reconcileErr
}

func (s *serviceTestStore) FindUnpublishedArticleForPublish(_ context.Context, userID, draftID string) (model.UnpublishedArticle, error) {
	s.findPublishUserID = userID
	s.findPublishDraftID = draftID
	return s.publishDraft, s.findPublishErr
}

func (s *serviceTestStore) PublishArticle(_ context.Context, _, _, _ string) (model.Article, error) {
	s.publishCalls++
	return s.publishResult, s.publishErr
}

func (s *serviceTestStore) AutoFindUnpublishedArticleForPublish(_ context.Context, draftID string) (model.UnpublishedArticle, error) {
	s.autoPublishDraftID = draftID
	return s.autoPublishDraft, s.autoFindErr
}

func (s *serviceTestStore) AutoPublishArticle(_ context.Context, draftID, gitBlobSHA string) (model.Article, error) {
	s.autoPublishCalls++
	s.autoPublishDraftID = draftID
	s.autoPublishBlobSHA = gitBlobSHA
	return s.autoPublishResult, s.autoPublishErr
}

type serviceTestLoader struct {
	loadedRepositories []model.Repository
	articlesByRepoID   map[string][]model.UnpublishedArticle
	err                error
}

func (l *serviceTestLoader) LoadUnpublishedArticlesByRepo(
	_ context.Context,
	repository model.Repository,
) ([]model.UnpublishedArticle, error) {
	l.loadedRepositories = append(l.loadedRepositories, repository)
	if l.err != nil {
		return nil, l.err
	}
	return l.articlesByRepoID[repository.ID], nil
}

func TestListForUserReturnsStoredArticlesWithoutRediscovery(t *testing.T) {
	store := &serviceTestStore{
		published:   []model.Article{{ID: "published-id"}},
		unpublished: []model.UnpublishedArticle{{ID: "draft-id"}},
	}
	loader := &serviceTestLoader{}
	service := articles.NewService(store, loader)

	articles, err := service.ListForUser(context.Background(), "user-id")
	if err != nil {
		t.Fatalf("ListForUser() error = %v", err)
	}
	if articles.UserID != "user-id" || len(articles.PublishedArticles) != 1 || len(articles.UnpublishedArticles) != 1 {
		t.Fatalf("ListForUser() = %#v", articles)
	}
	if len(loader.loadedRepositories) != 0 {
		t.Fatalf("loaded repositories = %#v, want none", loader.loadedRepositories)
	}
	if len(store.upsertedSourcePath) != 0 {
		t.Fatalf("upserted paths = %#v, want none", store.upsertedSourcePath)
	}
}

func TestListForUserDiscoversAndSavesArticlesWhenStorageIsEmpty(t *testing.T) {
	repository := model.Repository{ID: "repository-id", FullName: "octocat/docs"}
	store := &serviceTestStore{userRepositories: []model.Repository{repository}}
	loader := &serviceTestLoader{articlesByRepoID: map[string][]model.UnpublishedArticle{
		repository.ID: {{RepositoryID: repository.ID, SourcePath: "guides/setup.md"}},
	}}
	service := articles.NewService(store, loader)

	articles, err := service.ListForUser(context.Background(), "user-id")
	if err != nil {
		t.Fatalf("ListForUser() error = %v", err)
	}
	if len(loader.loadedRepositories) != 1 || loader.loadedRepositories[0].ID != repository.ID {
		t.Fatalf("loaded repositories = %#v", loader.loadedRepositories)
	}
	if len(store.upsertedSourcePath) != 1 || store.upsertedSourcePath[0] != "guides/setup.md" {
		t.Fatalf("upserted paths = %#v", store.upsertedSourcePath)
	}
	if len(articles.UnpublishedArticles) != 1 || articles.UnpublishedArticles[0].ID != "saved-guides/setup.md" {
		t.Fatalf("unpublished articles = %#v", articles.UnpublishedArticles)
	}
	if articles.UnpublishedArticles[0].ValidationError == nil {
		t.Fatal("discovered invalid draft has no validation warning")
	}
	if store.reconcileCalls != 1 {
		t.Fatalf("duplicate slug checks = %d, want 1", store.reconcileCalls)
	}
}

func TestListForUserDoesNotAutomaticallyPublishDiscoveredArticles(t *testing.T) {
	repository := model.Repository{ID: "repository-id", FullName: "octocat/docs"}
	store := &serviceTestStore{userRepositories: []model.Repository{repository}}
	loader := &serviceTestLoader{articlesByRepoID: map[string][]model.UnpublishedArticle{
		repository.ID: {{
			RepositoryID: repository.ID,
			SourcePath:   "automatic.md",
			Frontmatter:  model.Frontmatter{Title: "Automatic", Slug: "automatic", PublishMode: "auto"},
			GitBlobSHA:   "blob-sha",
			Present:      true,
		}},
	}}
	service := articles.NewService(store, loader)

	if _, err := service.ListForUser(context.Background(), "user-id"); err != nil {
		t.Fatalf("ListForUser() error = %v", err)
	}
	if store.autoPublishCalls != 0 {
		t.Fatalf("ListForUser() auto-publish calls = %d, want 0", store.autoPublishCalls)
	}
}

func TestRunOnceUsesSharedDiscovery(t *testing.T) {
	repository := model.Repository{ID: "repository-id", FullName: "octocat/docs"}
	store := &serviceTestStore{syncRepositories: []model.Repository{repository}}
	loader := &serviceTestLoader{articlesByRepoID: map[string][]model.UnpublishedArticle{
		repository.ID: {{RepositoryID: repository.ID, SourcePath: "README.md"}},
	}}
	worker := articles.NewArticlesWorker(articles.NewService(store, loader))

	articles, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(articles) != 1 || articles[0].SourcePath != "README.md" {
		t.Fatalf("RunOnce() = %#v", articles)
	}
	if len(loader.loadedRepositories) != 1 || loader.loadedRepositories[0].ID != repository.ID {
		t.Fatalf("loaded repositories = %#v", loader.loadedRepositories)
	}
}

func TestDiscoverIncludesRepositoryNameInLoaderError(t *testing.T) {
	wantErr := errors.New("GitHub unavailable")
	store := &serviceTestStore{userRepositories: []model.Repository{{FullName: "octocat/docs"}}}
	service := articles.NewService(store, &serviceTestLoader{err: wantErr})

	_, err := service.ListForUser(context.Background(), "user-id")
	if !errors.Is(err, wantErr) {
		t.Fatalf("discover() error = %v, want %v", err, wantErr)
	}
	if got := err.Error(); got != "load unpublished articles from octocat/docs: GitHub unavailable" {
		t.Fatalf("discover() error = %q", got)
	}
}

func TestPublishArticleWrapsNotFoundForMissingIdentity(t *testing.T) {
	service := articles.NewService(&serviceTestStore{}, &serviceTestLoader{})

	_, err := service.PublishArticle(context.Background(), "", "draft-id")
	if !errors.Is(err, model.ErrArticleNotFound) {
		t.Fatalf("PublishArticle() error = %v, want ErrArticleNotFound", err)
	}
}

func TestPublishArticleWrapsNotFoundForUnauthorizedDraft(t *testing.T) {
	store := &serviceTestStore{findPublishErr: model.ErrArticleNotFound}
	service := articles.NewService(store, &serviceTestLoader{})

	_, err := service.PublishArticle(context.Background(), "user-id", "draft-id")
	if !errors.Is(err, model.ErrArticleNotFound) {
		t.Fatalf("PublishArticle() error = %v, want ErrArticleNotFound", err)
	}
}

func TestPublishArticleRejectsInvalidDraft(t *testing.T) {
	store := &serviceTestStore{publishDraft: model.UnpublishedArticle{
		ID: "draft-id",
		Frontmatter: model.Frontmatter{
			Title:       "Article",
			Slug:        "article",
			PublishMode: "manual",
		},
		GitBlobSHA: "blob-sha",
		Present:    false,
	}}
	service := articles.NewService(store, &serviceTestLoader{})

	_, err := service.PublishArticle(context.Background(), "user-id", "draft-id")
	if !errors.Is(err, model.ErrArticleNotPublishable) {
		t.Fatalf("PublishArticle() error = %v, want ErrArticleNotPublishable", err)
	}
	if store.publishCalls != 0 {
		t.Fatalf("PublishArticle() store calls = %d, want 0", store.publishCalls)
	}
}

func TestPublishArticleTreatsDraftWarningAsCritical(t *testing.T) {
	store := &serviceTestStore{publishDraft: model.UnpublishedArticle{
		ID: "draft-id",
		Frontmatter: model.Frontmatter{
			Title:       strings.Repeat("a", 121),
			Slug:        "article",
			PublishMode: "manual",
		},
		GitBlobSHA: "blob-sha",
		Present:    true,
	}}
	service := articles.NewService(store, &serviceTestLoader{})

	_, err := service.PublishArticle(context.Background(), "user-id", "draft-id")
	if !errors.Is(err, model.ErrArticleNotPublishable) {
		t.Fatalf("PublishArticle() error = %v, want ErrArticleNotPublishable", err)
	}
	if store.publishCalls != 0 {
		t.Fatalf("PublishArticle() store calls = %d, want 0", store.publishCalls)
	}
}

func TestPublishArticleStoresAuthorizedDraft(t *testing.T) {
	store := &serviceTestStore{
		publishDraft: model.UnpublishedArticle{
			ID: "draft-id",
			Frontmatter: model.Frontmatter{
				Title:       "Article",
				Slug:        "article",
				PublishMode: "manual",
			},
			GitBlobSHA: "blob-sha",
			Content:    "# Hello\n\nPublished content.",
			Present:    true,
		},
		publishResult: model.Article{ID: "article-id"},
	}
	service := articles.NewService(store, &serviceTestLoader{})

	article, err := service.PublishArticle(context.Background(), "user-id", "draft-id")
	if err != nil {
		t.Fatalf("PublishArticle() error = %v", err)
	}
	if article.ID != "article-id" {
		t.Fatalf("PublishArticle() article = %#v", article)
	}
	if store.publishCalls != 1 {
		t.Fatalf("PublishArticle() store calls = %d, want 1", store.publishCalls)
	}
	if store.findPublishUserID != "user-id" || store.findPublishDraftID != "draft-id" {
		t.Fatalf("manual lookup = user:%q draft:%q", store.findPublishUserID, store.findPublishDraftID)
	}
}

func TestPublishArticleRejectsAutomaticDraft(t *testing.T) {
	store := &serviceTestStore{publishDraft: publishableDraft("auto")}
	service := articles.NewService(store, &serviceTestLoader{})

	_, err := service.PublishArticle(context.Background(), "user-id", "draft-id")
	if !errors.Is(err, model.ErrArticleNotPublishable) {
		t.Fatalf("PublishArticle() error = %v, want ErrArticleNotPublishable", err)
	}
	if store.publishCalls != 0 {
		t.Fatalf("PublishArticle() store calls = %d, want 0", store.publishCalls)
	}
}

func TestPublishArticleRejectsDuplicateSlugValidation(t *testing.T) {
	validationError := `duplicate slug: 'article' is used by multiple source files; choose a unique slug.`
	store := &serviceTestStore{publishDraft: model.UnpublishedArticle{
		ID:              "draft-id",
		Frontmatter:     model.Frontmatter{Title: "Article", Slug: "article", PublishMode: "manual"},
		GitBlobSHA:      "blob-sha",
		Present:         true,
		ValidationError: &validationError,
	}}
	service := articles.NewService(store, &serviceTestLoader{})

	_, err := service.PublishArticle(context.Background(), "user-id", "draft-id")
	if !errors.Is(err, model.ErrArticleNotPublishable) {
		t.Fatalf("PublishArticle() error = %v, want ErrArticleNotPublishable", err)
	}
	if store.publishCalls != 0 {
		t.Fatalf("PublishArticle() store calls = %d, want 0", store.publishCalls)
	}
}

func TestAutoPublishArticleRejectsManualDraft(t *testing.T) {
	store := &serviceTestStore{autoPublishDraft: publishableDraft("manual")}
	service := articles.NewService(store, &serviceTestLoader{})

	_, err := service.AutoPublishArticle(context.Background(), "draft-id")
	if !errors.Is(err, model.ErrArticleNotPublishable) {
		t.Fatalf("AutoPublishArticle() error = %v, want ErrArticleNotPublishable", err)
	}
	if store.autoPublishCalls != 0 {
		t.Fatalf("AutoPublishArticle() store calls = %d, want 0", store.autoPublishCalls)
	}
}

func TestAutoPublishArticleStoresAutomaticDraftWithoutUserID(t *testing.T) {
	store := &serviceTestStore{
		autoPublishDraft:  publishableDraft("auto"),
		autoPublishResult: model.Article{ID: "article-id"},
	}
	service := articles.NewService(store, &serviceTestLoader{})

	article, err := service.AutoPublishArticle(context.Background(), "draft-id")
	if err != nil {
		t.Fatalf("AutoPublishArticle() error = %v", err)
	}
	if article.ID != "article-id" {
		t.Fatalf("AutoPublishArticle() article = %#v", article)
	}
	if store.autoPublishCalls != 1 || store.autoPublishDraftID != "draft-id" || store.autoPublishBlobSHA != "blob-sha" {
		t.Fatalf("auto-publish call = count:%d draft:%q blob:%q", store.autoPublishCalls, store.autoPublishDraftID, store.autoPublishBlobSHA)
	}
}

func publishableDraft(publishMode string) model.UnpublishedArticle {
	return model.UnpublishedArticle{
		ID: "draft-id",
		Frontmatter: model.Frontmatter{
			Title:       "Article",
			Slug:        "article",
			PublishMode: publishMode,
		},
		GitBlobSHA: "blob-sha",
		Present:    true,
	}
}
