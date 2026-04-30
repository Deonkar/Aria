package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Deonkar/Aria/aria/internal/models"
)

func TestUserRepo_FindByGoogleID_NotFound(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	repo := NewUserRepo(pool)
	u, err := repo.FindByGoogleID(ctx, "non-existent-google-id")
	if err != nil {
		t.Fatalf("FindByGoogleID: %v", err)
	}
	if u != nil {
		t.Fatalf("expected nil user")
	}
}

func TestUserRepo_Create_And_Find(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	repo := NewUserRepo(pool)
	googleID := "test-google-id-userrepo"
	email := "test-userrepo@example.com"

	// best-effort cleanup
	_, _ = pool.Exec(ctx, "DELETE FROM users WHERE google_id=$1 OR email=$2", googleID, email)

	user := &models.User{
		GoogleID:  googleID,
		Email:     email,
		FullName:  "Test UserRepo",
		Role:      "agent",
		Timezone:  "Asia/Kolkata",
		IsActive:  true,
		AvatarURL: nil,
	}

	created, err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected ID set")
	}

	found, err := repo.FindByGoogleID(ctx, googleID)
	if err != nil {
		t.Fatalf("FindByGoogleID: %v", err)
	}
	if found == nil || found.Email != email {
		t.Fatalf("expected to find created user")
	}

	if err := repo.UpdateLastLogin(ctx, created.ID); err != nil {
		t.Fatalf("UpdateLastLogin: %v", err)
	}
}

