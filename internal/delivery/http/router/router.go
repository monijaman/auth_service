package router

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/monir/auth_service/internal/delivery/http/handler"
	mw "github.com/monir/auth_service/internal/middleware"
	redisCache "github.com/monir/auth_service/internal/repository/redis"
	"github.com/monir/auth_service/internal/service/jwt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Handlers struct {
	Auth *handler.AuthHandler
	User *handler.UserHandler
	Site *handler.SiteHandler
	Role *handler.RoleHandler
}

func New(h Handlers, jwtSvc *jwt.Service, cache *redisCache.TokenCache) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger())
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:8081"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
	}))

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Metrics
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	authMW := mw.Auth(jwtSvc)
	strictLimit := mw.RateLimit(cache, 10, time.Minute)   // 10 req/min for auth endpoints
	normalLimit := mw.RateLimit(cache, 100, time.Minute)  // 100 req/min for user endpoints

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		auth.Use(strictLimit)
		{
			auth.POST("/register", h.Auth.Register)
			auth.POST("/login", h.Auth.Login)
			auth.POST("/refresh", h.Auth.Refresh)
			auth.POST("/forgot-password", h.Auth.ForgotPassword)
			auth.POST("/reset-password", h.Auth.ResetPassword)

			// Authenticated auth endpoints
			authProtected := auth.Group("", authMW)
			{
				authProtected.POST("/logout", h.Auth.Logout)
				authProtected.POST("/verify-email", h.Auth.VerifyEmail)
				authProtected.POST("/resend-verification", h.Auth.ResendVerification)
			}
		}

		users := v1.Group("/users", authMW, normalLimit)
		{
			users.GET("/me", h.User.Me)
			users.GET("/:id", h.User.GetByID)
			users.PUT("/:id", h.User.Update)
			users.DELETE("/:id", h.User.Delete)
		}

		// Roles & permissions
		roles := v1.Group("/roles", authMW, normalLimit)
		{
			roles.GET("",                                        h.Role.List)
			roles.GET("/permissions",                            h.Role.ListPermissions)
			roles.GET("/:id",                                    h.Role.GetByID)
			roles.POST("",              mw.RequireRole("admin"), h.Role.Create)
			roles.DELETE("/:id",        mw.RequireRole("admin"), h.Role.Delete)
			roles.POST("/:id/permissions",                       mw.RequireRole("admin"), h.Role.AssignPermission)
			roles.DELETE("/:id/permissions/:permission",         mw.RequireRole("admin"), h.Role.RemovePermission)
		}

		// Site management — admin only
		sites := v1.Group("/sites", authMW, normalLimit)
		{
			sites.GET("",                                  h.Site.List)
			sites.GET("/:id",                             h.Site.GetByID)
			sites.POST("",          mw.RequireRole("admin"), h.Site.Create)
			sites.PUT("/:id",       mw.RequireRole("admin"), h.Site.Update)
			sites.DELETE("/:id",    mw.RequireRole("admin"), h.Site.Delete)
			sites.GET("/:id/users", mw.RequireRole("admin"), h.Site.ListUsers)
			sites.POST("/:id/users/:user_id/roles",          mw.RequireRole("admin"), h.Site.AssignUserRole)
			sites.DELETE("/:id/users/:user_id/roles/:role",  mw.RequireRole("admin"), h.Site.RemoveUserRole)
		}
	}

	return r
}

func requestLogger() gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health", "/metrics"},
	})
}
