package middleware

import (
	"context"
	"net/http"
)

// Auth is a middleware that authenticates requests and assigns user and property IDs to the request context.
// For simplicity, this example uses placeholder logic for authentication.
// In a real application, you would replace this with actual authentication logic.
// Including token validation, session management, etc.
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Placeholder for authentication logic
		// Put admin user id in context for further handlers to use
		userID := "admin-user-id" // This would be fetched from a real auth system
		ctx := context.WithValue(r.Context(), "userID", userID)

		// Assign a property ID for multi-tenancy
		propertyID := "default-property-id" // This would be fetched based on the user
		ctx = context.WithValue(ctx, "propertyID", propertyID)

		// If authentication is successful, proceed to the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
