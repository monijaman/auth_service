package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OTPType string

const (
	OTPEmailVerification OTPType = "email_verification"
	OTPPasswordReset     OTPType = "password_reset"
)

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	DeviceID  string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

type OTPCode struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Code      string
	Type      OTPType
	ExpiresAt time.Time
}

type TokenRepository interface {
	SaveRefreshToken(ctx context.Context, t *RefreshToken) error
	FindRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error
	DeleteExpiredTokens(ctx context.Context) error
}

type OTPRepository interface {
	SaveOTP(ctx context.Context, otp *OTPCode) error
	FindOTP(ctx context.Context, userID uuid.UUID, otpType OTPType) (*OTPCode, error)
	DeleteOTP(ctx context.Context, userID uuid.UUID, otpType OTPType) error
}
