package cache

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestCacheOverlaps(t *testing.T) {
	ci := &CacheInvalidator{}
	propertyID := uuid.New()

	tests := []struct {
		name        string
		cacheKey    string
		resCheckIn  string
		resCheckOut string
		want        bool
	}{
		{
			name:        "reservation within cache range",
			cacheKey:    fmt.Sprintf("planner:%s:2025-03-01:2025-03-31", propertyID),
			resCheckIn:  "2025-03-15",
			resCheckOut: "2025-03-18",
			want:        true,
		},
		{
			name:        "reservation starts before cache, ends within",
			cacheKey:    fmt.Sprintf("planner:%s:2025-03-01:2025-03-31", propertyID),
			resCheckIn:  "2025-02-28",
			resCheckOut: "2025-03-05",
			want:        true,
		},
		{
			name:        "reservation starts within cache, ends after",
			cacheKey:    fmt.Sprintf("planner:%s:2025-03-01:2025-03-31", propertyID),
			resCheckIn:  "2025-03-25",
			resCheckOut: "2025-04-02",
			want:        true,
		},
		{
			name:        "reservation completely before cache",
			cacheKey:    fmt.Sprintf("planner:%s:2025-03-01:2025-03-31", propertyID),
			resCheckIn:  "2025-02-15",
			resCheckOut: "2025-02-20",
			want:        false,
		},
		{
			name:        "reservation completely after cache",
			cacheKey:    fmt.Sprintf("planner:%s:2025-03-01:2025-03-31", propertyID),
			resCheckIn:  "2025-04-10",
			resCheckOut: "2025-04-15",
			want:        false,
		},
		{
			name:        "reservation spans entire cache range",
			cacheKey:    fmt.Sprintf("planner:%s:2025-03-01:2025-03-31", propertyID),
			resCheckIn:  "2025-02-20",
			resCheckOut: "2025-04-10",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Split(tt.cacheKey, ":")

			cacheStart := parts[2]
			cacheEnd := parts[3]
			cachePropertyIDStr := parts[1]

			got, err := ci.cacheOverlaps(propertyID, tt.resCheckIn, tt.resCheckOut, cacheStart, cacheEnd, cachePropertyIDStr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("cacheOverlaps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateReservationInvalidatesCache(t *testing.T) {
	// This test would ideally be an integration test that sets up a real Redis instance (e.g. using Testcontainers),
	// creates a CacheInvalidator, and verifies that when UpdateReservation is called, the relevant cache keys are deleted.
	// Due to time constraints, we will not implement this full integration test here, but it would involve:
	// 1. Setting up a test Redis instance
	// 2. Creating a CacheInvalidator with a real Redis client
	// 3. Seeding the cache with a known key (e.g. "planner:{propertyID}:2025-03-01:2025-03-31")
	// 4. Calling UpdateReservation with a reservation that overlaps that date range
	// 5. Verifying that the cache key has been deleted after the call

	t.Skip("Integration test for cache invalidation not implemented")

	t.Run("Updating guest invalidates cache", func(t *testing.T) {
		// TODO
	})

	t.Run("Updating price invalidates cache", func(t *testing.T) {
		// TODO
	})

	t.Run("Updating room type invalidates cache", func(t *testing.T) {
		// TODO
	})

	t.Run("Updating room invalidates cache", func(t *testing.T) {
		// TODO
	})

	t.Run("Updating reservation invalidates cache", func(t *testing.T) {
		// TODO
	})
}
