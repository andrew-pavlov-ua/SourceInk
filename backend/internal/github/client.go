package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"sourceink/backend/internal/model"
)

const (
	defaultAPIBaseURL    = "https://api.github.com"
	defaultOAuthTokenURL = "https://github.com/login/oauth/access_token"
	githubAPIVersion     = "2022-11-28"
	maxResponseBytes     = 1 << 20
	maxMarkdownBytes     = 512 << 10
)

type ClientConfig struct {
	AppJWT        func() (string, error)
	HTTPClient    *http.Client
	APIBaseURL    string
	ClientID      string
	ClientSecret  string
	OAuthTokenURL string
}

type GitHubClient struct {
	appJWT        func() (string, error)
	httpClient    *http.Client
	apiBaseURL    string
	clientID      string
	clientSecret  string
	oauthTokenURL string
}

type githubUserResponse struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

type RepositoryResp struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Private       bool   `json:"private"`
	Archived      bool   `json:"archived"`
	DefaultBranch string `json:"default_branch"`
	Owner         struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
	} `json:"owner"`
}

type repositoriesResponse struct {
	TotalCount   int              `json:"total_count"`
	Repositories []RepositoryResp `json:"repositories"`
}

type TreeResponse struct {
	SHA       string `json:"sha"`
	URL       string `json:"url"`
	Truncated bool   `json:"truncated"`
	Tree      []Tree `json:"tree"`
}

type Tree struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"` // blob
	Size int    `json:"size"`
	SHA  string `json:"sha"`
	URL  string `json:"url"`
}

type blobResponse struct {
	SHA      string `json:"sha"`
	URL      string `json:"url"`
	Size     int    `json:"size"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
}

// NewClient builds a GitHub App client. GitHub expires App JWTs quickly, so
// appJWT signs a new one for each request.
func NewClient(cfg ClientConfig) (*GitHubClient, error) {
	if cfg.AppJWT == nil {
		return nil, errors.New("github app JWT generator is required")
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if strings.TrimSpace(cfg.APIBaseURL) == "" {
		cfg.APIBaseURL = defaultAPIBaseURL
	}
	if strings.TrimSpace(cfg.OAuthTokenURL) == "" {
		cfg.OAuthTokenURL = defaultOAuthTokenURL
	}
	return &GitHubClient{
		appJWT:        cfg.AppJWT,
		httpClient:    cfg.HTTPClient,
		apiBaseURL:    strings.TrimRight(cfg.APIBaseURL, "/"),
		clientID:      cfg.ClientID,
		clientSecret:  cfg.ClientSecret,
		oauthTokenURL: strings.TrimRight(cfg.OAuthTokenURL, "/"),
	}, nil
}

// GetInstallation checks that this App owns the installation. User access is
// checked separately by UserCanAccessInstallation.
func (g *GitHubClient) GetInstallation(ctx context.Context, installationID int64) (*model.GitHubInstallation, error) {
	installationExists, err := g.installationExists(ctx, installationID)
	if err != nil {
		return nil, err
	}

	return installationExists, nil
}

func (g *GitHubClient) installationExists(ctx context.Context, installationID int64) (*model.GitHubInstallation, error) {
	if installationID <= 0 {
		return nil, errors.New("github installation id must be positive")
	}

	jwt, err := g.appJWT()
	if err != nil {
		return nil, fmt.Errorf("create github app JWT: %w", err)
	}

	endpoint := fmt.Sprintf("%s/app/installations/%d", g.apiBaseURL, installationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create get installation request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+jwt)
	setAPIHeaders(req)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get github installation: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read github installation response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("get github installation: unexpected status %d", resp.StatusCode)
	}

	var installation installationResponse
	err = json.Unmarshal(body, &installation)
	if err != nil {
		return nil, fmt.Errorf("decode github installation response: %w", err)
	}
	if installation.ID != installationID {
		return nil, errors.New("github returned a different installation id")
	}
	if installation.Account.ID <= 0 || installation.Account.Login == "" ||
		(installation.RepositorySelection != "all" && installation.RepositorySelection != "selected") {
		return nil, errors.New("github returned invalid installation metadata")
	}

	repos, err := g.FetchInstallationRepos(ctx, installationID)
	if err != nil {
		return nil, fmt.Errorf("load installation repositories: %w", err)
	}

	return &model.GitHubInstallation{
		ID:                  installation.ID,
		AccountID:           installation.Account.ID,
		AccountLogin:        installation.Account.Login,
		AccountType:         installation.Account.Type,
		RepositorySelection: installation.RepositorySelection,
		Suspended:           installation.SuspendedAt != nil,
		Repositories:        repos,
	}, nil
}

func (g *GitHubClient) FetchInstallationRepos(ctx context.Context, installationID int64) ([]model.Repository, error) {
	installationToken, err := g.CreateInstallationToken(ctx, installationID)
	if err != nil {
		return nil, err
	}

	var result []model.Repository
	for page := 1; page <= 100; page++ {
		endpoint := fmt.Sprintf("%s/installation/repositories?per_page=100&page=%d", g.apiBaseURL, page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("create list installation repositories request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+installationToken)
		setAPIHeaders(req)

		resp, err := g.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list installation repositories: %w", err)
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read installation repositories response: %w", readErr)
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil, fmt.Errorf("list installation repositories: unexpected status %d", resp.StatusCode)
		}

		var repositories repositoriesResponse
		if err := json.Unmarshal(body, &repositories); err != nil {
			return nil, fmt.Errorf("decode installation repositories response: %w", err)
		}
		result = append(result, repositoryRespToModel(repositories.Repositories, installationID)...)
		if len(repositories.Repositories) < 100 {
			return result, nil
		}
	}

	return nil, errors.New("github returned too many installation repositories")
}

func repositoryRespToModel(repos []RepositoryResp, installationID int64) []model.Repository {
	resultRepos := []model.Repository{}

	for _, rr := range repos {
		repo := model.Repository{
			InstallationID: installationID,
			GitHubID:       rr.ID,
			Owner:          rr.Owner.Login,
			Name:           rr.Name,
			FullName:       rr.FullName,
			DefaultBranch:  rr.DefaultBranch,
			Private:        rr.Private,
			Archived:       rr.Archived,
		}

		resultRepos = append(resultRepos, repo)
	}

	return resultRepos
}

func setAPIHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
}

type installationResponse struct {
	ID      int64 `json:"id"`
	Account struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"account"`
	RepositorySelection string     `json:"repository_selection"`
	SuspendedAt         *time.Time `json:"suspended_at"`
}

