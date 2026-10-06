package articles

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"sourceink/backend/internal/model"
)

type ArticlesWorker struct {
	service         *Service
	gatesMu         sync.Mutex
	repositoryGates map[string]chan struct{}
}

// Webhooks normally make pushes visible immediately. This reconciliation is a
// small safety net for missed deliveries and restarts.
const articlesSyncInterval = 15 * time.Minute

func NewArticlesWorker(service *Service) *ArticlesWorker {
	return &ArticlesWorker{service: service, repositoryGates: make(map[string]chan struct{})}
}

func (w *ArticlesWorker) Start(ctx context.Context) error {
	return runOnInterval(ctx, articlesSyncInterval, func() error {
		_, err := w.SyncOnce(ctx)
		return err
	})
}

func runOnInterval(ctx context.Context, interval time.Duration, run func() error) error {
	if interval <= 0 {
		return errors.New("articles worker interval must be positive")
	}
	if run == nil {
		return errors.New("articles worker run function is required")
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := run(); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
		}
	}
}

func (w *ArticlesWorker) RunOnce(ctx context.Context) ([]model.UnpublishedArticle, error) {
	if w.service == nil || w.service.loader == nil {
		return nil, errors.New("articles worker is not configured")
	}

	return w.service.discoverForSync(ctx)
}

func (w *ArticlesWorker) SyncOnce(ctx context.Context) ([]model.UnpublishedArticle, error) {
	if w.service == nil || w.service.loader == nil {
		return nil, errors.New("articles worker is not configured")
	}
	repositories, err := w.service.store.ListRepositoriesForArticleSync(ctx)
	if err != nil {
		return nil, fmt.Errorf("list repositories for article sync: %w", err)
	}

	unpublishedArticles := make([]model.UnpublishedArticle, 0)
	for _, repository := range repositories {
		repositoryArticles, err := w.syncRepository(ctx, repository)
		if err != nil {
			return nil, fmt.Errorf("sync repository %s: %w", repository.FullName, err)
		}
		unpublishedArticles = append(unpublishedArticles, repositoryArticles...)
	}
	if err := w.service.store.ReconcileDuplicateSlugs(ctx); err != nil {
		return nil, fmt.Errorf("validate article slugs: %w", err)
	}
	if err := w.service.AutoPublishArticles(ctx, unpublishedArticles); err != nil {
		return nil, err
	}

	return unpublishedArticles, nil
}

// SyncRepository handles a verified push for one connected repository.
func (w *ArticlesWorker) SyncRepository(ctx context.Context, installationID, githubRepositoryID int64) error {
	if w.service == nil || w.service.loader == nil {
		return errors.New("articles worker is not configured")
	}
	repository, err := w.service.store.RepositoryForWebhookSync(ctx, installationID, githubRepositoryID)
	if err != nil {
		return fmt.Errorf("find repository for webhook sync: %w", err)
	}
	unpublishedArticles, err := w.syncRepository(ctx, repository)
	if err != nil {
		return fmt.Errorf("sync repository %s: %w", repository.FullName, err)
	}
	if err := w.service.store.ReconcileDuplicateSlugs(ctx); err != nil {
		return fmt.Errorf("validate article slugs: %w", err)
	}
	if err := w.service.AutoPublishArticles(ctx, unpublishedArticles); err != nil {
		return err
	}
	return nil
}

func (w *ArticlesWorker) syncRepository(ctx context.Context, repository model.Repository) ([]model.UnpublishedArticle, error) {
	release, err := w.lockRepository(ctx, repository.ID)
	if err != nil {
		return nil, err
	}
	defer release()

	return w.service.syncRepository(ctx, repository)
}

func (w *ArticlesWorker) lockRepository(ctx context.Context, repositoryID string) (func(), error) {
	if repositoryID == "" {
		return nil, errors.New("repository ID is required")
	}

	w.gatesMu.Lock()
	gate := w.repositoryGates[repositoryID]
	if gate == nil {
		gate = make(chan struct{}, 1)
		w.repositoryGates[repositoryID] = gate
	}
	w.gatesMu.Unlock()

	select {
	case gate <- struct{}{}:
		return func() { <-gate }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
