package auth

import (
	"net/http"
	"time"
)

type session struct {
	token     string
	tokenHash []byte
	expiresAt time.Time
}

func newSession(ttl time.Duration) (session, error) {
	token, tokenHash, err := NewSessionToken()
	if err != nil {
		return session{}, err
	}
	return session{
		token:     token,
		tokenHash: tokenHash,
		expiresAt: time.Now().Add(ttl),
	}, nil
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := SessionUser(w, r, true, h.cookieSecure, h.logger, h.store.UserBySession)
	if !ok {
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if token, ok := sessionToken(r); ok {
		if err := h.store.DeleteSession(r.Context(), SessionTokenHash(token)); err != nil {
			h.logger.Error("delete session", "error", err)
			WriteError(w, http.StatusInternalServerError, "logout failed")
			return
		}
	}

	ClearSessionCookie(w, h.cookieSecure)
	w.WriteHeader(http.StatusNoContent)
}

func sessionToken(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}
