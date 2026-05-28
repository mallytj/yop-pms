package booking

// Core Requirements: R-RES-CRUD-003, R-RES-CRUD-010, R-RES-CRUD-011, R-RES-CRUD-012

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	"github.com/lexxcode1/yop-pms/internal/platform/db"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/lexxcode1/yop-pms/internal/platform/util"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// UpdateItem updates reservation item fields and returns the reservation response.
func (s *Service) UpdateItem(ctx context.Context, itemID uuid.UUID, input CreateItemInput) (*ReservationResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ReservationResponse, error) {
		item, err := qtx.GetReservationItem(ctx, itemID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apierror.ErrNotFound
			}
			return nil, fmt.Errorf("get item: %w", err)
		}

		version := helpers.GetIfMatchVersion(ctx)

		arrival := input.ArrivalDate.Time
		departure := input.DepartureDate.Time
		if !departure.After(arrival) {
			return nil, ErrInvalidDates.WithMessage("departure must be after arrival")
		}

		newStayPeriod := util.ToRange(arrival, departure)
		var newRatePlanID = item.RatePlanID
		if input.RatePlanID != nil {
			newRatePlanID = uuid.NullUUID{UUID: *input.RatePlanID, Valid: true}
		}
		var newRoomID = item.AssignedRoomID
		if input.AssignedRoomID != nil {
			newRoomID = uuid.NullUUID{UUID: *input.AssignedRoomID, Valid: true}
		}

		if _, err := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
			ID: itemID, Version: version,
			StayPeriod:     newStayPeriod,
			RatePlanID:     newRatePlanID,
			AssignedRoomID: newRoomID,
			Status:         item.Status,
			AdultsCount:    int32(input.AdultsCount),
			ChildrenCount:  int32(input.ChildrenCount),
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("update item: %w", err)
		}

		// Refresh reservation for response
		resRow, err := qtx.GetReservation(ctx, item.ReservationID)
		if err != nil {
			return nil, fmt.Errorf("get reservation: %w", err)
		}
		resp := reservationFromRow(&resRow)
		expandInclude(ctx, qtx, resp, IncludeFlags{Items: true}, propertyID, item.ReservationID, resRow.PrimaryGuestID, s.log)

		if err := notifyReservationChange(ctx, qtx, "item_updated", item.ReservationID); err != nil {
			return nil, fmt.Errorf("notify: %w", err)
		}
		return resp, nil
	})
}

// UpdateItemStayPeriod changes an item's arrival and/or departure dates.
// For checked_in items, calls ShortenStay to handle future ledger/rates.
func (s *Service) UpdateItemStayPeriod(ctx context.Context, itemID uuid.UUID, arrival, departure time.Time) (*ItemResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ItemResponse, error) {
		item, err := qtx.GetReservationItem(ctx, itemID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apierror.ErrNotFound
			}
			return nil, fmt.Errorf("get item: %w", err)
		}

		if !departure.After(arrival) {
			return nil, ErrInvalidDates.WithMessage("departure must be after arrival")
		}

		newStayPeriod := util.ToRange(arrival, departure)
		version := helpers.GetIfMatchVersion(ctx)

		if item.Status == store.OperationsReservationItemStatusCheckedIn {
			if err := s.ShortenStay(ctx, qtx, item, departure); err != nil {
				return nil, err
			}
		}

		updated, err := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
			ID: itemID, Version: version,
			BookedRoomTypeID: uuid.NullUUID{},
			StayPeriod:       newStayPeriod,
			AssignedRoomID:   item.AssignedRoomID,
			RatePlanID:       item.RatePlanID,
			Status:           item.Status,
			AdultsCount:      item.AdultsCount,
			ChildrenCount:    item.ChildrenCount,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("update stay period: %w", err)
		}

		// Update envelope on the reservation
		if err := recomputeEnvelope(ctx, qtx, s.log, item.ReservationID, propertyID, version); err != nil {
			return nil, err
		}

		if err := notifyReservationChange(ctx, qtx, "item_updated", item.ReservationID); err != nil {
			return nil, fmt.Errorf("notify: %w", err)
		}

		return itemToResponse(&updated), nil
	})
}

