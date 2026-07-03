package booking

// Remaining stubs for Phase 7 features not yet implemented.

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lexxcode1/yop-pms/internal/platform/db"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// UpdateBookedRates applies direct nightly rate overrides — sets base_price_pence
// on booked_daily_rates rows for the given dates. Unlike AdjustRate (which goes
// through the adjustment column + approval workflow), this sets the base price
// directly. Requires reservations:rate_override.
func (s *Service) UpdateBookedRates(ctx context.Context, itemID uuid.UUID, input RateAdjustInput) (*ReservationResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	_, err := db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (struct{}, error) {
		for _, adj := range input.Adjustments {
			calDate := pgtype.Date{Time: adj.CalendarDate, Valid: true}
			// Direct override: set base_price_pence to the adjustment value.
			// For a percentage override, compute the new price from existing rate.
			if adj.Type == AdjustmentPercent {
				rate, err := qtx.GetBaseRateForDate(ctx, &store.GetBaseRateForDateParams{
					ReservationItemID: itemID,
					CalendarDate:      calDate,
					PropertyID:        propertyID,
				})
				if err != nil {
					return struct{}{}, fmt.Errorf("get base rate for %s: %w", adj.CalendarDate.Format("2006-01-02"), err)
				}
				newPrice := max(int32(float64(rate)*(1+float64(adj.Value)/100.0)), 0)

				if err := qtx.SetBaseRateForDate(ctx, &store.SetBaseRateForDateParams{
					ReservationItemID: itemID,
					CalendarDate:      calDate,
					PropertyID:        propertyID,
					BasePricePence:    newPrice,
				}); err != nil {
					return struct{}{}, fmt.Errorf("set base rate for %s: %w", adj.CalendarDate.Format("2006-01-02"), err)
				}
				continue

			}

			// Fixed adjustment: value IS the new base price.
			newPrice := max(int32(adj.Value), 0)
			if err := qtx.SetBaseRateForDate(ctx, &store.SetBaseRateForDateParams{
				ReservationItemID: itemID,
				CalendarDate:      calDate,
				PropertyID:        propertyID,
				BasePricePence:    newPrice,
			}); err != nil {
				return struct{}{}, fmt.Errorf("set base rate for %s: %w", adj.CalendarDate.Format("2006-01-02"), err)
			}
		}
		return struct{}{}, nil
	})
	if err != nil {
		return nil, err
	}

	item, err := s.q.GetReservationItem(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("get item after rate update: %w", err)
	}
	return s.GetReservation(ctx, item.ReservationID, IncludeFlags{Items: true})
}
