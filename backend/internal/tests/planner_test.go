package integration_tests

import (
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

	"ollerod-pms/internal/cache"
	"ollerod-pms/internal/events"
	"ollerod-pms/internal/handlers"
	"ollerod-pms/internal/service"
)

// ---------------------------------------------------------------------------
// Service layer
// ---------------------------------------------------------------------------

func TestPlannerService_GetPlannerData(t *testing.T) {
	ctx := ctx0()
	propID := seedProperty(t, ctx)
	roomTypeID := seedRoomType(t, ctx, propID)
	_ = seedRoom(t, ctx, propID, roomTypeID)

	svc := service.NewPlannerService(*testQueries, testDB)
	svcCtx := withPropertyCtx(propID)

	start := time.Now().Truncate(24 * time.Hour).UTC()
	end := start.AddDate(0, 0, 13)

	t.Run("returns all rooms for the property even with no reservations", func(t *testing.T) {
		data, err := svc.GetPlannerData(svcCtx, start, end)
		require.NoError(t, err)
		require.NotNil(t, data)
		assert.GreaterOrEqual(t, len(data.Rooms), 1)
	})

	t.Run("reservation item is associated with its assigned room", func(t *testing.T) {
		guestID := seedGuest(t, ctx, propID)
		resID := seedReservation(t, ctx, propID, guestID)
		ratePlanID := seedRatePlan(t, ctx, propID, true)
		checkIn := start.AddDate(0, 0, 2)
		checkOut := checkIn.AddDate(0, 0, 3)
		roomID := seedRoom(t, ctx, propID, roomTypeID)

		resItemID := seedReservationItem(t, ctx, propID, resID, roomTypeID, ratePlanID, checkIn, checkOut)

		// Assign the reservation item to the room via raw SQL
		_, err := testDB.Exec(ctx,
			`UPDATE operations.reservation_items SET assigned_room_id = $1 WHERE id = $2`,
			roomID, resItemID,
		)
		require.NoError(t, err)

		data, err := svc.GetPlannerData(svcCtx, start, end)
		require.NoError(t, err)

		var found bool
		for _, room := range data.Rooms {
			if room.RoomID == roomID {
				for _, res := range room.Reservations {
					if res.ReservationItemID == resItemID {
						found = true
						assert.Equal(t, checkIn.Format("2006-01-02"), res.CheckInDate)
						assert.Equal(t, checkOut.Format("2006-01-02"), res.CheckOutDate)
						assert.NotEmpty(t, res.StatusColor)
						assert.Equal(t, "booked", res.ItemStatus)
					}
				}
			}
		}
		assert.True(t, found, "seeded reservation item not found in planner rooms")
	})

	t.Run("unassigned reservation items are not placed in any room", func(t *testing.T) {
		guestID := seedGuest(t, ctx, propID)
		resID := seedReservation(t, ctx, propID, guestID)
		ratePlanID := seedRatePlan(t, ctx, propID, true)
		checkIn := start.AddDate(0, 0, 5)
		checkOut := checkIn.AddDate(0, 0, 2)

		unassignedItemID := seedReservationItem(t, ctx, propID, resID, roomTypeID, ratePlanID, checkIn, checkOut)
		// No assigned_room_id — must NOT appear in any room's Reservations slice

		data, err := svc.GetPlannerData(svcCtx, start, end)
		require.NoError(t, err)

		for _, room := range data.Rooms {
			for _, res := range room.Reservations {
				assert.NotEqual(t, unassignedItemID, res.ReservationItemID,
					"unassigned item should not appear in room %s", room.RoomID)
			}
		}
	})

	t.Run("date range boundaries are populated correctly", func(t *testing.T) {
		data, err := svc.GetPlannerData(svcCtx, start, end)
		require.NoError(t, err)
		assert.Equal(t, start.Format("2006-01-02"), data.StartDate)
		assert.Equal(t, end.Format("2006-01-02"), data.EndDate)
	})

	t.Run("reservation outside the queried date range is excluded", func(t *testing.T) {
		guestID := seedGuest(t, ctx, propID)
		resID := seedReservation(t, ctx, propID, guestID)
		ratePlanID := seedRatePlan(t, ctx, propID, true)
		// far in the future — outside start→end
		futureIn := start.AddDate(1, 0, 0)
		futureOut := futureIn.AddDate(0, 0, 2)
		roomID := seedRoom(t, ctx, propID, roomTypeID)
		futureItem := seedReservationItem(t, ctx, propID, resID, roomTypeID, ratePlanID, futureIn, futureOut)
		_, _ = testDB.Exec(ctx,
			`UPDATE operations.reservation_items SET assigned_room_id = $1 WHERE id = $2`,
			roomID, futureItem,
		)

		data, err := svc.GetPlannerData(svcCtx, start, end)
		require.NoError(t, err)
		for _, room := range data.Rooms {
			for _, res := range room.Reservations {
				assert.NotEqual(t, futureItem, res.ReservationItemID,
					"future reservation should not appear in current date range")
			}
		}
	})

	t.Run("returns error when no property ID in context", func(t *testing.T) {
		_, err := svc.GetPlannerData(withPropertyCtx(zeroUUID()), start, end)
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// HTTP handler layer
// ---------------------------------------------------------------------------

func TestPlannerHandler_GetPlannerData(t *testing.T) {
	ctx := ctx0()
	propID := seedProperty(t, ctx)
	_ = seedRoomType(t, ctx, propID)

	svc := service.NewPlannerService(*testQueries, testDB)
	h := handlers.NewPlannerHandler(svc, testCache, testLogger)

	r := chi.NewRouter()
	r.Get("/planner", h.GetPlannerData)

	svcCtx := withPropertyCtx(propID)
	startStr := time.Now().Format("2006-01-02")
	endStr := time.Now().AddDate(0, 0, 13).Format("2006-01-02")
	validQuery := "startDate=" + startStr + "&endDate=" + endStr

	makeRequest := func(ctx context.Context, query string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/planner?"+query, nil).
			WithContext(ctx)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}

	t.Run("200 on cache miss — populates cache asynchronously", func(t *testing.T) {
		testMiniRedis.FlushAll()
		rr := makeRequest(svcCtx, validQuery)
		assert.Equal(t, http.StatusOK, rr.Code)

		var data map[string]interface{}
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &data))
		assert.Contains(t, data, "rooms")

		// Allow the async goroutine to settle
		time.Sleep(50 * time.Millisecond)
		cacheKey := cache.PlannerCacheKey(propID, startStr, endStr)
		v, err := testRedis.Get(ctx, cacheKey).Result()
		assert.NoError(t, err, "cache should be populated after handler call")
		assert.NotEmpty(t, v)
	})

	t.Run("200 served from cache on second request", func(t *testing.T) {
		rr := makeRequest(svcCtx, validQuery)
		assert.Equal(t, http.StatusOK, rr.Code)
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
		assert.Contains(t, body, "rooms")
	})

	t.Run("cache is invalidated when OnReservationChange fires for overlapping range", func(t *testing.T) {
		cacheKey := cache.PlannerCacheKey(propID, startStr, endStr)

		// Ensure cache is warm
		rr := makeRequest(svcCtx, validQuery)
		require.Equal(t, http.StatusOK, rr.Code)
		time.Sleep(50 * time.Millisecond)
		_, err := testRedis.Get(ctx, cacheKey).Result()
		require.NoError(t, err, "cache entry should exist before invalidation")

		// Simulate what Postgres NOTIFY would emit
		ev := events.Event{
			Channel: "reservation_change",
			Data: map[string]interface{}{
				"property_id":    propID.String(),
				"check_in_date":  startStr,
				"check_out_date": endStr,
			},
		}
		require.NoError(t, testCache.OnReservationChange(ctx, ev))

		val, _ := testRedis.Get(ctx, cacheKey).Result()
		assert.Empty(t, val, "cache entry should be invalidated after reservation change")
	})

	t.Run("unrelated property cache is NOT invalidated", func(t *testing.T) {
		otherPropID := seedProperty(t, ctx)
		otherKey := cache.PlannerCacheKey(otherPropID, startStr, endStr)

		// Seed a cache entry for a different property
		require.NoError(t, testCache.SetPlannerCache(ctx, otherPropID, startStr, endStr,
			&service.PlannerData{StartDate: startStr, EndDate: endStr},
			time.Minute))

		// Fire event for our propID
		ev := events.Event{
			Channel: "reservation_change",
			Data: map[string]interface{}{
				"property_id":    propID.String(),
				"check_in_date":  startStr,
				"check_out_date": endStr,
			},
		}
		require.NoError(t, testCache.OnReservationChange(ctx, ev))

		// Other property's cache must still be intact
		v, err := testRedis.Get(ctx, otherKey).Result()
		assert.NoError(t, err)
		assert.NotEmpty(t, v)
	})

	t.Run("400 when startDate is missing", func(t *testing.T) {
		rr := makeRequest(svcCtx, "endDate="+endStr)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("400 when endDate is missing", func(t *testing.T) {
		rr := makeRequest(svcCtx, "startDate="+startStr)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("400 when date format is invalid", func(t *testing.T) {
		rr := makeRequest(svcCtx, "startDate=01/06/2025&endDate=14/06/2025")
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("400 when startDate is after endDate", func(t *testing.T) {
		rr := makeRequest(svcCtx, "startDate=2025-07-01&endDate=2025-06-01")
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("500 when context carries no property ID", func(t *testing.T) {
		rr := makeRequest(withPropertyCtx(uuid.Nil), "startDate=2025-06-01&endDate=2025-06-14")
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}
