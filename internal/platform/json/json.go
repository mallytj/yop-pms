package json

import (
	"encoding/json"
	"net/http"

	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	"github.com/lexxcode1/yop-pms/internal/platform/logging"
)

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

// WriteError writes an error response in JSON format.
// It maps PostgreSQL errors through MapPostgresError, logs unexpected errors,
// and returns a structured error response to the client.
func WriteError(w http.ResponseWriter, r *http.Request, err error) error {
	if err == nil {
		// No error to write
		return nil
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
		if apiErr == apierror.ErrInternal {
			logger.Error("unexpected error", "error", err)
		}
	}

	// Handle case where MapPostgresError returned nil (shouldn't happen, but be safe)
	if apiErr == nil {
		apiErr = apierror.ErrInternal
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.Status)
	return json.NewEncoder(w).Encode(apiErr)
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
