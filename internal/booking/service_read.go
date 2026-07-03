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
	params.PropertyID = propertyID

	if params.Limit <= 0 || params.Limit > 200 {
		params.Limit = 50
	}

	// SQLC-generated ListReservations expects Status to be NULL (not empty string)
	// when no filter is applied. Use a raw query to avoid the type constraint.
	const listSQL = `
		SELECT r.*
		FROM operations.reservations r
		WHERE r.property_id = $1
		AND r.deleted_at IS NULL
		AND ($2::operations.reservation_status IS NULL OR r.status = $2)
		AND ($3::timestamptz IS NULL OR lower(r.stay_period_envelope) < $3
		    OR (lower(r.stay_period_envelope) = $3 AND r.id < $4))
		AND ($5::date IS NULL OR lower(r.stay_period_envelope) >= $5)
		AND ($6::date IS NULL OR upper(r.stay_period_envelope) <= $6)
		ORDER BY lower(r.stay_period_envelope) DESC, r.id DESC
		LIMIT $7
	`

	rawStatus := pgtype.Text{Valid: params.Status != nil}
	if params.Status != nil {
		rawStatus.String = *params.Status
	}
	rawCursorDate := pgtype.Timestamptz{Valid: params.CursorDate != nil}
	if params.CursorDate != nil {
		rawCursorDate.Time = *params.CursorDate
	}
	rawCursorID := pgtype.UUID{Valid: params.CursorID != nil}
	if params.CursorID != nil {
		rawCursorID.Bytes = *params.CursorID
	}
	rawStartDate := pgtype.Date{Valid: params.StartDate != nil}
	if params.StartDate != nil {
		rawStartDate.Time = *params.StartDate
	}
	rawEndDate := pgtype.Date{Valid: params.EndDate != nil}
	if params.EndDate != nil {
		rawEndDate.Time = *params.EndDate
	}

	rows, err := s.pool.Query(ctx, listSQL,
		propertyID,
		rawStatus,
		rawCursorDate,
		rawCursorID,
		rawStartDate,
		rawEndDate,
		params.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list reservations: %w", err)
	}
	defer rows.Close()

	result := make([]ReservationResponse, 0)
	for rows.Next() {
		var r store.OperationsReservation
		if err := rows.Scan(
			&r.ID, &r.PropertyID, &r.PrimaryGuestID, &r.GroupID,
			&r.Source, &r.TravelAgentID, &r.Notes, &r.Status,
			&r.Version, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt,
			&r.Sequential, &r.Code, &r.StayPeriodEnvelope, &r.ExpiresAt,
			&r.CancellationIntent,
		); err != nil {
			return nil, fmt.Errorf("scan reservation: %w", err)
		}
		result = append(result, *reservationToResponse(&r))
	}
	return result, nil
}

// UpdateMetadata applies partial updates to reservation-level metadata fields
// (notes, travel_agent_id, group_id, primary_guest_id). Version check prevents
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
		if input.TravelAgentID != nil {
			pgParams.TravelAgentID = uuid.NullUUID{UUID: *input.TravelAgentID, Valid: true}
		}
		if input.GroupID != nil {
			pgParams.GroupID = uuid.NullUUID{UUID: *input.GroupID, Valid: true}
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
