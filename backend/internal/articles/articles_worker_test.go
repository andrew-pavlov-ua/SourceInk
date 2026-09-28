package articles

import (
	"context"
	"errors"
	"testing"
	"time"

	"sourceink/backend/internal/model"
)

func TestRunOnIntervalWaitsForIntervalAndRepeatsUntilCancelled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	runs := 0
	err := runOnInterval(ctx, time.Millisecond, func() error {
		runs++
		if runs == 2 {
			cancel()
		}
		return nil
	})
	if err != nil {
		t.Fatalf("runOnInterval() error = %v", err)
	}
	if runs < 2 {
		t.Fatalf("runs = %d, want at least 2", runs)
	}
}

func TestSyncOnceDiscoversAndSavesArticles(t *testing.T) {
	repository := model.Repository{ID: "repository-id", FullName: "octocat/docs"}
	store := &serviceTestStore{syncRepositories: []model.Repository{repository}}
	loader := &serviceTestLoader{articlesByRepoID: map[string][]model.UnpublishedArticle{
		repository.ID: {{RepositoryID: repository.ID, SourcePath: "README.md"}},
	}}
	worker := NewArticlesWorker(NewService(store, loader))

	articles, err := worker.SyncOnce(context.Background())
	if err != nil {
		t.Fatalf("SyncOnce() error = %v", err)
	}
	if len(articles) != 1 || len(store.upsertedSourcePath) != 1 || store.upsertedSourcePath[0] != "README.md" {
		t.Fatalf("SyncOnce() articles = %#v, saved paths = %#v", articles, store.upsertedSourcePath)
	}
}

func TestRunOnIntervalReturnsRunError(t *testing.T) {
	wantErr := errors.New("sync failed")
	runs := 0

	err := runOnInterval(context.Background(), time.Millisecond, func() error {
		runs++
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("runOnInterval() error = %v, want %v", err, wantErr)
	}
	if runs != 1 {
		t.Fatalf("runs = %d, want 1", runs)
	}
}

func TestRunOnIntervalDoesNotRunAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	runs := 0
	err := runOnInterval(ctx, time.Millisecond, func() error {
		runs++
		return nil
	})
	if err != nil {
		t.Fatalf("runOnInterval() error = %v", err)
	}
	if runs != 0 {
		t.Fatalf("runs = %d, want 0", runs)
	}
}
