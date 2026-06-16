package permission

import (
	"context"

	"github.com/google/uuid"
)

type Permission struct {
	ID   uuid.UUID
	Name string
}

type Repository interface {
	Create(ctx context.Context, p *Permission) error
	FindByID(ctx context.Context, id uuid.UUID) (*Permission, error)
	FindByName(ctx context.Context, name string) (*Permission, error)
	List(ctx context.Context) ([]*Permission, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
