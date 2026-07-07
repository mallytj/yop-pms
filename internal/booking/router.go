// Package booking implements the reservation domain: CRUD, lifecycle state
// machine, availability, pricing, and outbox events.
//
// File layout:
//
//	handlers_*.go  — HTTP handlers (thin: parse → validate → service → respond)
//	service*.go    — Business logic, split by domain area
//	types.go       — Shared types, I/O structs, constants
//	router.go      — Route registration, Handler struct
//	state_machine.go — Status transition validation
//	availability.go  — Availability check + cache invalidation
//	errors.go      — Domain error sentinels
package booking

// Routes returns a chi sub-router function for all reservation endpoints.
// Pass it to r.Route("/reservations", booking.Routes(...)) in cmd/server/api.go.

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	yopMw "github.com/lexxcode1/yop-pms/internal/platform/middleware"
)

// Handler wraps the booking Service and exposes it as HTTP handlers.
// Handlers are thin: parse → validate → domain check → service → respond.
type Handler struct {
	svc *Service
}

// Routes builds the /reservations sub-router and wires all handlers.
func Routes(svc *Service, ifMatch func(http.Handler) http.Handler) func(chi.Router) {
	h := &Handler{svc: svc}

	return func(r chi.Router) {
		r.Get("/availability", h.Availability)

		r.With(yopMw.RequirePermission("reservations:create")).Post("/", h.Create)
		r.With(yopMw.RequirePermission("reservations:read")).Get("/", h.List)

		r.Route("/{id}", func(r chi.Router) {
			r.With(yopMw.RequirePermission("reservations:read")).Get("/", h.Get)
			r.With(ifMatch, yopMw.RequirePermission("reservations:update")).Patch("/", h.UpdateMetadata)
			r.With(ifMatch, yopMw.RequirePermission("reservations:confirm")).Post("/confirm", h.Confirm)
			r.With(ifMatch, yopMw.RequirePermission("reservations:cancel")).Post("/cancel", h.Cancel)
			r.With(ifMatch, yopMw.RequirePermission("reservations:reactivate")).Post("/reactivate", h.Reactivate)
			r.With(yopMw.RequirePermission("reservations:checkin")).Patch("/checkin", h.CheckinReservation)
			r.With(yopMw.RequirePermission("reservations:checkout")).Patch("/checkout", h.CheckoutReservation)
			r.With(yopMw.RequirePermission("reservations:read")).Get("/cancellation-quote", h.CancellationQuote)
			r.With(yopMw.RequirePermission("reservations:read")).Get("/folios/{folio_id}", h.GetFolio)
			r.With(ifMatch, yopMw.RequirePermission("reservations:add_item")).Post("/items", h.AddItem)

			r.Route("/items/{item_id}", func(r chi.Router) {
				r.With(ifMatch, yopMw.RequirePermission("reservations:update_item")).Patch("/", h.UpdateItem)
				r.With(ifMatch, yopMw.RequirePermission("reservations:checkin")).Patch("/checkin", h.CheckinItem)
				r.With(ifMatch, yopMw.RequirePermission("reservations:checkout")).Patch("/checkout", h.CheckoutItem)
				r.With(ifMatch, yopMw.RequirePermission("reservations:assign_room")).Patch("/assign-room", h.AssignRoom)
				r.With(ifMatch, yopMw.RequirePermission("reservations:mark_no_show")).Patch("/no-show", h.MarkNoShow)
				r.With(ifMatch, yopMw.RequirePermission("reservations:cancel")).Post("/cancel", h.CancelItem)
				r.With(ifMatch, yopMw.RequirePermission("reservations:rate_override")).Patch("/booked-rates", h.UpdateBookedRates)
				r.With(yopMw.RequirePermission("reservations:read")).Get("/booked-rates", h.GetBookedRates)
				r.With(ifMatch, yopMw.RequirePermission("reservations:rate_override")).Post("/booked-rates/approve", h.ApproveAdjustments)
				r.With(ifMatch, yopMw.RequirePermission("reservations:rate_override")).Post("/adjust-rate", h.AdjustRate)
			})
		})
	}
}
