package auth

import (
	"errors"
	"net/http"

	"sourceink/backend/internal/model"
)

type registerRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input registerRequest
	if err := decodeJSON(w, r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	email, username, err := NormalizeAndValidateRegistration(input.Email, input.Username, input.Password)
	if err != nil {
		WriteError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if !h.registerLimiter.Allow(clientIP(r) + "|" + email) {
		WriteError(w, http.StatusTooManyRequests, "too many registration attempts; try again later")
		return
	}
	releasePasswordWork, ok := h.beginPasswordWork()
	if !ok {
		WriteError(w, http.StatusTooManyRequests, "too many registration attempts; try again later")
		return
	}
	defer releasePasswordWork()

	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		h.logger.Error("hash registration password", "error", err)
		WriteError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	session, err := newSession(h.sessionTTL)
	if err != nil {
		h.logger.Error("create registration token", "error", err)
		WriteError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	user, err := h.store.CreateUserAndSession(
		r.Context(),
		email,
		username,
		passwordHash,
		session.tokenHash,
		session.expiresAt,
	)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrEmailTaken):
			WriteError(w, http.StatusConflict, model.ErrEmailTaken.Error())
		case errors.Is(err, model.ErrUsernameTaken):
			WriteError(w, http.StatusConflict, model.ErrUsernameTaken.Error())
		default:
			h.logger.Error("register user", "error", err)
			WriteError(w, http.StatusInternalServerError, "registration failed")
		}
		return
	}

	SetSessionCookie(w, session.token, session.expiresAt, h.cookieSecure)
	WriteJSON(w, http.StatusCreated, map[string]any{"user": user})
}
