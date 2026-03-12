package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"ollerod-pms/internal/service"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func (ci *CacheInvalidator) GetPlannerCache(ctx context.Context, propertyID uuid.UUID, resCheckIn, resCheckOut string) (*service.PlannerData, bool, error) {
	cacheKey := PlannerCacheKey(propertyID, resCheckIn, resCheckOut)

	cachedData, err := ci.redis.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		return nil, false, nil // Cache miss is not an error

	}

	if err != nil {
		return nil, false, fmt.Errorf("failed to get cache: %w", err)
	}

	var plannerData service.PlannerData
	if err := json.Unmarshal([]byte(cachedData), &plannerData); err != nil {
		// If unmarshaling fails, we should treat it as a cache miss and delete the corrupted cache entry
		// It may be done if the data structure has changed and old cache entries are no longer valid
		if delErr := ci.redis.Del(ctx, cacheKey).Err(); delErr != nil {
			ci.logger.Error("Failed to delete corrupted cache key", "key", cacheKey, "error", delErr)
		}
		return nil, false, fmt.Errorf("failed to unmarshal cached data: %w", err)
	}

	ci.logger.Debug("Planner cache hit", "key", cacheKey)

	return &plannerData, true, nil
}

func (ci *CacheInvalidator) SetPlannerCache(
	ctx context.Context,
	propertyID uuid.UUID,
	startDate, endDate string,
	data *service.PlannerData,
	ttl time.Duration,
) error {
	key := PlannerCacheKey(propertyID, startDate, endDate)

	// Marshal the data to JSON before caching
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal planner data: %w", err)
	}

	// Set the cache with the specified TTL
	if err := ci.redis.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	ci.logger.Debug("Planner cache set", "key", key, "ttl", ttl)

	return nil
}
