package auth_test

import (
	"bytes"
	"testing"

	"sourceink/backend/internal/auth"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	encoded, err := auth.HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	valid, err := auth.VerifyPassword("correct horse battery staple", encoded)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !valid {
		t.Fatal("expected password to verify")
	}

	valid, err = auth.VerifyPassword("wrong password", encoded)
	if err != nil {
		t.Fatalf("VerifyPassword wrong password: %v", err)
	}
	if valid {
		t.Fatal("wrong password verified")
	}
}

func TestSessionTokensAreRandomAndStoredAsHashes(t *testing.T) {
	first, firstHash, err := auth.NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken first: %v", err)
	}
	second, secondHash, err := auth.NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken second: %v", err)
	}
	if first == second || bytes.Equal(firstHash, secondHash) {
		t.Fatal("expected unique session tokens")
	}
	if bytes.Equal([]byte(first), firstHash) {
		t.Fatal("session token was not hashed")
	}
	if !bytes.Equal(firstHash, auth.SessionTokenHash(first)) {
		t.Fatal("stored hash does not match token")
	}
}
