package api

import (
	"encoding/json"
	"net/http"

	"sourceink/backend/internal/auth"
)

func (h *Handler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}

	repos, err := h.service.ListRepositories(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("list repositories", "error", err)
		auth.WriteError(w, http.StatusInternalServerError, "SourceInk couldn't load your repositories")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(repos)
	if err != nil {
		h.logger.Error("encode repositories response", "error", err)
	}
}
