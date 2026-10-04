package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const maxWebhookBodyBytes = 1 << 20

type webhookDeliveryStore interface {
	RecordGitHubWebhookDelivery(ctx context.Context, deliveryID, event string, installationID, repositoryID int64) (bool, error)
	CompleteGitHubWebhookDelivery(ctx context.Context, deliveryID string, processingErr error) error
}

type repositorySync func(context.Context, int64, int64) error

// WebhookHandler accepts GitHub push events and sends only the changed
// repository through the existing article sync flow.
type WebhookHandler struct {
	secret string
	store  webhookDeliveryStore
	sync   repositorySync
	logger *slog.Logger
}

type pushEvent struct {
	Ref          string `json:"ref"`
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
	Repository struct {
		ID            int64  `json:"id"`
		DefaultBranch string `json:"default_branch"`
	} `json:"repository"`
}

func NewWebhookHandler(secret string, store webhookDeliveryStore, sync repositorySync, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{secret: secret, store: store, sync: sync, logger: logger}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h == nil || h.secret == "" || h.store == nil || h.sync == nil {
		http.Error(w, "GitHub webhook is not configured", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes))
	if err != nil {
		http.Error(w, "invalid webhook body", http.StatusRequestEntityTooLarge)
		return
	}
	if !validWebhookSignature(h.secret, r.Header.Get("X-Hub-Signature-256"), body) {
		http.Error(w, "invalid webhook signature", http.StatusUnauthorized)
		return
	}

	deliveryID := strings.TrimSpace(r.Header.Get("X-GitHub-Delivery"))
	event := strings.TrimSpace(r.Header.Get("X-GitHub-Event"))
	if deliveryID == "" || event == "" {
		http.Error(w, "missing GitHub webhook headers", http.StatusBadRequest)
		return
	}
	if event != "push" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var payload pushEvent
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid push webhook payload", http.StatusBadRequest)
		return
	}
	if payload.Installation.ID <= 0 || payload.Repository.ID <= 0 || payload.Repository.DefaultBranch == "" {
		http.Error(w, "invalid push webhook payload", http.StatusBadRequest)
		return
	}
	if payload.Ref != "refs/heads/"+payload.Repository.DefaultBranch {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	newDelivery, err := h.store.RecordGitHubWebhookDelivery(r.Context(), deliveryID, event, payload.Installation.ID, payload.Repository.ID)
	if err != nil {
		h.logger.Error("record GitHub webhook delivery", "error", err, "delivery_id", deliveryID)
		http.Error(w, "could not accept GitHub webhook", http.StatusInternalServerError)
		return
	}
	if !newDelivery {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	go h.process(deliveryID, payload.Installation.ID, payload.Repository.ID)
	w.WriteHeader(http.StatusAccepted)
}

func (h *WebhookHandler) process(deliveryID string, installationID, repositoryID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	err := h.sync(ctx, installationID, repositoryID)
	if completeErr := h.store.CompleteGitHubWebhookDelivery(ctx, deliveryID, err); completeErr != nil {
		h.logger.Error("complete GitHub webhook delivery", "error", completeErr, "delivery_id", deliveryID)
	}
	if err != nil {
		h.logger.Error("sync GitHub webhook repository", "error", err, "delivery_id", deliveryID, "installation_id", installationID, "repository_id", repositoryID)
	}
}

func validWebhookSignature(secret, header string, body []byte) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	expected, err := hex.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil || len(expected) != sha256.Size {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hmac.Equal(expected, mac.Sum(nil))
}
