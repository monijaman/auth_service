# Auth Service

Centralized authentication and authorization microservice.
Other services authenticate users via **REST** (external) and **gRPC** (internal).

---

## Table of Contents

- [Status](#status)
- [Prerequisites](#prerequisites)
- [Run Locally](#run-locally)
- [Verify It Works](#verify-it-works)
- [Environment Variables](#environment-variables)
- [REST API](#rest-api)
- [gRPC API](#grpc-api-internal)
- [Events](#events-rabbitmq)
- [Database Migrations](#database-migrations)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Observability](#observability)
- [Security](#security)
- [Project Structure](#project-structure)

---

## Status

| Phase | Description | Status |
|-------|-------------|--------|
| 1 | Project setup, Docker, PostgreSQL, Migrations | ✅ Done |
| 2 | User registration, Login, JWT generation | ✅ Done |
| 3 | Refresh token flow, Logout | ✅ Done |
| 4 | RBAC (roles & permissions) | ✅ Done |
| 5 | gRPC services (ValidateToken, GetUser, HasPermission) | ✅ Done |
| 6 | Email verification | ✅ Done |
| 7 | Password reset | ✅ Done |
| 8 | RabbitMQ event publishing | ✅ Done |
| 9 | Prometheus metrics + OpenTelemetry tracing | ✅ Done |
| 10 | Kubernetes manifests | ✅ Done |

---

## Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.24+ | `sudo apt install golang-go` or [go.dev/dl](https://go.dev/dl/) |
| Docker | latest | `sudo apt install docker.io docker-compose-plugin` |
| migrate CLI | v4.18+ | See below (optional) |

```bash
# Install migrate CLI (optional — for manual DB control)
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.18.1/migrate.linux-amd64.tar.gz | tar xz
sudo mv migrate /usr/local/bin/
```

---

## Run Locally

```bash
# 1. Clone and enter the directory
cd auth_service

# 2. Start infrastructure (postgres, redis, rabbitmq)
docker compose up -d postgres redis rabbitmq

# 3. Run database migrations
docker compose run --rm migrate

# 4. Download Go dependencies
go mod tidy

# 5. Start the service
go run ./cmd/server
```

The service starts on:

| | URL |
|-|-----|
| REST API | `http://localhost:8080/api/v1` |
| Health   | `http://localhost:8080/health` |
| Metrics  | `http://localhost:8080/metrics` |
| gRPC     | `localhost:50051` |

---

## Verify It Works

```bash
# Health check
curl http://localhost:8080/health

# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"secret123"}'

# Login — save the access_token from the response
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"secret123"}'

# Get current user
curl http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer <access_token>"
```

---

## Environment Variables

Copy `.env.example` to `.env` and fill in the values.

| Variable | Required | Description | Default |
|----------|----------|-------------|---------|
| `JWT_ACCESS_SECRET` | ✅ | Min 32 chars | — |
| `JWT_REFRESH_SECRET` | ✅ | Min 32 chars | — |
| `DATABASE_URL` | | PostgreSQL DSN | from `configs/config.yaml` |
| `REDIS_ADDR` | | Redis address | `localhost:6379` |
| `RABBITMQ_URL` | | AMQP URL (leave empty to disable events) | — |
| `PORT` | | HTTP port | `8080` |
| `CONFIG_PATH` | | Path to config YAML | `configs/config.yaml` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | | OTLP endpoint for tracing | disabled |

---

## REST API

Base URL: `http://localhost:8080/api/v1`

### Public

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/register` | Register a new user |
| POST | `/auth/login` | Login — returns access + refresh tokens |
| POST | `/auth/refresh` | Rotate refresh token |
| POST | `/auth/forgot-password` | Send password reset OTP to email |
| POST | `/auth/reset-password` | Reset password using OTP |

### Protected

> Requires `Authorization: Bearer <access_token>`

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/logout` | Revoke refresh token |
| POST | `/auth/verify-email` | Verify email with OTP |
| POST | `/auth/resend-verification` | Resend verification email |
| GET | `/users/me` | Get current user profile |
| GET | `/users/:id` | Get user by ID |
| PUT | `/users/:id` | Update user |
| DELETE | `/users/:id` | Soft-delete user |

---

## gRPC API (internal)

Port `50051` — used by other microservices to validate tokens and check permissions.

```protobuf
service AuthService {
  rpc ValidateToken   (TokenRequest)      returns (TokenResponse);
  rpc GetUser         (UserRequest)       returns (UserResponse);
  rpc HasPermission   (PermissionRequest) returns (PermissionResponse);
}
```

### Connecting from another service

```go
import authpb "github.com/monir/auth_service/proto/gen/auth"

conn, _ := grpc.NewClient("auth-service:50051",
    grpc.WithTransportCredentials(insecure.NewCredentials()))
client := authpb.NewAuthServiceClient(conn)

resp, _ := client.ValidateToken(ctx, &authpb.TokenRequest{Token: bearerToken})
if resp.Valid {
    // resp.UserID, resp.Email, resp.Roles, resp.Permissions
}
```

> **Note:** gRPC currently uses JSON encoding (no `protoc` required).
> Run `make proto` after installing protoc to switch to binary protobuf for production.

---

## Events (RabbitMQ)

Exchange: `auth.events` (topic). Leave `RABBITMQ_URL` empty to disable.

| Routing Key | Triggered When |
|-------------|----------------|
| `UserRegistered` | Registration succeeded |
| `UserLoggedIn` | Login succeeded |
| `UserLoggedOut` | Logout called |
| `PasswordChanged` | Password reset completed |
| `EmailVerified` | Email verification completed |
| `UserDeleted` | User soft-deleted |

---

## Database Migrations

```bash
# Run all pending migrations
docker compose run --rm migrate

# Roll back the last migration
migrate -path migrations -database "$DATABASE_URL" down 1

# Create a new migration file
make migrate-create
```

---

## Kubernetes Deployment

K8s does not run migrations automatically. The migration Job runs first, then the Deployment starts.

```bash
# 1. Build and push the image
docker build -f docker/Dockerfile -t your-registry/auth-service:1.0.0 .
docker push your-registry/auth-service:1.0.0

# 2. Update the image name in k8s/deployment.yaml
# 3. Fill in real secrets in k8s/secret.yaml

# 4. Deploy (runs migrations first, then the service)
make k8s-deploy
```

`make k8s-deploy` applies secrets → configmaps → migration Job (waits for completion) → Deployment → Service → Ingress.

```bash
# Tear down
make k8s-delete

# View logs
make k8s-logs
```

---

## Observability

| Signal | How |
|--------|-----|
| Health | `GET /health` |
| Metrics | `GET /metrics` (Prometheus format) |
| Tracing | Set `OTEL_EXPORTER_OTLP_ENDPOINT` to enable OTLP export |

---

## Security

| Concern | Implementation |
|---------|---------------|
| Password hashing | Argon2id |
| Access tokens | HS256 JWT, 15 min expiry |
| Refresh tokens | Rotated on every use, stored as SHA-256 hash |
| OTP codes | 6-digit, 15 min TTL |
| Rate limiting | Redis-backed per IP |
| Soft deletes | Users are never hard-deleted |
| Email enumeration | Forgot-password always returns 200 |

---

## Project Structure

```
auth_service/
├── cmd/server/               # Entry point (main.go)
├── configs/config.yaml       # Default configuration
├── internal/
│   ├── config/               # Config loading (Viper)
│   ├── domain/               # Entities + repository interfaces
│   │   ├── user/
│   │   ├── auth/             # Tokens, OTPs
│   │   ├── role/
│   │   └── permission/
│   ├── usecase/              # Business logic (one package per operation)
│   ├── repository/
│   │   ├── postgres/         # pgx implementations
│   │   └── redis/            # Cache, blacklist, rate limiter
│   ├── service/
│   │   ├── jwt/              # HS256 token issue + validate
│   │   ├── password/         # Argon2id hashing
│   │   ├── email/            # SMTP
│   │   └── event/            # RabbitMQ publisher
│   ├── delivery/
│   │   ├── http/             # Gin handlers + router
│   │   └── grpc/             # gRPC server
│   ├── middleware/            # JWT auth, RBAC, rate limiting
│   └── observability/        # Prometheus metrics, OTEL tracing
├── proto/
│   ├── auth.proto
│   └── gen/auth/             # gRPC stubs (JSON codec — run make proto to replace)
├── migrations/               # SQL migration files (golang-migrate)
├── docker/Dockerfile
├── docker-compose.yml
├── k8s/                      # Kubernetes manifests
└── Makefile
```

---

## Useful Commands

```bash
make run            # go run ./cmd/server
make build          # build binary to bin/auth-service
make test           # go test -race -cover ./...
make lint           # golangci-lint run
make docker-up      # docker compose up -d --build
make docker-down    # docker compose down
make migrate-up     # run migrations via docker compose
make migrate-create # create a new migration file
make proto          # regenerate gRPC stubs from proto/auth.proto
make k8s-deploy     # full kubernetes deploy (migrate → deploy)
make k8s-delete     # tear down kubernetes resources
```
