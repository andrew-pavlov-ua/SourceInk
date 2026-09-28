package api

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	repos, err := h.service.ListRepositories(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("list repositories", "error", err)
		writeError(w, http.StatusInternalServerError, "SourceInk couldn't load your repositories")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(repos)
	if err != nil {
		h.logger.Error("encode repositories response", "error", err)
	}
}
