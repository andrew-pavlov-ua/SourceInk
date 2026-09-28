package auth

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
)

const invalidCredentialsMessage = "invalid email or password"

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input loginRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	if !h.loginLimiter.Allow(clientIP(r) + "|" + email) {
		writeError(w, http.StatusTooManyRequests, "too many login attempts; try again later")
		return
	}

	user, storedHash, err := h.store.FindUserByEmail(r.Context(), email)
	userExists := true
	if errors.Is(err, sql.ErrNoRows) {
		userExists = false
		storedHash = h.dummyPasswordHash
	} else if err != nil {
		h.logger.Error("find login user", "error", err)
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}

	passwordMatches, err := VerifyPassword(input.Password, storedHash)
	if err != nil {
		h.logger.Error("verify password", "error", err)
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	if !userExists || !passwordMatches {
		writeError(w, http.StatusUnauthorized, invalidCredentialsMessage)
		return
	}

	session, err := newSession(h.sessionTTL)
	if err != nil {
		h.logger.Error("create login token", "error", err)
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	if err := h.store.CreateSession(r.Context(), user.ID, session.tokenHash, session.expiresAt); err != nil {
		h.logger.Error("persist login session", "error", err, "user_id", user.ID)
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}

	h.setSessionCookie(w, session)
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}
