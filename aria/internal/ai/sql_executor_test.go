package ai

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Deonkar/Aria/aria/internal/db"
)

func TestExecute_SimpleSelect(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL_READONLY")
	if dsn == "" {
		t.Skip("DATABASE_URL_READONLY not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	rows, _, err := Execute(ctx, pool, "SELECT id, first_name FROM leads LIMIT 3")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
}

func TestExecute_WithParam(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL_READONLY")
	if dsn == "" {
		t.Skip("DATABASE_URL_READONLY not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	var agentID string
	err = pool.QueryRow(ctx, "SELECT assigned_agent_id FROM leads LIMIT 1").Scan(&agentID)
	if err != nil {
		t.Fatal(err)
	}

	rows, _, err := Execute(ctx, pool, "SELECT id FROM leads WHERE assigned_agent_id = $1 LIMIT 5", agentID)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r["id"] == nil {
			t.Fatal("missing id")
		}
	}
}

func TestExecute_Timeout(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL_READONLY")
	if dsn == "" {
		t.Skip("DATABASE_URL_READONLY not set")
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	_, _, err = Execute(ctx, pool, "SELECT pg_sleep(10)")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if err != ErrQueryTimeout {
		t.Fatalf("expected ErrQueryTimeout, got %v", err)
	}
}

func TestExecute_InvalidSQL(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL_READONLY")
	if dsn == "" {
		t.Skip("DATABASE_URL_READONLY not set")
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	_, _, err = Execute(ctx, pool, "SELEKT 1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExecute_ZeroRows(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL_READONLY")
	if dsn == "" {
		t.Skip("DATABASE_URL_READONLY not set")
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	rows, _, err := Execute(ctx, pool, "SELECT id FROM leads WHERE id = '00000000-0000-0000-0000-000000000000'")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}
