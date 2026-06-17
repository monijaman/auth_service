package site

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Site struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Domain    string
	IsActive  bool
	CreatedAt time.Time
}

type Repository interface {
	Create(ctx context.Context, s *Site) error
	FindByID(ctx context.Context, id uuid.UUID) (*Site, error)
	FindBySlug(ctx context.Context, slug string) (*Site, error)
	List(ctx context.Context) ([]*Site, error)
	Update(ctx context.Context, s *Site) error
	Delete(ctx context.Context, id uuid.UUID) error
	AssignUserRole(ctx context.Context, userID, siteID, roleID uuid.UUID) error
	RemoveUserRole(ctx context.Context, userID, siteID, roleID uuid.UUID) error
	GetUserRoles(ctx context.Context, userID, siteID uuid.UUID) ([]string, error)
	GetUserPermissions(ctx context.Context, userID, siteID uuid.UUID) ([]string, error)
	GetUserSites(ctx context.Context, userID uuid.UUID) ([]*Site, error)
	GetSiteUsers(ctx context.Context, siteID uuid.UUID) ([]*SiteUser, error)
}

type SiteUser struct {
	UserID uuid.UUID
	Email  string
	Roles  []string
}
