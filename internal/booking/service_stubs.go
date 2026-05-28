package booking

// Remaining stubs for Phase 7 features not yet implemented.

import (
	"context"

	"github.com/google/uuid"
)

// UpdateBookedRates applies a nightly rate override. Deferred — use ApplyRateAdjustments instead.
func (s *Service) UpdateBookedRates(_ context.Context, _ uuid.UUID, _ RateAdjustInput) (*ReservationResponse, error) {
	return nil, ErrNotImplemented
}
