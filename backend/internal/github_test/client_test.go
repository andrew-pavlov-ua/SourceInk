package github_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"sourceink/backend/internal/github"
	"sourceink/backend/internal/model"
)

func TestFetchInstallationReposPaginates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/installations/42/access_tokens":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"token":"installation-token"}`))
		case "/installation/repositories":
			if got := r.URL.Query().Get("per_page"); got != "100" {
				t.Errorf("per_page = %q, want 100", got)
			}
			page := r.URL.Query().Get("page")
			repositories := make([]map[string]any, 0, 100)
			if page == "1" {
				for id := 1; id <= 100; id++ {
					repositories = append(repositories, map[string]any{
						"id": id, "name": "repo", "full_name": "octocat/repo",
						"default_branch": "main", "owner": map[string]any{"id": 7, "login": "octocat"},
					})
				}
			} else if page == "2" {
				repositories = append(repositories, map[string]any{
					"id": 101, "name": "last-repo", "full_name": "octocat/last-repo",
					"default_branch": "main", "owner": map[string]any{"id": 7, "login": "octocat"},
				})
			} else {
				t.Errorf("unexpected page %q", page)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"repositories": repositories})
		default:
			t.Errorf("path = %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := newTestClient(server.Client(), server.URL, func() (string, error) { return "signed-app-jwt", nil })
	if err != nil {
		t.Fatal(err)
	}

	repositories, err := client.FetchInstallationRepos(context.Background(), 42)
	if err != nil {
		t.Fatalf("FetchInstallationRepos() error = %v", err)
	}
	if len(repositories) != 101 {
		t.Fatalf("len(repositories) = %d, want 101", len(repositories))
	}
	if repositories[100].FullName != "octocat/last-repo" || repositories[100].InstallationID != 42 {
		t.Errorf("last repository = %#v", repositories[100])
	}
}

func TestListTreeReturnsRecursiveGitHubTree(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/installations/42/access_tokens":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"token":"installation-token"}`))
		case "/repos/octocat/docs/git/trees/main":
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}
			if r.URL.Query().Get("recursive") != "1" {
				t.Errorf("recursive = %q, want 1", r.URL.Query().Get("recursive"))
			}
			if got := r.Header.Get("Authorization"); got != "Bearer installation-token" {
				t.Errorf("authorization = %q", got)
			}
			_, _ = w.Write([]byte(`{"sha":"tree-sha","url":"https://api.github.com/repos/octocat/docs/git/trees/tree-sha","truncated":true,"tree":[{"path":"README.md","mode":"100644","type":"blob","sha":"readme-sha"},{"path":"guides","mode":"040000","type":"tree","sha":"guides-sha"},{"path":"guides/setup.md","mode":"100644","type":"blob","sha":"setup-sha"}]}`))
		default:
			t.Errorf("path = %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := newTestClient(server.Client(), server.URL, func() (string, error) { return "signed-app-jwt", nil })
	if err != nil {
		t.Fatal(err)
	}

	tree, err := client.ListTree(context.Background(), model.Repository{
		Owner: "octocat", Name: "docs", DefaultBranch: "main",
	}, 42)
	if err != nil {
		t.Fatalf("ListTree() error = %v", err)
	}
	if tree.SHA != "tree-sha" || tree.URL != "https://api.github.com/repos/octocat/docs/git/trees/tree-sha" {
		t.Errorf("tree metadata = %#v", tree)
	}
	if !tree.Truncated {
		t.Error("Truncated = false, want true")
	}
	if len(tree.Tree) != 3 {
		t.Fatalf("len(Tree) = %d, want 3", len(tree.Tree))
	}
	if tree.Tree[1].Path != "guides" || tree.Tree[1].Type != "tree" || tree.Tree[1].Mode != "040000" {
		t.Errorf("Tree[1] = %#v", tree.Tree[1])
	}
	if tree.Tree[2].Path != "guides/setup.md" || tree.Tree[2].SHA != "setup-sha" {
		t.Errorf("Tree[2] = %#v", tree.Tree[2])
	}
}

