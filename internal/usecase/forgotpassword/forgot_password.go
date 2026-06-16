package forgotpassword

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/monir/auth_service/internal/domain/auth"
	"github.com/monir/auth_service/internal/domain/user"
	"github.com/monir/auth_service/internal/service/email"
)

const otpTTL = 15 * time.Minute

type Input struct {
	Email string `json:"email" validate:"required,email"`
}

type UseCase struct {
	userRepo user.Repository
	otpRepo  auth.OTPRepository
	emailSvc *email.Service
}

func New(userRepo user.Repository, otpRepo auth.OTPRepository, emailSvc *email.Service) *UseCase {
	return &UseCase{userRepo: userRepo, otpRepo: otpRepo, emailSvc: emailSvc}
}

// Execute always returns nil to prevent email enumeration.
func (uc *UseCase) Execute(ctx context.Context, in Input) error {
	u, err := uc.userRepo.FindByEmail(ctx, in.Email)
	if err != nil {
		return nil // silently ignore unknown emails
	}

	code := generateOTP()
	otp := &auth.OTPCode{
		ID:        uuid.New(),
		UserID:    u.ID,
		Code:      code,
		Type:      auth.OTPPasswordReset,
		ExpiresAt: time.Now().Add(otpTTL),
	}
	if err := uc.otpRepo.SaveOTP(ctx, otp); err != nil {
		return err
	}
	_ = uc.emailSvc.SendPasswordResetEmail(ctx, u.Email, code)
	return nil
}

func generateOTP() string {
	b := make([]byte, 3)
	rand.Read(b)
	n := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1000000
	return fmt.Sprintf("%06d", n)
}
