package api

import (
	"log/slog"
	"net/http"

	service "sourceink/backend/internal/api/service"
	"sourceink/backend/internal/auth"
	"sourceink/backend/internal/model"
)

type Handler struct {
	cookieSecure bool
	logger       *slog.Logger
	service      *service.Service
}

func NewHandler(service *service.Service, cookieSecure bool, logger *slog.Logger) Handler {
	return Handler{
		service:      service,
		cookieSecure: cookieSecure,
		logger:       logger,
	}
}

func (h *Handler) currentUser(w http.ResponseWriter, r *http.Request) (model.User, bool) {
	return auth.SessionUser(w, r, true, h.cookieSecure, h.logger, h.service.UserBySession)
}

func (h *Handler) optionalCurrentUser(w http.ResponseWriter, r *http.Request) (model.User, bool) {
	return auth.SessionUser(w, r, false, h.cookieSecure, h.logger, h.service.UserBySession)
}
