package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"tutor-platform/services/core/internal/apperr"
)

var (
	ErrBadRequest   = apperr.ErrBadRequest
	ErrUnauthorized = apperr.ErrUnauthorized
	ErrForbidden    = apperr.ErrForbidden
	ErrNotFound     = apperr.ErrNotFound
	ErrConflict     = apperr.ErrConflict
	ErrInvalidState = apperr.ErrInvalidState
	ErrInternal     = apperr.ErrInternal
)

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func WriteError(w http.ResponseWriter, err error) {
	app := mapError(err)
	WriteJSON(w, app.Status, map[string]AppError{"error": app})
}

func mapError(err error) AppError {
	switch {
	case errors.Is(err, ErrBadRequest):
		return AppError{Code: "bad_request", Message: "invalid request", Status: http.StatusBadRequest}
	case errors.Is(err, ErrUnauthorized):
		return AppError{Code: "unauthorized", Message: "authentication required", Status: http.StatusUnauthorized}
	case errors.Is(err, ErrForbidden):
		return AppError{Code: "forbidden", Message: "permission denied", Status: http.StatusForbidden}
	case errors.Is(err, ErrNotFound):
		return AppError{Code: "not_found", Message: "resource not found", Status: http.StatusNotFound}
	case errors.Is(err, ErrConflict):
		return AppError{Code: "conflict", Message: "resource conflict", Status: http.StatusConflict}
	case errors.Is(err, ErrInvalidState):
		return AppError{Code: "invalid_state", Message: "resource is not in a valid state", Status: http.StatusUnprocessableEntity}
	default:
		return AppError{Code: "internal_error", Message: "internal server error", Status: http.StatusInternalServerError}
	}
}
