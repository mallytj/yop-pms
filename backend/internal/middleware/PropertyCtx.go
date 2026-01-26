package middleware

import (
	"net/http"
)

// PropertyCtx is a middleware that extracts the propertyID from the URL parameters,
// converts it to a UUID, and stores it in the request context for downstream handlers to use.
// It expects the URL parameter to be named "propertyID".
// If the propertyID is missing or invalid, it responds with a 400 Bad Request.
func PropertyCtx(next http.Handler) http.Handler {
	return genericIDExtractor("propertyID", next)
}
