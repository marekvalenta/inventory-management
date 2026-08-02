package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/marekvalenta/inventory-management/internal/service"
)

type InstanceHandler struct {
	svc *service.InstanceService
}

func NewInstanceHandler(svc *service.InstanceService) *InstanceHandler {
	return &InstanceHandler{svc: svc}
}

type CreateInstanceRequest struct {
	DefinitionID     string                    `json:"definition_id"`
	Quantity         int                       `json:"quantity"`
	LocationID       *string                   `json:"location_id"`
	ParentInstanceID *string                   `json:"parent_instance_id"`
	FieldValues      []service.FieldValueInput `json:"field_values"`
}

type UpdateInstanceRequest struct {
	Quantity    *int                       `json:"quantity"`
	FieldValues []service.FieldValueInput  `json:"field_values"`
}

type MoveInstanceRequest struct {
	Quantity               int     `json:"quantity"`
	TargetLocationID       *string `json:"target_location_id"`
	TargetParentInstanceID *string `json:"target_parent_instance_id"`
}

func (h *InstanceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, service.ErrInvalidInput)
		return
	}

	if strings.TrimSpace(req.DefinitionID) == "" {
		RespondWithError(w, service.ErrInvalidInput)
		return
	}

	if req.Quantity <= 0 {
		RespondWithError(w, service.ErrInvalidInput)
		return
	}

	if (req.LocationID == nil) == (req.ParentInstanceID == nil) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "exactly one of location_id or parent_instance_id must be provided",
			Code:  "invalid_parent",
		})
		return
	}

	inst, err := h.svc.Create(r.Context(), service.CreateInstanceInput{
		DefinitionID:     req.DefinitionID,
		Quantity:         req.Quantity,
		LocationID:       req.LocationID,
		ParentInstanceID: req.ParentInstanceID,
		FieldValues:      req.FieldValues,
	})
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "not a container") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "not_a_container",
			})
			return
		}
		if strings.Contains(errStr, "required field") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "required_field_missing",
			})
			return
		}
		if strings.Contains(errStr, "must be a number") || strings.Contains(errStr, "must be 'true' or 'false'") || strings.Contains(errStr, "is not in enum_values") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "invalid_field_value",
			})
			return
		}
		if strings.Contains(errStr, "field_id") && strings.Contains(errStr, "does not belong") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "unknown_field",
			})
			return
		}
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(inst)
}

func (h *InstanceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	inst, err := h.svc.GetByID(id)
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inst)
}

func (h *InstanceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, service.ErrInvalidInput)
		return
	}

	inst, err := h.svc.Update(id, service.UpdateInstanceInput{
		Quantity:    req.Quantity,
		FieldValues: req.FieldValues,
	})
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "required field") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "required_field_missing",
			})
			return
		}
		if strings.Contains(errStr, "must be a number") || strings.Contains(errStr, "must be 'true' or 'false'") || strings.Contains(errStr, "is not in enum_values") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "invalid_field_value",
			})
			return
		}
		if strings.Contains(errStr, "field_id") && strings.Contains(errStr, "does not belong") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "unknown_field",
			})
			return
		}
		if strings.Contains(errStr, "quantity must be greater than 0") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "invalid_quantity",
			})
			return
		}
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inst)
}

func (h *InstanceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.svc.Delete(id)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "items are stored inside") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "instance_has_children",
			})
			return
		}
		RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *InstanceHandler) Move(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req MoveInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, service.ErrInvalidInput)
		return
	}

	if req.Quantity <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "quantity must be greater than 0",
			Code:  "invalid_quantity",
		})
		return
	}

	if (req.TargetLocationID == nil) == (req.TargetParentInstanceID == nil) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "exactly one of target_location_id or target_parent_instance_id must be provided",
			Code:  "invalid_parent",
		})
		return
	}

	result, err := h.svc.Move(id, service.MoveInstanceInput{
		Quantity:               req.Quantity,
		TargetLocationID:       req.TargetLocationID,
		TargetParentInstanceID: req.TargetParentInstanceID,
	})
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "not a container") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "not_a_container",
			})
			return
		}
		if strings.Contains(errStr, "cannot move instance into itself") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "self_parent",
			})
			return
		}
		if strings.Contains(errStr, "cycle detected") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "cycle_detected",
			})
			return
		}
		if strings.Contains(errStr, "already at this location") || strings.Contains(errStr, "already in this container") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "same_parent",
			})
			return
		}
		if strings.Contains(errStr, "Cannot move") && strings.Contains(errStr, "only") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "invalid_quantity",
			})
			return
		}
		if strings.Contains(errStr, "has") && strings.Contains(errStr, "children") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "instance_has_children",
			})
			return
		}
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *InstanceHandler) List(w http.ResponseWriter, r *http.Request) {
	var locationID, definitionID, parentInstanceID *string

	if v := r.URL.Query().Get("location_id"); v != "" {
		locationID = &v
	}
	if v := r.URL.Query().Get("definition_id"); v != "" {
		definitionID = &v
	}
	if v := r.URL.Query().Get("parent_instance_id"); v != "" {
		parentInstanceID = &v
	}

	result, err := h.svc.List(locationID, definitionID, parentInstanceID)
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *InstanceHandler) GetContents(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	instances, err := h.svc.GetContents(id)
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"instances": instances,
	})
}

func (h *InstanceHandler) GetBreadcrumb(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	breadcrumb, err := h.svc.GetBreadcrumb(id)
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(breadcrumb)
}

func (h *InstanceHandler) RegisterRoutes(r chi.Router) {
	r.Get("/instances", h.List)
	r.Get("/instances/{id}", h.Get)
	r.Post("/instances", h.Create)
	r.Put("/instances/{id}", h.Update)
	r.Delete("/instances/{id}", h.Delete)
	r.Post("/instances/{id}/move", h.Move)
	r.Get("/instances/{id}/contents", h.GetContents)
	r.Get("/instances/{id}/breadcrumb", h.GetBreadcrumb)
}
