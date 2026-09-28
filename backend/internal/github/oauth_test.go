package github

import (
	"errors"
	"testing"
	"time"
)

func TestOAuthAttemptCacheConsumesOnceAndBindsUser(t *testing.T) {
	now := time.Date(2026, time.September, 10, 12, 0, 0, 0, time.UTC)
	cache := newOAuthAttemptCache()
	attempt := oauthAttempt{userID: "user-a", expiresAt: now.Add(oauthAttemptTTL), pendingInstallationID: 42, pkceVerifier: "verifier"}
	if err := cache.Put("state", attempt, now); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if _, err := cache.Consume("user-b", "state", now); !errors.Is(err, errOAuthAttemptInvalid) {
		t.Fatalf("Consume() user mismatch error = %v", err)
	}
	if _, err := cache.Consume("user-a", "state", now); !errors.Is(err, errOAuthAttemptInvalid) {
		t.Fatalf("Consume() replay error = %v", err)
	}
}

func TestOAuthAttemptCacheRejectsExpiredAttempt(t *testing.T) {
	now := time.Date(2026, time.September, 10, 12, 0, 0, 0, time.UTC)
	cache := newOAuthAttemptCache()
	attempt := oauthAttempt{userID: "user-a", expiresAt: now.Add(-time.Second)}
	if err := cache.Put("state", attempt, now.Add(-time.Minute)); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if _, err := cache.Consume("user-a", "state", now); !errors.Is(err, errOAuthAttemptInvalid) {
		t.Fatalf("Consume() error = %v", err)
	}
}
