package api_test

import (
	"context"
	"errors"
	"testing"

	api "sourceink/backend/internal/api/service"
	"sourceink/backend/internal/model"
)

type reviewServiceStore struct {
	createCalls int
	deleteCalls int
	listCalls   int
	created     model.Review
	reviews     []model.Review
}

func (s *reviewServiceStore) ListReviewsForArticle(context.Context, string, string) ([]model.Review, error) {
	s.listCalls++
	if s.reviews != nil {
		return s.reviews, nil
	}
	return []model.Review{s.created}, nil
}

func (s *reviewServiceStore) CreateReview(_ context.Context, review *model.Review) error {
	s.createCalls++
	review.ID = "review-id"
	s.created = *review
	return nil
}

func (s *reviewServiceStore) DeleteReview(context.Context, string, string, string) error {
	s.deleteCalls++
	return nil
}

func TestReviewServiceCreatesValidatedReview(t *testing.T) {
	store := &reviewServiceStore{}
	service := api.NewReviewService(store)
	reason := model.ReasonOutdated

	review, err := service.CreateReview(
		context.Background(), "article-id", "blob-sha", "reviewer-id",
		model.VerdictRequestChanges, &reason,
	)
	if err != nil {
		t.Fatalf("CreateReview() error = %v", err)
	}
	if store.createCalls != 1 || review.ID != "review-id" || review.ReviewerID != "reviewer-id" {
		t.Fatalf("CreateReview() = %#v, calls = %d", review, store.createCalls)
	}
}

func TestReviewServiceRejectsInvalidReviewsBeforeStore(t *testing.T) {
	reason := model.ReasonOutdated
	unknownReason := model.Reason("spam")
	tests := []struct {
		name       string
		articleID  string
		gitBlobSHA string
		reviewerID string
		verdict    model.Verdict
		reason     *model.Reason
	}{
		{name: "missing article", gitBlobSHA: "blob-sha", reviewerID: "reviewer-id", verdict: model.VerdictApprove},
		{name: "missing revision", articleID: "article-id", reviewerID: "reviewer-id", verdict: model.VerdictApprove},
		{name: "missing reviewer", articleID: "article-id", gitBlobSHA: "blob-sha", verdict: model.VerdictApprove},
		{name: "unknown verdict", articleID: "article-id", gitBlobSHA: "blob-sha", reviewerID: "reviewer-id", verdict: "like"},
		{name: "approve with reason", articleID: "article-id", gitBlobSHA: "blob-sha", reviewerID: "reviewer-id", verdict: model.VerdictApprove, reason: &reason},
		{name: "request changes without reason", articleID: "article-id", gitBlobSHA: "blob-sha", reviewerID: "reviewer-id", verdict: model.VerdictRequestChanges},
		{name: "request changes with unknown reason", articleID: "article-id", gitBlobSHA: "blob-sha", reviewerID: "reviewer-id", verdict: model.VerdictRequestChanges, reason: &unknownReason},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &reviewServiceStore{}
			service := api.NewReviewService(store)
			_, err := service.CreateReview(
				context.Background(), test.articleID, test.gitBlobSHA, test.reviewerID,
				test.verdict, test.reason,
			)
			if !errors.Is(err, model.ErrArticleReviewInvalid) {
				t.Fatalf("CreateReview() error = %v, want ErrArticleReviewInvalid", err)
			}
			if store.createCalls != 0 {
				t.Fatalf("CreateReview() store calls = %d, want 0", store.createCalls)
			}
		})
	}
}

func TestReviewServiceBuildsPrivateAggregateSummary(t *testing.T) {
	outdated := model.ReasonOutdated
	unclear := model.ReasonUnclear
	store := &reviewServiceStore{reviews: []model.Review{
		{ReviewerID: "reader-one", Verdict: model.VerdictApprove},
		{ReviewerID: "reader-two", Verdict: model.VerdictRequestChanges, Reason: &outdated},
		{ReviewerID: "reader-three", Verdict: model.VerdictRequestChanges, Reason: &unclear},
	}}
	service := api.NewReviewService(store)

	summary, err := service.GetReviewSummary(context.Background(), "article-id", "blob-sha", "reader-two")
	if err != nil {
		t.Fatalf("GetReviewSummary() error = %v", err)
	}
	if summary.ApproveCount != 1 || summary.RequestChangesCount != 2 {
		t.Fatalf("GetReviewSummary() counts = approve:%d changes:%d", summary.ApproveCount, summary.RequestChangesCount)
	}
	if summary.RequestReasons.Outdated != 1 || summary.RequestReasons.Unclear != 1 || summary.RequestReasons.Incorrect != 0 {
		t.Fatalf("GetReviewSummary() reasons = %#v", summary.RequestReasons)
	}
	if !summary.Authenticated || summary.ViewerReview == nil || summary.ViewerReview.Reason == nil || *summary.ViewerReview.Reason != outdated {
		t.Fatalf("GetReviewSummary() viewer review = %#v", summary.ViewerReview)
	}
}

func TestReviewServiceValidatesListAndDeleteInputs(t *testing.T) {
	store := &reviewServiceStore{}
	service := api.NewReviewService(store)

	if _, err := service.ListReviewsForArticle(context.Background(), "", "blob-sha"); !errors.Is(err, model.ErrArticleReviewInvalid) {
		t.Fatalf("ListReviewsForArticle() error = %v, want ErrArticleReviewInvalid", err)
	}
	if err := service.DeleteReview(context.Background(), "article-id", "blob-sha", " "); !errors.Is(err, model.ErrArticleReviewInvalid) {
		t.Fatalf("DeleteReview() error = %v, want ErrArticleReviewInvalid", err)
	}
	if store.listCalls != 0 || store.deleteCalls != 0 {
		t.Fatalf("invalid requests reached store: list=%d delete=%d", store.listCalls, store.deleteCalls)
	}
}
