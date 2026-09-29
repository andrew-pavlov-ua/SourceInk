package articles

import (
	"context"
	"errors"
	"fmt"
	"time"

	"sourceink/backend/internal/model"
)

type ArticlesWorker struct {
	service *Service
}

const articlesSyncInterval = 15 * time.Second

func NewArticlesWorker(service *Service) *ArticlesWorker {
	return &ArticlesWorker{service: service}
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
	unpublishedArticles, err := w.RunOnce(ctx)
	if err != nil {
		return nil, fmt.Errorf("discover unpublished articles: %w", err)
	}

	if err := w.service.save(ctx, unpublishedArticles); err != nil {
		return nil, fmt.Errorf("save unpublished articles: %w", err)
	}
	if err := w.service.store.ReconcileDuplicateSlugs(ctx); err != nil {
		return nil, fmt.Errorf("validate article slugs: %w", err)
	}
	if err := w.AutoPublishArticles(ctx, unpublishedArticles); err != nil {
		return nil, err
	}

	return unpublishedArticles, nil
}

func (w *ArticlesWorker) AutoPublishArticles(ctx context.Context, unpublishedArticles []model.UnpublishedArticle) error {
	for _, article := range unpublishedArticles {
		if article.ValidationError != nil {
			continue
		}

		if article.Frontmatter.PublishMode == string(model.ArticlePublishModeAuto) {
			_, err := w.service.AutoPublishArticle(ctx, article.ID)
			if err != nil {
				return fmt.Errorf("auto-publish article %s: %w", article.ID, err)
			}
		}
	}

	return nil
}
