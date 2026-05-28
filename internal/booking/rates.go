package booking

// Core Requirements: R-RES-RATE-001, R-RES-RATE-002, R-RES-RATE-003, ADR-021

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lexxcode1/yop-pms/internal/platform/db"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// GetBookedRates returns booked daily rates for a reservation item.
func (s *Service) GetBookedRates(ctx context.Context, itemID uuid.UUID) ([]store.PricingBookedDailyRate, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	rates, err := s.q.GetBookedRates(ctx, &store.GetBookedRatesParams{
		ReservationItemID: itemID,
		PropertyID:        propertyID,
	})
	if err != nil {
		return nil, fmt.Errorf("get booked rates: %w", err)
	}
	return rates, nil
}

// OverrideNightlyRate sets a specific base price for a single night on an item.
func (s *Service) OverrideNightlyRate(ctx context.Context, itemID uuid.UUID, date time.Time, baseRatePence int) error {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return ErrNoPropertyContext
	}

	item, err := s.q.GetReservationItem(ctx, itemID)
	if err != nil {
		return fmt.Errorf("get item: %w", err)
	}
	if !item.RatePlanID.Valid {
		return fmt.Errorf("item has no rate plan")
	}

	return s.q.InsertBookedDailyRate(ctx, &store.InsertBookedDailyRateParams{
		PropertyID:        propertyID,
		ReservationItemID: itemID,
		CalendarDate:      pgtype.Date{Time: date, Valid: true},
		RatePlanID:        item.RatePlanID,
		BasePricePence:    int32(baseRatePence),
	})
}

// AdjustRate applies rate adjustments and returns a reservation response.
// Convenience wrapper for the AdjustRate handler which expects *ReservationResponse.
func (s *Service) AdjustRate(ctx context.Context, itemID uuid.UUID, input RateAdjustInput) (*ReservationResponse, error) {
	if err := s.ApplyRateAdjustments(ctx, itemID, input); err != nil {
		return nil, err
	}
	if len(input.Adjustments) == 0 {
		return &ReservationResponse{}, nil
	}
	// Re-fetch to get the item's reservation for a meaningful response.
	item, err := s.q.GetReservationItem(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("get item after adjust: %w", err)
	}
	return s.GetReservation(ctx, item.ReservationID, IncludeFlags{Items: true})
}

// ApplyRateAdjustments applies percentage or fixed adjustments to booked daily rates.
func (s *Service) ApplyRateAdjustments(ctx context.Context, itemID uuid.UUID, input RateAdjustInput) error {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return ErrNoPropertyContext
	}

	_, err := db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (struct{}, error) {
		// Fetch rates once, build a lookup map by date.
		rates, err := qtx.GetBookedRates(ctx, &store.GetBookedRatesParams{
			ReservationItemID: itemID, PropertyID: propertyID,
		})
		if err != nil {
			return struct{}{}, fmt.Errorf("get rates: %w", err)
		}
		rateByDate := make(map[time.Time]int32, len(rates))
		for _, r := range rates {
			if r.CalendarDate.Valid {
				rateByDate[r.CalendarDate.Time] = r.BasePricePence
			}
		}

		autoApprove := helpers.HasPermission(ctx, "reservations:rate_override")
		userID := helpers.GetUserIDFromCtx(ctx)

		for _, adj := range input.Adjustments {
			calDate := pgtype.Date{Time: adj.CalendarDate, Valid: true}

			base, ok := rateByDate[adj.CalendarDate]
			if !ok {
				return struct{}{}, fmt.Errorf("no rate found for date %s", adj.CalendarDate.Format("2006-01-02"))
			}

			switch adj.Type {
			case AdjustmentPercent:
				delta := math.Round(float64(base) * float64(adj.Value) / 100.0)
				if base+int32(delta) < 0 {
					s.log.Warn("adjustment resulted in negative rate, clamped to 0",
						"item_id", itemID, "date", adj.CalendarDate, "value", adj.Value)
				}

				if _, err := qtx.ApplyRateAdjustment(ctx, &store.ApplyRateAdjustmentParams{
					ReservationItemID: itemID, PropertyID: propertyID,
					CalendarDate: calDate, Type: string(adj.Type),
					Value: int32(adj.Value), Reason: adj.Reason,
				}); err != nil {
					return struct{}{}, fmt.Errorf("apply percentage adjustment: %w", err)
				}

			case AdjustmentFixed:
				if base+int32(adj.Value) < 0 {
					s.log.Warn("fixed adjustment resulted in negative rate, clamped to 0",
						"item_id", itemID, "date", adj.CalendarDate, "value", adj.Value)
				}

				if _, err := qtx.ApplyRateAdjustment(ctx, &store.ApplyRateAdjustmentParams{
					ReservationItemID: itemID, PropertyID: propertyID,
					CalendarDate: calDate, Type: string(adj.Type),
					Value: int32(adj.Value), Reason: adj.Reason,
				}); err != nil {
					return struct{}{}, fmt.Errorf("apply fixed adjustment: %w", err)
				}

			default:
				return struct{}{}, fmt.Errorf("unknown adjustment type: %s", adj.Type)
			}

			// Auto-approve when caller has rate_override permission
			if autoApprove && userID != uuid.Nil {
				if err := qtx.ApproveRateAdjustments(ctx, &store.ApproveRateAdjustmentsParams{
					ReservationItemID: itemID, PropertyID: propertyID,
					Dates:  []pgtype.Date{calDate},
					UserID: uuid.NullUUID{UUID: userID, Valid: true},
				}); err != nil {
					return struct{}{}, fmt.Errorf("auto-approve adjustment: %w", err)
				}
			}
		}
		return struct{}{}, nil
	})
	return err
}

// ApproveAdjustments approves all pending rate adjustments for an item.
func (s *Service) ApproveAdjustments(ctx context.Context, itemID uuid.UUID) (*ReservationResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	var hadPending bool
	_, err := db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ReservationResponse, error) {
		rates, err := qtx.GetBookedRates(ctx, &store.GetBookedRatesParams{
			ReservationItemID: itemID, PropertyID: propertyID,
		})
		if err != nil {
			return nil, fmt.Errorf("get rates: %w", err)
		}

		var pgDates []pgtype.Date
		for _, r := range rates {
			if r.CalendarDate.Valid && !r.AdjustmentApproved {
				pgDates = append(pgDates, r.CalendarDate)
			}
		}
		if len(pgDates) == 0 {
			return &ReservationResponse{}, nil
		}
		hadPending = true

		if err := qtx.ApproveRateAdjustments(ctx, &store.ApproveRateAdjustmentsParams{
			ReservationItemID: itemID, PropertyID: propertyID, Dates: pgDates,
		}); err != nil {
			return nil, fmt.Errorf("approve: %w", err)
		}
		return &ReservationResponse{}, nil
	})
	if err != nil {
		return nil, err
	}
	if !hadPending {
		return &ReservationResponse{}, nil
	}
	// Re-fetch to return populated reservation response.
	item, err := s.q.GetReservationItem(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("get item after approve: %w", err)
	}
	return s.GetReservation(ctx, item.ReservationID, IncludeFlags{Items: true})
}
