package model

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrEmailTaken                 = errors.New("email is already registered")
	ErrEmailSignInAlreadySet      = errors.New("email sign-in is already connected")
	ErrEmailSignInNotAllowed      = errors.New("email sign-in can only be added to a GitHub account")
	ErrUsernameTaken              = errors.New("username is already registered")
	ErrGitHubUsernameTaken        = errors.New("GitHub username is already connected to another account")
	ErrGitHubAccountConflict      = errors.New("GitHub account is connected to another account")
	ErrGitHubInstallationConflict = errors.New("github installation is connected to another user")
	ErrArticleNotFound            = errors.New("article draft not found")
	ErrArticleNotPublishable      = errors.New("article draft cannot be published")
	ErrPublishedArticleNotFound   = errors.New("published article not found")
	ErrArticleReviewStale         = errors.New("article review targets a stale revision")
	ErrArticleReviewNotFound      = errors.New("article review not found")
	ErrArticleReviewInvalid       = errors.New("article review is invalid")
)

type User struct {
	ID              string    `db:"id" json:"id"`
	Email           string    `db:"email" json:"email"`
	Username        string    `db:"username" json:"username"`
	GitHubUserID    *int64    `db:"github_user_id" json:"github_user_id,omitempty"`
	GitHubLogin     string    `db:"github_login" json:"github_login,omitempty"`
	GitHubAvatarURL string    `db:"github_avatar_url" json:"github_avatar_url,omitempty"`
	Role            string    `db:"role" json:"role"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

type GitHubUser struct {
	ID        int64
	Login     string
	AvatarURL string
}

type AccountStore interface {
	CreateUserAndSession(
		ctx context.Context,
		email, username, passwordHash string,
		tokenHash []byte,
		expiresAt time.Time,
	) (User, error)
	FindUserByEmail(ctx context.Context, email string) (User, string, error)
	CreateSession(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time) error
	FindOrCreateUserByGitHub(ctx context.Context, githubUser GitHubUser, tokenHash []byte, expiresAt time.Time) (User, error)
	SetEmailPasswordForGitHubUser(ctx context.Context, userID, email, passwordHash string) (User, error)
	UserBySession(ctx context.Context, tokenHash []byte) (User, error)
	DeleteSession(ctx context.Context, tokenHash []byte) error
}

type GitHubInstallation struct {
	ID                  int64        `db:"installation_id"`
	AccountID           int64        `db:"github_account_id"`
	AccountLogin        string       `db:"github_account_login"`
	AccountType         string       `db:"github_account_type"`
	RepositorySelection string       `db:"repository_selection"`
	Repositories        []Repository `db:"-"`
	Suspended           bool         `db:"suspended"`
}

type Repository struct {
	ID             string `db:"id" json:"-"`
	InstallationID int64  `db:"installation_id" json:"-"`
	GitHubID       int64  `db:"github_id" json:"github_id"`
	Owner          string `db:"owner" json:"owner"`
	Name           string `db:"name" json:"name"`
	FullName       string `db:"full_name" json:"full_name"`
	DefaultBranch  string `db:"default_branch" json:"default_branch"`
	Private        bool   `db:"private" json:"private"`
	Archived       bool   `db:"archived" json:"archived"`
}

type UnpublishedArticle struct {
	ID              string `db:"id" json:"id"`
	RepositoryID    string `db:"repository_id" json:"repository_id"`
	SourcePath      string `db:"source_path" json:"source_path"`
	GitBlobSHA      string `db:"git_blob_sha" json:"git_blob_sha"`
	Frontmatter     `json:"frontmatter"`
	Content         string    `db:"markdown" json:"content"`
	ValidationError *string   `db:"validation_error" json:"validation_error,omitempty"`
	Present         bool      `db:"present" json:"present"`
	DiscoveredAt    time.Time `db:"discovered_at" json:"discovered_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

type ArticlePublishMode string

const (
	ArticlePublishModeManual ArticlePublishMode = "manual"
	ArticlePublishModeAuto   ArticlePublishMode = "auto"
)

type ArticleSourceState string

const (
	ArticleSourceStateAvailable  ArticleSourceState = "available"
	ArticleSourceStateMissing    ArticleSourceState = "missing"
	ArticleSourceStateAccessLost ArticleSourceState = "access_lost"
)

type UserArticles struct {
	UserID              string               `json:"user_id"`
	UnpublishedArticles []UnpublishedArticle `json:"unpublished_articles"`
	PublishedArticles   []Article            `json:"published_articles"`
}

type Article struct {
	ID                   string             `db:"id" json:"id"`
	UnpublishedArticleID *string            `db:"unpublished_article_id" json:"unpublished_article_id,omitempty"`
	OwnerID              string             `db:"owner_id" json:"owner_id"`
	AuthorUsername       string             `db:"author_username" json:"author_username"`
	RepositoryID         *string            `db:"repository_id" json:"repository_id,omitempty"`
	SourcePath           string             `db:"source_path" json:"source_path"`
	Slug                 string             `db:"slug" json:"slug"`
	PublishMode          ArticlePublishMode `db:"publish_mode" json:"publish_mode"`
	SourceState          ArticleSourceState `db:"source_state" json:"source_state"`
	GitBlobSHA           string             `db:"git_blob_sha" json:"git_blob_sha"`
	Markdown             string             `db:"markdown" json:"markdown"`
	Title                string             `db:"title" json:"title"`
	Frontmatter          *json.RawMessage   `db:"frontmatter" json:"frontmatter,omitempty"`
	PublishedAt          time.Time          `db:"published_at" json:"published_at"`
	CreatedAt            time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time          `db:"updated_at" json:"updated_at"`
}

type FileList struct {
	TreeSHA   string
	Truncated bool
	Files     []RepositoryFile
}

type RepositoryFile struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	BlobSHA string `json:"sha"`
	BlobURL string `json:"url"`
	Size    int    `json:"size"`
}

