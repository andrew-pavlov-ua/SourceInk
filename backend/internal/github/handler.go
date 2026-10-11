package github

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"sourceink/backend/internal/auth"
	"sourceink/backend/internal/model"
)

const maxOAuthStartsPerMinute = 120

type Handler struct {
	service           *Service
	cookieSecure      bool
	logger            *slog.Logger
	installationURL   string
	repositoriesURL   string
	settingsURL       string
	successURL        string
	loginSuccessURL   string
	oauthAttempts     *oauthAttemptCache
	oauthStartLimiter *auth.Limiter

	githubClientID   string
	oauthCallbackURL string
	sessionTTL       time.Duration

	// SyncAfterConnect runs in the background once an installation is
	// connected so the dashboard shows drafts without waiting for a push.
	SyncAfterConnect func(context.Context) error
}

func NewHandler(service *Service, appSlug, appOrigin, githubClientID, oauthCallbackURL string, cookieSecure bool, sessionTTL time.Duration, logger *slog.Logger) Handler {
	repositoriesURL := strings.TrimRight(appOrigin, "/") + "/dashboard/repositories"
	return Handler{
		service:           service,
		logger:            logger,
		cookieSecure:      cookieSecure,
		installationURL:   "https://github.com/apps/" + url.PathEscape(appSlug) + "/installations/new",
		repositoriesURL:   repositoriesURL,
		settingsURL:       strings.TrimRight(appOrigin, "/") + "/dashboard/settings",
		successURL:        repositoriesURL + "?github=connected",
		loginSuccessURL:   strings.TrimRight(appOrigin, "/") + "/dashboard",
		oauthAttempts:     newOAuthAttemptCache(),
		oauthStartLimiter: auth.NewLimiter(maxOAuthStartsPerMinute, time.Minute),
		githubClientID:    githubClientID,
		oauthCallbackURL:  oauthCallbackURL,
		sessionTTL:        sessionTTL,
	}
}

func (h *Handler) InstallationRedirect(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	installation, err := h.service.InstallationForUser(r.Context(), user.ID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Redirect(w, r, h.installationURL, http.StatusFound)
		return
	}
	if err != nil {
		h.logger.Error("load GitHub installation", "error", err)
		auth.WriteError(w, http.StatusInternalServerError, "could not load GitHub installation")
		return
	}

	configurationURL, err := installationConfigurationURL(installation)
	if err != nil {
		h.logger.Error("build GitHub installation configuration URL", "error", err)
		auth.WriteError(w, http.StatusInternalServerError, "could not configure GitHub installation")
		return
	}
	http.Redirect(w, r, configurationURL, http.StatusFound)
}

func installationConfigurationURL(installation model.GitHubInstallation) (string, error) {
	if installation.ID <= 0 || strings.TrimSpace(installation.AccountLogin) == "" {
		return "", errors.New("invalid GitHub installation")
	}
	switch installation.AccountType {
	case "User":
		return fmt.Sprintf("https://github.com/settings/installations/%d", installation.ID), nil
	case "Organization":
		return fmt.Sprintf("https://github.com/organizations/%s/settings/installations/%d", url.PathEscape(installation.AccountLogin), installation.ID), nil
	default:
		return "", fmt.Errorf("unsupported GitHub installation account type %q", installation.AccountType)
	}
}

// SetupInstallation handles GitHub's post-install redirect. OAuth confirms the
// signed-in user can access that installation.
func (h *Handler) SetupInstallation(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	installationID, err := strconv.ParseInt(r.URL.Query().Get("installation_id"), 10, 64)
	if err != nil || installationID <= 0 {
		auth.WriteError(w, http.StatusBadRequest, "invalid GitHub installation response")
		return
	}
	h.redirectToAuth(w, r, user.ID, installationID, oauthPurposeConnectInstallation)
}

// LoginRedirect starts GitHub OAuth without a SourceInk session. State binds
// the callback to this short-lived attempt.
func (h *Handler) LoginRedirect(w http.ResponseWriter, r *http.Request) {
	h.redirectToAuth(w, r, "", 0, oauthPurposeLogin)
}

