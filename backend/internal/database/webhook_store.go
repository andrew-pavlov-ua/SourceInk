package database

import (
	"context"
	"fmt"
)

// RecordGitHubWebhookDelivery returns false for a delivery GitHub has already
// sent to this SourceInk instance.
func (s *Store) RecordGitHubWebhookDelivery(ctx context.Context, deliveryID, event string, installationID, repositoryID int64) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		insert into github_webhook_deliveries (
			delivery_id, event, installation_id, repository_github_id, status
		)
		values ($1, $2, $3, $4, 'received')
		on conflict (delivery_id) do nothing
	`, deliveryID, event, installationID, repositoryID)
	if err != nil {
		return false, fmt.Errorf("record GitHub webhook delivery: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read GitHub webhook delivery result: %w", err)
	}
	return inserted == 1, nil
}

func (s *Store) CompleteGitHubWebhookDelivery(ctx context.Context, deliveryID string, processingErr error) error {
	status := "processed"
	var failure any
	if processingErr != nil {
		status = "failed"
		failure = processingErr.Error()
	}
	_, err := s.db.ExecContext(ctx, `
		update github_webhook_deliveries
		set status = $2, error = $3, processed_at = now()
		where delivery_id = $1
	`, deliveryID, status, failure)
	if err != nil {
		return fmt.Errorf("complete GitHub webhook delivery: %w", err)
	}
	return nil
}
