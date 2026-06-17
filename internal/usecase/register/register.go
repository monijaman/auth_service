package register

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/monir/auth_service/internal/domain/role"
	"github.com/monir/auth_service/internal/domain/site"
	"github.com/monir/auth_service/internal/domain/user"
	"github.com/monir/auth_service/internal/repository/postgres"
	"github.com/monir/auth_service/internal/service/event"
	"github.com/monir/auth_service/internal/service/password"
)

var (
	ErrEmailTaken   = errors.New("email already registered")
	ErrSiteNotFound = errors.New("site not found")
	ErrSiteRequired = errors.New("site_id is required")
	ErrRoleNotFound = errors.New("role not found")
)

type Input struct {
	Email    string    `json:"email"    validate:"required,email"`
	Phone    string    `json:"phone"    validate:"omitempty,e164"`
	Password string    `json:"password" validate:"required,min=8"`
	SiteID   uuid.UUID `json:"site_id"  validate:"required"`
	Role     string    `json:"role"     validate:"required"`
}

type Output struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	SiteID uuid.UUID `json:"site_id"`
	Role   string    `json:"role"`
}

type UseCase struct {
	userRepo user.Repository
	roleRepo role.Repository
	siteRepo site.Repository
	pwdSvc   *password.Service
	eventPub event.Publisher
}

func New(userRepo user.Repository, roleRepo role.Repository, siteRepo site.Repository, pwdSvc *password.Service, eventPub event.Publisher) *UseCase {
	return &UseCase{userRepo: userRepo, roleRepo: roleRepo, siteRepo: siteRepo, pwdSvc: pwdSvc, eventPub: eventPub}
}

func (uc *UseCase) Execute(ctx context.Context, in Input) (*Output, error) {
	if in.SiteID == uuid.Nil {
		return nil, ErrSiteRequired
	}
	targetSite, err := uc.siteRepo.FindByID(ctx, in.SiteID)
	if errors.Is(err, postgres.ErrNotFound) {
		return nil, ErrSiteNotFound
	}
	if err != nil {
		return nil, err
	}

	hash, err := uc.pwdSvc.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	u := &user.User{
		ID:            uuid.New(),
		Email:         in.Email,
		Phone:         in.Phone,
		PasswordHash:  hash,
		EmailVerified: false,
		Status:        user.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := uc.userRepo.Create(ctx, u); err != nil {
		if errors.Is(err, postgres.ErrDuplicate) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	r, err := uc.roleRepo.FindByName(ctx, in.Role)
	if err != nil {
		return nil, ErrRoleNotFound
	}
	if err := uc.siteRepo.AssignUserRole(ctx, u.ID, targetSite.ID, r.ID); err != nil {
		return nil, err
	}

	_ = uc.eventPub.Publish(ctx, event.Event{
		Type:      event.UserRegistered,
		Payload:   map[string]string{"user_id": u.ID.String(), "email": u.Email, "site_id": targetSite.ID.String()},
		OccuredAt: now,
	})

	return &Output{UserID: u.ID, Email: u.Email, SiteID: targetSite.ID, Role: in.Role}, nil
}
