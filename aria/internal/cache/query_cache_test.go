package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestCacheKey_Deterministic(t *testing.T) {
	a := CacheKey("u1", "hello")
	b := CacheKey("u1", "hello")
	if a != b {
		t.Fatalf("keys differ")
	}
}

func TestCacheKey_CaseInsensitive(t *testing.T) {
	a := CacheKey("u1", "My Leads")
	b := CacheKey("u1", "my leads")
	if a != b {
		t.Fatalf("keys differ %q vs %q", a, b)
	}
}

func TestGetSet_RoundTrip(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	cr := &CachedResult{Answer: "a", SQL: "s", RowCount: 2, GeneratedAt: now}
	if err := SetCachedQuery(ctx, rdb, "u1", "q", cr, time.Minute); err != nil {
		t.Fatal(err)
	}
	got, ok, err := GetCachedQuery(ctx, rdb, "u1", "q")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected hit")
	}
	if got.Answer != "a" || got.SQL != "s" || got.RowCount != 2 {
		t.Fatalf("got %+v", got)
	}
}

func TestGet_Miss(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	ctx := context.Background()
	_, ok, err := GetCachedQuery(ctx, rdb, "u1", "nope")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected miss")
	}
}

func TestInvalidate(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	ctx := context.Background()
	_ = SetCachedQuery(ctx, rdb, "u1", "q", &CachedResult{Answer: "x"}, time.Minute)
	if err := InvalidateCachedQuery(ctx, rdb, "u1", "q"); err != nil {
		t.Fatal(err)
	}
	_, ok, err := GetCachedQuery(ctx, rdb, "u1", "q")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected miss after invalidate")
	}
}

func TestCacheTTL(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	ctx := context.Background()
	_ = SetCachedQuery(ctx, rdb, "u1", "q", &CachedResult{Answer: "x"}, time.Second)
	s.FastForward(2 * time.Second)
	_, ok, err := GetCachedQuery(ctx, rdb, "u1", "q")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected expiry")
	}
}
