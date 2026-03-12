package integration_tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	"ollerod-pms/internal/handlers"
	hf "ollerod-pms/internal/helpers"
	"ollerod-pms/internal/service"
)

// ---------------------------------------------------------------------------
// Service layer
// ---------------------------------------------------------------------------

func TestReservationService_UpdateItem(t *testing.T) {
	ctx := ctx0()
	propID := seedProperty(t, ctx)
	roomTypeID := seedRoomType(t, ctx, propID)
	guestID := seedGuest(t, ctx, propID)
	resID := seedReservation(t, ctx, propID, guestID)
	ratePlanID := seedRatePlan(t, ctx, propID, true)

	checkIn := time.Now().AddDate(0, 0, 10)
	checkOut := checkIn.AddDate(0, 0, 3)
	resItemID := seedReservationItem(t, ctx, propID, resID, roomTypeID, ratePlanID, checkIn, checkOut)
	svc := service.NewReservationService(*testQueries, testDB)
	ctxWithIDs := withResItemCtx(propID, resItemID)

	t.Run("updates status to checked_in", func(t *testing.T) {
		newStatus := repo.OperationsReservationItemStatusCheckedIn
		updated, err := svc.UpdateItem(ctxWithIDs, service.UpdateReservationItemData{
			Status: &newStatus,
		})
		require.NoError(t, err)
		assert.Equal(t, repo.OperationsReservationItemStatusCheckedIn, updated.Status)
	})

	t.Run("updates assigned room", func(t *testing.T) {
		roomID := seedRoom(t, ctx, propID, roomTypeID)
		updated, err := svc.UpdateItem(ctxWithIDs, service.UpdateReservationItemData{
			AssignedRoomID: &roomID,
		})
		require.NoError(t, err)
		assert.True(t, updated.AssignedRoomID.Valid)
		assert.Equal(t, roomID, updated.AssignedRoomID.UUID)
	})

	t.Run("updates stay period dates", func(t *testing.T) {
		newIn := checkIn.AddDate(0, 0, 1)
		newOut := checkOut.AddDate(0, 0, 1)
		updated, err := svc.UpdateItem(ctxWithIDs, service.UpdateReservationItemData{
			CheckInDate:  &newIn,
			CheckOutDate: &newOut,
		})
		require.NoError(t, err)
		if updated.StayPeriod.Lower.Valid {
			assert.Equal(t,
				newIn.UTC().Truncate(time.Second),
				updated.StayPeriod.Lower.Time.UTC().Truncate(time.Second),
			)
		}
	})

	t.Run("updates all fields simultaneously", func(t *testing.T) {
		roomID := seedRoom(t, ctx, propID, roomTypeID)
		newStatus := repo.OperationsReservationItemStatusBooked
		newIn := checkIn.AddDate(0, 0, 2)
		newOut := checkOut.AddDate(0, 0, 2)
		updated, err := svc.UpdateItem(ctxWithIDs, service.UpdateReservationItemData{
			AssignedRoomID: &roomID,
			CheckInDate:    &newIn,
			CheckOutDate:   &newOut,
			Status:         &newStatus,
		})
		require.NoError(t, err)
		assert.Equal(t, repo.OperationsReservationItemStatusBooked, updated.Status)
		assert.True(t, updated.AssignedRoomID.Valid)
	})

	t.Run("Status=nil defaults to booked", func(t *testing.T) {
		// Fresh item so its starting state is known
		resItemID2 := seedReservationItem(t, ctx, propID, resID, roomTypeID, ratePlanID,
			checkIn.AddDate(1, 0, 0), checkOut.AddDate(1, 0, 0))
		updated, err := svc.UpdateItem(withResItemCtx(propID, resItemID2),
			service.UpdateReservationItemData{Status: nil})
		require.NoError(t, err)
		assert.Equal(t, repo.OperationsReservationItemStatusBooked, updated.Status)
	})

	t.Run("returns ErrRelatedEntityNotFound when resItem ID is missing from context", func(t *testing.T) {
		noItemCtx := withPropertyCtx(propID)
		_, err := svc.UpdateItem(noItemCtx, service.UpdateReservationItemData{})
		assert.ErrorIs(t, err, hf.ErrRelatedEntityNotFound)
	})

	t.Run("returns error when no property ID in context", func(t *testing.T) {
		noPropertyCtx := hf.SetIDInCtx(context.Background(), hf.ReservationItemIDKey, resItemID)
		_, err := svc.UpdateItem(noPropertyCtx, service.UpdateReservationItemData{})
		assert.Error(t, err)
	})

	t.Run("returns error when reservation item does not exist", func(t *testing.T) {
		ghostCtx := withResItemCtx(propID, uuid.New())
		_, err := svc.UpdateItem(ghostCtx, service.UpdateReservationItemData{})
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// HTTP handler layer
// ---------------------------------------------------------------------------

func TestReservationHandler_UpdateReservationItem(t *testing.T) {
	ctx := ctx0()
	propID := seedProperty(t, ctx)
	roomTypeID := seedRoomType(t, ctx, propID)
	guestID := seedGuest(t, ctx, propID)
	resID := seedReservation(t, ctx, propID, guestID)
	ratePlanID := seedRatePlan(t, ctx, propID, true)

	checkIn := time.Now().AddDate(0, 0, 15)
	checkOut := checkIn.AddDate(0, 0, 3)
	resItemID := seedReservationItem(t, ctx, propID, resID, roomTypeID, ratePlanID, checkIn, checkOut)

	svc := service.NewReservationService(*testQueries, testDB)
	h := handlers.NewReservationHandler(svc)

	r := chi.NewRouter()
	r.Put("/reservation_item/{reservationID}", h.UpdateReservationItem)

	serve := func(ctx context.Context, body []byte) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPut,
			"/reservation_item/"+resItemID.String(),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}

	validCtx := withResItemCtx(propID, resItemID)

	t.Run("200 with valid status update", func(t *testing.T) {
		newStatus := repo.OperationsReservationItemStatusCheckedIn
		body, _ := json.Marshal(service.UpdateReservationItemData{Status: &newStatus})
		rr := serve(validCtx, body)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("200 with empty update (all fields nil — defaults apply)", func(t *testing.T) {
		body, _ := json.Marshal(service.UpdateReservationItemData{})
		rr := serve(validCtx, body)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("400 with malformed JSON body", func(t *testing.T) {
		rr := serve(validCtx, []byte(`{bad json`))
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("400 with empty body", func(t *testing.T) {
		rr := serve(validCtx, []byte(``))
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
