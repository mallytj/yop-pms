package middleware

import (
	"net/http"
)

// PropertyAmenityCtx is a middleware that extracts the propertyAmenityID from the URL parameters,
// converts it to a UUID, and stores it in the request context for downstream handlers to use.
// It expects the URL parameter to be named "propertyAmenityID".
// If the propertyAmenityID is missing or invalid, it responds with a 400 Bad Request.
func PropertyAmenityCtx(next http.Handler) http.Handler {
	return genericIDExtractor("propertyAmenityID", next)
}
