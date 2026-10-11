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
		WriteError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	user, err := h.store.UserBySession(r.Context(), SessionTokenHash(token))
	if errors.Is(err, sql.ErrNoRows) {
		ClearSessionCookie(w, h.cookieSecure)
		WriteError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if err != nil {
		h.logger.Error("load account for email connection", "error", err)
		WriteError(w, http.StatusInternalServerError, "SourceInk couldn't load your account")
		return
	}

	var input connectEmailRequest
	if err := decodeJSON(w, r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	email, err := NormalizeAndValidateEmailCredentials(input.Email, input.Password)
	if err != nil {
		WriteError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if !h.registerLimiter.Allow(clientIP(r) + "|connect-email|" + email) {
		WriteError(w, http.StatusTooManyRequests, "too many attempts; try again later")
		return
	}
	releasePasswordWork, ok := h.beginPasswordWork()
	if !ok {
		WriteError(w, http.StatusTooManyRequests, "too many attempts; try again later")
		return
	}
	defer releasePasswordWork()

	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		h.logger.Error("hash connected email password", "error", err, "user_id", user.ID)
		WriteError(w, http.StatusInternalServerError, "email sign-in connection failed")
		return
	}

	updatedUser, err := h.store.SetEmailPasswordForGitHubUser(r.Context(), user.ID, email, passwordHash)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrEmailTaken):
			WriteError(w, http.StatusConflict, model.ErrEmailTaken.Error())
		case errors.Is(err, model.ErrEmailSignInAlreadySet), errors.Is(err, model.ErrEmailSignInNotAllowed):
			WriteError(w, http.StatusConflict, err.Error())
		default:
			h.logger.Error("connect email sign-in", "error", err, "user_id", user.ID)
			WriteError(w, http.StatusInternalServerError, "email sign-in connection failed")
		}
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{"user": updatedUser})
}
