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
