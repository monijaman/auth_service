package resetpassword

import (
	"context"
	"errors"
	"time"

	"github.com/monir/auth_service/internal/domain/auth"
	"github.com/monir/auth_service/internal/domain/user"
	"github.com/monir/auth_service/internal/repository/postgres"
	"github.com/monir/auth_service/internal/service/event"
	"github.com/monir/auth_service/internal/service/password"
)

var (
	ErrInvalidOTP   = errors.New("invalid or expired reset code")
	ErrWeakPassword = errors.New("password must be at least 8 characters")
)

type Input struct {
	Email       string `json:"email"        validate:"required,email"`
	Code        string `json:"code"         validate:"required,len=6"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

type UseCase struct {
	userRepo  user.Repository
	otpRepo   auth.OTPRepository
	tokenRepo auth.TokenRepository
	pwdSvc    *password.Service
	eventPub  event.Publisher
}

func New(
	userRepo user.Repository,
	otpRepo auth.OTPRepository,
	tokenRepo auth.TokenRepository,
	pwdSvc *password.Service,
	eventPub event.Publisher,
) *UseCase {
	return &UseCase{
		userRepo:  userRepo,
		otpRepo:   otpRepo,
		tokenRepo: tokenRepo,
		pwdSvc:    pwdSvc,
		eventPub:  eventPub,
	}
}

func (uc *UseCase) Execute(ctx context.Context, in Input) error {
	u, err := uc.userRepo.FindByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrInvalidOTP
		}
		return err
	}

	otp, err := uc.otpRepo.FindOTP(ctx, u.ID, auth.OTPPasswordReset)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrInvalidOTP
		}
		return err
	}

	if otp.Code != in.Code || time.Now().After(otp.ExpiresAt) {
		return ErrInvalidOTP
	}

	hash, err := uc.pwdSvc.Hash(in.NewPassword)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	if err := uc.userRepo.Update(ctx, u); err != nil {
		return err
	}

	_ = uc.otpRepo.DeleteOTP(ctx, u.ID, auth.OTPPasswordReset)
	_ = uc.tokenRepo.RevokeAllUserTokens(ctx, u.ID)
	_ = uc.eventPub.Publish(ctx, event.Event{
		Type:    event.PasswordChanged,
		Payload: map[string]string{"user_id": u.ID.String()},
	})
	return nil
}