func TestListMarkdownFilesFiltersRecursiveTree(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/installations/42/access_tokens":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"token":"installation-token"}`))
		case "/repos/octocat/docs/git/trees/main":
			_, _ = w.Write([]byte(`{"sha":"tree-sha","url":"tree-url","truncated":false,"tree":[{"path":"README.md","mode":"100644","type":"blob","size":120,"sha":"readme-sha","url":"readme-url"},{"path":"guides","mode":"040000","type":"tree"},{"path":"guides/setup.MD","mode":"100644","type":"blob","size":240,"sha":"setup-sha","url":"setup-url"},{"path":"guides/link.md","mode":"120000","type":"blob","size":16,"sha":"link-sha"},{"path":"assets/logo.png","mode":"100644","type":"blob","size":480,"sha":"logo-sha"},{"path":"vendor/docs","mode":"160000","type":"commit"}]}`))
		default:
			t.Errorf("path = %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := newTestClient(server.Client(), server.URL, func() (string, error) { return "signed-app-jwt", nil })
	if err != nil {
		t.Fatal(err)
	}

	tree, err := client.ListMarkdownFiles(context.Background(), model.Repository{
		Owner: "octocat", Name: "docs", DefaultBranch: "main",
	}, 42)
	if err != nil {
		t.Fatalf("ListMarkdownFiles() error = %v", err)
	}
	if tree.TreeSHA != "tree-sha" || tree.Truncated {
		t.Errorf("tree metadata = %#v", tree)
	}
	wantFiles := []model.RepositoryFile{
		{Path: "README.md", BlobSHA: "readme-sha", BlobURL: "readme-url", Size: 120},
		{Path: "guides/setup.MD", BlobSHA: "setup-sha", BlobURL: "setup-url", Size: 240},
	}
	if len(tree.Files) != len(wantFiles) {
		t.Fatalf("Files = %#v, want %#v", tree.Files, wantFiles)
	}
	for i, want := range wantFiles {
		if tree.Files[i] != want {
			t.Errorf("Files[%d] = %#v, want %#v", i, tree.Files[i], want)
		}
	}
}

func TestLoadUnpublishedArticlesReturnsDecodedMarkdown(t *testing.T) {
	tokenRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/installations/42/access_tokens":
			tokenRequests++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"token":"installation-token"}`))
		case "/repos/octocat/docs/git/trees/main":
			_, _ = w.Write([]byte(`{"sha":"tree-sha","truncated":false,"tree":[{"path":"guides/setup.md","mode":"100644","type":"blob","size":8,"sha":"setup-sha","url":"blob-url"}]}`))
		case "/repos/octocat/docs/git/blobs/setup-sha":
			if got := r.Header.Get("Authorization"); got != "Bearer installation-token" {
				t.Errorf("authorization = %q", got)
			}
			_, _ = w.Write([]byte(`{"sha":"setup-sha","url":"blob-url","size":105,"encoding":"base64","content":"LS0tCnRpdGxlOiBIZWxsbwpzbHVnOiBoZWxsbwpkZXNjcmlwdGlvbjogR3JlZXRpbmcKdGFnczoKICAtIGludHJvCnB1Ymxpc2hfbW9kZTogbWFudWFsCi0tLQojIEhlbGxvCg=="}`))
		default:
			t.Errorf("path = %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := newTestClient(server.Client(), server.URL, func() (string, error) { return "signed-app-jwt", nil })
	if err != nil {
		t.Fatal(err)
	}

	articles, err := client.LoadUnpublishedArticlesByRepo(context.Background(), model.Repository{
		ID: "repository-id", InstallationID: 42, Owner: "octocat", Name: "docs", DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("LoadUnpublishedArticlesByRepo() error = %v", err)
	}
	if len(articles) != 1 {
		t.Fatalf("articles = %#v", articles)
	}
	if tokenRequests != 1 {
		t.Fatalf("installation token requests = %d, want 1 per repository sync", tokenRequests)
	}
	article := articles[0]
	if article.RepositoryID != "repository-id" || article.SourcePath != "guides/setup.md" ||
		article.GitBlobSHA != "setup-sha" || article.Content != "# Hello\n" || !article.Present ||
		article.ValidationError != nil || article.Title != "Hello" || article.Slug != "hello" ||
		article.Description != "Greeting" || article.PublishMode != "manual" ||
		len(article.Tags) != 1 || article.Tags[0] != "intro" {
		t.Errorf("article = %#v", article)
	}
}

func TestLoadUnpublishedArticlesRejectsOversizedMarkdownBeforeFetchingBlob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/installations/42/access_tokens":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"token":"installation-token"}`))
		case "/repos/octocat/docs/git/trees/main":
			_, _ = w.Write([]byte(`{"sha":"tree-sha","truncated":false,"tree":[{"path":"huge.md","mode":"100644","type":"blob","size":600000,"sha":"huge-sha"}]}`))
		default:
			t.Fatalf("unexpected request for oversized Markdown: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := newTestClient(server.Client(), server.URL, func() (string, error) { return "signed-app-jwt", nil })
	if err != nil {
		t.Fatal(err)
	}

	articles, err := client.LoadUnpublishedArticlesByRepo(context.Background(), model.Repository{
		ID: "repository-id", InstallationID: 42, Owner: "octocat", Name: "docs", DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("LoadUnpublishedArticlesByRepo() error = %v", err)
	}
	if len(articles) != 1 || articles[0].ValidationError == nil || articles[0].SourcePath != "huge.md" {
		t.Fatalf("oversized article = %#v", articles)
	}
}

func TestLoadUnpublishedArticlesSkipsMarkdownWithoutCompleteFrontmatter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/installations/42/access_tokens":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"token":"installation-token"}`))
		case "/repos/octocat/docs/git/trees/main":
			_, _ = w.Write([]byte(`{"sha":"tree-sha","truncated":false,"tree":[{"path":"README.md","mode":"100644","type":"blob","sha":"plain-sha"},{"path":"unfinished.md","mode":"100644","type":"blob","sha":"unfinished-sha"},{"path":"invalid.md","mode":"100644","type":"blob","sha":"invalid-sha"}]}`))
		case "/repos/octocat/docs/git/blobs/plain-sha":
			_, _ = w.Write([]byte(`{"sha":"plain-sha","encoding":"base64","content":"IyBQbGFpbiBNYXJrZG93bgo="}`))
		case "/repos/octocat/docs/git/blobs/unfinished-sha":
			_, _ = w.Write([]byte(`{"sha":"unfinished-sha","encoding":"base64","content":"LS0tCnRpdGxlOiBVbmZpbmlzaGVkCg=="}`))
		case "/repos/octocat/docs/git/blobs/invalid-sha":
			_, _ = w.Write([]byte(`{"sha":"invalid-sha","encoding":"base64","content":"LS0tCnRpdGxlOiBJbnZhbGlkCi0tLQojIEJvZHkK"}`))
		default:
			t.Errorf("path = %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := newTestClient(server.Client(), server.URL, func() (string, error) { return "signed-app-jwt", nil })
	if err != nil {
		t.Fatal(err)
	}

	articles, err := client.LoadUnpublishedArticlesByRepo(context.Background(), model.Repository{
		ID: "repository-id", InstallationID: 42, Owner: "octocat", Name: "docs", DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("LoadUnpublishedArticlesByRepo() error = %v", err)
	}
	if len(articles) != 1 {
		t.Fatalf("articles = %#v, want only the file with a complete frontmatter block", articles)
	}
	if articles[0].SourcePath != "invalid.md" || articles[0].ValidationError == nil {
		t.Fatalf("article = %#v, want invalid.md with a validation error", articles[0])
	}
}

func TestGetInstallation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/installations/42":
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer signed-app-jwt" {
				t.Errorf("authorization = %q", got)
			}
			_, _ = w.Write([]byte(`{"id":42,"account":{"id":7,"login":"octocat","type":"User"},"repository_selection":"selected","suspended_at":null}`))
		case "/app/installations/42/access_tokens":
			if r.Method != http.MethodPost {
				t.Errorf("method = %s, want POST", r.Method)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"token":"installation-token"}`))
		case "/installation/repositories":
			if got := r.Header.Get("Authorization"); got != "Bearer installation-token" {
				t.Errorf("authorization = %q", got)
			}
			_, _ = w.Write([]byte(`{"total_count":1,"repositories":[{"id":100,"name":"docs","full_name":"octocat/docs","default_branch":"main","owner":{"id":7,"login":"octocat"}}]}`))
		case "/repos/octocat/docs/git/trees/main":
			_, _ = w.Write([]byte(`{"sha":"tree-sha","tree":[]}`))
		default:
			t.Errorf("path = %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := newTestClient(server.Client(), server.URL, func() (string, error) {
		return "signed-app-jwt", nil
	})
	if err != nil {
		t.Fatal(err)
	}

	installation, err := client.GetInstallation(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetInstallation() error = %v", err)
	}
	if installation.ID != 42 || installation.AccountLogin != "octocat" || installation.Suspended {
		t.Errorf("installation = %#v", installation)
	}
}

func TestGetInstallationRejectsInvalidIDBeforeCallingGitHub(t *testing.T) {
	called := false
	client, err := newTestClient(nil, "https://api.github.com", func() (string, error) {
		called = true
		return "signed-app-jwt", nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := client.GetInstallation(context.Background(), 0); err == nil {
		t.Fatal("GetInstallation() accepted an invalid installation id")
	}
	if called {
		t.Fatal("GetInstallation() generated a JWT for an invalid installation id")
	}
}

func TestGetInstallationRejectsTokenAndResponseFailures(t *testing.T) {
	t.Run("token", func(t *testing.T) {
		client, err := newTestClient(nil, "https://api.github.com", func() (string, error) {
			return "", errors.New("key unavailable")
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := client.GetInstallation(context.Background(), 42); err == nil {
			t.Fatal("GetInstallation() accepted a failed JWT generator")
		}
	})

	t.Run("unexpected status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		}))
		defer server.Close()

		client, err := newTestClient(server.Client(), server.URL, func() (string, error) {
			return "signed-app-jwt", nil
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := client.GetInstallation(context.Background(), 42); err == nil {
			t.Fatal("GetInstallation() accepted a non-success response")
		}
	})

	t.Run("mismatched id", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"id": 99, "account": {"id": 7, "login": "octocat", "type": "User"}}`))
		}))
		defer server.Close()

		client, err := newTestClient(server.Client(), server.URL, func() (string, error) {
			return "signed-app-jwt", nil
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := client.GetInstallation(context.Background(), 42); err == nil {
			t.Fatal("GetInstallation() accepted a mismatched installation id")
		}
	})
}

func TestExchangeUserCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/login/oauth/access_token" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("authorization = %q, want empty", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("accept = %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := r.Form.Get("client_id"); got != "client-id" {
			t.Errorf("client_id = %q", got)
		}
		if got := r.Form.Get("client_secret"); got != "client-secret" {
			t.Errorf("client_secret = %q", got)
		}
		if got := r.Form.Get("code"); got != "authorization-code" {
			t.Errorf("code = %q", got)
		}
		if got := r.Form.Get("code_verifier"); got != "pkce-verifier" {
			t.Errorf("code_verifier = %q", got)
		}
		if got := r.Form.Get("redirect_uri"); got != "http://localhost:3000/api/github/callback" {
			t.Errorf("redirect_uri = %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"ghu_user_token","token_type":"bearer"}`))
	}))
	defer server.Close()

	client, err := github.NewClient(github.ClientConfig{
		AppJWT:        func() (string, error) { return "signed-app-jwt", nil },
		HTTPClient:    server.Client(),
		APIBaseURL:    "https://api.github.com",
		ClientID:      "client-id",
		ClientSecret:  "client-secret",
		OAuthTokenURL: server.URL + "/login/oauth/access_token",
	})
	if err != nil {
		t.Fatal(err)
	}

	token, err := client.ExchangeUserCode(
		context.Background(),
		"authorization-code",
		"pkce-verifier",
		"http://localhost:3000/api/github/callback",
	)
	if err != nil {
		t.Fatalf("ExchangeUserCode() error = %v", err)
	}
	if token != "ghu_user_token" {
		t.Errorf("token = %q", token)
	}
}

