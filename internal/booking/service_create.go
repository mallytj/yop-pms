package booking

// Core Requirements: R-RES-CRUD-001, R-RES-CRUD-013, R-RES-CRUD-014, R-RES-CRUD-018, ADR-015, ADR-018

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lexxcode1/yop-pms/internal/platform/db"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/lexxcode1/yop-pms/internal/platform/util"
	"github.com/lexxcode1/yop-pms/internal/platform/worker"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// CreateReservation creates a new reservation with items, ledger rows,
// booked daily rates, and a stub Folio A. The initial status depends on
// source and walkin flag (see SourceToInitialStatus map).
//
// Guest dedup by email is deferred to the guest-profile PR. Frontend will
// use pg_trgm autocomplete against identity.guests for live search.
//
// include controls response depth. When flags.Items is false, the response
// omits the items array. When flags.Guest is true and primary_guest_id is
// set, the guest object is expanded inline.
func (s *Service) CreateReservation(ctx context.Context, input *CreateReservationInput, include IncludeFlags) (*ReservationResponse, error) {
	initial, ok := SourceToInitialStatus[input.Source]
	if !ok {
		return nil, ErrSourceDeferred
	}

	// Walk-in override: internal + is_walkin=true → checked_in
	if SourceIsWalkin(input.Source, input.IsWalkin) {
		initial = struct {
			ReservationStatus ReservationStatus
			ItemStatus        ItemStatus
		}{StatusCheckedIn, ItemStatusCheckedIn}

		// If source is walk-in, every room must be assigned
		for i, item := range input.Items {
			if item.AssignedRoomID == nil {
				return nil, ErrUnassignedItems.WithMessage(fmt.Sprintf("item %d: walk-in requires assigned_room_id", i))
			}
		}
	}

	res, txErr := db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ReservationResponse, error) {
		// Past-date check uses property timezone, not server UTC. Skip for walk-in
		// (R-RES-VALID-002); retroactive_create is gate-checked in handler.
		if !SourceIsWalkin(input.Source, input.IsWalkin) {
			propertyID := helpers.GetPropertyIDFromCtx(ctx)
			tzName, err := qtx.GetPropertyTimezone(ctx, propertyID)
			if err != nil {
				return nil, fmt.Errorf("load property timezone: %w", err)
			}
			loc, err := time.LoadLocation(tzName)
			if err != nil {
				return nil, fmt.Errorf("invalid property timezone %q: %w", tzName, err)
			}
			now := time.Now().In(loc)
			midnightToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
			for _, item := range input.Items {
				if item.ArrivalDate.Before(midnightToday) {
					return nil, ErrInvalidDates.WithMessage("arrival date must be today or later")
				}
			}
		}

		result, callErr := s.createReservationInTx(ctx, qtx, input, initial, include)
		if callErr != nil {
			s.log.Error("createReservationInTx failed", "error", callErr, "source", input.Source)
		}
		return result, callErr
	})
	if txErr != nil {
		return nil, txErr
	}

	// Invalidate availability cache after successful creation
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	s.invalidateAvailabilityAfterCreate(ctx, propertyID, input)

	return res, nil
}

// invalidateAvailabilityAfterCreate clears the availability cache for all room types
// and date ranges affected by a newly created reservation.
func (s *Service) invalidateAvailabilityAfterCreate(ctx context.Context, propertyID uuid.UUID, input *CreateReservationInput) {
	seen := make(map[uuid.UUID]bool)
	for _, item := range input.Items {
		if seen[item.RoomTypeID] {
			continue
		}
		seen[item.RoomTypeID] = true
		s.InvalidateAvailabilityCache(ctx, propertyID, item.RoomTypeID, item.ArrivalDate.Time, item.DepartureDate.Time)
	}
}

