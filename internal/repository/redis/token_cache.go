package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyBlacklist = "blacklist:%s"
	keyOTP       = "otp:%s:%s"
	keyRateLimit = "ratelimit:%s"
	keySession   = "session:%s"
)

type TokenCache struct {
	client *redis.Client
}

func NewTokenCache(client *redis.Client) *TokenCache {
	return &TokenCache{client: client}
}

// BlacklistToken marks an access token as revoked until its expiry.
func (c *TokenCache) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	return c.client.Set(ctx, fmt.Sprintf(keyBlacklist, jti), 1, ttl).Err()
}

// IsBlacklisted returns true if the token has been revoked.
func (c *TokenCache) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	n, err := c.client.Exists(ctx, fmt.Sprintf(keyBlacklist, jti)).Result()
	return n > 0, err
}

// SetOTP caches an OTP code.
func (c *TokenCache) SetOTP(ctx context.Context, userID, otpType, code string, ttl time.Duration) error {
	return c.client.Set(ctx, fmt.Sprintf(keyOTP, userID, otpType), code, ttl).Err()
}

// GetOTP retrieves a cached OTP code.
func (c *TokenCache) GetOTP(ctx context.Context, userID, otpType string) (string, error) {
	return c.client.Get(ctx, fmt.Sprintf(keyOTP, userID, otpType)).Result()
}

// DeleteOTP removes a cached OTP.
func (c *TokenCache) DeleteOTP(ctx context.Context, userID, otpType string) error {
	return c.client.Del(ctx, fmt.Sprintf(keyOTP, userID, otpType)).Err()
}

// IncrRateLimit increments the request counter for a key and sets TTL on first call.
// Returns current count and any error.
func (c *TokenCache) IncrRateLimit(ctx context.Context, key string, window time.Duration) (int64, error) {
	pipe := c.client.Pipeline()
	incr := pipe.Incr(ctx, fmt.Sprintf(keyRateLimit, key))
	pipe.Expire(ctx, fmt.Sprintf(keyRateLimit, key), window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}
