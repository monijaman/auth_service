package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/monir/auth_service/internal/config"
	grpcDelivery "github.com/monir/auth_service/internal/delivery/grpc"
	"github.com/monir/auth_service/internal/delivery/http/handler"
	"github.com/monir/auth_service/internal/delivery/http/router"
	"github.com/monir/auth_service/internal/observability"
	postgresRepo "github.com/monir/auth_service/internal/repository/postgres"
	redisRepo "github.com/monir/auth_service/internal/repository/redis"
	"github.com/monir/auth_service/internal/service/email"
	"github.com/monir/auth_service/internal/service/event"
	"github.com/monir/auth_service/internal/service/jwt"
	"github.com/monir/auth_service/internal/service/password"
	"github.com/monir/auth_service/internal/usecase/forgotpassword"
	"github.com/monir/auth_service/internal/usecase/login"
	"github.com/monir/auth_service/internal/usecase/logout"
	"github.com/monir/auth_service/internal/usecase/refresh"
	"github.com/monir/auth_service/internal/usecase/register"
	"github.com/monir/auth_service/internal/usecase/resetpassword"
	"github.com/monir/auth_service/internal/usecase/verifyemail"
	"github.com/monir/auth_service/pkg/logger"
	authpb "github.com/monir/auth_service/proto/gen/auth"
)

func main() {
	cfgPath := envOrDefault("CONFIG_PATH", "configs/config.yaml")
	cfg, err := config.Load(cfgPath)
	must(err, "load config")

	log, err := logger.New(cfg.Server.Mode)
	must(err, "init logger")
	defer log.Sync() //nolint:errcheck

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── Observability ────────────────────────────────────────────────────────
	if otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); otlpEndpoint != "" {
		shutdown, err := observability.InitTracer(ctx, otlpEndpoint)
		if err != nil {
			log.Warn("tracing init failed", zap.Error(err))
		} else {
			defer shutdown(ctx) //nolint:errcheck
		}
	}

	// ── Infrastructure ───────────────────────────────────────────────────────
	db, err := postgresRepo.NewPool(ctx, cfg.Postgres)
	must(err, "postgres")

	redisClient, err := redisRepo.NewClient(cfg.Redis)
	must(err, "redis")

	// ── Event publisher ──────────────────────────────────────────────────────
	var eventPub event.Publisher
	if cfg.RabbitMQ.URL != "" {
		pub, err := event.NewRabbitPublisher(cfg.RabbitMQ.URL, "auth.events")
		if err != nil {
			log.Warn("rabbitmq unavailable, using noop publisher", zap.Error(err))
			eventPub = event.NoopPublisher{}
		} else {
			eventPub = pub
			defer pub.Close() //nolint:errcheck
		}
	} else {
		eventPub = event.NoopPublisher{}
	}

	// ── Repositories ─────────────────────────────────────────────────────────
	userRepo    := postgresRepo.NewUserRepo(db)
	roleRepo    := postgresRepo.NewRoleRepo(db)
	siteRepo    := postgresRepo.NewSiteRepo(db)
	tokenRepo   := postgresRepo.NewTokenRepo(db)
	cache       := redisRepo.NewTokenCache(redisClient)

	// ── Services ─────────────────────────────────────────────────────────────
	jwtService  := jwt.New(cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret, cfg.JWT.AccessExpMinutes, cfg.JWT.RefreshExpDays)
	pwdService  := password.New()
	emailService := email.New(cfg.Email.Host, cfg.Email.Port, cfg.Email.Username, cfg.Email.Password, cfg.Email.From)

	// ── Use Cases ────────────────────────────────────────────────────────────
	registerUC    := register.New(userRepo, roleRepo, siteRepo, pwdService, eventPub)
	loginUC       := login.New(userRepo, siteRepo, tokenRepo, pwdService, jwtService, eventPub)
	refreshUC     := refresh.New(userRepo, siteRepo, tokenRepo, jwtService)
	logoutUC      := logout.New(tokenRepo, jwtService, eventPub)
	forgotPwUC    := forgotpassword.New(userRepo, tokenRepo, emailService)
	resetPwUC     := resetpassword.New(userRepo, tokenRepo, tokenRepo, pwdService, eventPub)
	verifyEmailUC := verifyemail.New(userRepo, tokenRepo, emailService, eventPub)

	// ── HTTP Handlers ────────────────────────────────────────────────────────
	permRepo    := postgresRepo.NewPermissionRepo(db)

	authHandler := handler.NewAuthHandler(registerUC, loginUC, refreshUC, logoutUC, forgotPwUC, resetPwUC, verifyEmailUC)
	userHandler := handler.NewUserHandler(userRepo)
	siteHandler := handler.NewSiteHandler(siteRepo, roleRepo)
	roleHandler := handler.NewRoleHandler(roleRepo, permRepo)

	httpEngine := router.New(
		router.Handlers{Auth: authHandler, User: userHandler, Site: siteHandler, Role: roleHandler},
		jwtService,
		cache,
	)

	// ── gRPC Server ──────────────────────────────────────────────────────────
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(grpcLoggingInterceptor(log)),
	)
	authpb.RegisterAuthServiceServer(grpcServer, grpcDelivery.NewAuthServer(jwtService, userRepo))

	// ── Start servers ────────────────────────────────────────────────────────
	httpServer := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      httpEngine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("HTTP server starting", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	go func() {
		lis, err := net.Listen("tcp", ":"+cfg.GRPC.Port)
		if err != nil {
			log.Fatal("gRPC listen error", zap.Error(err))
		}
		log.Info("gRPC server starting", zap.String("addr", lis.Addr().String()))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("gRPC server error", zap.Error(err))
		}
	}()

	// ── Graceful shutdown ────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP shutdown error", zap.Error(err))
	}
	db.Close()
	redisClient.Close()

	log.Info("server stopped")
}

func grpcLoggingInterceptor(log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			log.Error("gRPC error", zap.String("method", info.FullMethod), zap.Error(err))
		}
		return resp, err
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func must(err error, label string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s: %v\n", label, err)
		os.Exit(1)
	}
}