// createReservationInTx is the transactional body of CreateReservation.
func (s *Service) createReservationInTx(
	ctx context.Context,
	qtx *store.Queries,
	input *CreateReservationInput,
	initial struct {
		ReservationStatus ReservationStatus
		ItemStatus        ItemStatus
	},
	include IncludeFlags,
) (*ReservationResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	now := time.Now()

	// Compute stay_period_envelope across all items (ADR-020)
	envLower, envUpper := computeEnvelope(input.Items)

	expiresAt := computeExpiresAt(input.Source, initial.ReservationStatus, now)

	// --- 2. Resolve guest before creating reservation (FK requires valid primary_guest_id) ---
	primaryGuestID, err := resolvePrimaryGuest(ctx, qtx, input, propertyID)
	if err != nil {
		return nil, fmt.Errorf("resolve guest: %w", err)
	}

	// --- 3. Insert reservation ---
	res, err := qtx.CreateReservation(ctx, &store.CreateReservationParams{
		PropertyID:         propertyID,
		PrimaryGuestID:     primaryGuestID,
		GroupID:            uuid.NullUUID{UUID: util.PtrUUID(input.GroupID), Valid: input.GroupID != nil},
		Source:             store.OperationsReservationSource(input.Source),
		TravelAgentID:      uuid.NullUUID{UUID: util.PtrUUID(input.TravelAgentID), Valid: input.TravelAgentID != nil},
		Notes:              pgtype.Text{String: input.Notes, Valid: input.Notes != ""},
		Status:             store.OperationsReservationStatus(initial.ReservationStatus),
		StayPeriodEnvelope: util.ToRange(envLower, envUpper),
		ExpiresAt:          expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("create reservation: %w", err)
	}

	// --- 4. Insert items + ledger rows + booked daily rates ---
	itemResponses, err := insertAllItems(ctx, qtx, input, initial, propertyID, res.ID)
	if err != nil {
		return nil, err
	}
	if _, err := qtx.CreateFolio(ctx, &store.CreateFolioParams{
		PropertyID:    propertyID,
		ReservationID: uuid.NullUUID{UUID: res.ID, Valid: true},
		FolioPart:     store.FinanceFolioPartA,
	}); err != nil {
		return nil, fmt.Errorf("create folio a: %w", err)
	}

	if err := notifyReservationChange(ctx, qtx, "created", res.ID); err != nil {
		return nil, fmt.Errorf("notify reservation_changes: %w", err)
	}

	// Only enqueue on create for sources that start confirmed (e.g. OTA).
	// Hold reservations get the email when ConfirmReservation is called.
	if initial.ReservationStatus != StatusHold {
		if err := worker.Enqueue(ctx, qtx, worker.EventConfirmationEmail, worker.ConfirmationEmailPayload{
			ReservationID: res.ID.String(),
		}); err != nil {
			s.log.Error("enqueue confirmation email", "error", err, "reservation_id", res.ID)
		}
	}

	response := reservationToResponse(&res)

	if include.IncludeItems() {
		response.Items = itemResponses
	}

	expandInclude(ctx, qtx, response, include, propertyID, res.ID, primaryGuestID, s.log)
	return response, nil
}

// ConfirmReservation transitions a hold reservation to confirmed.
// Requires the reservation to be in 'hold' status and have a guest attached.
// If-Match version is extracted from context (set by RequireIfMatch middleware).
//
// include controls response depth (items, guest expansion).
func (s *Service) ConfirmReservation(ctx context.Context, id uuid.UUID, include IncludeFlags) (*ReservationResponse, error) {
	version := helpers.GetIfMatchVersion(ctx)

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ReservationResponse, error) {
		resRow, err := qtx.GetReservation(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get reservation: %w", err)
		}

		// State machine check: only hold → confirmed is allowed
		if err := ValidateReservationTransition(
			ReservationStatus(resRow.Status),
			StatusConfirmed,
		); err != nil {
			return nil, err
		}

		// Guest must be attached (R-RES-CRUD-018)
		if resRow.PrimaryGuestID == uuid.Nil {
			return nil, ErrGuestNotAttached
		}

		rowsAffected, err := qtx.UpdateReservationStatus(ctx, &store.UpdateReservationStatusParams{
			ID:      id,
			Version: version,
			Status:  store.OperationsReservationStatusConfirmed,
		})
		if err != nil {
			return nil, fmt.Errorf("confirm reservation: %w", err)
		}
		if rowsAffected == 0 {
			return nil, ErrVersionMismatch
		}

		// Re-fetch to get updated row
		updatedRow, err := qtx.GetReservation(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("re-fetch reservation: %w", err)
		}

		response := reservationFromRow(&updatedRow)

		expandInclude(ctx, qtx, response, include, resRow.PropertyID, id, resRow.PrimaryGuestID, s.log)

		if err := notifyReservationChange(ctx, qtx, "confirmed", id); err != nil {
			return nil, fmt.Errorf("notify reservation_changes: %w", err)
		}

		// Enqueue confirmation email outbox event
		if err := worker.Enqueue(ctx, qtx, worker.EventConfirmationEmail, worker.ConfirmationEmailPayload{
			ReservationID: id.String(),
		}); err != nil {
			s.log.Error("enqueue confirmation email", "error", err, "reservation_id", id)
		}

		return response, nil
	})
}
