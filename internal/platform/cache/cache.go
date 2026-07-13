// Package cache provides a Redis-backed cache client with hierarchical keys
// (colon-separated, e.g. "yop:planner:<property>:<date>") and pattern
// invalidation via "yop:foo:*" wildcards. Cache lives in the service layer —
// handlers stay cache-unaware. See ADR-008 and ADR-010.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrCacheMiss is returned when a key is not found in the cache
var ErrCacheMiss = errors.New("cache miss")

// Client provides a simple interface for caching operations with Redis.
type Client struct {
	rdb    *redis.Client
	prefix string
	logger *slog.Logger
}

// New creates a new cache client with the given Redis client, key prefix, and logger.
// The prefix is used to namespace all cache keys (e.g., "yop:").
func New(rdb *redis.Client, prefix string, logger *slog.Logger) *Client {
	return &Client{
		rdb:    rdb,
		prefix: prefix,
		logger: logger,
	}
}

// Set stores a value in the cache with the given key and TTL.
// The value is JSON-encoded.
func (c *Client) Set(ctx context.Context, key string, v any, ttl time.Duration) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	fullKey := c.prefix + key
	return c.rdb.Set(ctx, fullKey, string(data), ttl).Err()
}

// Get retrieves a value from the cache and JSON-decodes it into dst.
// Returns ErrCacheMiss if the key is not found.
func (c *Client) Get(ctx context.Context, key string, dst any) error {
	fullKey := c.prefix + key

	data, err := c.rdb.Get(ctx, fullKey).Result()
	if err == redis.Nil {
		return ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("redis error: %w", err)
	}

	return json.Unmarshal([]byte(data), dst)
}

// Delete removes a key from the cache.
func (c *Client) Delete(ctx context.Context, key string) error {
	fullKey := c.prefix + key
	return c.rdb.Del(ctx, fullKey).Err()
}

// Invalidate removes all keys matching a pattern from the cache.
// Uses SCAN + DEL to avoid blocking on large datasets.
// Pattern should include the prefix if needed (e.g., "yop:availability:*").
func (c *Client) Invalidate(ctx context.Context, pattern string) error {
	var cursor uint64
	var deleteCount int64

	for {
		keys, newCursor, err := c.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("scan error: %w", err)
		}

		if len(keys) > 0 {
			if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("delete error: %w", err)
			}
			deleteCount += int64(len(keys))
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	if deleteCount > 0 {
		c.logger.Debug("cache invalidated", "pattern", pattern, "keys_deleted", deleteCount)
	}

	return nil
}

// InvalidateIf removes all keys matching a pattern where shouldDelete returns true.
// Uses SCAN + filter + DEL, so it never blocks on large key sets.
// Pattern should include the full prefix (e.g., "yop:planner:prop-1:*").
func (c *Client) InvalidateIf(ctx context.Context, pattern string, shouldDelete func(key string) bool) error {
	var cursor uint64
	var deleteCount int64

	for {
		keys, newCursor, err := c.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("scan error: %w", err)
		}

		var toDelete []string
		for _, key := range keys {
			if shouldDelete(key) {
				toDelete = append(toDelete, key)
			}
		}

		if len(toDelete) > 0 {
			if err := c.rdb.Del(ctx, toDelete...).Err(); err != nil {
				return fmt.Errorf("delete error: %w", err)
			}
			deleteCount += int64(len(toDelete))
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	if deleteCount > 0 {
		c.logger.Debug("cache conditionally invalidated", "pattern", pattern, "keys_deleted", deleteCount)
	}

	return nil
}

// GetOrSet is a read-through cache helper.
// If the key exists and can be decoded into dst, it returns nil.
// If the key doesn't exist, it calls the loader function to retrieve the value,
// stores it in the cache with the given TTL, and returns it in dst.
func (c *Client) GetOrSet(ctx context.Context, key string, dst any, ttl time.Duration, loader func(context.Context) (any, error)) error {
	// Try to get from cache
	if err := c.Get(ctx, key, dst); err == nil {
		c.logger.Debug("cache hit", "key", key)
		return nil
	} else if err != ErrCacheMiss {
		// Log Redis errors but don't fail - try the loader
		c.logger.Warn("cache read error", "key", key, "error", err)
	}

	// Cache miss or error - load the value
	c.logger.Debug("cache miss", "key", key)

	value, err := loader(ctx)
	if err != nil {
		return fmt.Errorf("loader error: %w", err)
	}

	// Store in cache (silently fail if cache write fails - don't block the response)
	if err := c.Set(ctx, key, value, ttl); err != nil {
		c.logger.Warn("cache write error", "key", key, "error", err)
	}

	// Return the loaded value
	// For this to work, dst should be a pointer to the same type as value
	// This is a limitation of the current design but acceptable for now
	valueJSON, _ := json.Marshal(value)
	return json.Unmarshal(valueJSON, dst)
}
