package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/marekvalenta/inventory-management/internal/service"
)

type StackHandler struct {
	svc *service.StackService
}

func NewStackHandler(svc *service.StackService) *StackHandler {
	return &StackHandler{svc: svc}
}

func (h *StackHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var locationID, parentInstanceID *string
	if v := q.Get("location_id"); v != "" {
		locationID = &v
	}
	if v := q.Get("parent_instance_id"); v != "" {
		parentInstanceID = &v
	}

	result, err := h.svc.List(locationID, parentInstanceID)
	if err != nil {
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *StackHandler) Detail(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	definitionID := q.Get("definition_id")
	if strings.TrimSpace(definitionID) == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "definition_id is required",
			Code:  "invalid_params",
		})
		return
	}

	var locationID, parentInstanceID *string
	if v := q.Get("location_id"); v != "" {
		locationID = &v
	}
	if v := q.Get("parent_instance_id"); v != "" {
		parentInstanceID = &v
	}

	if (locationID == nil) == (parentInstanceID == nil) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "exactly one of location_id or parent_instance_id must be provided",
			Code:  "invalid_params",
		})
		return
	}

	page := 1
	perPage := 50
	if v := q.Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := q.Get("per_page"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}

	detail, err := h.svc.GetDetail(definitionID, locationID, parentInstanceID, page, perPage)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "not found") || err == service.ErrNotFound {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: "stack not found",
				Code:  "not_found",
			})
			return
		}
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

func (h *StackHandler) Move(w http.ResponseWriter, r *http.Request) {
	var req service.MoveStackInput
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
	if (req.SourceLocationID == nil) == (req.SourceParentInstanceID == nil) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "exactly one of source_location_id or source_parent_instance_id must be provided",
			Code:  "invalid_parent",
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

	result, err := h.svc.Move(req)
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
		if strings.Contains(errStr, "only") && strings.Contains(errStr, "available") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "insufficient_quantity",
			})
			return
		}
		RespondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *StackHandler) Delete(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	definitionID := q.Get("definition_id")
	if strings.TrimSpace(definitionID) == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "definition_id is required",
			Code:  "invalid_params",
		})
		return
	}

	var locationID, parentInstanceID *string
	if v := q.Get("location_id"); v != "" {
		locationID = &v
	}
	if v := q.Get("parent_instance_id"); v != "" {
		parentInstanceID = &v
	}

	if (locationID == nil) == (parentInstanceID == nil) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "exactly one of location_id or parent_instance_id must be provided",
			Code:  "invalid_params",
		})
		return
	}

	err := h.svc.Delete(definitionID, locationID, parentInstanceID)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "children") || strings.Contains(errStr, "items stored") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: errStr,
				Code:  "stack_has_children",
			})
			return
		}
		if err == service.ErrNotFound {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: "stack not found",
				Code:  "not_found",
			})
			return
		}
		RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