type MarkdownFile struct {
	RepositoryFile
	Frontmatter Frontmatter
	Content     string `json:"content"`
}

type Frontmatter struct {
	Title       string   `db:"title" yaml:"title" json:"title"`
	Slug        string   `db:"slug" yaml:"slug" json:"slug"`
	Description string   `db:"description" yaml:"description" json:"description"`
	Tags        []string `db:"tags" yaml:"tags" json:"tags"`
	PublishMode string   `db:"publish_mode" yaml:"publish_mode" json:"publish_mode"`
}

type Verdict string

const (
	VerdictApprove        Verdict = "approve"
	VerdictRequestChanges Verdict = "request_changes"
)

type Reason string

const (
	ReasonIncorrect Reason = "incorrect"
	ReasonOutdated  Reason = "outdated"
	ReasonUnclear   Reason = "unclear"
)

type Review struct {
	ID         string    `db:"id" json:"id"`
	ArticleID  string    `db:"article_id" json:"article_id"`
	GitBlobSHA string    `db:"git_blob_sha" json:"git_blob_sha"`
	ReviewerID string    `db:"reviewer_id" json:"-"`
	Verdict    Verdict   `db:"verdict" json:"verdict"`
	Reason     *Reason   `db:"reason" json:"reason,omitempty"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

type ReviewReasonCounts struct {
	Incorrect int `json:"incorrect"`
	Outdated  int `json:"outdated"`
	Unclear   int `json:"unclear"`
}

type ViewerReview struct {
	Verdict Verdict `json:"verdict"`
	Reason  *Reason `json:"reason,omitempty"`
}

type ReviewSummary struct {
	ArticleID           string             `json:"article_id"`
	GitBlobSHA          string             `json:"git_blob_sha"`
	ApproveCount        int                `json:"approve_count"`
	RequestChangesCount int                `json:"request_changes_count"`
	RequestReasons      ReviewReasonCounts `json:"request_reasons"`
	ViewerReview        *ViewerReview      `json:"viewer_review"`
	Authenticated       bool               `json:"authenticated"`
}

type PublishedArticle struct {
	Article
	Reviews ReviewSummary `json:"reviews"`
}

type PublishedArticleOrder string

const (
	PublishedArticleOrderNewest     PublishedArticleOrder = "newest"
	PublishedArticleOrderOldest     PublishedArticleOrder = "oldest"
	PublishedArticleOrderRatingDesc PublishedArticleOrder = "rating-desc"
	PublishedArticleOrderRatingAsc  PublishedArticleOrder = "rating-asc"
)
