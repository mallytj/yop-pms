package apierror

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
)

type Suggestions []string

type APIError struct {
	Code        string      `json:"code" enums:"NOT_FOUND,BAD_REQUEST,CONFLICT,INTERNAL_ERROR,UNPROCESSABLE_ENTITY"`
	Message     string      `json:"message"`
	Status      int         `json:"-"`
	Suggestions Suggestions `json:"suggestions,omitempty"`
}

// Error implements the error interface for *APIError
func (e *APIError) Error() string {
	return e.Message
}

// New creates a new APIError with the given code, message, and HTTP status
func New(code, message string, status int) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

// WithMessage returns a shallow copy of the error with a new message,
// preserving Code and Status. This allows sentinel errors to be customized.
func (e *APIError) WithMessage(msg string) *APIError {
	return &APIError{
		Code:    e.Code,
		Message: msg,
		Status:  e.Status,
	}
}

// WithSuggestions returns a shallow copy of the error with suggestions,
// preserving Code, Message, and Status. This allows sentinel errors to be customized.
func (e *APIError) WithSuggestions(suggestions Suggestions) *APIError {
	return &APIError{
		Code:        e.Code,
		Message:     e.Message,
		Status:      e.Status,
		Suggestions: suggestions,
	}
}

// Sentinel errors
var (
	ErrNotFound      = New("NOT_FOUND", "resource not found", http.StatusNotFound)
	ErrBadRequest    = New("BAD_REQUEST", "invalid request", http.StatusBadRequest)
	ErrUnauthorized  = New("UNAUTHORIZED", "authentication required", http.StatusUnauthorized)
	ErrForbidden     = New("FORBIDDEN", "insufficient permissions", http.StatusForbidden)
	ErrConflict      = New("CONFLICT", "resource already exists", http.StatusConflict)
	ErrInternal      = New("INTERNAL_ERROR", "internal server error", http.StatusInternalServerError)
	ErrUnprocessable = New("UNPROCESSABLE_ENTITY", "the request contains invalid data", http.StatusUnprocessableEntity)
)

// MapStoreError maps a store (database) error to an APIError.
// It handles pgx.ErrNoRows as 404 Not Found and delegates all other
// errors to MapPostgresError. Use this in service methods instead of
// calling MapPostgresError directly.
func MapStoreError(err error) *APIError {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return MapPostgresError(err)
}

// MapPostgresError maps a postgres error to an APIError
// Example:
//
//	err := store.SetCurrentPropertyID(ctx, "property_id")
//	if err != nil {
//		return apierror.MapPostgresError(err)
//	}
func MapPostgresError(err error) *APIError {
	if err == nil {
		return nil
	}

	// Check for specific PostgreSQL error codes
	switch {
	case helpers.CheckErrorCode(err, helpers.UniqueViolationCode):
		return ErrConflict.WithMessage("a record with this value already exists")

	case helpers.CheckErrorCode(err, helpers.ForeignKeyViolationCode):
		return ErrBadRequest.WithMessage("referenced resource does not exist")

	case helpers.CheckErrorCode(err, helpers.CheckViolationCode):
		return ErrUnprocessable.WithMessage("the value violates a database constraint")

	case helpers.CheckErrorCode(err, helpers.ExclusionViolationCode):
		return ErrConflict.WithMessage("the dates overlap with an existing reservation")

	case helpers.CheckErrorCode(err, helpers.RaiseExceptionCode):
		// Try to extract PgError.Detail for custom error messages from pl/pgsql
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Detail != "" {
			return ErrUnprocessable.WithMessage(pgErr.Detail)
		}
		return ErrUnprocessable.WithMessage("the request violates a business rule")

	default:
		return ErrInternal.WithMessage("an unexpected database error occurred")
	}
}
