package role

import (
	"context"

	"github.com/google/uuid"
)

type Role struct {
	ID   uuid.UUID
	Name string
}

type Repository interface {
	Create(ctx context.Context, r *Role) error
	FindByID(ctx context.Context, id uuid.UUID) (*Role, error)
	FindByName(ctx context.Context, name string) (*Role, error)
	List(ctx context.Context) ([]*Role, error)
	Delete(ctx context.Context, id uuid.UUID) error
	AssignPermission(ctx context.Context, roleID, permissionID uuid.UUID) error
	RemovePermission(ctx context.Context, roleID, permissionID uuid.UUID) error
}
