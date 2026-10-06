package api

import (
	"context"
	"fmt"

	"sourceink/backend/internal/model"
)

func (s *Service) ListArticles(ctx context.Context, userID string) (model.UserArticles, error) {
	return s.articles.ListForUser(ctx, userID)
}

func (s *Service) ListPublishedArticles(ctx context.Context, viewerID string) ([]model.PublishedArticle, error) {
	return s.store.ListPublishedArticles(ctx, viewerID)
}

func (s *Service) PublishedArticleBySlug(ctx context.Context, slug, viewerID string) (model.PublishedArticle, error) {
	article, err := s.store.PublishedArticleBySlug(ctx, slug, viewerID)
	if err != nil {
		return model.PublishedArticle{}, fmt.Errorf("load published article %q: %w", slug, err)
	}
	return article, nil
}

func (s *Service) PublishArticle(ctx context.Context, userID, draftID string) (model.Article, error) {
	article, err := s.articles.PublishArticle(ctx, userID, draftID)
	if err != nil {
		return model.Article{}, fmt.Errorf("publish article %s: %w", draftID, err)
	}
	return article, nil
}
