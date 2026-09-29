package github

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"sourceink/backend/internal/model"
)

type installationStore interface {
	UserBySession(ctx context.Context, tokenHash []byte) (model.User, error)
	InstallationForUser(ctx context.Context, userID string) (model.GitHubInstallation, error)
	SaveInstallation(ctx context.Context, userID string, installation model.GitHubInstallation) error
	LinkGitHubUser(ctx context.Context, userID string, githubUser model.GitHubUser) (model.User, error)
	FindOrCreateUserByGitHub(ctx context.Context, githubUser model.GitHubUser, tokenHash []byte, expiresAt time.Time) (model.User, error)
}

type githubClient interface {
	GetInstallation(ctx context.Context, installationID int64) (*model.GitHubInstallation, error)
	ExchangeUserCode(
		ctx context.Context,
		code string,
		codeVerifier string,
		redirectURI string,
	) (string, error)
	UserCanAccessInstallation(ctx context.Context, userToken string, installationID int64) (bool, error)
	GetUser(ctx context.Context, userToken string) (model.GitHubUser, error)
}

type Service struct {
	client githubClient
	store  installationStore
}

func NewService(store installationStore, client githubClient) *Service {
	return &Service{store: store, client: client}
}

func (s *Service) UserBySession(ctx context.Context, tokenHash []byte) (model.User, error) {
	return s.store.UserBySession(ctx, tokenHash)
}

func (s *Service) InstallationForUser(ctx context.Context, userID string) (model.GitHubInstallation, error) {
	installation, err := s.store.InstallationForUser(ctx, userID)
	if err != nil {
		return model.GitHubInstallation{}, err
	}
	if installation.ID <= 0 {
		return model.GitHubInstallation{}, sql.ErrNoRows
	}
	return installation, nil
}

func (s *Service) ConnectInstallation(ctx context.Context, userID string, installationID int64) error {
	installation, err := s.client.GetInstallation(ctx, installationID)
	if err != nil {
		return err
	}
	return s.store.SaveInstallation(ctx, userID, *installation)
}

func (s *Service) RefreshInstallationForUser(ctx context.Context, userID string) (bool, error) {
	installation, err := s.InstallationForUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := s.ConnectInstallation(ctx, userID, installation.ID); err != nil {
		return true, err
	}
	return true, nil
}

func (s *Service) ExchangeUserCode(ctx context.Context, code, codeVerifier, redirectURI string) (string, error) {
	return s.client.ExchangeUserCode(ctx, code, codeVerifier, redirectURI)
}

func (s *Service) UserCanAccessInstallation(ctx context.Context, userToken string, installationID int64) (bool, error) {
	return s.client.UserCanAccessInstallation(ctx, userToken, installationID)
}

func (s *Service) LoginWithGitHub(ctx context.Context, userToken string, tokenHash []byte, expiresAt time.Time) (model.User, error) {
	githubUser, err := s.client.GetUser(ctx, userToken)
	if err != nil {
		return model.User{}, err
	}
	return s.store.FindOrCreateUserByGitHub(ctx, githubUser, tokenHash, expiresAt)
}

func (s *Service) LinkGitHubUser(ctx context.Context, userID, userToken string) (model.User, error) {
	githubUser, err := s.client.GetUser(ctx, userToken)
	if err != nil {
		return model.User{}, err
	}
	return s.store.LinkGitHubUser(ctx, userID, githubUser)
}
