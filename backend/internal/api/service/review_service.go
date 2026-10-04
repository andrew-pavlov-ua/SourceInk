package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"sourceink/backend/internal/model"
)

type ReviewStore interface {
	ListReviewsForArticle(ctx context.Context, articleID, gitBlobSHA string) ([]model.Review, error)
	CreateReview(ctx context.Context, review *model.Review) error
	DeleteReview(ctx context.Context, articleID, gitBlobSHA, reviewerID string) error
}

type ReviewService struct {
	store ReviewStore
}

func NewReviewService(store ReviewStore) *ReviewService {
	return &ReviewService{store: store}
}

func (s *Service) ListReviewsForArticle(
	ctx context.Context,
	articleID string,
	gitBlobSHA string,
) ([]model.Review, error) {
	return s.reviews.ListReviewsForArticle(ctx, articleID, gitBlobSHA)
}

func (s *Service) GetReviewSummary(
	ctx context.Context,
	articleID string,
	gitBlobSHA string,
	viewerID string,
) (model.ReviewSummary, error) {
	return s.reviews.GetReviewSummary(ctx, articleID, gitBlobSHA, viewerID)
}

func (s *Service) CreateReview(
	ctx context.Context,
	articleID string,
	gitBlobSHA string,
	reviewerID string,
	verdict model.Verdict,
	reason *model.Reason,
) (model.Review, error) {
	return s.reviews.CreateReview(ctx, articleID, gitBlobSHA, reviewerID, verdict, reason)
}

func (s *Service) DeleteReview(
	ctx context.Context,
	articleID string,
	gitBlobSHA string,
	reviewerID string,
) error {
	return s.reviews.DeleteReview(ctx, articleID, gitBlobSHA, reviewerID)
}

func (s *ReviewService) ListReviewsForArticle(
	ctx context.Context,
	articleID string,
	gitBlobSHA string,
) ([]model.Review, error) {
	if err := validateArticleRevision(articleID, gitBlobSHA); err != nil {
		return nil, err
	}
	if s == nil || s.store == nil {
		return nil, errors.New("review service is not configured")
	}

	reviews, err := s.store.ListReviewsForArticle(ctx, articleID, gitBlobSHA)
	if err != nil {
		return nil, fmt.Errorf("list reviews for article %s: %w", articleID, err)
	}
	return reviews, nil
}

func (s *ReviewService) GetReviewSummary(
	ctx context.Context,
	articleID string,
	gitBlobSHA string,
	viewerID string,
) (model.ReviewSummary, error) {
	reviews, err := s.ListReviewsForArticle(ctx, articleID, gitBlobSHA)
	if err != nil {
		return model.ReviewSummary{}, err
	}

	summary := model.ReviewSummary{
		ArticleID:     articleID,
		GitBlobSHA:    gitBlobSHA,
		Authenticated: viewerID != "",
	}
	for _, review := range reviews {
		switch review.Verdict {
		case model.VerdictApprove:
			summary.ApproveCount++
		case model.VerdictRequestChanges:
			summary.RequestChangesCount++
			if review.Reason != nil {
				switch *review.Reason {
				case model.ReasonIncorrect:
					summary.RequestReasons.Incorrect++
				case model.ReasonOutdated:
					summary.RequestReasons.Outdated++
				case model.ReasonUnclear:
					summary.RequestReasons.Unclear++
				}
			}
		}
		if viewerID != "" && review.ReviewerID == viewerID {
			summary.ViewerReview = &model.ViewerReview{
				Verdict: review.Verdict,
				Reason:  review.Reason,
			}
		}
	}
	return summary, nil
}

func (s *ReviewService) CreateReview(
	ctx context.Context,
	articleID string,
	gitBlobSHA string,
	reviewerID string,
	verdict model.Verdict,
	reason *model.Reason,
) (model.Review, error) {
	if err := validateArticleRevision(articleID, gitBlobSHA); err != nil {
		return model.Review{}, err
	}
	if err := validateReviewerID(reviewerID); err != nil {
		return model.Review{}, err
	}
	if err := validateReviewVerdict(verdict, reason); err != nil {
		return model.Review{}, err
	}
	if s == nil || s.store == nil {
		return model.Review{}, errors.New("review service is not configured")
	}

	review := model.Review{
		ArticleID:  articleID,
		GitBlobSHA: gitBlobSHA,
		ReviewerID: reviewerID,
		Verdict:    verdict,
		Reason:     reason,
	}
	if err := s.store.CreateReview(ctx, &review); err != nil {
		return model.Review{}, fmt.Errorf("create review for article %s: %w", articleID, err)
	}
	return review, nil
}

func (s *ReviewService) DeleteReview(
	ctx context.Context,
	articleID string,
	gitBlobSHA string,
	reviewerID string,
) error {
	if err := validateArticleRevision(articleID, gitBlobSHA); err != nil {
		return err
	}
	if err := validateReviewerID(reviewerID); err != nil {
		return err
	}
	if s == nil || s.store == nil {
		return errors.New("review service is not configured")
	}

	if err := s.store.DeleteReview(ctx, articleID, gitBlobSHA, reviewerID); err != nil {
		return fmt.Errorf("delete review for article %s: %w", articleID, err)
	}
	return nil
}

func validateArticleRevision(articleID, gitBlobSHA string) error {
	switch {
	case strings.TrimSpace(articleID) == "":
		return fmt.Errorf("article ID is required: %w", model.ErrArticleReviewInvalid)
	case strings.TrimSpace(gitBlobSHA) == "":
		return fmt.Errorf("Git blob SHA is required: %w", model.ErrArticleReviewInvalid)
	}
	return nil
}

func validateReviewerID(reviewerID string) error {
	if strings.TrimSpace(reviewerID) == "" {
		return fmt.Errorf("reviewer ID is required: %w", model.ErrArticleReviewInvalid)
	}
	return nil
}

func validateReviewVerdict(verdict model.Verdict, reason *model.Reason) error {
	switch verdict {
	case model.VerdictApprove:
		if reason != nil {
			return fmt.Errorf("approve review cannot have a reason: %w", model.ErrArticleReviewInvalid)
		}
	case model.VerdictRequestChanges:
		if reason == nil || !validReviewReason(*reason) {
			return fmt.Errorf("request changes review requires a valid reason: %w", model.ErrArticleReviewInvalid)
		}
	default:
		return fmt.Errorf("unknown review verdict %q: %w", verdict, model.ErrArticleReviewInvalid)
	}
	return nil
}

func validReviewReason(reason model.Reason) bool {
	switch reason {
	case model.ReasonIncorrect, model.ReasonOutdated, model.ReasonUnclear:
		return true
	default:
		return false
	}
}
