package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/marekvalenta/inventory-management/internal/service"
)

type DefinitionHandler struct {
	svc *service.DefinitionService
}

func NewDefinitionHandler(svc *service.DefinitionService) *DefinitionHandler {
	return &DefinitionHandler{svc: svc}
}

type CreateDefinitionRequest struct {
	Name        *string                    `json:"name" validate:"required"`
	Description *string                    `json:"description"`
	ParentDefID *string                    `json:"parent_def_id"`
	Unit        *string                    `json:"unit"`
	IsContainer *bool                       `json:"is_container"`
	Fields      []CreateFieldRequest       `json:"fields"`
	TagIDs      []string                   `json:"tag_ids"`
}

type CreateFieldRequest struct {
	FieldName       string           `json:"field_name" validate:"required"`
	FieldType       string           `json:"field_type" validate:"required,oneof=text number boolean date enum"`
	EnumValues      *json.RawMessage `json:"enum_values"`
	IsRequired      bool             `json:"is_required"`
	DisplayOrder    int              `json:"display_order"`
	DefaultValue    *string          `json:"default_value"`
	IsChildEditable bool             `json:"is_child_editable"`
}

type UpdateDefinitionRequest struct {
	Name        *string             `json:"name"`
	Description *string             `json:"description"`
	ParentDefID *string             `json:"parent_def_id"`
	Unit        *string             `json:"unit"`
	IsContainer *bool                `json:"is_container"`
	Fields      *[]service.CreateFieldInput `json:"fields"`
	TagIDs      *[]string           `json:"tag_ids"`
}

type OverrideRequest struct {
	ParentFieldID string  `json:"parent_field_id"`
	DefaultValue  *string `json:"default_value"`
}

type UpdateOverridesRequest struct {
	Overrides []OverrideRequest `json:"overrides"`
}

func (h *DefinitionHandler) List(w http.ResponseWriter, r *http.Request) {
	defs, err := h.svc.GetAll()
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(defs)
}

func (h *DefinitionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	def, err := h.svc.GetByID(id)
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(def)
}

func (h *DefinitionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, service.ErrInvalidInput)
		return
	}

	if req.Name == nil {
		RespondWithError(w, service.ErrInvalidInput)
		return
	}

	if req.Fields == nil {
		req.Fields = []CreateFieldRequest{}
	}

	fieldInputs := make([]service.CreateFieldInput, len(req.Fields))
	for i, f := range req.Fields {
		fieldInputs[i] = service.CreateFieldInput{
			FieldName:       f.FieldName,
			FieldType:       f.FieldType,
			EnumValues:      f.EnumValues,
			IsRequired:      f.IsRequired,
			DisplayOrder:    f.DisplayOrder,
			DefaultValue:    f.DefaultValue,
			IsChildEditable: f.IsChildEditable,
		}
	}

	def, err := h.svc.Create(service.CreateDefinitionInput{
		Name:        *req.Name,
		Description: req.Description,
		ParentDefID: req.ParentDefID,
		Unit:        req.Unit,
		IsContainer: req.IsContainer,
		Fields:      fieldInputs,
		TagIDs:      req.TagIDs,
	})
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
	json.NewEncoder(w).Encode(def)
}

func (h *DefinitionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, service.ErrInvalidInput)
		return
	}

	def, err := h.svc.Update(id, service.UpdateDefinitionInput{
		Name:        req.Name,
		Description: req.Description,
		ParentDefID: req.ParentDefID,
		Unit:        req.Unit,
		IsContainer: req.IsContainer,
		Fields:      req.Fields,
		TagIDs:      req.TagIDs,
	})
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
	json.NewEncoder(w).Encode(def)
}

func (h *DefinitionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.svc.Delete(id)
	if err != nil {
		if strings.Contains(err.Error(), "child definitions") || strings.Contains(err.Error(), "item instances") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: err.Error(),
				Code:  "conflict",
			})
			return
		}
		RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DefinitionHandler) UpdateOverrides(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateOverridesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, service.ErrInvalidInput)
		return
	}

	overrides := make([]service.OverrideInput, len(req.Overrides))
	for i, o := range req.Overrides {
		overrides[i] = service.OverrideInput{
			ParentFieldID: o.ParentFieldID,
			DefaultValue:  o.DefaultValue,
		}
	}

	result, err := h.svc.UpdateOverrides(id, overrides)
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"overrides": result,
	})
}

func (h *DefinitionHandler) RegisterRoutes(r chi.Router) {
	r.Get("/definitions", h.List)
	r.Get("/definitions/{id}", h.Get)
	r.Post("/definitions", h.Create)
	r.Put("/definitions/{id}", h.Update)
	r.Delete("/definitions/{id}", h.Delete)
	r.Put("/definitions/{id}/overrides", h.UpdateOverrides)
}
