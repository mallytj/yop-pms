package middleware

import (
	"context"
	"ollerod-pms/internal/json"

	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func UserCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Get the userID from url parameters
		idStr := chi.URLParam(r, "userID")

		// 2. Convert to UUID
		id, err := uuid.Parse(idStr)
		if err != nil {
			json.Write(w, http.StatusBadRequest, "invalid user ID")
			return
		}

		// 3. Store the userID in the request context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "userID", id)

		// 4. Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
