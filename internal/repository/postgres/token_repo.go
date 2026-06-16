package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/monir/auth_service/internal/domain/auth"
)

type TokenRepo struct {
	db *pgxpool.Pool
}

func NewTokenRepo(db *pgxpool.Pool) *TokenRepo {
	return &TokenRepo{db: db}
}

func (r *TokenRepo) SaveRefreshToken(ctx context.Context, t *auth.RefreshToken) error {
	q := `INSERT INTO refresh_tokens(id, user_id, token_hash, device_id, expires_at, revoked, created_at)
	      VALUES($1,$2,$3,$4,$5,$6,$7)`
	_, err := r.db.Exec(ctx, q, t.ID, t.UserID, t.TokenHash, t.DeviceID, t.ExpiresAt, t.Revoked, t.CreatedAt)
	return err
}

func (r *TokenRepo) FindRefreshToken(ctx context.Context, tokenHash string) (*auth.RefreshToken, error) {
	q := `SELECT id, user_id, token_hash, device_id, expires_at, revoked, created_at
	      FROM refresh_tokens WHERE token_hash=$1`
	t := &auth.RefreshToken{}
	err := r.db.QueryRow(ctx, q, tokenHash).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.DeviceID, &t.ExpiresAt, &t.Revoked, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return t, nil
}

func (r *TokenRepo) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked=true WHERE token_hash=$1`, tokenHash)
	return err
}

func (r *TokenRepo) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked=true WHERE user_id=$1`, userID)
	return err
}

func (r *TokenRepo) DeleteExpiredTokens(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE expires_at < $1`, time.Now())
	return err
}

// OTP operations

func (r *TokenRepo) SaveOTP(ctx context.Context, otp *auth.OTPCode) error {
	// Upsert — replace any existing OTP of the same type for this user
	q := `INSERT INTO otp_codes(id, user_id, code, type, expires_at)
	      VALUES($1,$2,$3,$4,$5)
	      ON CONFLICT (user_id, type) DO UPDATE
	      SET id=$1, code=$3, expires_at=$5`
	_, err := r.db.Exec(ctx, q, otp.ID, otp.UserID, otp.Code, string(otp.Type), otp.ExpiresAt)
	return err
}

func (r *TokenRepo) FindOTP(ctx context.Context, userID uuid.UUID, otpType auth.OTPType) (*auth.OTPCode, error) {
	q := `SELECT id, user_id, code, type, expires_at FROM otp_codes WHERE user_id=$1 AND type=$2`
	o := &auth.OTPCode{}
	var t string
	err := r.db.QueryRow(ctx, q, userID, string(otpType)).Scan(&o.ID, &o.UserID, &o.Code, &t, &o.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	o.Type = auth.OTPType(t)
	return o, nil
}

func (r *TokenRepo) DeleteOTP(ctx context.Context, userID uuid.UUID, otpType auth.OTPType) error {
	_, err := r.db.Exec(ctx, `DELETE FROM otp_codes WHERE user_id=$1 AND type=$2`, userID, string(otpType))
	return err
}
