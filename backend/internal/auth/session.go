package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func NewSessionToken() (string, []byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := SessionTokenHash(token)
	return token, hash, nil
}

func SessionTokenHash(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}
