package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"sourceink/backend/internal/model"
)

func (h *Handler) ListArticles(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}

	articles, err := h.service.ListArticles(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("list articles", "user_id", user.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "SourceInk couldn't load your articles")
		return
	}
	writeJSON(w, http.StatusOK, articles)
}

func (h *Handler) ListPublishedArticles(w http.ResponseWriter, r *http.Request) {
	user, ok := h.optionalCurrentUser(w, r)
	if !ok {
		return
	}

	articles, err := h.service.ListPublishedArticles(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("list published articles", "error", err)
		writeError(w, http.StatusInternalServerError, "SourceInk couldn't load published articles")
		return
	}

	writeJSON(w, http.StatusOK, articles)
}

func (h *Handler) PublishedArticleBySlug(w http.ResponseWriter, r *http.Request) {
	user, ok := h.optionalCurrentUser(w, r)
	if !ok {
		return
	}

	slug := chi.URLParam(r, "slug")
	article, err := h.service.PublishedArticleBySlug(r.Context(), slug, user.ID)
	if errors.Is(err, model.ErrPublishedArticleNotFound) {
		writeError(w, http.StatusNotFound, model.ErrPublishedArticleNotFound.Error())
		return
	}
	if err != nil {
		h.logger.Error("load published article", "slug", slug, "error", err)
		writeError(w, http.StatusInternalServerError, "SourceInk couldn't load the article")
		return
	}

	writeJSON(w, http.StatusOK, article)
}

func (h *Handler) PublishArticle(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return // currentUser already writes 401
	}

	draftID := chi.URLParam(r, "draftID")

	article, err := h.service.PublishArticle(r.Context(), user.ID, draftID)
	switch {
	case errors.Is(err, model.ErrArticleNotFound):
		writeError(w, http.StatusNotFound, model.ErrArticleNotFound.Error())
	case errors.Is(err, model.ErrArticleNotPublishable):
		writeError(w, http.StatusUnprocessableEntity, model.ErrArticleNotPublishable.Error())
	case err != nil:
		h.logger.Error("publish article", "user_id", user.ID, "draft_id", draftID, "error", err)
		writeError(w, http.StatusInternalServerError, "SourceInk couldn't publish the article")
	default:
		article.AuthorUsername = user.Username
		writeJSON(w, http.StatusCreated, article)
	}
}
