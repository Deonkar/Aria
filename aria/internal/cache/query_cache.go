package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type CachedResult struct {
	Answer     string    `json:"answer"`
	SQL        string    `json:"sql"`
	RowCount   int       `json:"row_count"`
	GeneratedAt time.Time `json:"generated_at"`
}

var spaceCollapse = regexp.MustCompile(`\s+`)

func NormaliseQuestion(question string) string {
	s := strings.ToLower(strings.TrimSpace(question))
	s = spaceCollapse.ReplaceAllString(s, " ")
	return s
}

func CacheKey(userID, question string) string {
	h := sha256.Sum256([]byte(NormaliseQuestion(question)))
	return fmt.Sprintf("query_cache:%s:%s", userID, hex.EncodeToString(h[:]))
}

func GetCachedQuery(ctx context.Context, rdb *redis.Client, userID, question string) (*CachedResult, bool, error) {
	if rdb == nil {
		return nil, false, nil
	}
	key := CacheKey(userID, question)
	raw, err := rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var c CachedResult
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, false, err
	}
	return &c, true, nil
}

func SetCachedQuery(ctx context.Context, rdb *redis.Client, userID, question string, result *CachedResult, ttl time.Duration) error {
	if rdb == nil || result == nil {
		return nil
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	key := CacheKey(userID, question)
	return rdb.Set(ctx, key, raw, ttl).Err()
}

func InvalidateCachedQuery(ctx context.Context, rdb *redis.Client, userID, question string) error {
	if rdb == nil {
		return nil
	}
	key := CacheKey(userID, question)
	return rdb.Del(ctx, key).Err()
}
