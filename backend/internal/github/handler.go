package github

import (
	"database/sql"
	"encoding/json"
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

type Handler struct {
	service         *Service
	cookieSecure    bool
	logger          *slog.Logger
	installationURL string
	repositoriesURL string
	successURL      string
	loginSuccessURL string
	oauthAttempts   *oauthAttemptCache

	githubClientID   string
	oauthCallbackURL string
	sessionTTL       time.Duration
}

func NewHandler(service *Service, appSlug, appOrigin, githubClientID, oauthCallbackURL string, cookieSecure bool, sessionTTL time.Duration, logger *slog.Logger) Handler {
	repositoriesURL := strings.TrimRight(appOrigin, "/") + "/dashboard/repositories"
	return Handler{
		service:          service,
		logger:           logger,
		cookieSecure:     cookieSecure,
		installationURL:  "https://github.com/apps/" + url.PathEscape(appSlug) + "/installations/new",
		repositoriesURL:  repositoriesURL,
		successURL:       repositoriesURL + "?github=connected",
		loginSuccessURL:  strings.TrimRight(appOrigin, "/") + "/dashboard",
		oauthAttempts:    newOAuthAttemptCache(),
		githubClientID:   githubClientID,
		oauthCallbackURL: oauthCallbackURL,
		sessionTTL:       sessionTTL,
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
		writeError(w, http.StatusInternalServerError, "could not load GitHub installation")
		return
	}

	configurationURL, err := installationConfigurationURL(installation)
	if err != nil {
		h.logger.Error("build GitHub installation configuration URL", "error", err)
		writeError(w, http.StatusInternalServerError, "could not configure GitHub installation")
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

// SetupInstallation receives GitHub's App setup redirect. OAuth must confirm
// that the signed-in GitHub user can access the installation ID.
func (h *Handler) SetupInstallation(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	installationID, err := strconv.ParseInt(r.URL.Query().Get("installation_id"), 10, 64)
	if err != nil || installationID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid GitHub installation response")
		return
	}
	h.redirectToAuth(w, r, user.ID, installationID, oauthPurposeConnectInstallation)
}

// LoginRedirect starts GitHub OAuth without a SourceInk session. A short-lived,
// single-use state value binds the callback to this attempt.
func (h *Handler) LoginRedirect(w http.ResponseWriter, r *http.Request) {
	h.redirectToAuth(w, r, "", 0, oauthPurposeLogin)
}

func (h *Handler) currentUser(w http.ResponseWriter, r *http.Request) (model.User, bool) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return model.User{}, false
	}
	user, err := h.service.UserBySession(r.Context(), auth.SessionTokenHash(cookie.Value))
	if errors.Is(err, sql.ErrNoRows) {
		h.clearSessionCookie(w)
		writeError(w, http.StatusUnauthorized, "authentication required")
		return model.User{}, false
	}
	if err != nil {
		h.logger.Error("load session user", "error", err)
		writeError(w, http.StatusInternalServerError, "could not load account")
		return model.User{}, false
	}
	return user, true
}

func (h *Handler) redirectToAuth(w http.ResponseWriter, r *http.Request, userID string, pendingInstallationID int64, purpose oauthPurpose) {
	now := time.Now()
	if strings.TrimSpace(h.githubClientID) == "" || strings.TrimSpace(h.oauthCallbackURL) == "" {
		h.logger.Error("GitHub OAuth callback is not configured")
		writeError(w, http.StatusInternalServerError, "GitHub connection is not configured")
		return
	}
	state, err := newOAuthState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start GitHub connection")
		return
	}
	verifier, err := newOAuthState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start GitHub connection")
		return
	}
	attempt := oauthAttempt{
		userID:                userID,
		expiresAt:             now.Add(oauthAttemptTTL),
		pendingInstallationID: pendingInstallationID,
		pkceVerifier:          verifier,
		purpose:               purpose,
	}
	if err := h.oauthAttempts.Put(state, attempt, now); err != nil {
		h.logger.Error("store GitHub OAuth attempt", "error", err)
		writeError(w, http.StatusServiceUnavailable, "could not start GitHub connection")
		return
	}

	authorizeURL, _ := url.Parse("https://github.com/login/oauth/authorize")
	query := authorizeURL.Query()
	query.Set("client_id", h.githubClientID)
	query.Set("redirect_uri", h.oauthCallbackURL)
	query.Set("state", state)
	query.Set("code_challenge", pkceChallenge(verifier))
	query.Set("code_challenge_method", "S256")
	if purpose == oauthPurposeLogin {
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
		writeError(w, http.StatusForbidden, "GitHub authorization was not completed")
		return
	}
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		writeError(w, http.StatusBadRequest, "missing GitHub authorization response")
		return
	}
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	attempt, err := h.oauthAttempts.Consume(user.ID, state, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or expired GitHub authorization response")
		return
	}

	userToken, err := h.service.ExchangeUserCode(r.Context(), code, attempt.pkceVerifier, h.oauthCallbackURL)
	if err != nil {
		h.logger.Error("exchange GitHub OAuth code", "error", err)
		writeError(w, http.StatusBadGateway, "GitHub authorization failed")
		return
	}
	canAccess, err := h.service.UserCanAccessInstallation(r.Context(), userToken, attempt.pendingInstallationID)
	if err != nil {
		h.logger.Error("verify GitHub user installation access", "error", err)
		writeError(w, http.StatusBadGateway, "could not verify GitHub installation access")
		return
	}
	if !canAccess {
		writeError(w, http.StatusForbidden, "you cannot connect this GitHub installation")
		return
	}

	err = h.service.ConnectInstallation(r.Context(), user.ID, attempt.pendingInstallationID)
	switch {
	case errors.Is(err, model.ErrGitHubInstallationConflict):
		writeError(w, http.StatusConflict, "this GitHub installation is already connected")
	case err != nil:
		h.logger.Error("connect github installation", "error", err)
		writeError(w, http.StatusInternalServerError, "could not connect GitHub installation")
	default:
		http.Redirect(w, r, h.successURL, http.StatusFound)
	}
}

func (h *Handler) ValidateLoginCallback(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("error") != "" {
		writeError(w, http.StatusForbidden, "GitHub authorization was not completed")
		return
	}
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		writeError(w, http.StatusBadRequest, "missing GitHub authorization response")
		return
	}
	attempt, err := h.oauthAttempts.ConsumeLogin(state, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or expired GitHub authorization response")
		return
	}
	userToken, err := h.service.ExchangeUserCode(r.Context(), code, attempt.pkceVerifier, h.oauthCallbackURL)
	if err != nil {
		h.logger.Error("exchange GitHub login code", "error", err)
		writeError(w, http.StatusBadGateway, "GitHub authorization failed")
		return
	}
	token, tokenHash, err := auth.NewSessionToken()
	if err != nil {
		h.logger.Error("create GitHub login session", "error", err)
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	expiresAt := time.Now().Add(h.sessionTTL)
	user, err := h.service.LoginWithGitHub(r.Context(), userToken, tokenHash, expiresAt)
	if errors.Is(err, model.ErrGitHubUsernameTaken) {
		writeError(w, http.StatusConflict, "a SourceInk account already uses this GitHub username; sign in with that account first")
		return
	}
	if err != nil {
		h.logger.Error("complete GitHub login", "error", err)
		writeError(w, http.StatusInternalServerError, "GitHub login failed")
		return
	}
	h.setLoginSessionCookie(w, token, expiresAt)
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

func (h *Handler) setLoginSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{Name: auth.SessionCookieName, Value: token, Path: "/", Expires: expiresAt, MaxAge: int(time.Until(expiresAt).Seconds()), HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
