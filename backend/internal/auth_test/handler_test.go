package auth_test

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sourceink/backend/internal/auth"
	"sourceink/backend/internal/model"
)

type fakeAccountStore struct {
	user                 model.User
	passwordHash         string
	findUserErr          error
	createUserErr        error
	createSessionErr     error
	sessionUserErr       error
	deleteSessionErr     error
	createdSessionHash   []byte
	loadedSessionHash    []byte
	deletedSessionHash   []byte
	registeredEmail      string
	registeredUsername   string
	registeredPassword   string
	registeredTokenHash  []byte
	registeredExpiration time.Time
}

func (s *fakeAccountStore) CreateUserAndSession(
	_ context.Context,
	email, username, passwordHash string,
	tokenHash []byte,
	expiresAt time.Time,
) (model.User, error) {
	s.registeredEmail = email
	s.registeredUsername = username
	s.registeredPassword = passwordHash
	s.registeredTokenHash = tokenHash
	s.registeredExpiration = expiresAt
	return s.user, s.createUserErr
}

func (s *fakeAccountStore) FindUserByEmail(context.Context, string) (model.User, string, error) {
	return s.user, s.passwordHash, s.findUserErr
}

func (s *fakeAccountStore) CreateSession(_ context.Context, _ string, tokenHash []byte, _ time.Time) error {
	s.createdSessionHash = tokenHash
	return s.createSessionErr
}

func (s *fakeAccountStore) FindOrCreateUserByGitHub(_ context.Context, _ model.GitHubUser, tokenHash []byte, _ time.Time) (model.User, error) {
	s.createdSessionHash = tokenHash
	return s.user, s.createUserErr
}

func (s *fakeAccountStore) UserBySession(_ context.Context, tokenHash []byte) (model.User, error) {
	s.loadedSessionHash = tokenHash
	return s.user, s.sessionUserErr
}

func (s *fakeAccountStore) DeleteSession(_ context.Context, tokenHash []byte) error {
	s.deletedSessionHash = tokenHash
	return s.deleteSessionErr
}

func newTestHandler(t *testing.T, store model.AccountStore, secure bool) *auth.Handler {
	t.Helper()
	handler, err := auth.NewHandler(store, slog.New(slog.NewTextHandler(io.Discard, nil)), secure, time.Hour)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return handler
}

func jsonRequest(method, target, body string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func responseCookie(t *testing.T, response *http.Response) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Cookies() {
		if cookie.Name == auth.SessionCookieName {
			return cookie
		}
	}
	t.Fatalf("response did not set %s cookie", auth.SessionCookieName)
	return nil
}

func TestRegisterNormalizesCredentialsAndCreatesSession(t *testing.T) {
	store := &fakeAccountStore{user: model.User{ID: "user-1", Email: "writer@example.com", Username: "writer"}}
	handler := newTestHandler(t, store, false)
	recorder := httptest.NewRecorder()

	handler.Register(recorder, jsonRequest(
		http.MethodPost,
		"/api/auth/register",
		`{"email":" Writer@Example.com ","username":"writer","password":"correct horse battery staple"}`,
	))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("Register() status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if store.registeredEmail != "writer@example.com" || store.registeredUsername != "writer" {
		t.Fatalf("registered credentials = %q, %q", store.registeredEmail, store.registeredUsername)
	}
	valid, err := auth.VerifyPassword("correct horse battery staple", store.registeredPassword)
	if err != nil || !valid {
		t.Fatalf("stored password hash did not verify: valid=%v err=%v", valid, err)
	}
	cookie := responseCookie(t, recorder.Result())
	if string(store.registeredTokenHash) != string(auth.SessionTokenHash(cookie.Value)) {
		t.Fatal("registration stored a token hash that does not match the session cookie")
	}
}

func TestLoginUsesSafeSessionCookie(t *testing.T) {
	passwordHash, err := auth.HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	store := &fakeAccountStore{
		user:         model.User{ID: "user-1", Email: "writer@example.com", Username: "writer"},
		passwordHash: passwordHash,
	}
	handler := newTestHandler(t, store, true)
	recorder := httptest.NewRecorder()

	handler.Login(recorder, jsonRequest(
		http.MethodPost,
		"/api/auth/login",
		`{"email":"writer@example.com","password":"correct horse battery staple"}`,
	))

	if recorder.Code != http.StatusOK {
		t.Fatalf("Login() status = %d, want %d", recorder.Code, http.StatusOK)
	}
	cookie := responseCookie(t, recorder.Result())
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
		t.Fatalf("session cookie flags = HttpOnly:%v Secure:%v SameSite:%v Path:%q", cookie.HttpOnly, cookie.Secure, cookie.SameSite, cookie.Path)
	}
	if string(store.createdSessionHash) != string(auth.SessionTokenHash(cookie.Value)) {
		t.Fatal("login stored a token hash that does not match the session cookie")
	}
}

func TestLoginHidesWhetherAccountExists(t *testing.T) {
	store := &fakeAccountStore{findUserErr: sql.ErrNoRows}
	handler := newTestHandler(t, store, false)
	recorder := httptest.NewRecorder()

	handler.Login(recorder, jsonRequest(
		http.MethodPost,
		"/api/auth/login",
		`{"email":"missing@example.com","password":"wrong password"}`,
	))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("Login() status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(recorder.Body.String(), "invalid email or password") {
		t.Fatalf("Login() response = %q, want generic credentials error", recorder.Body.String())
	}
}

func TestMeAndLogoutHashTheCookieToken(t *testing.T) {
	store := &fakeAccountStore{user: model.User{ID: "user-1", Email: "writer@example.com", Username: "writer"}}
	handler := newTestHandler(t, store, false)

	meRequest := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meRequest.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-token"})
	meRecorder := httptest.NewRecorder()
	handler.Me(meRecorder, meRequest)
	if meRecorder.Code != http.StatusOK {
		t.Fatalf("Me() status = %d, want %d", meRecorder.Code, http.StatusOK)
	}
	if string(store.loadedSessionHash) != string(auth.SessionTokenHash("session-token")) {
		t.Fatal("Me() did not hash the session token")
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutRequest.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-token"})
	logoutRecorder := httptest.NewRecorder()
	handler.Logout(logoutRecorder, logoutRequest)
	if logoutRecorder.Code != http.StatusNoContent {
		t.Fatalf("Logout() status = %d, want %d", logoutRecorder.Code, http.StatusNoContent)
	}
	if string(store.deletedSessionHash) != string(auth.SessionTokenHash("session-token")) {
		t.Fatal("Logout() did not hash the session token")
	}
	if cookie := responseCookie(t, logoutRecorder.Result()); cookie.MaxAge != -1 {
		t.Fatalf("logout cookie MaxAge = %d, want -1", cookie.MaxAge)
	}
}
