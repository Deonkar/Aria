package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/Deonkar/Aria/aria/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) FindByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	const q = `
SELECT id, google_id, email, full_name, avatar_url, role, is_active, department, team_id, manager_id, timezone, last_login_at, created_at, updated_at
FROM users
WHERE google_id = $1
LIMIT 1`

	var u models.User
	err := r.pool.QueryRow(ctx, q, googleID).Scan(
		&u.ID, &u.GoogleID, &u.Email, &u.FullName, &u.AvatarURL, &u.Role, &u.IsActive,
		&u.Department, &u.TeamID, &u.ManagerID, &u.Timezone, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == nil {
		return &u, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return nil, fmt.Errorf("find user by google_id: %w", err)
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	const q = `
SELECT id, google_id, email, full_name, avatar_url, role, is_active, department, team_id, manager_id, timezone, last_login_at, created_at, updated_at
FROM users
WHERE id = $1
LIMIT 1`

	var u models.User
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.GoogleID, &u.Email, &u.FullName, &u.AvatarURL, &u.Role, &u.IsActive,
		&u.Department, &u.TeamID, &u.ManagerID, &u.Timezone, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == nil {
		return &u, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return nil, fmt.Errorf("find user by id: %w", err)
}

func (r *UserRepo) Create(ctx context.Context, user *models.User) (*models.User, error) {
	const q = `
INSERT INTO users (google_id, email, full_name, avatar_url, role, department, team_id, manager_id, timezone, is_active)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,TRUE)
RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q,
		user.GoogleID, user.Email, user.FullName, user.AvatarURL, user.Role,
		user.Department, user.TeamID, user.ManagerID, user.Timezone,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (r *UserRepo) UpdateLastLogin(ctx context.Context, id string) error {
	const q = `UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`
	ct, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("update last_login: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("update last_login: user not found")
	}
	return nil
}

