package github

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"sourceink/backend/internal/model"
)

type serviceTestStore struct {
	user         model.User
	userErr      error
	saved        bool
	savedUserID  string
	installation model.GitHubInstallation
	saveErr      error
}

func (s *serviceTestStore) UserBySession(context.Context, []byte) (model.User, error) {
	return s.user, s.userErr
}

func (s *serviceTestStore) InstallationForUser(context.Context, string) (model.GitHubInstallation, error) {
	if s.installation.ID == 0 {
		return model.GitHubInstallation{}, sql.ErrNoRows
	}
	return s.installation, nil
}

func (s *serviceTestStore) SaveInstallation(_ context.Context, userID string, installation model.GitHubInstallation) error {
	s.saved = true
	s.savedUserID = userID
	s.installation = installation
	return s.saveErr
}

func (s *serviceTestStore) FindOrCreateUserByGitHub(_ context.Context, _ model.GitHubUser, _ []byte, _ time.Time) (model.User, error) {
	return s.user, s.userErr
}

type serviceTestClient struct {
	installation         *model.GitHubInstallation
	getInstallationID    int64
	err                  error
	userToken            string
	exchangeErr          error
	canAccess            bool
	accessErr            error
	exchangeCode         string
	exchangeVerifier     string
	accessToken          string
	accessInstallationID int64
}

func (c *serviceTestClient) GetInstallation(_ context.Context, installationID int64) (*model.GitHubInstallation, error) {
	c.getInstallationID = installationID
	return c.installation, c.err
}

func (c *serviceTestClient) ExchangeUserCode(_ context.Context, code, verifier, _ string) (string, error) {
	c.exchangeCode = code
	c.exchangeVerifier = verifier
	return c.userToken, c.exchangeErr
}

func (c *serviceTestClient) UserCanAccessInstallation(_ context.Context, token string, installationID int64) (bool, error) {
	c.accessToken = token
	c.accessInstallationID = installationID
	return c.canAccess, c.accessErr
}

func (c *serviceTestClient) GetUser(context.Context, string) (model.GitHubUser, error) {
	return model.GitHubUser{ID: 9, Login: "octocat"}, nil
}

func TestConnectInstallationChecksGitHubBeforeSaving(t *testing.T) {
	store := &serviceTestStore{}
	client := &serviceTestClient{err: errors.New("not a SourceInk installation")}
	service := NewService(store, client)

	if err := service.ConnectInstallation(context.Background(), "user-id", 42); err == nil {
		t.Fatal("ConnectInstallation() accepted a failed GitHub check")
	}
	if store.saved {
		t.Fatal("installation was saved after GitHub rejected it")
	}
}

func TestConnectInstallationSavesVerifiedMetadata(t *testing.T) {
	installation := &model.GitHubInstallation{
		ID: 42, AccountID: 9, AccountLogin: "octocat", AccountType: "User", RepositorySelection: "selected",
	}
	store := &serviceTestStore{}
	service := NewService(store, &serviceTestClient{installation: installation})

	if err := service.ConnectInstallation(context.Background(), "user-id", 42); err != nil {
		t.Fatalf("ConnectInstallation() error = %v", err)
	}
	if !store.saved || store.savedUserID != "user-id" || store.installation.ID != 42 {
		t.Fatalf("saved = %t, user = %q, installation = %#v", store.saved, store.savedUserID, store.installation)
	}
}

func TestRefreshInstallationForUserRefreshesExistingRepositories(t *testing.T) {
	stored := model.GitHubInstallation{ID: 42, AccountLogin: "octocat", AccountType: "User"}
	refreshed := &model.GitHubInstallation{
		ID: 42, AccountID: 9, AccountLogin: "octocat", AccountType: "User", RepositorySelection: "selected",
		Repositories: []model.Repository{{GitHubID: 100, FullName: "octocat/docs"}},
	}
	store := &serviceTestStore{installation: stored}
	client := &serviceTestClient{installation: refreshed}
	service := NewService(store, client)

	installed, err := service.RefreshInstallationForUser(context.Background(), "user-id")
	if err != nil {
		t.Fatalf("RefreshInstallationForUser() error = %v", err)
	}
	if !installed || client.getInstallationID != 42 || !store.saved {
		t.Fatalf("installed = %t, requested installation = %d, saved = %t", installed, client.getInstallationID, store.saved)
	}
	if len(store.installation.Repositories) != 1 || store.installation.Repositories[0].FullName != "octocat/docs" {
		t.Fatalf("saved repositories = %#v", store.installation.Repositories)
	}
}

func TestRefreshInstallationForUserSkipsUsersWithoutInstallation(t *testing.T) {
	store := &serviceTestStore{}
	client := &serviceTestClient{}
	service := NewService(store, client)

	installed, err := service.RefreshInstallationForUser(context.Background(), "user-id")
	if err != nil || installed {
		t.Fatalf("installed = %t, error = %v", installed, err)
	}
	if client.getInstallationID != 0 || store.saved {
		t.Fatalf("requested installation = %d, saved = %t", client.getInstallationID, store.saved)
	}
}
