package apperr

import "errors"

var (
	ErrBadRequest   = errors.New("bad_request")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not_found")
	ErrConflict     = errors.New("conflict")
	ErrInvalidState = errors.New("invalid_state")
	ErrInternal     = errors.New("internal_error")
)
