package auth

import (
	"errors"
	"log/slog"
	"time"

	"sourceink/backend/internal/model"
)

const (
	SessionCookieName = "sourceink_session"

	maxConcurrentPasswordWork = 4
	passwordWorkLimit         = 60
	passwordWorkWindow        = time.Minute
)

type Handler struct {
	store               model.AccountStore
	logger              *slog.Logger
	cookieSecure        bool
	sessionTTL          time.Duration
	dummyPasswordHash   string
	loginLimiter        *Limiter
	registerLimiter     *Limiter
	passwordWorkLimiter *Limiter
	passwordWorkSlots   chan struct{}
}

func NewHandler(store model.AccountStore, logger *slog.Logger, cookieSecure bool, sessionTTL time.Duration) (*Handler, error) {
	if store == nil {
		return nil, errors.New("auth store is required")
	}
	if logger == nil {
		return nil, errors.New("auth logger is required")
	}
	if sessionTTL <= 0 {
		return nil, errors.New("session TTL must be positive")
	}

	dummyPasswordHash, err := HashPassword("this-password-is-never-valid")
	if err != nil {
		return nil, err
	}

	return &Handler{
		store:               store,
		logger:              logger,
		cookieSecure:        cookieSecure,
		sessionTTL:          sessionTTL,
		dummyPasswordHash:   dummyPasswordHash,
		loginLimiter:        NewLimiter(10, 10*time.Minute),
		registerLimiter:     NewLimiter(5, time.Hour),
		passwordWorkLimiter: NewLimiter(passwordWorkLimit, passwordWorkWindow),
		passwordWorkSlots:   make(chan struct{}, maxConcurrentPasswordWork),
	}, nil
}

// beginPasswordWork reserves capacity for an Argon2 operation. Every endpoint
// that hashes or verifies a password shares this admission control.
func (h *Handler) beginPasswordWork() (release func(), ok bool) {
	select {
	case h.passwordWorkSlots <- struct{}{}:
	default:
		return nil, false
	}

	if !h.passwordWorkLimiter.Allow("password-work") {
		<-h.passwordWorkSlots
		return nil, false
	}

	return func() { <-h.passwordWorkSlots }, true
}
