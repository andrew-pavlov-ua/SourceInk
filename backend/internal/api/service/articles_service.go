package api

import (
	"context"

	"sourceink/backend/internal/model"
)

func (s *Service) ListArticles(ctx context.Context, userID string) (model.UserArticles, error) {
	return s.articles.ListForUser(ctx, userID)
}
