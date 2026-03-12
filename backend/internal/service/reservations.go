package service

import (
	"context"
	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	hf "ollerod-pms/internal/helpers"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Reservation interface {
	UpdateItem(ctx context.Context, updateData UpdateReservationItemData) (repo.OperationsReservationItem, error)
}

type reservationService struct {
	*Svc
}

func NewReservationService(r repo.Queries, db *pgxpool.Pool) Reservation {
	return &reservationService{
		&Svc{
			repo: &r,
			db:   db,
		},
	}
}

type UpdateReservationItemData struct {
	// Optional: ID of the assigned room (for validation)
	AssignedRoomID *uuid.UUID `json:"assigned_room_id,omitempty" example:"123e4567-e89b-12d3-a456-426614174000" validate:"omitempty,uuid7"`
	// Optional: ID of the booked room type (for validation)
	BookedRoomTypeID *uuid.UUID `json:"booked_room_type_id,omitempty" example:"123e4567-e89b-12d3-a456-464897010000" validate:"omitempty,uuid7"`
	// Optional: Check-in date in YYYY-MM-DD format
	CheckInDate *time.Time `json:"check_in_date,omitempty" validate:"omitempty" example:"2024-06-01"`
	// Optional: Check-out date in YYYY-MM-DD format
	CheckOutDate *time.Time `json:"check_out_date,omitempty" validate:"omitempty" example:"2024-06-05"`
	// Optional: New status for the reservation item (e.g., "booked", "checked_in")
	Status *repo.OperationsReservationItemStatus `json:"status,omitempty" validate:"omitempty" example:"booked"`
}

func (s *reservationService) UpdateItem(ctx context.Context, updateData UpdateReservationItemData) (repo.OperationsReservationItem, error) {
	resID := hf.GetResItemIDFromCtx(ctx)

	if resID == uuid.Nil {
		return repo.OperationsReservationItem{}, hf.ErrRelatedEntityNotFound
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return repo.OperationsReservationItem{}, hf.PsqlErrToCustomErr(err)
	}
	defer tx.Rollback(ctx)

	qtx := s.repo.WithTx(tx)

	if updateData.Status == nil {
		defaultStatus := repo.OperationsReservationItemStatusBooked
		updateData.Status = &defaultStatus
	}

	propertyID := hf.GetPropertyIDFromCtx(ctx).String()
	qtx.SetCurrentPropertyID(ctx, propertyID)

	// TODO - Add authorization check here to ensure the user has permission to update this reservation item

	resItem, err := qtx.UpdateReservationItem(ctx, repo.UpdateReservationItemParams{
		ReservationItemID: resID,
		AssignedRoomID:    hf.ToNullUUID(updateData.AssignedRoomID),
		BookedRoomTypeID:  hf.ToNullUUID(updateData.BookedRoomTypeID),
		CheckInDate:       pgtype.Timestamptz{Time: hf.Deref(updateData.CheckInDate), Valid: true},
		CheckOutDate:      pgtype.Timestamptz{Time: hf.Deref(updateData.CheckOutDate), Valid: true},
		Status:            repo.NullOperationsReservationItemStatus{Valid: updateData.Status != nil, OperationsReservationItemStatus: hf.Deref(updateData.Status)},
	})

	if err != nil {
		return repo.OperationsReservationItem{}, hf.PsqlErrToCustomErr(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return repo.OperationsReservationItem{}, hf.PsqlErrToCustomErr(err)
	}

	return resItem, nil
}
