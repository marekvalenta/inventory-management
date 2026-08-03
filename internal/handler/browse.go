package handler

import (
	"encoding/json"
	"net/http"

	"github.com/marekvalenta/inventory-management/internal/service"
)

type BrowseHandler struct {
	svc *service.BrowseService
}

func NewBrowseHandler(svc *service.BrowseService) *BrowseHandler {
	return &BrowseHandler{svc: svc}
}

func (h *BrowseHandler) Browse(w http.ResponseWriter, r *http.Request) {
	tree, err := h.svc.GetBrowse()
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tree)
}
