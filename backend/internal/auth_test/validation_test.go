package auth_test

import (
	"testing"

	"sourceink/backend/internal/auth"
)

func TestNormalizeAndValidateRegistration(t *testing.T) {
	email, username, err := auth.NormalizeAndValidateRegistration(
		"  Writer@Example.com ",
		"git_writer",
		"long enough password",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if email != "writer@example.com" {
		t.Fatalf("email = %q", email)
	}
	if username != "git_writer" {
		t.Fatalf("username = %q", username)
	}
}

func TestRegistrationRejectsWeakInput(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		username string
		password string
	}{
		{"invalid email", "writer", "git_writer", "long enough password"},
		{"invalid username", "writer@example.com", "bad handle", "long enough password"},
		{"short password", "writer@example.com", "git_writer", "short"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := auth.NormalizeAndValidateRegistration(test.email, test.username, test.password); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
