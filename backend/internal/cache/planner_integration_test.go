package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ollerod-pms/internal/events"
	"ollerod-pms/internal/service"
)

// ---------------------------------------------------------------------------
// Test setup
// ---------------------------------------------------------------------------

var (
	intTestMR     *miniredis.Miniredis
	intTestRedis  *redis.Client
	intTestLogger *slog.Logger
)

func TestMain(m *testing.M) {
	intTestLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	var err error
	intTestMR, err = miniredis.Run()
	if err != nil {
		panic("miniredis: " + err.Error())
	}
	defer intTestMR.Close()

	intTestRedis = redis.NewClient(&redis.Options{Addr: intTestMR.Addr()})
	m.Run()
}

func newCI() *CacheInvalidator {
	intTestMR.FlushAll()
	return NewCacheInvalidator(intTestRedis, intTestLogger)
}

// ---------------------------------------------------------------------------
// Planner cache: Get/Set
// ---------------------------------------------------------------------------

func TestPlannerCache_SetAndGet(t *testing.T) {
	ci := newCI()
	ctx := context.Background()
	propID := uuid.New()
	start, end := "2026-01-01", "2026-01-14"

	payload := &service.PlannerData{
		StartDate: start,
		EndDate:   end,
		Rooms: []service.PlannerRoom{
			{RoomName: "101", RoomTypeCode: "STD"},
		},
	}

	t.Run("Set stores marshalled JSON in Redis", func(t *testing.T) {
		err := ci.SetPlannerCache(ctx, propID, start, end, payload, time.Minute)
		require.NoError(t, err)

		key := PlannerCacheKey(propID, start, end)
		raw, err := intTestRedis.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Contains(t, raw, "101")
	})

	t.Run("Get returns the stored value on cache hit", func(t *testing.T) {
		ci.SetPlannerCache(ctx, propID, start, end, payload, time.Minute)

		data, found, err := ci.GetPlannerCache(ctx, propID, start, end)
		require.NoError(t, err)
		assert.True(t, found)
		require.NotNil(t, data)
		assert.Equal(t, "101", data.Rooms[0].RoomName)
	})

	t.Run("Get returns (nil, false, nil) on cache miss", func(t *testing.T) {
		intTestMR.FlushAll()
		data, found, err := ci.GetPlannerCache(ctx, propID, start, end)
		assert.NoError(t, err)
		assert.False(t, found)
		assert.Nil(t, data)
	})

	t.Run("Get returns error and deletes key on corrupted JSON", func(t *testing.T) {
		key := PlannerCacheKey(propID, start, end)
		require.NoError(t, intTestRedis.Set(ctx, key, `{not_valid_json`, time.Minute).Err())

		data, found, err := ci.GetPlannerCache(ctx, propID, start, end)
		assert.Error(t, err)
		assert.False(t, found)
		assert.Nil(t, data)

		// Key must have been auto-deleted
		exists := intTestRedis.Exists(ctx, key).Val()
		assert.Equal(t, int64(0), exists, "corrupted key should be auto-deleted")
	})

	t.Run("TTL expires the key correctly", func(t *testing.T) {
		ci.SetPlannerCache(ctx, propID, start, end, payload, 100*time.Millisecond)
		intTestMR.FastForward(200 * time.Millisecond)

		data, found, err := ci.GetPlannerCache(ctx, propID, start, end)
		assert.NoError(t, err)
		assert.False(t, found)
		assert.Nil(t, data)
	})
}

// ---------------------------------------------------------------------------
// Cache invalidation: OnReservationChange
// ---------------------------------------------------------------------------

func TestCacheInvalidation_OnReservationChange(t *testing.T) {
	ctx := context.Background()
	propID := uuid.New()
	ci := newCI()

	seedKey := func(start, end string) string {
		key := PlannerCacheKey(propID, start, end)
		data, _ := json.Marshal(&service.PlannerData{StartDate: start, EndDate: end})
		intTestRedis.Set(ctx, key, data, time.Hour)
		return key
	}

	t.Run("invalidates a cache entry whose range overlaps the reservation", func(t *testing.T) {
		key := seedKey("2026-06-01", "2026-06-30")

		ev := events.Event{
			Data: map[string]interface{}{
				"property_id":    propID.String(),
				"check_in_date":  "2026-06-15",
				"check_out_date": "2026-06-20",
			},
		}
		require.NoError(t, ci.OnReservationChange(ctx, ev))
		assert.Equal(t, int64(0), intTestRedis.Exists(ctx, key).Val(), "overlapping key must be deleted")
	})

	t.Run("does NOT invalidate a non-overlapping cache entry", func(t *testing.T) {
		key := seedKey("2026-08-01", "2026-08-31")

		ev := events.Event{
			Data: map[string]interface{}{
				"property_id":    propID.String(),
				"check_in_date":  "2026-06-15",
				"check_out_date": "2026-06-20",
			},
		}
		require.NoError(t, ci.OnReservationChange(ctx, ev))
		assert.Equal(t, int64(1), intTestRedis.Exists(ctx, key).Val(), "non-overlapping key must survive")
	})

	t.Run("does NOT invalidate a different property's cache", func(t *testing.T) {
		otherProp := uuid.New()
		otherKey := PlannerCacheKey(otherProp, "2026-06-01", "2026-06-30")
		otherData, _ := json.Marshal(&service.PlannerData{StartDate: "2026-06-01"})
		intTestRedis.Set(ctx, otherKey, otherData, time.Hour)

		ev := events.Event{
			Data: map[string]interface{}{
				"property_id":    propID.String(),
				"check_in_date":  "2026-06-15",
				"check_out_date": "2026-06-20",
			},
		}
		require.NoError(t, ci.OnReservationChange(ctx, ev))
		assert.Equal(t, int64(1), intTestRedis.Exists(ctx, otherKey).Val(), "other property's key must survive")
	})

	t.Run("invalidates multiple overlapping keys for same property", func(t *testing.T) {
		k1 := seedKey("2026-06-01", "2026-06-15")
		k2 := seedKey("2026-06-10", "2026-06-30")
		k3 := seedKey("2026-07-01", "2026-07-31") // non-overlapping

		ev := events.Event{
			Data: map[string]interface{}{
				"property_id":    propID.String(),
				"check_in_date":  "2026-06-12",
				"check_out_date": "2026-06-14",
			},
		}
		require.NoError(t, ci.OnReservationChange(ctx, ev))
		assert.Equal(t, int64(0), intTestRedis.Exists(ctx, k1).Val(), "k1 should be deleted (overlaps)")
		assert.Equal(t, int64(0), intTestRedis.Exists(ctx, k2).Val(), "k2 should be deleted (overlaps)")
		assert.Equal(t, int64(1), intTestRedis.Exists(ctx, k3).Val(), "k3 should survive (no overlap)")
	})

	t.Run("returns error on malformed property_id", func(t *testing.T) {
		ev := events.Event{
			Data: map[string]interface{}{
				"property_id":    "not-a-uuid",
				"check_in_date":  "2026-06-12",
				"check_out_date": "2026-06-14",
			},
		}
		err := ci.OnReservationChange(ctx, ev)
		assert.Error(t, err, "should reject invalid property_id")
	})
}

// ---------------------------------------------------------------------------
// Cache key format
// ---------------------------------------------------------------------------

func TestPlannerCacheKey(t *testing.T) {
	propID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	key := PlannerCacheKey(propID, "2026-06-01", "2026-06-30")
	expected := fmt.Sprintf("planner:%s:2026-06-01:2026-06-30", propID.String())
	assert.Equal(t, expected, key)
}
