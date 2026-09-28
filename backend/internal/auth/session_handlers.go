package auth

import (
	"database/sql"
	"errors"
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
		h.logger.Error("load session user", "error", err)
		writeError(w, http.StatusInternalServerError, "SourceInk couldn't load your account")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if token, ok := sessionToken(r); ok {
		if err := h.store.DeleteSession(r.Context(), SessionTokenHash(token)); err != nil {
			h.logger.Error("delete session", "error", err)
			writeError(w, http.StatusInternalServerError, "logout failed")
			return
		}
	}

	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func sessionToken(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, session session) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    session.token,
		Path:     "/",
		Expires:  session.expiresAt,
		MaxAge:   int(time.Until(session.expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
