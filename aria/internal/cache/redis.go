package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewClient(redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	c := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := c.Ping(ctx).Err(); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return c, nil
}

func SetWithTTL(ctx context.Context, c *redis.Client, key string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}
	if err := c.Set(ctx, key, b, ttl).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

func Get(ctx context.Context, c *redis.Client, key string) (string, bool, error) {
	val, err := c.Get(ctx, key).Result()
	if err == nil {
		return val, true, nil
	}
	if err == redis.Nil {
		return "", false, nil
	}
	return "", false, fmt.Errorf("redis get: %w", err)
}

func Delete(ctx context.Context, c *redis.Client, key string) error {
	if err := c.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del: %w", err)
	}
	return nil
}

func Exists(ctx context.Context, c *redis.Client, key string) (bool, error) {
	n, err := c.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return n > 0, nil
}

