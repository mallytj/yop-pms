package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ollerod-pms/internal/handlers"
	"ollerod-pms/internal/service"
)

// ---------------------------------------------------------------------------
// Service layer
// ---------------------------------------------------------------------------

func TestRatePlanService_Get(t *testing.T) {
	ctx := ctx0()

	t.Run("returns only active plans for the property", func(t *testing.T) {
		propID := seedProperty(t, ctx)
		svcCtx := withPropertyCtx(propID)

		_ = seedRatePlan(t, ctx, propID, true)  // active  — should appear
		_ = seedRatePlan(t, ctx, propID, false) // inactive — must NOT appear

		svc := service.NewRatePlanService(*testQueries, testDB)
		plans, err := svc.Get(svcCtx)
		require.NoError(t, err)
		assert.Len(t, plans, 1, "expected exactly 1 active rate plan")
	})

	t.Run("returns empty slice for a property with no plans", func(t *testing.T) {
		emptyPropID := seedProperty(t, ctx)
		svc := service.NewRatePlanService(*testQueries, testDB)
		plans, err := svc.Get(withPropertyCtx(emptyPropID))
		require.NoError(t, err)
		assert.Empty(t, plans)
	})

	t.Run("multiple active plans are all returned", func(t *testing.T) {
		propID := seedProperty(t, ctx)
		svcCtx := withPropertyCtx(propID)
		_ = seedRatePlan(t, ctx, propID, true)
		_ = seedRatePlan(t, ctx, propID, true)
		_ = seedRatePlan(t, ctx, propID, true)

		svc := service.NewRatePlanService(*testQueries, testDB)
		plans, err := svc.Get(svcCtx)
		require.NoError(t, err)
		assert.Len(t, plans, 3)
	})

	t.Run("returns error when no property ID in context", func(t *testing.T) {
		svc := service.NewRatePlanService(*testQueries, testDB)
		_, err := svc.Get(withPropertyCtx(zeroUUID()))
		assert.Error(t, err, "ExecuteTx should reject a nil property ID")
	})

	t.Run("plans from another property are not visible", func(t *testing.T) {
		propA := seedProperty(t, ctx)
		propB := seedProperty(t, ctx)
		_ = seedRatePlan(t, ctx, propB, true) // seeded under propB

		svc := service.NewRatePlanService(*testQueries, testDB)
		plans, err := svc.Get(withPropertyCtx(propA))
		require.NoError(t, err)
		assert.Empty(t, plans, "plans from another property must not be visible")
	})
}

// ---------------------------------------------------------------------------
// HTTP handler layer
// ---------------------------------------------------------------------------

func TestRatePlanHandler_GetRatePlans(t *testing.T) {
	ctx := ctx0()

	propID := seedProperty(t, ctx)
	svcCtx := withPropertyCtx(propID)
	_ = seedRatePlan(t, ctx, propID, true)

	svc := service.NewRatePlanService(*testQueries, testDB)
	h := handlers.NewRatePlanHandler(svc, testCache, testLogger)

	r := chi.NewRouter()
	r.Get("/rate-plans", h.GetRatePlans)

	t.Run("200 with active rate plans in body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/rate-plans", nil).
			WithContext(svcCtx)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var plans []map[string]interface{}
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &plans))
		assert.NotEmpty(t, plans)
	})

	t.Run("200 with null/empty body for property with no active plans", func(t *testing.T) {
		emptyPropID := seedProperty(t, ctx)
		req := httptest.NewRequest(http.MethodGet, "/rate-plans", nil).
			WithContext(withPropertyCtx(emptyPropID))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("response body contains expected fields", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/rate-plans", nil).
			WithContext(svcCtx)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		var plans []map[string]interface{}
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &plans))
		require.NotEmpty(t, plans)

		first := plans[0]
		assert.Contains(t, first, "id")
		assert.Contains(t, first, "name")
		assert.Contains(t, first, "code")
		assert.Contains(t, first, "currency_code")
	})
}
