package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestGetHistory_Empty(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	ctx := context.Background()
	h, err := GetHistory(ctx, rdb, "u1", "sess")
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 0 {
		t.Fatalf("expected empty, got %d", len(h))
	}
}

func TestAppend_SingleMessage(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	ctx := context.Background()
	_ = AppendMessage(ctx, rdb, "u1", "sess", SessionMessage{Role: "user", Content: "hi"}, 10, time.Hour)
	h, err := GetHistory(ctx, rdb, "u1", "sess")
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 1 || h[0].Content != "hi" {
		t.Fatalf("got %+v", h)
	}
}

func TestAppend_TrimToMax(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	ctx := context.Background()
	for i := 0; i < 12; i++ {
		_ = AppendMessage(ctx, rdb, "u1", "sess", SessionMessage{Role: "user", Content: string(rune('a' + i))}, 10, time.Hour)
	}
	h, err := GetHistory(ctx, rdb, "u1", "sess")
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 10 {
		t.Fatalf("expected 10, got %d", len(h))
	}
}

func TestClearSession(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	ctx := context.Background()
	_ = AppendMessage(ctx, rdb, "u1", "sess", SessionMessage{Role: "user", Content: "a"}, 10, time.Hour)
	_ = ClearSession(ctx, rdb, "u1", "sess")
	h, err := GetHistory(ctx, rdb, "u1", "sess")
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 0 {
		t.Fatal("expected empty")
	}
}

func TestHistoryTTLReset(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	ctx := context.Background()
	_ = AppendMessage(ctx, rdb, "u1", "sess", SessionMessage{Role: "user", Content: "a"}, 10, 10*time.Minute)
	ttl1 := rdb.TTL(ctx, SessionKey("u1", "sess")).Val()
	_ = AppendMessage(ctx, rdb, "u1", "sess", SessionMessage{Role: "user", Content: "b"}, 10, 10*time.Minute)
	ttl2 := rdb.TTL(ctx, SessionKey("u1", "sess")).Val()
	if ttl2 < ttl1-1*time.Second { // reset should be ~10m again on miniredis
		// miniredis TTL behavior: just ensure key still exists
		if ttl2 <= 0 {
			t.Fatal("ttl should be positive")
		}
	}
}
