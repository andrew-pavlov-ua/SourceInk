package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"sourceink/backend/internal/model"
)

const maxRequestBodyBytes = 32 * 1024

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errors.New("Content-Type must be application/json")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New("invalid JSON body")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func SetSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// SessionUser resolves the signed-in user from the session cookie. A missing
// or stale session answers 401 when required, otherwise it yields an empty
// user. The second result is false only after an error response was written.
func SessionUser(
	w http.ResponseWriter,
	r *http.Request,
	required bool,
	secure bool,
	logger *slog.Logger,
	lookup func(context.Context, []byte) (model.User, error),
) (model.User, bool) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		return noSession(w, required)
	}
	user, err := lookup(r.Context(), SessionTokenHash(cookie.Value))
	if errors.Is(err, sql.ErrNoRows) {
		ClearSessionCookie(w, secure)
		return noSession(w, required)
	}
	if err != nil {
		logger.Error("load session user", "error", err)
		WriteError(w, http.StatusInternalServerError, "SourceInk couldn't load your account")
		return model.User{}, false
	}
	return user, true
}

func noSession(w http.ResponseWriter, required bool) (model.User, bool) {
	if required {
		WriteError(w, http.StatusUnauthorized, "authentication required")
	}
	return model.User{}, !required
}
