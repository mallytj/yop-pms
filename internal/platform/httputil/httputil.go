// Package httputil provides shared HTTP handler utilities used across domains.
package httputil

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
)

// ParseUUIDParam extracts a named chi URL parameter and parses it as a UUID.
// Returns an APIError with status 400 if the parameter is missing or not a valid UUID.
func ParseUUIDParam(r *http.Request, param string) (uuid.UUID, *apierror.APIError) {
	raw := chi.URLParam(r, param)
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apierror.ErrBadRequest.WithMessage(param + " must be a valid UUID")
	}
	return id, nil
}

// ParseDateParam reads a named query parameter and parses it as a YYYY-MM-DD date.
// Returns a 400 APIError if the parameter is missing or not a valid date.
func ParseDateParam(r *http.Request, param string) (time.Time, *apierror.APIError) {
	raw := r.URL.Query().Get(param)
	if raw == "" {
		return time.Time{}, apierror.ErrBadRequest.WithMessage(param + " is required")
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, apierror.ErrBadRequest.WithMessage(param + " must be YYYY-MM-DD")
	}
	return t, nil
}