func (g *GitHubClient) ExchangeUserCode(
	ctx context.Context,
	code string,
	codeVerifier string,
	redirectURI string,
) (string, error) {
	if strings.TrimSpace(code) == "" {
		return "", errors.New("github authorization code is required")
	}
	if strings.TrimSpace(codeVerifier) == "" {
		return "", errors.New("github PKCE verifier is required")
	}
	if strings.TrimSpace(redirectURI) == "" {
		return "", errors.New("github OAuth redirect URI is required")
	}
	if strings.TrimSpace(g.clientID) == "" || strings.TrimSpace(g.clientSecret) == "" {
		return "", errors.New("github OAuth client credentials are not configured")
	}

	form := url.Values{}
	form.Set("client_id", g.clientID)
	form.Set("client_secret", g.clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", codeVerifier)

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		g.oauthTokenURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("create GitHub token exchange request: %w", err)
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := g.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("exchange GitHub authorization code: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("read GitHub token exchange response: %w", err)
	}

	var result struct {
		AccessToken      string `json:"access_token"`
		TokenType        string `json:"token_type"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode GitHub token exchange response: %w", err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("GitHub token exchange returned status %d", response.StatusCode)
	}

	if result.Error != "" {
		return "", fmt.Errorf("GitHub rejected authorization: %s", result.Error)
	}

	if result.AccessToken == "" {
		return "", errors.New("GitHub returned no user access token")
	}

	return result.AccessToken, nil
}

func (g *GitHubClient) GetUser(ctx context.Context, userToken string) (model.GitHubUser, error) {
	if strings.TrimSpace(userToken) == "" {
		return model.GitHubUser{}, errors.New("github user token is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.apiBaseURL+"/user", nil)
	if err != nil {
		return model.GitHubUser{}, fmt.Errorf("create get GitHub user request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+userToken)
	setAPIHeaders(req)
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return model.GitHubUser{}, fmt.Errorf("get GitHub user: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return model.GitHubUser{}, fmt.Errorf("read GitHub user response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return model.GitHubUser{}, fmt.Errorf("get GitHub user: unexpected status %d", resp.StatusCode)
	}
	var githubUser githubUserResponse
	if err := json.Unmarshal(body, &githubUser); err != nil {
		return model.GitHubUser{}, fmt.Errorf("decode GitHub user response: %w", err)
	}
	if githubUser.ID <= 0 || strings.TrimSpace(githubUser.Login) == "" {
		return model.GitHubUser{}, errors.New("GitHub returned invalid user identity")
	}
	return model.GitHubUser{ID: githubUser.ID, Login: githubUser.Login, AvatarURL: githubUser.AvatarURL}, nil
}

// UserCanAccessInstallation checks whether the OAuth user can see an installation.
func (g *GitHubClient) UserCanAccessInstallation(ctx context.Context, userToken string, installationID int64) (bool, error) {
	if strings.TrimSpace(userToken) == "" {
		return false, errors.New("github user token is required")
	}
	if installationID <= 0 {
		return false, errors.New("github installation id must be positive")
	}

	for page := 1; page <= 100; page++ {
		endpoint := fmt.Sprintf("%s/user/installations?per_page=100&page=%d", g.apiBaseURL, page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return false, fmt.Errorf("create list user installations request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+userToken)
		setAPIHeaders(req)

		resp, err := g.httpClient.Do(req)
		if err != nil {
			return false, fmt.Errorf("list user installations: %w", err)
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		resp.Body.Close()
		if readErr != nil {
			return false, fmt.Errorf("read user installations response: %w", readErr)
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return false, fmt.Errorf("list user installations: unexpected status %d", resp.StatusCode)
		}

		var result struct {
			Installations []struct {
				ID int64 `json:"id"`
			} `json:"installations"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return false, fmt.Errorf("decode user installations response: %w", err)
		}
		for _, installation := range result.Installations {
			if installation.ID == installationID {
				return true, nil
			}
		}
		if len(result.Installations) < 100 {
			return false, nil
		}
	}
	return false, errors.New("github returned too many user installations")
}

type installationTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (g *GitHubClient) CreateInstallationToken(
	ctx context.Context,
	installationID int64,
) (string, error) {
	if installationID <= 0 {
		return "", errors.New("github installation id must be positive")
	}

	jwt, err := g.appJWT()
	if err != nil {
		return "", fmt.Errorf("create GitHub App JWT: %w", err)
	}

	endpoint := fmt.Sprintf(
		"%s/app/installations/%d/access_tokens",
		g.apiBaseURL,
		installationID,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create installation-token request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+jwt)
	setAPIHeaders(req)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create installation token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create installation token: unexpected status %d", resp.StatusCode)
	}

	var result installationTokenResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&result); err != nil {
		return "", fmt.Errorf("decode installation token: %w", err)
	}
	if result.Token == "" {
		return "", errors.New("GitHub returned an empty installation token")
	}
	return result.Token, nil
}

func (g *GitHubClient) LoadUnpublishedArticlesByRepo(
	ctx context.Context,
	repository model.Repository,
) ([]model.UnpublishedArticle, error) {
	if repository.InstallationID <= 0 {
		return nil, errors.New("load unpublished articles: installation ID must be positive")
	}

	token, err := g.CreateInstallationToken(ctx, repository.InstallationID)
	if err != nil {
		return nil, fmt.Errorf("load unpublished articles: create installation token: %w", err)
	}

	mdFiles, err := g.listMarkdownFilesWithToken(ctx, repository, token)
	if err != nil {
		return nil, fmt.Errorf("load unpublished articles: list Markdown files: %w", err)
	}
	if mdFiles.Truncated {
		return nil, errors.New("load unpublished articles: GitHub returned a truncated repository tree")
	}

	unpublishedArticles := make([]model.UnpublishedArticle, 0, len(mdFiles.Files))
	for _, file := range mdFiles.Files {
		if file.Size > maxMarkdownBytes {
			validationError := fmt.Sprintf("Markdown file is too large: maximum size is %d KiB", maxMarkdownBytes/1024)
			unpublishedArticles = append(unpublishedArticles, model.UnpublishedArticle{
				RepositoryID:    repository.ID,
				SourcePath:      file.Path,
				GitBlobSHA:      file.BlobSHA,
				ValidationError: &validationError,
				Present:         true,
			})
			continue
		}
		markdownFile, err := g.fetchMarkdownFileWithToken(ctx, repository, token, file)
		if err != nil {
			return nil, fmt.Errorf("load unpublished articles: fetch %s: %w", file.Path, err)
		}

		parsedMarkdown, parseErr := ParseFrontmatter(markdownFile.Content)
		if parseErr != nil {
			if errors.Is(parseErr, errFrontmatterBlockMissing) {
				continue
			}

			validationError := parseErr.Error()
			unpublishedArticles = append(unpublishedArticles, model.UnpublishedArticle{
				RepositoryID:    repository.ID,
				SourcePath:      file.Path,
				GitBlobSHA:      file.BlobSHA,
				Content:         markdownFile.Content,
				ValidationError: &validationError,
				Present:         true,
			})
			continue
		}

		unpublishedArticles = append(unpublishedArticles, model.UnpublishedArticle{
			RepositoryID: repository.ID,
			SourcePath:   file.Path,
			GitBlobSHA:   file.BlobSHA,
			Frontmatter:  parsedMarkdown.Frontmatter,
			Content:      parsedMarkdown.Content,
			Present:      true,
		})
	}

	return unpublishedArticles, nil
}

