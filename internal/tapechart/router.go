// Package tapechart serves the room-availability grid: reservations,
// inventory, and maintenance blocks joined for a property/date-range window.
package tapechart

import (
	"github.com/go-chi/chi/v5"

	yopMw "github.com/lexxcode1/yop-pms/internal/platform/middleware"
)

// Handler holds service dependencies for tape chart endpoints.
type Handler struct {
	svc *Service
}

// Routes returns a chi.Router configuration function for tape chart endpoints.
func Routes(svc *Service) func(chi.Router) {
	h := &Handler{svc: svc}
	return func(r chi.Router) {
		r.With(yopMw.RequirePermission("reservations:read")).Get("/", h.GetTapeChart)
	}
}