// ConnectAccountRedirect links GitHub to the signed-in SourceInk account.
func (h *Handler) ConnectAccountRedirect(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	if user.GitHubUserID != nil {
		http.Redirect(w, r, h.settingsURL+"?github=already-connected", http.StatusFound)
		return
	}
	h.redirectToAuth(w, r, user.ID, 0, oauthPurposeConnectAccount)
}

func (h *Handler) currentUser(w http.ResponseWriter, r *http.Request) (model.User, bool) {
	return auth.SessionUser(w, r, true, h.cookieSecure, h.logger, h.service.UserBySession)
}

func (h *Handler) redirectToAuth(w http.ResponseWriter, r *http.Request, userID string, pendingInstallationID int64, purpose oauthPurpose) {
	if !h.oauthStartLimiter.Allow("oauth-start") {
		auth.WriteError(w, http.StatusTooManyRequests, "too many GitHub authorization attempts; try again later")
		return
	}

	now := time.Now()
	if strings.TrimSpace(h.githubClientID) == "" || strings.TrimSpace(h.oauthCallbackURL) == "" {
		h.logger.Error("GitHub OAuth callback is not configured")
		auth.WriteError(w, http.StatusInternalServerError, "GitHub connection is not configured")
		return
	}
	state, err := newOAuthState()
	if err != nil {
		auth.WriteError(w, http.StatusInternalServerError, "could not start GitHub connection")
		return
	}
	verifier, err := newOAuthState()
	if err != nil {
		auth.WriteError(w, http.StatusInternalServerError, "could not start GitHub connection")
		return
	}
	var loginBinding string
	if purpose == oauthPurposeLogin {
		loginBinding, err = newOAuthState()
		if err != nil {
			auth.WriteError(w, http.StatusInternalServerError, "could not start GitHub connection")
			return
		}
	}
	attempt := oauthAttempt{
		userID:                userID,
		expiresAt:             now.Add(oauthAttemptTTL),
		pendingInstallationID: pendingInstallationID,
		pkceVerifier:          verifier,
		purpose:               purpose,
	}
	if purpose == oauthPurposeLogin {
		attempt.loginBindingHash = sha256.Sum256([]byte(loginBinding))
	}
	if err := h.oauthAttempts.Put(state, attempt, now); err != nil {
		h.logger.Error("store GitHub OAuth attempt", "error", err)
		auth.WriteError(w, http.StatusServiceUnavailable, "could not start GitHub connection")
		return
	}
	if purpose == oauthPurposeLogin {
		expiresAt := now.Add(oauthAttemptTTL)
		http.SetCookie(w, &http.Cookie{Name: loginBindingCookieName, Value: loginBinding, Path: "/api/github/callback", Expires: expiresAt, MaxAge: int(oauthAttemptTTL.Seconds()), HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode})
	}

	authorizeURL, _ := url.Parse("https://github.com/login/oauth/authorize")
	query := authorizeURL.Query()
	query.Set("client_id", h.githubClientID)
	query.Set("redirect_uri", h.oauthCallbackURL)
	query.Set("state", state)
	query.Set("code_challenge", pkceChallenge(verifier))
	query.Set("code_challenge_method", "S256")
	if purpose == oauthPurposeLogin || purpose == oauthPurposeConnectAccount {
		query.Set("scope", "read:user")
	}
	authorizeURL.RawQuery = query.Encode()
	http.Redirect(w, r, authorizeURL.String(), http.StatusFound)
}

