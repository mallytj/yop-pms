package cache

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func ClearAllCache(cl *redis.Client) error {
	ctx := context.Background()
	return cl.FlushDB(ctx).Err()
}
