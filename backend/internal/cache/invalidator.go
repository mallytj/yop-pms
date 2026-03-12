package cache

import (
	"context"
	"fmt"
	"log/slog"
	"ollerod-pms/internal/events"
	"strings"
	"time"

	"ollerod-pms/internal/validator"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type CacheInvalidator struct {
	redis  *redis.Client
	logger *slog.Logger
}

func NewCacheInvalidator(redis *redis.Client, logger *slog.Logger) *CacheInvalidator {
	return &CacheInvalidator{redis: redis, logger: logger}
}

type ReservationChangeData struct {
	PropertyID   uuid.UUID `json:"property_id" validate:""`
	CheckInDate  string    `json:"check_in_date" validate:""`
	CheckOutDate string    `json:"check_out_date" validate:""`
}

// Reservation changes handler
func (ci *CacheInvalidator) OnReservationChange(ctx context.Context, event events.Event) error {
	// Unmarshal event data into ReservationChangeData
	parsedPropertyID, err := uuid.Parse(fmt.Sprintf("%v", event.Data["property_id"]))
	if err != nil {
		return fmt.Errorf("invalid property_id in event data: %w", err)
	}
	data := ReservationChangeData{
		PropertyID:   parsedPropertyID,
		CheckInDate:  event.Data["check_in_date"].(string),
		CheckOutDate: event.Data["check_out_date"].(string),
	}
	if err := validator.ValidateStruct(data); err != nil {
		return err
	}

	return ci.invalidatePlannerCache(ctx, data.PropertyID, data.CheckInDate, data.CheckOutDate)
}
func (ci *CacheInvalidator) invalidatePlannerCache(
	ctx context.Context,
	propertyID uuid.UUID,
	resCheckIn, resCheckOut string,
) error {
	pattern := fmt.Sprintf("planner:%s:*", propertyID)

	var cursor uint64
	deleted := 0

	for {
		var keys []string
		var err error

		// Scan in batches of 100
		keys, cursor, err = ci.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("scan error: %w", err)
		}

		for _, key := range keys {
			parts := strings.Split(key, ":")
			if len(parts) != 4 {
				continue
			}

			cachePropertyIDStr := parts[1]
			cacheStart := parts[2]
			cacheEnd := parts[3]

			if cacheStart > resCheckOut || cacheEnd < resCheckIn {
				continue // No overlap, skip
			}

			shouldInvalidate, err := ci.cacheOverlaps(
				propertyID,
				resCheckIn,
				resCheckOut,
				cacheStart,
				cacheEnd,
				cachePropertyIDStr)

			if err != nil {
				ci.logger.Warn("Failed to parse cache key", "key", key, "error", err)
				continue
			}

			if !shouldInvalidate {
				continue
			}

			if err := ci.redis.Del(ctx, key).Err(); err != nil {
				ci.logger.Error("Failed to delete cache key", "key", key, "error", err)
				continue
			}

			deleted++
		}

		// Done when cursor returns to 0
		if cursor == 0 {
			break
		}
	}

	if deleted > 0 {
		ci.logger.Info("Invalidated planner cache",
			"property_id", propertyID,
			"keys_deleted", deleted)
	}

	return nil
}

func (ci *CacheInvalidator) cacheOverlaps(propertyID uuid.UUID, checkInDate, checkOutDate, cacheStart, cacheEnd, cachePropertyIDStr string) (bool, error) {
	cachePropertyID, err := uuid.Parse(cachePropertyIDStr)
	if err != nil {
		return false, fmt.Errorf("failed to parse property ID from cache key: %w", err)
	}

	// Failsafe check to ensure we only invalidate cache keys for the affected property,
	// even if the date parsing fails later on.
	// This prevents accidentally invalidating unrelated cache entries due to a malformed key.
	if cachePropertyID != propertyID {
		return false, nil // Cache key is for a different property, so it doesn't overlap
	}

	cacheStartDate, err := time.Parse("2006-01-02", cacheStart[:10])
	if err != nil {
		return false, fmt.Errorf("failed to parse cache start date: %w", err)
	}

	cacheEndDate, err := time.Parse("2006-01-02", cacheEnd[:10])
	if err != nil {
		return false, fmt.Errorf("failed to parse cache end date: %w", err)
	}

	checkIn, err := time.Parse("2006-01-02", checkInDate[:10])
	if err != nil {
		return false, fmt.Errorf("failed to parse check-in date: %w", err)
	}

	checkOut, err := time.Parse("2006-01-02", checkOutDate[:10])
	if err != nil {
		return false, fmt.Errorf("failed to parse check-out date: %w", err)
	}

	// Check if the cache period overlaps with the reservation period
	overlaps := !cacheEndDate.Before(checkIn) && !cacheStartDate.After(checkOut)
	return overlaps, nil
}
