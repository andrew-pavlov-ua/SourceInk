package auth_test

import (
	"context"
	"strings"
	"testing"

	"sourceink/backend/internal/auth"
)

func TestEnsureDevelopmentAdminDoesNothingOutsideDevelopment(t *testing.T) {
	if err := auth.EnsureDevelopmentAdmin(context.Background(), nil, "production", "", "", ""); err != nil {
		t.Fatalf("EnsureDevelopmentAdmin() error = %v", err)
	}
}

func TestEnsureDevelopmentAdminRequiresAllCredentials(t *testing.T) {
	err := auth.EnsureDevelopmentAdmin(context.Background(), nil, "development", "admin@example.com", "admin", "")
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("EnsureDevelopmentAdmin() error = %v, want required credentials error", err)
	}
}

func TestEnsureDevelopmentUserDoesNothingOutsideDevelopment(t *testing.T) {
	if err := auth.EnsureDevelopmentUser(context.Background(), nil, "production", "", "", ""); err != nil {
		t.Fatalf("EnsureDevelopmentUser() error = %v", err)
	}
}

func TestEnsureDevelopmentUserRequiresAllCredentials(t *testing.T) {
	err := auth.EnsureDevelopmentUser(context.Background(), nil, "development", "user@example.com", "sourceink-user", "")
	if err == nil || !strings.Contains(err.Error(), "DEV_USER_EMAIL") {
		t.Fatalf("EnsureDevelopmentUser() error = %v, want required credentials error", err)
	}
}
