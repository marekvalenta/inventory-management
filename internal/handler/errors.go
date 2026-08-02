package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/marekvalenta/inventory-management/internal/service"
)

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func RespondWithError(w http.ResponseWriter, err error) {
	var status int
	var code string

	switch {
	case errors.Is(err, service.ErrNotFound):
		status = http.StatusNotFound
		code = "not_found"
	case errors.Is(err, service.ErrInvalidInput):
		status = http.StatusBadRequest
		code = "invalid_input"
	case errors.Is(err, service.ErrConflict):
		status = http.StatusConflict
		code = "conflict"
	default:
		status = http.StatusInternalServerError
		code = "internal_error"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: err.Error(),
		Code:  code,
	})
}
