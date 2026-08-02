package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/marekvalenta/inventory-management/internal/service"
)

type TagHandler struct {
	svc *service.TagService
}

func NewTagHandler(svc *service.TagService) *TagHandler {
	return &TagHandler{svc: svc}
}

type CreateTagRequest struct {
	Name  string  `json:"name"`
	Color *string `json:"color"`
}

type UpdateTagRequest struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

type DeleteTagResponse struct {
	Deleted                bool `json:"deleted"`
	LinkedDefinitionsCount int  `json:"linked_definitions_count"`
}

func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	tags, err := h.svc.GetAll()
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

func (h *TagHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	tag, err := h.svc.GetByID(id)
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tag)
}

func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, service.ErrInvalidInput)
		return
	}

	name := strings.TrimSpace(req.Name)
	if len(name) < 2 || len(name) > 100 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "name must be between 2 and 100 characters",
			Code:  "validation_failed",
		})
		return
	}

	if req.Color != nil {
		trimmed := strings.TrimSpace(*req.Color)
		if len(trimmed) > 10 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: "color must be at most 10 characters",
				Code:  "validation_failed",
			})
			return
		}
	}

	tag, err := h.svc.Create(req.Name, req.Color)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: err.Error(),
				Code:  "duplicate_name",
			})
			return
		}
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tag)
}

func (h *TagHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, service.ErrInvalidInput)
		return
	}

	tag, err := h.svc.Update(id, req.Name, req.Color)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: err.Error(),
				Code:  "duplicate_name",
			})
			return
		}
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tag)
}

func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	linkedCount, err := h.svc.Delete(id)
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DeleteTagResponse{
		Deleted:                true,
		LinkedDefinitionsCount: linkedCount,
	})
}

func (h *TagHandler) RegisterRoutes(r chi.Router) {
	r.Get("/tags", h.List)
	r.Get("/tags/{id}", h.Get)
	r.Post("/tags", h.Create)
	r.Put("/tags/{id}", h.Update)
	r.Delete("/tags/{id}", h.Delete)
}