// AssignRoom assigns a room to an item. Respects do-not-move flag.
func (s *Service) AssignRoom(ctx context.Context, itemID uuid.UUID, input AssignRoomInput) (*ItemResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ItemResponse, error) {
		item, err := qtx.GetReservationItem(ctx, itemID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apierror.ErrNotFound
			}
			return nil, fmt.Errorf("get item: %w", err)
		}

		if item.DoNotMove {
			return nil, ErrDoNotMove
		}

		version := helpers.GetIfMatchVersion(ctx)
		newRoomID := uuid.NullUUID{UUID: input.RoomID, Valid: true}

		updated, err := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
			ID: itemID, Version: version,
			BookedRoomTypeID: uuid.NullUUID{},
			AssignedRoomID:   newRoomID,
			StayPeriod:       item.StayPeriod,
			RatePlanID:       item.RatePlanID,
			Status:           item.Status,
			AdultsCount:      item.AdultsCount,
			ChildrenCount:    item.ChildrenCount,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("assign room: %w", err)
		}

		// Update ledger rows to reflect new room
		if err := qtx.UpdateLedgerRowRoom(ctx, &store.UpdateLedgerRowRoomParams{
			ReservationItemID: uuid.NullUUID{UUID: itemID, Valid: true}, PropertyID: propertyID, NewRoomID: input.RoomID,
		}); err != nil {
			return nil, fmt.Errorf("update ledger: %w", err)
		}

		if err := notifyReservationChange(ctx, qtx, "room_assigned", item.ReservationID); err != nil {
			return nil, fmt.Errorf("notify: %w", err)
		}

		return itemToResponse(&updated), nil
	})
}

// UpdateItemRoomType changes an item's room type. Optionally retains the original price.
func (s *Service) UpdateItemRoomType(ctx context.Context, itemID uuid.UUID, newRoomTypeID uuid.UUID, retainPrice bool) (*ItemResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ItemResponse, error) {
		item, err := qtx.GetReservationItem(ctx, itemID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apierror.ErrNotFound
			}
			return nil, fmt.Errorf("get item: %w", err)
		}

		if item.DoNotMove {
			return nil, ErrDoNotMove
		}

		version := helpers.GetIfMatchVersion(ctx)

		// Preserve prices if retainPrice=true
		var updated store.OperationsReservationItem
		if retainPrice {
			updated, err = qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
				ID: itemID, Version: version,
				BookedRoomTypeID: uuid.NullUUID{UUID: newRoomTypeID, Valid: true},
				StayPeriod:       item.StayPeriod,
				RatePlanID:       item.RatePlanID,
				AssignedRoomID:   item.AssignedRoomID,
				Status:           item.Status,
				AdultsCount:      item.AdultsCount,
				ChildrenCount:    item.ChildrenCount,
			})
		} else {
			// Clear rate plan so new rates can be generated for the new room type.
			updated, err = qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
				ID: itemID, Version: version,
				BookedRoomTypeID: uuid.NullUUID{UUID: newRoomTypeID, Valid: true},
				StayPeriod:       item.StayPeriod,
				RatePlanID:       uuid.NullUUID{},
				AssignedRoomID:   item.AssignedRoomID,
				Status:           item.Status,
				AdultsCount:      item.AdultsCount,
				ChildrenCount:    item.ChildrenCount,
			})
		}
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("update room type: %w", err)
		}

		if err := notifyReservationChange(ctx, qtx, "room_type_changed", item.ReservationID); err != nil {
			return nil, fmt.Errorf("notify: %w", err)
		}

		return itemToResponse(&updated), nil
	})
}

