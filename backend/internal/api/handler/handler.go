package api

import (
	"database/sql"
	"encoding/json"
	"errors"
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
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return model.User{}, false
	}
	user, err := h.service.UserBySession(r.Context(), auth.SessionTokenHash(cookie.Value))
	if errors.Is(err, sql.ErrNoRows) {
		h.clearSessionCookie(w)
		writeError(w, http.StatusUnauthorized, "authentication required")
		return model.User{}, false
	}
	if err != nil {
		h.logger.Error("load session user", "error", err)
		writeError(w, http.StatusInternalServerError, "SourceInk couldn't load your account")
		return model.User{}, false
	}
	return user, true
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
