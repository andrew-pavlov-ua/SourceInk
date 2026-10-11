package api

import (
	"context"

	"sourceink/backend/internal/model"
)

func (s *Service) ListArticles(ctx context.Context, userID string) (model.UserArticles, error) {
	return s.articles.ListForUser(ctx, userID)
}

func (s *Service) ListPublishedArticles(ctx context.Context, viewerID string, order model.PublishedArticleOrder) ([]model.PublishedArticle, error) {
	return s.store.ListPublishedArticles(ctx, viewerID, order)
}

func (s *Service) PublishedArticleBySlug(ctx context.Context, slug, viewerID string) (model.PublishedArticle, error) {
	return s.store.PublishedArticleBySlug(ctx, slug, viewerID)
}

func (s *Service) IncrementArticleViewCount(ctx context.Context, articleID string) error {
	return s.store.IncrementArticleViewCount(ctx, articleID)
}

func (s *Service) PublishArticle(ctx context.Context, userID, draftID string) (model.Article, error) {
	return s.articles.PublishArticle(ctx, userID, draftID)
}
