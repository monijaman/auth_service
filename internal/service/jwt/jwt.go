package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken  = errors.New("token expired")
)

type Claims struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	SiteID      uuid.UUID `json:"site_id"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
	jwt.RegisteredClaims
}

type Service struct {
	accessSecret  []byte
	refreshSecret []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func New(accessSecret, refreshSecret string, accessExpMinutes, refreshExpDays int) *Service {
	return &Service{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessExpiry:  time.Duration(accessExpMinutes) * time.Minute,
		refreshExpiry: time.Duration(refreshExpDays) * 24 * time.Hour,
	}
}

func (s *Service) GenerateAccessToken(userID uuid.UUID, email string, siteID uuid.UUID, roles, permissions []string) (string, error) {
	claims := &Claims{
		UserID:      userID,
		Email:       email,
		SiteID:      siteID,
		Roles:       roles,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessExpiry)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.accessSecret)
}

func (s *Service) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	claims := &jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshExpiry)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.refreshSecret)
}

func (s *Service) ValidateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.accessSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (s *Service) ValidateRefreshToken(tokenStr string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.refreshSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return uuid.Nil, ErrExpiredToken
		}
		return uuid.Nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return uuid.Nil, ErrInvalidToken
	}
	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	return id, nil
}

func (s *Service) RefreshExpiry() time.Duration {
	return s.refreshExpiry
}
