package articles

import (
	"context"
	"errors"
	"fmt"

	"sourceink/backend/internal/model"
)

type Store interface {
	ListUserArticles(ctx context.Context, userID string) ([]model.Article, error)
	ListUserUnpublishedArticles(ctx context.Context, userID string) ([]model.UnpublishedArticle, error)
	ListUnpublishedArticlesByRepository(ctx context.Context, repositoryID string) ([]model.UnpublishedArticle, error)
	ListRepositoriesForArticleSync(ctx context.Context) ([]model.Repository, error)
	RepositoryForWebhookSync(ctx context.Context, installationID, githubRepositoryID int64) (model.Repository, error)
	UpsertUnpublishedArticle(ctx context.Context, article *model.UnpublishedArticle) error
	MarkMissingUnpublishedArticles(ctx context.Context, repositoryID string, presentPaths []string) error
	ReconcileDuplicateSlugs(ctx context.Context) error
	FindUnpublishedArticleForPublish(ctx context.Context, userID, draftID string) (model.UnpublishedArticle, error)
	AutoFindUnpublishedArticleForPublish(ctx context.Context, draftID string) (model.UnpublishedArticle, error)
	PublishArticle(ctx context.Context, userID, draftID, gitBlobSHA string) (model.Article, error)
	AutoPublishArticle(ctx context.Context, draftID, gitBlobSHA string) (model.Article, error)
}

type ArticleLoader interface {
	// Files whose blob SHA matches knownBlobSHAs[path] may come back without content.
	LoadUnpublishedArticlesByRepo(ctx context.Context, repository model.Repository, knownBlobSHAs map[string]string) ([]model.UnpublishedArticle, error)
}

type Service struct {
	store  Store
	loader ArticleLoader
}

func NewService(store Store, loader ArticleLoader) *Service {
	return &Service{store: store, loader: loader}
}

func (s *Service) ListForUser(ctx context.Context, userID string) (model.UserArticles, error) {
	publishedArticles, err := s.store.ListUserArticles(ctx, userID)
	if err != nil {
		return model.UserArticles{}, fmt.Errorf("list published articles: %w", err)
	}
	unpublishedArticles, err := s.store.ListUserUnpublishedArticles(ctx, userID)
	if err != nil {
		return model.UserArticles{}, fmt.Errorf("list unpublished articles: %w", err)
	}
	return model.UserArticles{
		UserID:              userID,
		UnpublishedArticles: unpublishedArticles,
		PublishedArticles:   publishedArticles,
	}, nil
}

// PublishArticle publishes a manual draft on behalf of its owner.
func (s *Service) PublishArticle(ctx context.Context, userID, draftID string) (model.Article, error) {
	if userID == "" {
		return model.Article{}, fmt.Errorf("publish article: %w", model.ErrArticleNotFound)
	}
	return s.publish(ctx, draftID, model.ArticlePublishModeManual,
		func() (model.UnpublishedArticle, error) {
			return s.store.FindUnpublishedArticleForPublish(ctx, userID, draftID)
		},
		func(gitBlobSHA string) (model.Article, error) {
			return s.store.PublishArticle(ctx, userID, draftID, gitBlobSHA)
		},
	)
}

// AutoPublishArticle publishes an automatic draft after a successful sync.
func (s *Service) AutoPublishArticle(ctx context.Context, draftID string) (model.Article, error) {
	return s.publish(ctx, draftID, model.ArticlePublishModeAuto,
		func() (model.UnpublishedArticle, error) {
			return s.store.AutoFindUnpublishedArticleForPublish(ctx, draftID)
		},
		func(gitBlobSHA string) (model.Article, error) {
			return s.store.AutoPublishArticle(ctx, draftID, gitBlobSHA)
		},
	)
}

func (s *Service) publish(
	ctx context.Context,
	draftID string,
	mode model.ArticlePublishMode,
	find func() (model.UnpublishedArticle, error),
	store func(gitBlobSHA string) (model.Article, error),
) (model.Article, error) {
	if err := ctx.Err(); err != nil {
		return model.Article{}, fmt.Errorf("publish article: %w", err)
	}
	if draftID == "" {
		return model.Article{}, fmt.Errorf("publish article: %w", model.ErrArticleNotFound)
	}

	draft, err := find()
	if errors.Is(err, model.ErrArticleNotFound) {
		return model.Article{}, fmt.Errorf("publish article %s: %w", draftID, model.ErrArticleNotFound)
	}
	if err != nil {
		return model.Article{}, fmt.Errorf("load article draft %s for publication: %w", draftID, err)
	}

	draft, err = s.ValidateArticle(ctx, draft)
	if errors.Is(err, model.ErrArticleNotPublishable) || (err == nil && draft.PublishMode != string(mode)) {
		return model.Article{}, fmt.Errorf("publish article %s: %w", draftID, model.ErrArticleNotPublishable)
	}
	if err != nil {
		return model.Article{}, fmt.Errorf("validate article draft %s for publication: %w", draftID, err)
	}

	article, err := store(draft.GitBlobSHA)
	if err != nil {
		return model.Article{}, fmt.Errorf("publish article %s: %w", draftID, err)
	}
	return article, nil
}

func (s *Service) AutoPublishArticles(ctx context.Context, unpublishedArticles []model.UnpublishedArticle) error {
	for _, article := range unpublishedArticles {
		if article.ValidationError != nil || article.PublishMode != string(model.ArticlePublishModeAuto) {
			continue
		}
		if _, err := s.AutoPublishArticle(ctx, article.ID); err != nil {
			return fmt.Errorf("auto-publish article %s: %w", article.ID, err)
		}
	}
	return nil
}

func (s *Service) syncRepository(ctx context.Context, repository model.Repository) ([]model.UnpublishedArticle, error) {
	if s.loader == nil {
		return nil, errors.New("article discovery is not configured")
	}
	stored, err := s.store.ListUnpublishedArticlesByRepository(ctx, repository.ID)
	if err != nil {
		return nil, fmt.Errorf("list stored drafts: %w", err)
	}
	known := make(map[string]model.UnpublishedArticle, len(stored))
	knownBlobSHAs := make(map[string]string, len(stored))
	for _, draft := range stored {
		if draft.Present {
			known[draft.SourcePath] = draft
			knownBlobSHAs[draft.SourcePath] = draft.GitBlobSHA
		}
	}

	articles, err := s.loader.LoadUnpublishedArticlesByRepo(ctx, repository, knownBlobSHAs)
	if err != nil {
		return nil, fmt.Errorf("load unpublished articles from %s: %w", repository.FullName, err)
	}

	presentPaths := make([]string, 0, len(articles))
	for i := range articles {
		article := &articles[i]
		presentPaths = append(presentPaths, article.SourcePath)
		if draft, ok := known[article.SourcePath]; ok && draft.GitBlobSHA == article.GitBlobSHA {
			*article = draft // same revision as last sync: nothing to validate or store
			continue
		}
		validated, err := s.ValidateArticle(ctx, *article)
		if err != nil && !errors.Is(err, model.ErrArticleNotPublishable) {
			return nil, fmt.Errorf("validate unpublished article %s: %w", article.SourcePath, err)
		}
		*article = validated
		if err := s.store.UpsertUnpublishedArticle(ctx, article); err != nil {
			return nil, fmt.Errorf("upsert unpublished article %s: %w", article.SourcePath, err)
		}
	}
	if err := s.store.MarkMissingUnpublishedArticles(ctx, repository.ID, presentPaths); err != nil {
		return nil, fmt.Errorf("mark missing unpublished articles: %w", err)
	}
	return articles, nil
}
