package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"sourceink/backend/internal/model"
)

const maxReviewRequestBytes = 16 << 10

type reviewRequest struct {
	GitBlobSHA string        `json:"git_blob_sha"`
	Verdict    model.Verdict `json:"verdict"`
	Reason     *model.Reason `json:"reason"`
}

func (h *Handler) GetReviewSummary(w http.ResponseWriter, r *http.Request) {
	user, ok := h.optionalCurrentUser(w, r)
	if !ok {
		return
	}

	articleID := chi.URLParam(r, "articleID")
	gitBlobSHA := r.URL.Query().Get("git_blob_sha")
	summary, err := h.service.GetReviewSummary(r.Context(), articleID, gitBlobSHA, user.ID)
	if err != nil {
		h.writeReviewError(w, r, "get review summary", articleID, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}

	var request reviewRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxReviewRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "Review request must be valid JSON")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "Review request must contain one JSON object")
		return
	}

	articleID := chi.URLParam(r, "articleID")
	if _, err := h.service.CreateReview(
		r.Context(),
		articleID,
		request.GitBlobSHA,
		user.ID,
		request.Verdict,
		request.Reason,
	); err != nil {
		h.writeReviewError(w, r, "create review", articleID, err)
		return
	}

	summary, err := h.service.GetReviewSummary(r.Context(), articleID, request.GitBlobSHA, user.ID)
	if err != nil {
		h.writeReviewError(w, r, "reload review summary", articleID, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}

	articleID := chi.URLParam(r, "articleID")
	gitBlobSHA := r.URL.Query().Get("git_blob_sha")
	if err := h.service.DeleteReview(r.Context(), articleID, gitBlobSHA, user.ID); err != nil {
		h.writeReviewError(w, r, "delete review", articleID, err)
		return
	}

	summary, err := h.service.GetReviewSummary(r.Context(), articleID, gitBlobSHA, user.ID)
	if err != nil {
		h.writeReviewError(w, r, "reload review summary", articleID, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) writeReviewError(
	w http.ResponseWriter,
	r *http.Request,
	operation string,
	articleID string,
	err error,
) {
	switch {
	case errors.Is(err, model.ErrArticleReviewInvalid):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, model.ErrArticleReviewStale):
		writeError(w, http.StatusConflict, "A newer article revision is available. Refresh before reviewing.")
	case errors.Is(err, model.ErrPublishedArticleNotFound), errors.Is(err, model.ErrArticleReviewNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		h.logger.Error(operation, "article_id", articleID, "path", r.URL.Path, "error", err)
		writeError(w, http.StatusInternalServerError, "SourceInk couldn't update this review")
	}
}
