package middleware

import (
	"net/http"
)

// LicenceCtx is a middleware that extracts the licenceID from the URL parameters,
// converts it to a UUID, and stores it in the request context for downstream handlers to use.
// It expects the URL parameter to be named "licenceID".
// If the licenceID is missing or invalid, it responds with a 400 Bad Request.
func LicenceCtx(next http.Handler) http.Handler {
	return genericIDExtractor("licenceID", next)
}