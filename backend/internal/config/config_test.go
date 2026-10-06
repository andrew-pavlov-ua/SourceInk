package config

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestLoadRequiresSecureCookiesInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://sourceink:password@localhost/sourceink")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("GITHUB_PUBLISHER_APP_SLUG", "")
	t.Setenv("GITHUB_PUBLISHER_CLIENT_ID", "")
	t.Setenv("GITHUB_PUBLISHER_CLIENT_SECRET", "")
	t.Setenv("GITHUB_PUBLISHER_PRIVATE_KEY_PATH", "")
	t.Setenv("GITHUB_WEBHOOK_SECRET", "")

	if _, err := Load(); err == nil || err.Error() != "COOKIE_SECURE must be true when APP_ENV=production" {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestGitHubAppJWT(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		GitHubPublisherClientID:   "Iv1.example",
		GitHubPublisherPrivateKey: privateKey,
	}

	rawToken, err := cfg.GitHubAppJWT()
	if err != nil {
		t.Fatalf("GitHubAppJWT() error = %v", err)
	}

	claims := jwt.RegisteredClaims{}
	_, err = jwt.ParseWithClaims(rawToken, &claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodRS256 {
			t.Fatalf("signing method = %v, want RS256", token.Method.Alg())
		}
		return &privateKey.PublicKey, nil
	})
	if err != nil {
		t.Fatalf("parse signed token: %v", err)
	}

	if claims.Issuer != cfg.GitHubPublisherClientID {
		t.Errorf("issuer = %q, want %q", claims.Issuer, cfg.GitHubPublisherClientID)
	}
	if claims.IssuedAt == nil || claims.ExpiresAt == nil {
		t.Fatal("issued-at and expiration claims must be set")
	}
	if lifetime := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time); lifetime > 10*time.Minute {
		t.Errorf("token lifetime = %s, must not exceed 10 minutes", lifetime)
	}
}
