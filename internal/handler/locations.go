package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/marekvalenta/inventory-management/internal/service"
)

type LocationHandler struct {
	svc *service.LocationService
}

func NewLocationHandler(svc *service.LocationService) *LocationHandler {
	return &LocationHandler{svc: svc}
}

type CreateLocationRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	ParentID    *string `json:"parent_id"`
}

type UpdateLocationRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	ParentID    *string `json:"parent_id"`
}

func (h *LocationHandler) List(w http.ResponseWriter, r *http.Request) {
	parentIDStr := r.URL.Query().Get("parent_id")
	var parentID *string
	if parentIDStr != "" {
		parentID = &parentIDStr
	}

	locations, err := h.svc.List(parentID)
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(locations)
}

func (h *LocationHandler) Tree(w http.ResponseWriter, r *http.Request) {
	tree, err := h.svc.GetTree()
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tree)
}

func (h *LocationHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	location, err := h.svc.GetByID(id)
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(location)
}

func (h *LocationHandler) Children(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	children, err := h.svc.GetChildren(id)
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(children)
}

func (h *LocationHandler) Contents(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	contents, err := h.svc.GetContents(id)
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contents)
}

func (h *LocationHandler) Breadcrumb(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	breadcrumb, err := h.svc.GetBreadcrumb(id)
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(breadcrumb)
}

func (h *LocationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, service.ErrInvalidInput)
		return
	}

	name := strings.TrimSpace(req.Name)
	if len(name) < 2 || len(name) > 200 {
		RespondWithError(w, service.ErrInvalidInput)
		return
	}

	if req.Description != nil {
		trimmed := strings.TrimSpace(*req.Description)
		if len(trimmed) > 2000 {
			RespondWithError(w, service.ErrInvalidInput)
			return
		}
	}

	location, err := h.svc.Create(name, req.Description, req.ParentID)
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(location)
}

func (h *LocationHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, service.ErrInvalidInput)
		return
	}

	location, err := h.svc.Update(id, req.Name, req.Description, req.ParentID)
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(location)
}

func (h *LocationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	block, err := h.svc.Delete(id)
	if err != nil {
		if block != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: "Cannot delete: location has sub-locations or item instances",
				Code:  "location_not_empty",
			})
			return
		}
		RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *LocationHandler) RegisterRoutes(r chi.Router) {
	r.Get("/locations", h.List)
	r.Get("/locations/tree", h.Tree)
	r.Get("/locations/{id}", h.Get)
	r.Get("/locations/{id}/children", h.Children)
	r.Get("/locations/{id}/contents", h.Contents)
	r.Get("/locations/{id}/breadcrumb", h.Breadcrumb)
	r.Post("/locations", h.Create)
	r.Put("/locations/{id}", h.Update)
	r.Delete("/locations/{id}", h.Delete)
}
