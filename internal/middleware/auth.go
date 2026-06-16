package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/monir/auth_service/internal/service/jwt"
	"github.com/monir/auth_service/pkg/response"
)

const claimsKey = "claims"

func Auth(jwtSvc *jwt.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			response.Unauthorized(c, "missing authorization header")
			c.Abort()
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := jwtSvc.ValidateAccessToken(token)
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set(claimsKey, claims)
		c.Next()
	}
}

func RequirePermission(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := c.Get(claimsKey)
		if !ok {
			response.Unauthorized(c, "not authenticated")
			c.Abort()
			return
		}
		jwtClaims, ok := claims.(*jwt.Claims)
		if !ok {
			response.Unauthorized(c, "invalid claims")
			c.Abort()
			return
		}
		for _, p := range jwtClaims.Permissions {
			if p == perm {
				c.Next()
				return
			}
		}
		response.Forbidden(c, "insufficient permissions")
		c.Abort()
	}
}

func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := c.Get(claimsKey)
		if !ok {
			response.Unauthorized(c, "not authenticated")
			c.Abort()
			return
		}
		jwtClaims, ok := claims.(*jwt.Claims)
		if !ok {
			response.Unauthorized(c, "invalid claims")
			c.Abort()
			return
		}
		for _, r := range jwtClaims.Roles {
			if r == role {
				c.Next()
				return
			}
		}
		response.Forbidden(c, "insufficient role")
		c.Abort()
	}
}

func ClaimsFromContext(c *gin.Context) *jwt.Claims {
	v, _ := c.Get(claimsKey)
	claims, _ := v.(*jwt.Claims)
	return claims
}
