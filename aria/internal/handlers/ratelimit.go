package handlers

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const aiRateLimitPerMinute = 30

// TryAIRateLimit increments the per-user counter and reports whether the request is allowed.
func TryAIRateLimit(ctx context.Context, rdb *redis.Client, userID string) (allowed bool, retryAfterSec int, err error) {
	if rdb == nil {
		return true, 0, nil
	}
	key := "rate_limit:" + userID
	n, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return true, 0, err
	}
	if n == 1 {
		_ = rdb.Expire(ctx, key, time.Minute).Err()
	}
	if n > aiRateLimitPerMinute {
		return false, 60, nil
	}
	return true, 0, nil
}