func (g *GitHubClient) ListMarkdownFiles(ctx context.Context, repo model.Repository, installationID int64) (*model.FileList, error) {
	token, err := g.CreateInstallationToken(ctx, installationID)
	if err != nil {
		return nil, fmt.Errorf("create installation token: %w", err)
	}
	return g.listMarkdownFilesWithToken(ctx, repo, token)
}

func (g *GitHubClient) listMarkdownFilesWithToken(ctx context.Context, repo model.Repository, token string) (*model.FileList, error) {
	treeResponse, err := g.listTreeWithToken(ctx, repo, token)
	if err != nil {
		return nil, fmt.Errorf("list repository tree: %w", err)
	}

	var files []model.RepositoryFile
	for _, entry := range treeResponse.Tree {
		if entry.Type == "blob" &&
			entry.Mode != "120000" &&
			strings.EqualFold(path.Ext(entry.Path), ".md") {
			file := model.RepositoryFile{
				Path:    entry.Path,
				BlobURL: entry.URL,
				BlobSHA: entry.SHA,
				Size:    entry.Size,
			}
			files = append(files, file)
		}
	}

	return &model.FileList{
		TreeSHA:   treeResponse.SHA,
		Truncated: treeResponse.Truncated,
		Files:     files,
	}, nil
}

func (g *GitHubClient) ListTree(ctx context.Context, repo model.Repository, installationID int64) (*TreeResponse, error) {
	token, err := g.CreateInstallationToken(ctx, installationID)
	if err != nil {
		return nil, fmt.Errorf("create installation token: %w", err)
	}
	return g.listTreeWithToken(ctx, repo, token)
}

func (g *GitHubClient) listTreeWithToken(ctx context.Context, repo model.Repository, token string) (*TreeResponse, error) {
	endpoint := fmt.Sprintf(
		"%s/repos/%s/%s/git/trees/%s?recursive=1",
		g.apiBaseURL,
		repo.Owner,
		repo.Name,
		repo.DefaultBranch,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create Git tree request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	setAPIHeaders(req)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get Git tree: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get Git tree: unexpected status %d", resp.StatusCode)
	}

	var treeResponse TreeResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&treeResponse); err != nil {
		return nil, fmt.Errorf("decode Git tree response: %w", err)
	}

	return &treeResponse, nil
}

func (g *GitHubClient) FetchMarkdownFile(
	ctx context.Context,
	repo model.Repository,
	installationID int64,
	file model.RepositoryFile,
) (model.MarkdownFile, error) {
	token, err := g.CreateInstallationToken(ctx, installationID)
	if err != nil {
		return model.MarkdownFile{}, fmt.Errorf("create installation token: %w", err)
	}
	return g.fetchMarkdownFileWithToken(ctx, repo, token, file)
}

func (g *GitHubClient) fetchMarkdownFileWithToken(
	ctx context.Context,
	repo model.Repository,
	token string,
	file model.RepositoryFile,
) (model.MarkdownFile, error) {
	endpoint := fmt.Sprintf(
		"%s/repos/%s/%s/git/blobs/%s",
		g.apiBaseURL,
		repo.Owner,
		repo.Name,
		file.BlobSHA,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return model.MarkdownFile{}, fmt.Errorf("create Git blob request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	setAPIHeaders(req)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return model.MarkdownFile{}, fmt.Errorf("get Git blob: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return model.MarkdownFile{}, fmt.Errorf("get Git blob: unexpected status %d", resp.StatusCode)
	}

	var blob blobResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&blob); err != nil {
		return model.MarkdownFile{}, fmt.Errorf("decode GitHub blob: %w", err)
	}
	if blob.SHA != file.BlobSHA {
		return model.MarkdownFile{}, errors.New("GitHub returned a different Markdown blob")
	}
	if blob.Encoding != "base64" {
		return model.MarkdownFile{}, fmt.Errorf("GitHub returned unsupported blob encoding %q", blob.Encoding)
	}
	content, err := base64.StdEncoding.DecodeString(blob.Content)
	if err != nil {
		return model.MarkdownFile{}, fmt.Errorf("decode Markdown blob content: %w", err)
	}

	return model.MarkdownFile{
		RepositoryFile: model.RepositoryFile{
			Path:    file.Path,
			Name:    file.Name,
			BlobSHA: blob.SHA,
			BlobURL: blob.URL,
			Size:    blob.Size,
		},
		Content: string(content),
	}, nil
}
