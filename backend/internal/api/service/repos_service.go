package api

import (
	"context"
	"sourceink/backend/internal/model"
)

func (s *Service) ListRepositories(ctx context.Context, userID string) ([]model.Repository, error) {
	return s.store.ListRepositoriesForUser(ctx, userID)
}
