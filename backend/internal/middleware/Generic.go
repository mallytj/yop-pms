package middleware

import (
	"context"
	"ollerod-pms/internal/json"
	"ollerod-pms/internal/types"

	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// genericIDExtractor is a middleware that extracts a generic ID from the URL parameters,
// converts it to a UUID, and stores it in the request context for downstream handlers to use.
// It expects the URL parameter name to be provided as an argument.
// If the ID is missing or invalid, it responds with a 400 Bad Request.
func genericIDExtractor(paramName types.ContextKey, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Get the param name from url parameters
		idStr := chi.URLParam(r, string(paramName))

		// Use the paramName as the context key
		if idStr == "" {
			json.Write(w, http.StatusBadRequest, string(paramName)+" is required")
			return
		}

		// 2. Convert to UUID
		id, err := uuid.Parse(idStr)
		if err != nil {
			json.Write(w, http.StatusBadRequest, "invalid "+string(paramName))
			return
		}

		// 3. Store the paramName in the request context
		ctx := r.Context()
		ctx = context.WithValue(ctx, paramName, id)

		// 4. Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
