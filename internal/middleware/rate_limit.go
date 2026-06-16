package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	redisCache "github.com/monir/auth_service/internal/repository/redis"
	"github.com/monir/auth_service/pkg/response"
)

// RateLimit limits requests to maxReqs per window per IP for the given route key.
func RateLimit(cache *redisCache.TokenCache, maxReqs int64, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP() + ":" + c.FullPath()
		count, err := cache.IncrRateLimit(c.Request.Context(), key, window)
		if err != nil {
			// Fail open — don't block requests if Redis is unavailable
			c.Next()
			return
		}
		if count > maxReqs {
			response.TooManyRequests(c)
			c.Abort()
			return
		}
		c.Next()
	}
}
