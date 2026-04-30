package cache

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func newTestClient(t *testing.T) *redis.Client {
	t.Helper()
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not set")
	}
	c, err := NewClient(url)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

type testPayload struct {
	A string `json:"a"`
}

func TestRedis_SetGet(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	key := "test:key:setget"
	_ = Delete(ctx, c, key)

	if err := SetWithTTL(ctx, c, key, testPayload{A: "v"}, 10*time.Second); err != nil {
		t.Fatalf("SetWithTTL: %v", err)
	}

	val, found, err := Get(ctx, c, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatalf("expected found=true")
	}
	if val == "" {
		t.Fatalf("expected non-empty val")
	}
}

func TestRedis_GetMissing(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	_, found, err := Get(ctx, c, "test:key:missing:"+time.Now().Format(time.RFC3339Nano))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Fatalf("expected found=false")
	}
}

func TestRedis_TTLExpiry(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	key := "test:key:ttl"
	_ = Delete(ctx, c, key)

	if err := SetWithTTL(ctx, c, key, testPayload{A: "v"}, 1*time.Second); err != nil {
		t.Fatalf("SetWithTTL: %v", err)
	}
	time.Sleep(2 * time.Second)

	_, found, err := Get(ctx, c, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Fatalf("expected key to expire")
	}
}

func TestRedis_Delete(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	key := "test:key:delete"
	_ = Delete(ctx, c, key)
	if err := SetWithTTL(ctx, c, key, testPayload{A: "v"}, 10*time.Second); err != nil {
		t.Fatalf("SetWithTTL: %v", err)
	}
	if err := Delete(ctx, c, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, found, err := Get(ctx, c, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Fatalf("expected deleted key not found")
	}
}