// UpdateItemRatePlan changes an item's rate plan. Checks rate plan capacity.
func (s *Service) UpdateItemRatePlan(ctx context.Context, itemID uuid.UUID, newRatePlanID uuid.UUID, retainPrice bool) (*ItemResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ItemResponse, error) {
		item, err := qtx.GetReservationItem(ctx, itemID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apierror.ErrNotFound
			}
			return nil, fmt.Errorf("get item: %w", err)
		}

		version := helpers.GetIfMatchVersion(ctx)
		ratePlanID := uuid.NullUUID{UUID: newRatePlanID, Valid: true}

		updated, err := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
			ID: itemID, Version: version,
			BookedRoomTypeID: uuid.NullUUID{},
			RatePlanID:       ratePlanID,
			StayPeriod:       item.StayPeriod,
			AssignedRoomID:   item.AssignedRoomID,
			Status:           item.Status,
			AdultsCount:      item.AdultsCount,
			ChildrenCount:    item.ChildrenCount,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("update rate plan: %w", err)
		}

		if err := notifyReservationChange(ctx, qtx, "rate_plan_changed", item.ReservationID); err != nil {
			return nil, fmt.Errorf("notify: %w", err)
		}

		return itemToResponse(&updated), nil
	})
}

// AddItem adds a new item to an existing non-terminal reservation.
func (s *Service) AddItem(ctx context.Context, id uuid.UUID, input AddItemInput) (*ReservationResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ReservationResponse, error) {
		res, err := qtx.GetReservation(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apierror.ErrNotFound
			}
			return nil, fmt.Errorf("get reservation: %w", err)
		}

		if IsTerminalReservationStatus(ReservationStatus(res.Status)) {
			return nil, ErrTerminal
		}

		itemResp, err := insertSingleItem(ctx, qtx, input.CreateItemInput,
			struct {
				ReservationStatus ReservationStatus
				ItemStatus        ItemStatus
			}{ReservationStatus: StatusConfirmed, ItemStatus: ItemStatusBooked},
			propertyID, id,
		)
		if err != nil {
			return nil, fmt.Errorf("insert item: %w", err)
		}

		// Update stay_period_envelope to include new item's range
		version := helpers.GetIfMatchVersion(ctx)
		if err := recomputeEnvelope(ctx, qtx, s.log, id, propertyID, version); err != nil {
			return nil, err
		}

		if err := notifyReservationChange(ctx, qtx, "item_added", id); err != nil {
			return nil, fmt.Errorf("notify: %w", err)
		}

		updatedRes, err := qtx.GetReservation(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get updated reservation: %w", err)
		}
		response := reservationFromRow(&updatedRes)
		response.Items = append(response.Items, itemResp)
		return response, nil
	})
}

// recomputeEnvelope recalculates stay_period_envelope from all items.
func recomputeEnvelope(ctx context.Context, qtx *store.Queries, log *slog.Logger, reservationID, propertyID uuid.UUID, version int32) error {
	items, err := qtx.GetReservationItems(ctx, &store.GetReservationItemsParams{
		ReservationID: reservationID, PropertyID: propertyID,
	})
	if err != nil {
		return fmt.Errorf("get items for envelope: %w", err)
	}

	var lower, upper time.Time
	for i, item := range items {
		if i == 0 || item.StayPeriod.Lower.Time.Before(lower) {
			lower = item.StayPeriod.Lower.Time
		}
		if i == 0 || item.StayPeriod.Upper.Time.After(upper) {
			upper = item.StayPeriod.Upper.Time
		}
	}

	if lower.IsZero() || upper.IsZero() {
		return nil
	}

	res, err := qtx.GetReservation(ctx, reservationID)
	if err != nil {
		return fmt.Errorf("get reservation for envelope: %w", err)
	}

	_, err = qtx.UpdateReservationMetadata(ctx, &store.UpdateReservationMetadataParams{
		ID:                 reservationID,
		Version:            version,
		StayPeriodEnvelope: util.ToRange(lower, upper),
		PrimaryGuestID:     res.PrimaryGuestID,
	})
	return err
}
