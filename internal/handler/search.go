package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/marekvalenta/inventory-management/internal/service"
)

type SearchHandler struct {
	svc *service.SearchService
}

func NewSearchHandler(svc *service.SearchService) *SearchHandler {
	return &SearchHandler{svc: svc}
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := q.Get("q")
	searchType := q.Get("type")
	if searchType == "" {
		searchType = "all"
	}
	limitStr := q.Get("limit")
	limit := 0
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	result, err := h.svc.Search(service.SearchParams{
		Query: query,
		Type:  searchType,
		Limit: limit,
	})
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
