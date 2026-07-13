// Package json wraps net/http with response helpers: WriteJSON for success,
// WriteError for failures (auto-maps pg errors via apierror), ReadJSON for
// request bodies. All errors flow through the apierror pipeline.
package json

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	"github.com/lexxcode1/yop-pms/internal/platform/logging"
)

// WriteJSON writes a JSON response with the given status code.
// Encoding errors are logged internally via slog.Default(); the function
// does not return an error because there is no meaningful recovery at the
// call site (the connection is likely broken).
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Default().Error("platformjson: encoding JSON response", "error", err)
	}
}

// WriteError writes an error response in JSON format.
// It maps PostgreSQL errors through MapPostgresError, logs unexpected errors,
// and returns a structured error response to the client.
// Encoding errors are logged internally; the function does not return an
// error because there is no meaningful recovery at the call site.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	logger := logging.FromContext(r.Context())

	var apiErr *apierror.APIError

	// Try to cast to *APIError, or map from database error
	if ae, ok := err.(*apierror.APIError); ok {
		apiErr = ae
	} else {
		// Attempt to map PostgreSQL error
		apiErr = apierror.MapPostgresError(err)

		// If not a database error, log as unexpected error
		if apiErr.Code == apierror.ErrInternal.Code {
			logger.Error("unexpected error", "error", err)
		}
	}

	// Handle case where MapPostgresError returned nil (shouldn't happen, but be safe)
	if apiErr == nil {
		apiErr = apierror.ErrInternal
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.Status)
	if err := json.NewEncoder(w).Encode(apiErr); err != nil {
		logger.Error("platformjson: encoding error response", "error", err)
	}
}

// ReadJSON parses a JSON request body into the destination struct.
// It returns a 400 Bad Request error if the JSON is malformed or contains unknown fields.
func ReadJSON(r *http.Request, dst any) *apierror.APIError {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return apierror.ErrBadRequest.WithMessage("request body is not valid JSON or contains unknown fields")
	}

	return nil
}
