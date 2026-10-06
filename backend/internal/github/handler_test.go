package github

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"sourceink/backend/internal/auth"
	"sourceink/backend/internal/model"
)

func TestInstallationRedirectRequiresSession(t *testing.T) {
	store := &serviceTestStore{user: model.User{ID: "user-id"}}
	handler := newTestHandler(store, &serviceTestClient{})

	unauthenticated := httptest.NewRecorder()
	handler.InstallationRedirect(unauthenticated, httptest.NewRequest(http.MethodGet, "/api/github/install", nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", unauthenticated.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/github/install", nil)
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.InstallationRedirect(response, request)
	if response.Code != http.StatusFound {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	location, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Path != "/apps/sourceink/installations/new" || location.RawQuery != "" {
		t.Fatalf("redirect = %q", location.String())
	}
}

func TestInstallationRedirectUsesExistingInstallationConfiguration(t *testing.T) {
	store := &serviceTestStore{
		user:         model.User{ID: "user-id"},
		installation: model.GitHubInstallation{ID: 42, AccountLogin: "octocat", AccountType: "User"},
	}
	handler := newTestHandler(store, &serviceTestClient{})
	request := httptest.NewRequest(http.MethodGet, "/api/github/install", nil)
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()

	handler.InstallationRedirect(response, request)
	if response.Code != http.StatusFound {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Location"); got != "https://github.com/settings/installations/42" {
		t.Fatalf("redirect = %q", got)
	}
}

func TestSetupInstallationStartsOAuthWithoutSaving(t *testing.T) {
	store := &serviceTestStore{user: model.User{ID: "user-id"}}
	handler := newTestHandler(store, &serviceTestClient{})
	request := httptest.NewRequest(http.MethodGet, "/api/github/setup?installation_id=42", nil)
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()

	handler.SetupInstallation(response, request)
	if response.Code != http.StatusFound {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if store.saved {
		t.Fatal("setup saved an installation before GitHub user authorization")
	}
	location, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Host != "github.com" || location.Path != "/login/oauth/authorize" {
		t.Fatalf("redirect = %q", location.String())
	}
	query := location.Query()
	if query.Get("client_id") != "client-id" || query.Get("redirect_uri") != "http://localhost:3000/api/github/callback" ||
		query.Get("state") == "" || query.Get("code_challenge") == "" || query.Get("code_challenge_method") != "S256" {
		t.Fatalf("OAuth query = %#v", query)
	}
}

func TestValidateOAuthCallbackSavesOnlyVerifiedInstallation(t *testing.T) {
	store := &serviceTestStore{user: model.User{ID: "user-id"}}
	client := &serviceTestClient{
		userToken: "user-token",
		canAccess: true,
		installation: &model.GitHubInstallation{
			ID: 42, AccountID: 9, AccountLogin: "octocat", AccountType: "User", RepositorySelection: "selected",
		},
	}
	handler := newTestHandler(store, client)
	setupRequest := httptest.NewRequest(http.MethodGet, "/api/github/setup?installation_id=42", nil)
	setupRequest.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-token"})
	setupResponse := httptest.NewRecorder()
	handler.SetupInstallation(setupResponse, setupRequest)
	state := mustRedirectURL(t, setupResponse).Query().Get("state")

	callbackRequest := httptest.NewRequest(http.MethodGet, "/api/github/callback?code=code-value&state="+url.QueryEscape(state), nil)
	callbackRequest.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-token"})
	callbackResponse := httptest.NewRecorder()
	handler.ValidateOAuthCallback(callbackResponse, callbackRequest)

	if callbackResponse.Code != http.StatusFound {
		t.Fatalf("status = %d, body = %s", callbackResponse.Code, callbackResponse.Body.String())
	}
	if !store.saved || store.savedUserID != "user-id" || store.installation.ID != 42 {
		t.Fatalf("saved = %t, user = %q, installation = %#v", store.saved, store.savedUserID, store.installation)
	}
	if client.exchangeCode != "code-value" || client.exchangeVerifier == "" || client.accessToken != "user-token" || client.accessInstallationID != 42 {
		t.Fatalf("exchange/access calls were not made correctly: %#v", client)
	}
}

func TestValidateOAuthCallbackRejectsUnknownState(t *testing.T) {
	store := &serviceTestStore{user: model.User{ID: "user-id"}}
	client := &serviceTestClient{}
	handler := newTestHandler(store, client)
	request := httptest.NewRequest(http.MethodGet, "/api/github/callback?code=code-value&state=unknown", nil)
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()

	handler.ValidateOAuthCallback(response, request)
	if response.Code != http.StatusBadRequest || store.saved || client.exchangeCode != "" {
		t.Fatalf("status = %d, saved = %t, exchanged = %q", response.Code, store.saved, client.exchangeCode)
	}
}

func TestGitHubLoginCreatesSessionWithoutExistingSourceInkSession(t *testing.T) {
	store := &serviceTestStore{user: model.User{ID: "user-id", Username: "octocat"}}
	client := &serviceTestClient{userToken: "user-token"}
	handler := newTestHandler(store, client)

	start := httptest.NewRecorder()
	handler.LoginRedirect(start, httptest.NewRequest(http.MethodGet, "/api/auth/github", nil))
	if start.Code != http.StatusFound {
		t.Fatalf("start status = %d", start.Code)
	}
	state := mustRedirectURL(t, start).Query().Get("state")
	binding := mustLoginBindingCookie(t, start)

	callback := httptest.NewRecorder()
	callbackRequest := httptest.NewRequest(http.MethodGet, "/api/github/callback?code=code-value&state="+url.QueryEscape(state), nil)
	callbackRequest.AddCookie(binding)
	handler.ValidateOAuthCallback(callback, callbackRequest)
	if callback.Code != http.StatusFound {
		t.Fatalf("callback status = %d, body = %s", callback.Code, callback.Body.String())
	}
	if callback.Header().Get("Location") != "http://localhost:3000/dashboard" {
		t.Fatalf("location = %q", callback.Header().Get("Location"))
	}
	if client.exchangeCode != "code-value" || client.accessToken != "" {
		t.Fatalf("client = %#v", client)
	}
	if len(callback.Result().Cookies()) == 0 || callback.Result().Cookies()[0].Name != auth.SessionCookieName {
		t.Fatal("GitHub login did not set the session cookie")
	}
}

func TestGitHubLoginCallbackRequiresTheInitiatingBrowserBinding(t *testing.T) {
	store := &serviceTestStore{user: model.User{ID: "user-id", Username: "octocat"}}
	client := &serviceTestClient{userToken: "user-token"}
	handler := newTestHandler(store, client)

	start := httptest.NewRecorder()
	handler.LoginRedirect(start, httptest.NewRequest(http.MethodGet, "/api/auth/github", nil))
	state := mustRedirectURL(t, start).Query().Get("state")

	callback := httptest.NewRecorder()
	handler.ValidateOAuthCallback(callback, httptest.NewRequest(http.MethodGet, "/api/github/callback?code=code-value&state="+url.QueryEscape(state), nil))
	if callback.Code != http.StatusBadRequest || client.exchangeCode != "" {
		t.Fatalf("status = %d, exchanged = %q", callback.Code, client.exchangeCode)
	}
}

func TestGitHubOAuthStartsUseSharedAdmissionLimit(t *testing.T) {
	handler := newTestHandler(&serviceTestStore{}, &serviceTestClient{})
	handler.oauthStartLimiter = auth.NewLimiter(1, time.Hour)

	first := httptest.NewRecorder()
	handler.LoginRedirect(first, httptest.NewRequest(http.MethodGet, "/api/auth/github", nil))
	if first.Code != http.StatusFound {
		t.Fatalf("first login start status = %d", first.Code)
	}
	if len(handler.oauthAttempts.attempts) != 1 {
		t.Fatalf("OAuth attempts after first start = %d, want 1", len(handler.oauthAttempts.attempts))
	}

	second := httptest.NewRecorder()
	handler.LoginRedirect(second, httptest.NewRequest(http.MethodGet, "/api/auth/github", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second login start status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
	if len(handler.oauthAttempts.attempts) != 1 {
		t.Fatalf("OAuth attempts after rejected start = %d, want 1", len(handler.oauthAttempts.attempts))
	}
}

func TestConnectAccountLinksGitHubToSignedInUser(t *testing.T) {
	store := &serviceTestStore{user: model.User{ID: "email-user", Email: "writer@example.com", Username: "writer"}}
	client := &serviceTestClient{userToken: "user-token"}
	handler := newTestHandler(store, client)

	startRequest := httptest.NewRequest(http.MethodGet, "/api/github/connect", nil)
	startRequest.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-token"})
	start := httptest.NewRecorder()
	handler.ConnectAccountRedirect(start, startRequest)
	if start.Code != http.StatusFound {
		t.Fatalf("start status = %d, body = %s", start.Code, start.Body.String())
	}
	state := mustRedirectURL(t, start).Query().Get("state")

	callbackRequest := httptest.NewRequest(http.MethodGet, "/api/github/callback?code=code-value&state="+url.QueryEscape(state), nil)
	callbackRequest.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-token"})
	callback := httptest.NewRecorder()
	handler.ValidateOAuthCallback(callback, callbackRequest)

	if callback.Code != http.StatusFound {
		t.Fatalf("callback status = %d, body = %s", callback.Code, callback.Body.String())
	}
	if callback.Header().Get("Location") != "http://localhost:3000/dashboard/settings?github=connected" {
		t.Fatalf("location = %q", callback.Header().Get("Location"))
	}
	if !store.linked || store.linkedUserID != "email-user" || store.linkedGitHub.ID != 9 {
		t.Fatalf("linked = %t, user = %q, GitHub user = %#v", store.linked, store.linkedUserID, store.linkedGitHub)
	}
}

func TestGitHubLoginRefreshesExistingInstallation(t *testing.T) {
	store := &serviceTestStore{
		user: model.User{ID: "user-id", Username: "octocat"},
		installation: model.GitHubInstallation{
			ID: 42, AccountID: 9, AccountLogin: "octocat", AccountType: "User", RepositorySelection: "selected",
		},
	}
	client := &serviceTestClient{
		userToken: "user-token",
		installation: &model.GitHubInstallation{
			ID: 42, AccountID: 9, AccountLogin: "octocat", AccountType: "User", RepositorySelection: "selected",
			Repositories: []model.Repository{{GitHubID: 100, FullName: "octocat/docs"}},
		},
	}
	handler := newTestHandler(store, client)

	start := httptest.NewRecorder()
	handler.LoginRedirect(start, httptest.NewRequest(http.MethodGet, "/api/auth/github", nil))
	state := mustRedirectURL(t, start).Query().Get("state")
	binding := mustLoginBindingCookie(t, start)

	callback := httptest.NewRecorder()
	callbackRequest := httptest.NewRequest(http.MethodGet, "/api/github/callback?code=code-value&state="+url.QueryEscape(state), nil)
	callbackRequest.AddCookie(binding)
	handler.ValidateOAuthCallback(callback, callbackRequest)

	if callback.Code != http.StatusFound {
		t.Fatalf("callback status = %d, body = %s", callback.Code, callback.Body.String())
	}
	if callback.Header().Get("Location") != "http://localhost:3000/dashboard/repositories?github=existing" {
		t.Fatalf("location = %q", callback.Header().Get("Location"))
	}
	if !store.saved || client.getInstallationID != 42 || len(store.installation.Repositories) != 1 {
		t.Fatalf("saved = %t, installation = %d, repositories = %#v", store.saved, client.getInstallationID, store.installation.Repositories)
	}
}

func mustRedirectURL(t *testing.T, response *httptest.ResponseRecorder) *url.URL {
	t.Helper()
	location, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	return location
}

func mustLoginBindingCookie(t *testing.T, response *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == loginBindingCookieName {
			if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/api/github/callback" || cookie.MaxAge <= 0 {
				t.Fatalf("OAuth binding cookie = %#v", cookie)
			}
			return cookie
		}
	}
	t.Fatalf("OAuth login did not set %s", loginBindingCookieName)
	return nil
}

func newTestHandler(store installationStore, client githubClient) Handler {
	service := NewService(store, client)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewHandler(service, "sourceink", "http://localhost:3000", "client-id", "http://localhost:3000/api/github/callback", false, time.Hour, logger)
}