func (h *Handler) ValidateOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if h.oauthAttempts.IsLogin(r.URL.Query().Get("state"), time.Now()) {
		h.ValidateLoginCallback(w, r)
		return
	}
	if r.URL.Query().Get("error") != "" {
		auth.WriteError(w, http.StatusForbidden, "GitHub authorization was not completed")
		return
	}
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		auth.WriteError(w, http.StatusBadRequest, "missing GitHub authorization response")
		return
	}
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	attempt, err := h.oauthAttempts.Consume(user.ID, state, time.Now())
	if err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid or expired GitHub authorization response")
		return
	}

	userToken, err := h.service.ExchangeUserCode(r.Context(), code, attempt.pkceVerifier, h.oauthCallbackURL)
	if err != nil {
		h.logger.Error("exchange GitHub OAuth code", "error", err)
		auth.WriteError(w, http.StatusBadGateway, "GitHub authorization failed")
		return
	}
	if attempt.purpose == oauthPurposeConnectAccount {
		_, err = h.service.LinkGitHubUser(r.Context(), user.ID, userToken)
		switch {
		case errors.Is(err, model.ErrGitHubAccountConflict):
			http.Redirect(w, r, h.settingsURL+"?github=conflict", http.StatusFound)
		case err != nil:
			h.logger.Error("connect GitHub account", "error", err, "user_id", user.ID)
			http.Redirect(w, r, h.settingsURL+"?github=failed", http.StatusFound)
		default:
			http.Redirect(w, r, h.settingsURL+"?github=connected", http.StatusFound)
		}
		return
	}
	if attempt.purpose != oauthPurposeConnectInstallation {
		auth.WriteError(w, http.StatusBadRequest, "invalid GitHub authorization response")
		return
	}
	canAccess, err := h.service.UserCanAccessInstallation(r.Context(), userToken, attempt.pendingInstallationID)
	if err != nil {
		h.logger.Error("verify GitHub user installation access", "error", err)
		auth.WriteError(w, http.StatusBadGateway, "could not verify GitHub installation access")
		return
	}
	if !canAccess {
		auth.WriteError(w, http.StatusForbidden, "you cannot connect this GitHub installation")
		return
	}

	err = h.service.ConnectInstallation(r.Context(), user.ID, attempt.pendingInstallationID)
	switch {
	case errors.Is(err, model.ErrGitHubInstallationConflict):
		auth.WriteError(w, http.StatusConflict, "this GitHub installation is already connected")
	case err != nil:
		h.logger.Error("connect github installation", "error", err)
		auth.WriteError(w, http.StatusInternalServerError, "could not connect GitHub installation")
	default:
		if h.SyncAfterConnect != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				if err := h.SyncAfterConnect(ctx); err != nil {
					h.logger.Error("sync articles after connecting installation", "error", err)
				}
			}()
		}
		http.Redirect(w, r, h.successURL, http.StatusFound)
	}
}

func (h *Handler) ValidateLoginCallback(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("error") != "" {
		auth.WriteError(w, http.StatusForbidden, "GitHub authorization was not completed")
		return
	}
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		auth.WriteError(w, http.StatusBadRequest, "missing GitHub authorization response")
		return
	}
	binding, err := r.Cookie(loginBindingCookieName)
	if err != nil || binding.Value == "" {
		auth.WriteError(w, http.StatusBadRequest, "invalid or expired GitHub authorization response")
		return
	}
	attempt, err := h.oauthAttempts.ConsumeLogin(state, binding.Value, time.Now())
	if err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid or expired GitHub authorization response")
		return
	}
	userToken, err := h.service.ExchangeUserCode(r.Context(), code, attempt.pkceVerifier, h.oauthCallbackURL)
	if err != nil {
		h.logger.Error("exchange GitHub login code", "error", err)
		auth.WriteError(w, http.StatusBadGateway, "GitHub authorization failed")
		return
	}
	token, tokenHash, err := auth.NewSessionToken()
	if err != nil {
		h.logger.Error("create GitHub login session", "error", err)
		auth.WriteError(w, http.StatusInternalServerError, "login failed")
		return
	}
	expiresAt := time.Now().Add(h.sessionTTL)
	user, err := h.service.LoginWithGitHub(r.Context(), userToken, tokenHash, expiresAt)
	if errors.Is(err, model.ErrGitHubUsernameTaken) {
		auth.WriteError(w, http.StatusConflict, "a SourceInk account already uses this GitHub username; sign in with that account first")
		return
	}
	if err != nil {
		h.logger.Error("complete GitHub login", "error", err)
		auth.WriteError(w, http.StatusInternalServerError, "GitHub login failed")
		return
	}
	auth.SetSessionCookie(w, token, expiresAt, h.cookieSecure)
	installed, refreshErr := h.service.RefreshInstallationForUser(r.Context(), user.ID)
	if refreshErr != nil {
		h.logger.Error("refresh GitHub installation after login", "error", refreshErr)
		http.Redirect(w, r, h.repositoriesURL+"?github=refresh-failed", http.StatusFound)
		return
	}
	if installed {
		http.Redirect(w, r, h.repositoriesURL+"?github=existing", http.StatusFound)
		return
	}
	http.Redirect(w, r, h.loginSuccessURL, http.StatusFound)
}
