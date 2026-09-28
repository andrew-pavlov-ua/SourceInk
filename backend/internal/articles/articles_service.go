package articles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"sourceink/backend/internal/model"
)

type Store interface {
	ListUserArticles(ctx context.Context, userID string) ([]model.Article, error)
	ListUserUnpublishedArticles(ctx context.Context, userID string) ([]model.UnpublishedArticle, error)
	ListRepositoriesForUser(ctx context.Context, userID string) ([]model.Repository, error)
	ListRepositoriesForArticleSync(ctx context.Context) ([]model.Repository, error)
	UpsertUnpublishedArticle(ctx context.Context, article *model.UnpublishedArticle) error
}

type ArticleLoader interface {
	LoadUnpublishedArticlesByRepo(ctx context.Context, repository model.Repository) ([]model.UnpublishedArticle, error)
}

type Service struct {
	store  Store
	loader ArticleLoader
}

func NewService(store Store, loader ArticleLoader) *Service {
	return &Service{
		store:  store,
		loader: loader,
	}
}

func (s *Service) ListForUser(ctx context.Context, userID string) (model.UserArticles, error) {
	publishedArticles, err := s.store.ListUserArticles(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		publishedArticles = []model.Article{}
	} else if err != nil {
		return model.UserArticles{}, fmt.Errorf("list published articles: %w", err)
	}

	unpublishedArticles, err := s.store.ListUserUnpublishedArticles(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		unpublishedArticles = []model.UnpublishedArticle{}
	} else if err != nil {
		return model.UserArticles{}, fmt.Errorf("list unpublished articles: %w", err)
	}

	if len(publishedArticles) == 0 && len(unpublishedArticles) == 0 {
		unpublishedArticles, err = s.discoverForUser(ctx, userID)
		if err != nil {
			return model.UserArticles{}, err
		}
		if err := s.save(ctx, unpublishedArticles); err != nil {
			return model.UserArticles{}, fmt.Errorf("save discovered articles: %w", err)
		}
	}

	return model.UserArticles{
		UserID:              userID,
		UnpublishedArticles: unpublishedArticles,
		PublishedArticles:   publishedArticles,
	}, nil
}

func (s *Service) discoverForUser(ctx context.Context, userID string) ([]model.UnpublishedArticle, error) {
	repositories, err := s.store.ListRepositoriesForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list repositories for article discovery: %w", err)
	}

	return s.discover(ctx, repositories)
}

func (s *Service) discoverForSync(ctx context.Context) ([]model.UnpublishedArticle, error) {
	repositories, err := s.store.ListRepositoriesForArticleSync(ctx)
	if err != nil {
		return nil, fmt.Errorf("list repositories for article sync: %w", err)
	}

	return s.discover(ctx, repositories)
}

func (s *Service) discover(ctx context.Context, repositories []model.Repository) ([]model.UnpublishedArticle, error) {
	if s.loader == nil {
		return nil, errors.New("article discovery is not configured")
	}

	unpublishedArticles := make([]model.UnpublishedArticle, 0)
	for _, repository := range repositories {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		repositoryArticles, err := s.loader.LoadUnpublishedArticlesByRepo(ctx, repository)
		if err != nil {
			return nil, fmt.Errorf("load unpublished articles from %s: %w", repository.FullName, err)
		}
		unpublishedArticles = append(unpublishedArticles, repositoryArticles...)
	}

	return unpublishedArticles, nil
}

func (s *Service) save(ctx context.Context, articles []model.UnpublishedArticle) error {
	for i := range articles {
		article := &articles[i]
		if err := s.store.UpsertUnpublishedArticle(ctx, article); err != nil {
			return fmt.Errorf("upsert unpublished article %s: %w", article.SourcePath, err)
		}
	}

	return nil
}
