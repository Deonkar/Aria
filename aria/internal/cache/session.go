package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionMessage struct {
	Role         string    `json:"role"`
	Content      string    `json:"content"`
	GeneratedSQL string    `json:"generated_sql,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

func SessionKey(userID, sessionID string) string {
	return fmt.Sprintf("session:%s:%s", userID, sessionID)
}

func GetHistory(ctx context.Context, rdb *redis.Client, userID, sessionID string) ([]SessionMessage, error) {
	if rdb == nil {
		return nil, nil
	}
	raw, err := rdb.Get(ctx, SessionKey(userID, sessionID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var hist []SessionMessage
	if err := json.Unmarshal(raw, &hist); err != nil {
		return nil, err
	}
	return hist, nil
}

func AppendMessage(ctx context.Context, rdb *redis.Client, userID, sessionID string, msg SessionMessage, maxMessages int, ttl time.Duration) error {
	if rdb == nil {
		return nil
	}
	if maxMessages <= 0 {
		maxMessages = 10
	}
	hist, err := GetHistory(ctx, rdb, userID, sessionID)
	if err != nil {
		return err
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}
	hist = append(hist, msg)
	if len(hist) > maxMessages {
		hist = hist[len(hist)-maxMessages:]
	}
	raw, err := json.Marshal(hist)
	if err != nil {
		return err
	}
	key := SessionKey(userID, sessionID)
	return rdb.Set(ctx, key, raw, ttl).Err()
}

func ClearSession(ctx context.Context, rdb *redis.Client, userID, sessionID string) error {
	if rdb == nil {
		return nil
	}
	return rdb.Del(ctx, SessionKey(userID, sessionID)).Err()
}
