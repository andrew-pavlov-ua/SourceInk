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

func (s *Service) PublishArticle(ctx context.Context, userID, draftID string) (model.Article, error) {
	article, err := s.articles.PublishArticle(ctx, userID, draftID)
	if err != nil {
		return model.Article{}, fmt.Errorf("publish article %s: %w", draftID, err)
	}
	return article, nil
}
