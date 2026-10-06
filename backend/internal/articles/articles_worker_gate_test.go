package articles

import (
	"context"
	"testing"
	"time"
)

func TestRepositoryGateSerializesTheSameRepository(t *testing.T) {
	worker := NewArticlesWorker(nil)
	firstRelease, err := worker.lockRepository(context.Background(), "repository-a")
	if err != nil {
		t.Fatalf("lock first repository sync: %v", err)
	}

	secondAcquired := make(chan func(), 1)
	go func() {
		release, err := worker.lockRepository(context.Background(), "repository-a")
		if err == nil {
			secondAcquired <- release
		}
	}()

	select {
	case release := <-secondAcquired:
		release()
		t.Fatal("second sync acquired the repository gate before the first released it")
	case <-time.After(20 * time.Millisecond):
	}

	firstRelease()
	select {
	case release := <-secondAcquired:
		release()
	case <-time.After(time.Second):
		t.Fatal("second sync did not acquire the repository gate after release")
	}
}
