package middleware

import (
	"net/http"
)

// Gets the reservation ID from the URL and adds it to the request context for downstream handlers to use.
func ReservationItemCtx(next http.Handler) http.Handler {
	return genericIDExtractor("reservationItemID", next)
}
