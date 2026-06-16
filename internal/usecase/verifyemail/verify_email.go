package verifyemail

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/monir/auth_service/internal/domain/auth"
	"github.com/monir/auth_service/internal/domain/user"
	"github.com/monir/auth_service/internal/repository/postgres"
	"github.com/monir/auth_service/internal/service/email"
	"github.com/monir/auth_service/internal/service/event"
)

var (
	ErrInvalidOTP  = errors.New("invalid or expired verification code")
	ErrAlreadyVerified = errors.New("email already verified")
)

const otpTTL = 15 * time.Minute

type SendInput struct {
	UserID uuid.UUID
}

type VerifyInput struct {
	UserID uuid.UUID
	Code   string `json:"code" validate:"required,len=6"`
}

type UseCase struct {
	userRepo  user.Repository
	otpRepo   auth.OTPRepository
	emailSvc  *email.Service
	eventPub  event.Publisher
}

func New(userRepo user.Repository, otpRepo auth.OTPRepository, emailSvc *email.Service, eventPub event.Publisher) *UseCase {
	return &UseCase{userRepo: userRepo, otpRepo: otpRepo, emailSvc: emailSvc, eventPub: eventPub}
}

func (uc *UseCase) SendCode(ctx context.Context, in SendInput) error {
	u, err := uc.userRepo.FindByID(ctx, in.UserID)
	if err != nil {
		return err
	}
	if u.EmailVerified {
		return ErrAlreadyVerified
	}

	code := generateOTP()
	otp := &auth.OTPCode{
		ID:        uuid.New(),
		UserID:    u.ID,
		Code:      code,
		Type:      auth.OTPEmailVerification,
		ExpiresAt: time.Now().Add(otpTTL),
	}
	if err := uc.otpRepo.SaveOTP(ctx, otp); err != nil {
		return err
	}
	return uc.emailSvc.SendVerificationEmail(ctx, u.Email, code)
}

func (uc *UseCase) Verify(ctx context.Context, in VerifyInput) error {
	otp, err := uc.otpRepo.FindOTP(ctx, in.UserID, auth.OTPEmailVerification)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrInvalidOTP
		}
		return err
	}
	if otp.Code != in.Code || time.Now().After(otp.ExpiresAt) {
		return ErrInvalidOTP
	}

	u, err := uc.userRepo.FindByID(ctx, in.UserID)
	if err != nil {
		return err
	}
	u.EmailVerified = true
	if err := uc.userRepo.Update(ctx, u); err != nil {
		return err
	}

	_ = uc.otpRepo.DeleteOTP(ctx, in.UserID, auth.OTPEmailVerification)
	_ = uc.eventPub.Publish(ctx, event.Event{
		Type:    event.EmailVerified,
		Payload: map[string]string{"user_id": in.UserID.String()},
	})
	return nil
}

func generateOTP() string {
	b := make([]byte, 3)
	rand.Read(b)
	n := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1000000
	return fmt.Sprintf("%06d", n)
}
