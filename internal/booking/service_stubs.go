package booking

// Phase 7 stubs — replaced when each action/mutation/rates domain is implemented.

import (
	"context"

	"github.com/google/uuid"

	"github.com/lexxcode1/yop-pms/internal/store"
)

// CancelReservation cancels a reservation. Implemented in Phase 7 (actions.go).
// When implemented: enqueue EventCancellationEmail via worker.Enqueue on success.
func (s *Service) CancelReservation(_ context.Context, _ uuid.UUID, _ CancelInput) (*ReservationResponse, error) {
	return nil, ErrNotImplemented
}

// ReactivateReservation reactivates a cancelled reservation. Implemented in Phase 7 (actions.go).
// When implemented: enqueue EventConfirmationEmail via worker.Enqueue on success.
func (s *Service) ReactivateReservation(_ context.Context, _ uuid.UUID) (*ReservationResponse, error) {
	return nil, ErrNotImplemented
}

// CheckinReservation checks in all items on a reservation (207 Multi-Status). Implemented in Phase 7 (actions.go).
func (s *Service) CheckinReservation(_ context.Context, _ uuid.UUID) (*BatchResult, error) {
	return nil, ErrNotImplemented
}

// CheckinItem checks in a single reservation item. Implemented in Phase 7 (actions.go).
func (s *Service) CheckinItem(_ context.Context, _ uuid.UUID) (*ItemResponse, error) {
	return nil, ErrNotImplemented
}

// CheckoutReservation checks out all items on a reservation (207 Multi-Status). Implemented in Phase 7 (actions.go).
func (s *Service) CheckoutReservation(_ context.Context, _ uuid.UUID) (*BatchResult, error) {
	return nil, ErrNotImplemented
}

// CheckoutItem checks out a single reservation item. Implemented in Phase 7 (actions.go).
func (s *Service) CheckoutItem(_ context.Context, _ uuid.UUID) (*ItemResponse, error) {
	return nil, ErrNotImplemented
}

// MarkNoShow marks an item as no-show. Implemented in Phase 7 (actions.go).
func (s *Service) MarkNoShow(_ context.Context, _ uuid.UUID) (*ItemResponse, error) {
	return nil, ErrNotImplemented
}

// CancelItem cancels a single reservation item. Implemented in Phase 7 (actions.go).
// When implemented: enqueue EventCancellationEmail via worker.Enqueue on success.
func (s *Service) CancelItem(_ context.Context, _ uuid.UUID, _ CancelInput) (*ItemResponse, error) {
	return nil, ErrNotImplemented
}

// AddItem adds a new item to an existing reservation. Implemented in Phase 7 (mutations.go).
// When implemented: enqueue EventConfirmationEmail via worker.Enqueue on success.
func (s *Service) AddItem(_ context.Context, _ uuid.UUID, _ AddItemInput) (*ReservationResponse, error) {
	return nil, ErrNotImplemented
}

// UpdateItem updates a reservation item's stay period, room type, or rate plan. Implemented in Phase 7 (mutations.go).
func (s *Service) UpdateItem(_ context.Context, _ uuid.UUID, _ CreateItemInput) (*ReservationResponse, error) {
	return nil, ErrNotImplemented
}

// AssignRoom assigns a specific room to an item. Implemented in Phase 7 (mutations.go).
func (s *Service) AssignRoom(_ context.Context, _ uuid.UUID, _ AssignRoomInput) (*ReservationResponse, error) {
	return nil, ErrNotImplemented
}

// GetBookedRates returns booked daily rates for an item. Implemented in Phase 7 (rates.go).
func (s *Service) GetBookedRates(_ context.Context, _ uuid.UUID) ([]store.PricingBookedDailyRate, error) {
	return nil, ErrNotImplemented
}

// UpdateBookedRates applies a nightly rate override. Implemented in Phase 7 (rates.go).
func (s *Service) UpdateBookedRates(_ context.Context, _ uuid.UUID, _ RateAdjustInput) (*ReservationResponse, error) {
	return nil, ErrNotImplemented
}

// AdjustRate applies a rate adjustment to a specific night. Implemented in Phase 7 (rates.go).
func (s *Service) AdjustRate(_ context.Context, _ uuid.UUID, _ RateAdjustInput) (*ReservationResponse, error) {
	return nil, ErrNotImplemented
}

// ApproveAdjustments approves pending rate adjustments. Implemented in Phase 7 (rates.go).
func (s *Service) ApproveAdjustments(_ context.Context, _ uuid.UUID) (*ReservationResponse, error) {
	return nil, ErrNotImplemented
}
