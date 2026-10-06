package auth

import (
	"database/sql"
	"errors"
	"net/http"

	"sourceink/backend/internal/model"
)

type connectEmailRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) ConnectEmail(w http.ResponseWriter, r *http.Request) {
	token, ok := sessionToken(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	user, err := h.store.UserBySession(r.Context(), SessionTokenHash(token))
	if errors.Is(err, sql.ErrNoRows) {
		h.clearSessionCookie(w)
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if err != nil {
		h.logger.Error("load account for email connection", "error", err)
		writeError(w, http.StatusInternalServerError, "SourceInk couldn't load your account")
		return
	}

	var input connectEmailRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	email, err := NormalizeAndValidateEmailCredentials(input.Email, input.Password)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if !h.registerLimiter.Allow(clientIP(r) + "|connect-email|" + email) {
		writeError(w, http.StatusTooManyRequests, "too many attempts; try again later")
		return
	}
	releasePasswordWork, ok := h.beginPasswordWork()
	if !ok {
		writeError(w, http.StatusTooManyRequests, "too many attempts; try again later")
		return
	}
	defer releasePasswordWork()

	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		h.logger.Error("hash connected email password", "error", err, "user_id", user.ID)
		writeError(w, http.StatusInternalServerError, "email sign-in connection failed")
		return
	}

	updatedUser, err := h.store.SetEmailPasswordForGitHubUser(r.Context(), user.ID, email, passwordHash)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrEmailTaken):
			writeError(w, http.StatusConflict, model.ErrEmailTaken.Error())
		case errors.Is(err, model.ErrEmailSignInAlreadySet), errors.Is(err, model.ErrEmailSignInNotAllowed):
			writeError(w, http.StatusConflict, err.Error())
		default:
			h.logger.Error("connect email sign-in", "error", err, "user_id", user.ID)
			writeError(w, http.StatusInternalServerError, "email sign-in connection failed")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"user": updatedUser})
}
