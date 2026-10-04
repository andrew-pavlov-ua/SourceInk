package api

import (
	"context"

	articleService "sourceink/backend/internal/articles"
	db "sourceink/backend/internal/database"
	"sourceink/backend/internal/model"
)

type Service struct {
	store    *db.Store
	articles *articleService.Service
	reviews  *ReviewService
}

func NewService(store *db.Store, articles *articleService.Service) Service {
	return Service{
		store:    store,
		articles: articles,
		reviews:  NewReviewService(store),
	}
}

func (s *Service) UserBySession(ctx context.Context, tokenHash []byte) (model.User, error) {
	return s.store.UserBySession(ctx, tokenHash)
}
