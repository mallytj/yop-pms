package middleware

import (
	"net/http"

	hf "ollerod-pms/internal/helpers"

	"github.com/google/uuid"
)

// PropertyCtx is a middleware that extracts the propertyID from the URL parameters,
// converts it to a UUID, and stores it in the request context for downstream handlers to use.
// It expects the URL parameter to be named "propertyID".
// If the propertyID is missing or invalid, it responds with a 400 Bad Request.
func PropertyCtx(next http.Handler) http.Handler {
	return genericIDExtractor("propertyID", next)
}

func EnforcePropertyContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockID := uuid.MustParse(hf.TestPropertyID)

		ctx := hf.SetIDInCtx(r.Context(), "propertyID", mockID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
