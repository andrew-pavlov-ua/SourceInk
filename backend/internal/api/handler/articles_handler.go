package api

import "net/http"

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
