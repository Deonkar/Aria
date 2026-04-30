package models

import "time"

type User struct {
	ID            string
	GoogleID      string
	Email         string
	FullName      string
	AvatarURL     *string
	Role          string
	IsActive      bool
	Department    *string
	TeamID        *string
	ManagerID     *string
	Timezone      string
	LastLoginAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

