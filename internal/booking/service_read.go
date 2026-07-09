package booking

// Core Requirements: R-RES-CRUD-002, R-RES-CRUD-010

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	"github.com/lexxcode1/yop-pms/internal/platform/db"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// GetReservation fetches a single reservation by ID. The reservation must belong
// to the property in context (enforced by RLS + SQL WHERE clause).
func (s *Service) GetReservation(ctx context.Context, id uuid.UUID, include IncludeFlags) (*ReservationResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	row, err := s.q.GetReservation(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.ErrNotFound
		}
		return nil, fmt.Errorf("get reservation: %w", err)
	}

	response := reservationFromRow(&row)

	expandInclude(ctx, s.q, response, include, propertyID, id, uuid.NullUUID{UUID: row.PrimaryGuestID, Valid: true}, s.log)

	return response, nil
}

// ListReservations returns a cursor-paginated list of reservations for the property
// in context (ADR-014). Default limit is 50; max is 200.
func (s *Service) ListReservations(ctx context.Context, params ListParams) ([]ReservationResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	if params.Limit <= 0 || params.Limit > 200 {
		params.Limit = 50
	}

	queryParams := &store.ListReservationsParams{
		PropertyID: propertyID,
		Limit:      int32(params.Limit),
	}
	if params.Status != nil {
		queryParams.Status = *params.Status
	}
	if params.CursorDate != nil {
		queryParams.CursorDate = pgtype.Timestamptz{Time: *params.CursorDate, Valid: true}
	}
	if params.CursorID != nil {
		queryParams.CursorID = *params.CursorID
	}
	if params.StartDate != nil {
		queryParams.StartDate = pgtype.Date{Time: *params.StartDate, Valid: true}
	}
	if params.EndDate != nil {
		queryParams.EndDate = pgtype.Date{Time: *params.EndDate, Valid: true}
	}

	rows, err := s.q.ListReservations(ctx, queryParams)
	if err != nil {
		return nil, fmt.Errorf("list reservations: %w", err)
	}

	result := make([]ReservationResponse, 0, len(rows))
	for i := range rows {
		result = append(result, *reservationToResponse(&rows[i]))
	}
	return result, nil
}

// UpdateMetadata applies partial updates to reservation-level metadata fields
// (notes, primary_guest_id). Version check prevents
// concurrent overwrites (ErrVersionMismatch on mismatch).
func (s *Service) UpdateMetadata(ctx context.Context, id uuid.UUID, input UpdateMetadataInput) (*ReservationResponse, error) {
	version := helpers.GetIfMatchVersion(ctx)

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ReservationResponse, error) {
		pgParams := &store.UpdateReservationMetadataParams{
			ID:      id,
			Version: version,
		}
		if input.Notes != nil {
			pgParams.Notes = pgtype.Text{String: *input.Notes, Valid: true}
		}

		if input.PrimaryGuestID != nil {
			pgParams.PrimaryGuestID = *input.PrimaryGuestID
		} else {
			// PrimaryGuestID in the SQLC params struct is uuid.UUID (not nullable).
			// When omitted from PATCH, fetch the current value to avoid sending
			// a zero UUID which would cause a FK violation.
			current, err := qtx.GetReservation(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("get current reservation: %w", err)
			}
			pgParams.PrimaryGuestID = current.PrimaryGuestID
		}

		updated, err := qtx.UpdateReservationMetadata(ctx, pgParams)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("update metadata: %w", err)
		}

		if err := notifyReservationChange(ctx, qtx, "updated", id); err != nil {
			s.log.Warn("notify reservation_changes after metadata update", "error", err)
		}

		return reservationToResponse(&updated), nil
	})
}
