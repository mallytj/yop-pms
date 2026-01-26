package middleware

import (
	"net/http"
)

// UserCtx is a middleware that extracts the userID from the URL parameters,
// converts it to a UUID, and stores it in the request context for downstream handlers to use.
// It expects the URL parameter to be named "userID".
// If the userID is missing or invalid, it responds with a 400 Bad Request.
func UserCtx(next http.Handler) http.Handler {
	return genericIDExtractor("userID", next)
}
