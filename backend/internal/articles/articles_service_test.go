package articles

import (
	"context"
	"errors"
	"testing"

	"sourceink/backend/internal/model"
)

type serviceTestStore struct {
	published          []model.Article
	unpublished        []model.UnpublishedArticle
	userRepositories   []model.Repository
	syncRepositories   []model.Repository
	upsertedSourcePath []string
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
	return nil
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
	service := NewService(store, loader)

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
	service := NewService(store, loader)

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
}

func TestRunOnceUsesSharedDiscovery(t *testing.T) {
	repository := model.Repository{ID: "repository-id", FullName: "octocat/docs"}
	store := &serviceTestStore{syncRepositories: []model.Repository{repository}}
	loader := &serviceTestLoader{articlesByRepoID: map[string][]model.UnpublishedArticle{
		repository.ID: {{RepositoryID: repository.ID, SourcePath: "README.md"}},
	}}
	worker := NewArticlesWorker(NewService(store, loader))

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
	service := NewService(&serviceTestStore{}, &serviceTestLoader{err: wantErr})

	_, err := service.discover(context.Background(), []model.Repository{{FullName: "octocat/docs"}})
	if !errors.Is(err, wantErr) {
		t.Fatalf("discover() error = %v, want %v", err, wantErr)
	}
	if got := err.Error(); got != "load unpublished articles from octocat/docs: GitHub unavailable" {
		t.Fatalf("discover() error = %q", got)
	}
}
