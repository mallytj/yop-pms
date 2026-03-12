package integration_tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ollerod-pms/internal/handlers"
	"ollerod-pms/internal/service"
)

// ---------------------------------------------------------------------------
// Service layer
// ---------------------------------------------------------------------------

func TestPricingService_GetRateMap(t *testing.T) {
	ctx := ctx0()
	propID := seedProperty(t, ctx)
	roomTypeID := seedRoomType(t, ctx, propID)
	ratePlanID := seedRatePlan(t, ctx, propID, true)

	// Seed base rates for every day-of-week at £100
	seedBaseRate(t, ctx, propID, roomTypeID, ratePlanID, 10000)

	// Seed a DPG override for a specific date at £150
	start := time.Now().Truncate(24 * time.Hour).UTC()
	overrideDate := start.AddDate(0, 0, 2).Format("2006-01-02")
	overridePricePence := int32(15000)
	seedDailyPriceGrid(t, ctx, propID, roomTypeID, ratePlanID, overrideDate, overridePricePence)

	svc := service.NewPricingService(*testQueries, testDB)
	svcCtx := withPropertyCtx(propID)

	end := start.AddDate(0, 0, 6)

	t.Run("returns rates for all dates in range", func(t *testing.T) {
		rm, err := svc.GetRateMap(svcCtx, start, end)
		require.NoError(t, err)
		require.NotNil(t, rm)
		// 7 days × 1 room type × 1 rate plan = 7 entries
		assert.Len(t, rm.Rates, 7)
	})

	t.Run("override date has source=override and elevated price", func(t *testing.T) {
		rm, err := svc.GetRateMap(svcCtx, start, end)
		require.NoError(t, err)
		var found bool
		for _, r := range rm.Rates {
			if r.CalendarDate.Format("2006-01-02") == overrideDate {
				found = true
				assert.Equal(t, "override", r.Source)
				assert.Equal(t, int(overridePricePence), r.Price)
			}
		}
		assert.True(t, found, "override date not found in rate map")
	})

	t.Run("base dates have source=base", func(t *testing.T) {
		rm, err := svc.GetRateMap(svcCtx, start, end)
		require.NoError(t, err)
		for _, r := range rm.Rates {
			if r.CalendarDate.Format("2006-01-02") != overrideDate {
				assert.Equal(t, "base", r.Source, "expected base source for non-overridden date %s", r.CalendarDate)
			}
		}
	})

	t.Run("returns empty rates when no base rates exist for property", func(t *testing.T) {
		emptyPropID := seedProperty(t, ctx)
		rm, err := svc.GetRateMap(withPropertyCtx(emptyPropID), start, end)
		require.NoError(t, err)
		assert.Empty(t, rm.Rates)
	})

	t.Run("returns error when no property in context", func(t *testing.T) {
		_, err := svc.GetRateMap(withPropertyCtx(zeroUUID()), start, end)
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// HTTP handler layer
// ---------------------------------------------------------------------------

func TestPricingHandler_GetRateMap(t *testing.T) {
	ctx := ctx0()
	propID := seedProperty(t, ctx)
	roomTypeID := seedRoomType(t, ctx, propID)
	ratePlanID := seedRatePlan(t, ctx, propID, true)
	seedBaseRate(t, ctx, propID, roomTypeID, ratePlanID, 10000)

	svc := service.NewPricingService(*testQueries, testDB)
	h := handlers.NewPricingHandler(svc, testCache, testLogger)

	r := chi.NewRouter()
	r.Get("/rate-map", h.GetRateMap)

	svcCtx := withPropertyCtx(propID)

	makeRequest := func(query string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/rate-map?"+query, nil).
			WithContext(svcCtx)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}

	t.Run("200 with valid date range", func(t *testing.T) {
		start := time.Now().Format("2006-01-02")
		end := time.Now().AddDate(0, 0, 6).Format("2006-01-02")
		rr := makeRequest("startDate=" + start + "&endDate=" + end)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("400 when startDate is missing", func(t *testing.T) {
		rr := makeRequest("endDate=2025-06-30")
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("400 when endDate is missing", func(t *testing.T) {
		rr := makeRequest("startDate=2025-06-01")
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("400 when date format is invalid", func(t *testing.T) {
		rr := makeRequest("startDate=01-06-2025&endDate=07-06-2025")
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("400 when startDate is after endDate", func(t *testing.T) {
		rr := makeRequest("startDate=2025-06-30&endDate=2025-06-01")
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
