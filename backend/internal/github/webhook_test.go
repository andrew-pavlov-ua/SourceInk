package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

const webhookBody = `{"ref":"refs/heads/main","installation":{"id":44},"repository":{"id":99,"default_branch":"main"}}`

type webhookTestStore struct {
	mu           sync.Mutex
	newDelivery  bool
	recordedID   string
	completedID  string
	completedErr error
}

func (s *webhookTestStore) RecordGitHubWebhookDelivery(_ context.Context, deliveryID, _ string, _, _ int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recordedID = deliveryID
	return s.newDelivery, nil
}

func (s *webhookTestStore) CompleteGitHubWebhookDelivery(_ context.Context, deliveryID string, processingErr error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.completedID = deliveryID
	s.completedErr = processingErr
	return nil
}

func TestWebhookPushVerifiesDeduplicatesAndSyncsRepository(t *testing.T) {
	store := &webhookTestStore{newDelivery: true}
	synced := make(chan [2]int64, 1)
	handler := NewWebhookHandler("secret", store, func(_ context.Context, installationID, repositoryID int64) error {
		synced <- [2]int64{installationID, repositoryID}
		return nil
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	request := signedWebhookRequest(t, webhookBody, "delivery-1")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	select {
	case got := <-synced:
		if got != [2]int64{44, 99} {
			t.Fatalf("sync target = %#v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("webhook did not start repository sync")
	}
	deadline := time.Now().Add(time.Second)
	for {
		store.mu.Lock()
		complete := store.completedID
		store.mu.Unlock()
		if complete == "delivery-1" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("webhook did not complete delivery")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestWebhookRejectsInvalidSignature(t *testing.T) {
	store := &webhookTestStore{newDelivery: true}
	handler := NewWebhookHandler("secret", store, func(context.Context, int64, int64) error {
		t.Fatal("sync should not run")
		return nil
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/api/github/webhook", strings.NewReader(webhookBody))
	request.Header.Set("X-GitHub-Event", "push")
	request.Header.Set("X-GitHub-Delivery", "delivery-1")
	request.Header.Set("X-Hub-Signature-256", "sha256=bad")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if store.recordedID != "" {
		t.Fatalf("recorded delivery = %q", store.recordedID)
	}
}

func TestWebhookDoesNotResyncDuplicateDelivery(t *testing.T) {
	store := &webhookTestStore{newDelivery: false}
	handler := NewWebhookHandler("secret", store, func(context.Context, int64, int64) error {
		t.Fatal("sync should not run")
		return errors.New("unexpected sync")
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, signedWebhookRequest(t, webhookBody, "delivery-1"))
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
}

func TestWebhookIgnoresNonDefaultBranchPush(t *testing.T) {
	store := &webhookTestStore{newDelivery: true}
	handler := NewWebhookHandler("secret", store, func(context.Context, int64, int64) error {
		t.Fatal("sync should not run")
		return nil
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	body := strings.Replace(webhookBody, "refs/heads/main", "refs/heads/feature", 1)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, signedWebhookRequest(t, body, "delivery-1"))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if store.recordedID != "" {
		t.Fatalf("recorded delivery = %q", store.recordedID)
	}
}

func signedWebhookRequest(t *testing.T, body, deliveryID string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/github/webhook", strings.NewReader(body))
	request.Header.Set("X-GitHub-Event", "push")
	request.Header.Set("X-GitHub-Delivery", deliveryID)
	request.Header.Set("X-Hub-Signature-256", testWebhookSignature("secret", []byte(body)))
	return request
}

func testWebhookSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return fmt.Sprintf("sha256=%x", mac.Sum(nil))
}
