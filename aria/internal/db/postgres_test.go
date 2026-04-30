package db

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewPool_Connect(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPool error: %v", err)
	}
	pool.Close()
}

func TestReadOnlyPool_NoWrite(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL_READONLY")
	if dsn == "" {
		t.Skip("DATABASE_URL_READONLY not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPool error: %v", err)
	}
	defer pool.Close()

	_, err = pool.Exec(ctx, "INSERT INTO teams(name, department) VALUES('x','pre_sales')")
	if err == nil {
		t.Fatalf("expected permission error for readonly role")
	}
}

