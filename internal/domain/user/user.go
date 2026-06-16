package user

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusInactive  Status = "inactive"
	StatusSuspended Status = "suspended"
)

type User struct {
	ID            uuid.UUID
	Email         string
	Phone         string
	PasswordHash  string
	EmailVerified bool
	Status        Status
	DeletedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Repository interface {
	Create(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, u *User) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	// RBAC helpers
	AssignRole(ctx context.Context, userID, roleID uuid.UUID) error
	RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error
	GetRoles(ctx context.Context, userID uuid.UUID) ([]string, error)
	GetPermissions(ctx context.Context, userID uuid.UUID) ([]string, error)
}