func TestExchangeUserCodeRequiresOAuthCredentials(t *testing.T) {
	client, err := github.NewClient(github.ClientConfig{
		AppJWT: func() (string, error) { return "signed-app-jwt", nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.ExchangeUserCode(
		context.Background(),
		"authorization-code",
		"pkce-verifier",
		"http://localhost:3000/api/github/callback",
	)
	if err == nil {
		t.Fatal("ExchangeUserCode() accepted missing OAuth credentials")
	}
}

func TestGetUserUsesOAuthToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/user" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer user-token" {
			t.Fatalf("authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"id": 7, "login": "octocat", "avatar_url": "https://avatars.example/octocat"}`))
	}))
	defer server.Close()
	client, err := newTestClient(server.Client(), server.URL, func() (string, error) { return "app-jwt", nil })
	if err != nil {
		t.Fatal(err)
	}
	user, err := client.GetUser(context.Background(), "user-token")
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if user.ID != 7 || user.Login != "octocat" {
		t.Fatalf("user = %#v", user)
	}
}

func newTestClient(httpClient *http.Client, apiBaseURL string, appJWT func() (string, error)) (*github.GitHubClient, error) {
	return github.NewClient(github.ClientConfig{
		AppJWT:     appJWT,
		HTTPClient: httpClient,
		APIBaseURL: apiBaseURL,
	})
}
