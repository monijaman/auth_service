package login

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/monir/auth_service/internal/domain/auth"
	"github.com/monir/auth_service/internal/domain/site"
	"github.com/monir/auth_service/internal/domain/user"
	"github.com/monir/auth_service/internal/repository/postgres"
	"github.com/monir/auth_service/internal/service/event"
	jwtSvc "github.com/monir/auth_service/internal/service/jwt"
	"github.com/monir/auth_service/internal/service/password"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAccountInactive    = errors.New("account is not active")
	ErrSiteNotFound       = errors.New("site not found")
	ErrSiteRequired       = errors.New("site_id is required")
	ErrNoSiteAccess       = errors.New("user does not have access to this site")
)

type Input struct {
	Email    string    `json:"email"    validate:"required,email"`
	Password string    `json:"password" validate:"required"`
	SiteID   uuid.UUID `json:"site_id"  validate:"required"`
	DeviceID string    `json:"device_id"`
}

type Output struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	UserID       uuid.UUID `json:"user_id"`
	Email        string    `json:"email"`
	SiteID       uuid.UUID `json:"site_id"`
	Roles        []string  `json:"roles"`
}

type UseCase struct {
	userRepo  user.Repository
	siteRepo  site.Repository
	tokenRepo auth.TokenRepository
	pwdSvc    *password.Service
	jwtSvc    *jwtSvc.Service
	eventPub  event.Publisher
}

func New(
	userRepo user.Repository,
	siteRepo site.Repository,
	tokenRepo auth.TokenRepository,
	pwdSvc *password.Service,
	jwtSvc *jwtSvc.Service,
	eventPub event.Publisher,
) *UseCase {
	return &UseCase{
		userRepo:  userRepo,
		siteRepo:  siteRepo,
		tokenRepo: tokenRepo,
		pwdSvc:    pwdSvc,
		jwtSvc:    jwtSvc,
		eventPub:  eventPub,
	}
}

func (uc *UseCase) Execute(ctx context.Context, in Input) (*Output, error) {
	u, err := uc.userRepo.FindByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if u.Status != user.StatusActive {
		return nil, ErrAccountInactive
	}

	ok, err := uc.pwdSvc.Verify(in.Password, u.PasswordHash)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}

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

	// Get roles and permissions scoped to this site
	roles, err := uc.siteRepo.GetUserRoles(ctx, u.ID, targetSite.ID)
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, ErrNoSiteAccess
	}

	perms, err := uc.siteRepo.GetUserPermissions(ctx, u.ID, targetSite.ID)
	if err != nil {
		return nil, err
	}

	accessToken, err := uc.jwtSvc.GenerateAccessToken(u.ID, u.Email, targetSite.ID, roles, perms)
	if err != nil {
		return nil, err
	}

	rawRefresh, err := uc.jwtSvc.GenerateRefreshToken(u.ID)
	if err != nil {
		return nil, err
	}

	tokenHash := hashToken(rawRefresh)
	if err := uc.tokenRepo.SaveRefreshToken(ctx, &auth.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		SiteID:    targetSite.ID,
		TokenHash: tokenHash,
		DeviceID:  in.DeviceID,
		ExpiresAt: time.Now().Add(uc.jwtSvc.RefreshExpiry()),
		Revoked:   false,
		CreatedAt: time.Now(),
	}); err != nil {
		return nil, err
	}

	_ = uc.eventPub.Publish(ctx, event.Event{
		Type:      event.UserLoggedIn,
		Payload:   map[string]string{"user_id": u.ID.String(), "site_id": targetSite.ID.String()},
		OccuredAt: time.Now(),
	})

	return &Output{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		UserID:       u.ID,
		Email:        u.Email,
		SiteID:       targetSite.ID,
		Roles:        roles,
	}, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}
