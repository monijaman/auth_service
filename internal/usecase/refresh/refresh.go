package refresh

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
	jwtSvc "github.com/monir/auth_service/internal/service/jwt"
)

var (
	ErrInvalidToken = errors.New("invalid or expired refresh token")
	ErrTokenRevoked = errors.New("refresh token has been revoked")
)

type Input struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type Output struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type UseCase struct {
	userRepo  user.Repository
	siteRepo  site.Repository
	tokenRepo auth.TokenRepository
	jwtSvc    *jwtSvc.Service
}

func New(userRepo user.Repository, siteRepo site.Repository, tokenRepo auth.TokenRepository, jwtSvc *jwtSvc.Service) *UseCase {
	return &UseCase{userRepo: userRepo, siteRepo: siteRepo, tokenRepo: tokenRepo, jwtSvc: jwtSvc}
}

func (uc *UseCase) Execute(ctx context.Context, in Input) (*Output, error) {
	userID, err := uc.jwtSvc.ValidateRefreshToken(in.RefreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	tokenHash := hashToken(in.RefreshToken)
	stored, err := uc.tokenRepo.FindRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	if stored.Revoked || time.Now().After(stored.ExpiresAt) {
		return nil, ErrTokenRevoked
	}

	if err := uc.tokenRepo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return nil, err
	}

	u, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Use site_id preserved from the original token
	siteID := stored.SiteID
	roles, err := uc.siteRepo.GetUserRoles(ctx, u.ID, siteID)
	if err != nil {
		return nil, err
	}
	perms, err := uc.siteRepo.GetUserPermissions(ctx, u.ID, siteID)
	if err != nil {
		return nil, err
	}

	newAccess, err := uc.jwtSvc.GenerateAccessToken(u.ID, u.Email, siteID, roles, perms)
	if err != nil {
		return nil, err
	}

	newRefresh, err := uc.jwtSvc.GenerateRefreshToken(u.ID)
	if err != nil {
		return nil, err
	}

	if err := uc.tokenRepo.SaveRefreshToken(ctx, &auth.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		SiteID:    siteID,
		TokenHash: hashToken(newRefresh),
		DeviceID:  stored.DeviceID,
		ExpiresAt: time.Now().Add(uc.jwtSvc.RefreshExpiry()),
		CreatedAt: time.Now(),
	}); err != nil {
		return nil, err
	}

	return &Output{AccessToken: newAccess, RefreshToken: newRefresh}, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}
