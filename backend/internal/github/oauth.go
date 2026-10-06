package github

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

const (
	oauthAttemptTTL        = 10 * time.Minute
	maxOAuthAttempts       = 10_000
	loginBindingCookieName = "sourceink_github_login"
)

var errOAuthAttemptInvalid = errors.New("github OAuth attempt is invalid or expired")

type oauthAttempt struct {
	userID                string
	expiresAt             time.Time
	pendingInstallationID int64
	pkceVerifier          string
	purpose               oauthPurpose
	loginBindingHash      [sha256.Size]byte
}

type oauthPurpose uint8

const (
	oauthPurposeConnectInstallation oauthPurpose = iota + 1
	oauthPurposeLogin
	oauthPurposeConnectAccount
)

// oauthAttemptCache stores hashed OAuth state in this process.
type oauthAttemptCache struct {
	mu       sync.Mutex
	attempts map[[sha256.Size]byte]oauthAttempt
}

func newOAuthAttemptCache() *oauthAttemptCache {
	return &oauthAttemptCache{attempts: make(map[[sha256.Size]byte]oauthAttempt)}
}

func (c *oauthAttemptCache) Put(rawState string, attempt oauthAttempt, now time.Time) error {
	stateHash := sha256.Sum256([]byte(rawState))

	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeExpired(now)
	if len(c.attempts) >= maxOAuthAttempts {
		return errors.New("github OAuth attempt cache is full")
	}
	c.attempts[stateHash] = attempt
	return nil
}

// Consume reads and removes an attempt under one lock. A GitHub authorization
// code can only be used once.
func (c *oauthAttemptCache) Consume(userID, rawState string, now time.Time) (oauthAttempt, error) {
	stateHash := sha256.Sum256([]byte(rawState))

	c.mu.Lock()
	defer c.mu.Unlock()
	attempt, ok := c.attempts[stateHash]
	if !ok {
		return oauthAttempt{}, errOAuthAttemptInvalid
	}
	delete(c.attempts, stateHash)
	if attempt.userID != userID || !attempt.expiresAt.After(now) {
		return oauthAttempt{}, errOAuthAttemptInvalid
	}
	return attempt, nil
}

func (c *oauthAttemptCache) ConsumeLogin(rawState, rawBinding string, now time.Time) (oauthAttempt, error) {
	stateHash := sha256.Sum256([]byte(rawState))
	bindingHash := sha256.Sum256([]byte(rawBinding))

	c.mu.Lock()
	defer c.mu.Unlock()

	attempt, ok := c.attempts[stateHash]
	if !ok || attempt.userID != "" || attempt.purpose != oauthPurposeLogin || !attempt.expiresAt.After(now) || attempt.loginBindingHash != bindingHash {
		return oauthAttempt{}, errOAuthAttemptInvalid
	}
	delete(c.attempts, stateHash)
	return attempt, nil
}

func (c *oauthAttemptCache) IsLogin(rawState string, now time.Time) bool {
	stateHash := sha256.Sum256([]byte(rawState))
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeExpired(now)
	attempt, ok := c.attempts[stateHash]
	return ok && attempt.purpose == oauthPurposeLogin
}

func (c *oauthAttemptCache) removeExpired(now time.Time) {
	for stateHash, attempt := range c.attempts {
		if !attempt.expiresAt.After(now) {
			delete(c.attempts, stateHash)
		}
	}
}

func newOAuthState() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
