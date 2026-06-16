package logout

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/google/uuid"
	"github.com/monir/auth_service/internal/domain/auth"
	"github.com/monir/auth_service/internal/service/event"
	"github.com/monir/auth_service/internal/service/jwt"
)

type Input struct {
	RefreshToken string `json:"refresh_token"`
	UserID       uuid.UUID
}

type UseCase struct {
	tokenRepo auth.TokenRepository
	jwtSvc    *jwt.Service
	eventPub  event.Publisher
}

func New(tokenRepo auth.TokenRepository, jwtSvc *jwt.Service, eventPub event.Publisher) *UseCase {
	return &UseCase{tokenRepo: tokenRepo, jwtSvc: jwtSvc, eventPub: eventPub}
}

func (uc *UseCase) Execute(ctx context.Context, in Input) error {
	if in.RefreshToken != "" {
		hash := hashToken(in.RefreshToken)
		_ = uc.tokenRepo.RevokeRefreshToken(ctx, hash)
	}

	_ = uc.eventPub.Publish(ctx, event.Event{
		Type:    event.UserLoggedOut,
		Payload: map[string]string{"user_id": in.UserID.String()},
	})
	return nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}
