package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	Environment      string
	HTTPAddr         string
	DatabaseURL      string
	AppOrigin        string
	CookieSecure     bool
	AutoMigrate      bool
	SessionTTL       time.Duration
	DevAdminEmail    string
	DevAdminUsername string
	DevAdminPassword string
	DevUserEmail     string
	DevUserUsername  string
	DevUserPassword  string

	GitHubPublisherAppSlug      string
	GitHubPublisherClientID     string
	GitHubPublisherClientSecret string
	GitHubWebhookSecret         string
	GitHubPublisherAPIBaseURL   string
	GitHubPublisherPrivateKey   *rsa.PrivateKey
}

func Load() (Config, error) {
	devUserPassword := os.Getenv("DEV_USER_PASSWORD")
	if devUserPassword == "" {
		devUserPassword = os.Getenv("DEV_ADMIN_PASSWORD")
	}
	cfg := Config{
		Environment:                 valueOrDefault("APP_ENV", "production"),
		HTTPAddr:                    valueOrDefault("HTTP_ADDR", ":8080"),
		DatabaseURL:                 os.Getenv("DATABASE_URL"),
		AppOrigin:                   valueOrDefault("APP_ORIGIN", "http://localhost:3000"),
		SessionTTL:                  30 * 24 * time.Hour,
		DevAdminEmail:               os.Getenv("DEV_ADMIN_EMAIL"),
		DevAdminUsername:            os.Getenv("DEV_ADMIN_USERNAME"),
		DevAdminPassword:            os.Getenv("DEV_ADMIN_PASSWORD"),
		DevUserEmail:                os.Getenv("DEV_USER_EMAIL"),
		DevUserUsername:             os.Getenv("DEV_USER_USERNAME"),
		DevUserPassword:             devUserPassword,
		GitHubPublisherAppSlug:      os.Getenv("GITHUB_PUBLISHER_APP_SLUG"),
		GitHubPublisherClientID:     os.Getenv("GITHUB_PUBLISHER_CLIENT_ID"),
		GitHubPublisherClientSecret: os.Getenv("GITHUB_PUBLISHER_CLIENT_SECRET"),
		GitHubWebhookSecret:         os.Getenv("GITHUB_WEBHOOK_SECRET"),
		GitHubPublisherAPIBaseURL:   valueOrDefault("GITHUB_API_BASE_URL", "https://api.github.com"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	privateKeyPath := strings.TrimSpace(os.Getenv("GITHUB_PUBLISHER_PRIVATE_KEY_PATH"))
	githubConfigured := cfg.GitHubPublisherAppSlug != "" || cfg.GitHubPublisherClientID != "" || privateKeyPath != "" || cfg.GitHubWebhookSecret != ""
	if githubConfigured {
		if cfg.GitHubPublisherAppSlug == "" || cfg.GitHubPublisherClientID == "" || cfg.GitHubPublisherClientSecret == "" || privateKeyPath == "" || cfg.GitHubWebhookSecret == "" {
			return Config{}, errors.New("GITHUB_PUBLISHER_APP_SLUG, GITHUB_PUBLISHER_CLIENT_ID, GITHUB_PUBLISHER_CLIENT_SECRET, GITHUB_PUBLISHER_PRIVATE_KEY_PATH, and GITHUB_WEBHOOK_SECRET must be set together")
		}

		privateKeyPEM, err := os.ReadFile(privateKeyPath)
		if err != nil {
			return Config{}, fmt.Errorf("read github publisher private key: %w", err)
		}
		cfg.GitHubPublisherPrivateKey, err = parseRSAPrivateKey(privateKeyPEM)
		if err != nil {
			return Config{}, fmt.Errorf("parse github publisher private key: %w", err)
		}
		if err := validateHTTPURL("GITHUB_API_BASE_URL", cfg.GitHubPublisherAPIBaseURL); err != nil {
			return Config{}, err
		}
	}

	var err error
	cfg.CookieSecure, err = boolValue("COOKIE_SECURE", false)
	if err != nil {
		return Config{}, err
	}
	cfg.AutoMigrate, err = boolValue("AUTO_MIGRATE", false)
	if err != nil {
		return Config{}, err
	}

	if raw := os.Getenv("SESSION_TTL"); raw != "" {
		cfg.SessionTTL, err = time.ParseDuration(raw)
		if err != nil || cfg.SessionTTL <= 0 {
			return Config{}, fmt.Errorf("SESSION_TTL must be a positive duration")
		}
	}

	return cfg, nil
}

func validateHTTPURL(name, value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%s must be an absolute HTTP(S) URL", name)
	}
	return nil
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func boolValue(key string, fallback bool) (bool, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", key)
	}
	return value, nil
}

// GitHubAppJWT signs a short-lived token for GitHub App API requests.
func (c Config) GitHubAppJWT() (string, error) {
	if c.GitHubPublisherClientID == "" || c.GitHubPublisherPrivateKey == nil {
		return "", errors.New("github app credentials are not configured")
	}

	now := time.Now()

	claims := jwt.RegisteredClaims{
		Issuer:    c.GitHubPublisherClientID,
		IssuedAt:  jwt.NewNumericDate(now.Add(-60 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(now.Add(9 * time.Minute)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	signed, err := token.SignedString(c.GitHubPublisherPrivateKey)
	if err != nil {
		return "", fmt.Errorf("sign GitHub app JWT: %w", err)
	}

	return signed, nil
}

func parseRSAPrivateKey(privateKeyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return nil, errors.New("private key is not valid PEM")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKCS#1 or PKCS#8 key: %w", err)
	}

	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return key, nil
}
