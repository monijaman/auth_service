# Auth Service

Centralized authentication and authorization microservice for the platform.
Used by all other services via REST (external) and gRPC (internal).

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

## Quick Start

### 1. Prerequisites

- Docker & Docker Compose
- Go 1.24+
- (Optional) `protoc` for regenerating gRPC stubs
- (Optional) `migrate` CLI for running migrations manually

### 2. Run locally with Docker

```bash
cp .env.example .env
# Edit .env — set JWT secrets at minimum

docker compose up -d                    # Start postgres, redis, rabbitmq
docker compose run --rm migrate         # Run DB migrations
go run ./cmd/server                     # Start the service
```

Or build and run everything in Docker:

```bash
docker compose --profile migrate up -d  # Starts all services + runs migrations
```

### 3. Environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL DSN | from `configs/config.yaml` |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `RABBITMQ_URL` | RabbitMQ AMQP URL | `amqp://guest:guest@localhost:5672/` |
| `JWT_ACCESS_SECRET` | **Required.** Min 32 chars | — |
| `JWT_REFRESH_SECRET` | **Required.** Min 32 chars | — |
| `CONFIG_PATH` | Path to config YAML | `configs/config.yaml` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP gRPC endpoint for tracing | disabled |

---

## REST API

Base URL: `http://localhost:8080/api/v1`

### Public endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/register` | Register a new user |
| POST | `/auth/login` | Login, returns access + refresh tokens |
| POST | `/auth/refresh` | Rotate refresh token |
| POST | `/auth/forgot-password` | Send password reset OTP |
| POST | `/auth/reset-password` | Reset password with OTP |

### Authenticated endpoints

> Requires `Authorization: Bearer <access_token>`

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/logout` | Revoke refresh token |
| POST | `/auth/verify-email` | Verify email with OTP |
| POST | `/auth/resend-verification` | Resend verification email |
| GET | `/users/me` | Get current user |
| GET | `/users/:id` | Get user by ID |
| PUT | `/users/:id` | Update user |
| DELETE | `/users/:id` | Soft-delete user |

### Example: Register

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}'
```

### Example: Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}'
```

---

## gRPC API (internal)

Port: `50051`

Used by other microservices to validate tokens and check permissions.

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

conn, _ := grpc.NewClient("auth-service:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
client := authpb.NewAuthServiceClient(conn)

resp, _ := client.ValidateToken(ctx, &authpb.TokenRequest{Token: bearerToken})
if resp.Valid {
    // resp.UserID, resp.Email, resp.Roles, resp.Permissions
}
```

> **Note on gRPC encoding**: The service currently uses JSON encoding over gRPC (no protoc required).
> To switch to binary protobuf (recommended for production), install protoc and run `make proto`,
> then remove the `init()` codec override in `proto/gen/auth/auth.go`.

---

## Events (RabbitMQ)

Exchange: `auth.events` (topic)

| Routing key | When |
|-------------|------|
| `UserRegistered` | After successful registration |
| `UserLoggedIn` | After successful login |
| `UserLoggedOut` | After logout |
| `PasswordChanged` | After password reset |
| `EmailVerified` | After email verification |
| `UserDeleted` | After soft delete |

---

## Observability

- **Metrics**: Prometheus — `GET /metrics`
- **Tracing**: OpenTelemetry (OTLP gRPC) — set `OTEL_EXPORTER_OTLP_ENDPOINT`
- **Health**: `GET /health`

---

## Database Migrations

```bash
# Up
migrate -path migrations -database "$DATABASE_URL" up

# Down one
migrate -path migrations -database "$DATABASE_URL" down 1

# Create a new migration
make migrate-create
```

---

## Project Structure

```
auth_service/
├── cmd/server/            # Entry point
├── internal/
│   ├── config/            # Config loading (Viper)
│   ├── delivery/
│   │   ├── http/          # REST handlers + router (Gin)
│   │   └── grpc/          # gRPC server
│   ├── domain/            # Entities + repository interfaces
│   │   ├── user/
│   │   ├── auth/          # Tokens, OTPs
│   │   ├── role/
│   │   └── permission/
│   ├── middleware/        # JWT auth, RBAC, rate limiting
│   ├── observability/     # Prometheus metrics, OTEL tracing
│   ├── repository/
│   │   ├── postgres/      # pgx implementations
│   │   └── redis/         # Cache, blacklist, rate limiter
│   ├── service/
│   │   ├── jwt/           # JWT issue + validate (HS256)
│   │   ├── password/      # Argon2id hashing
│   │   ├── email/         # SMTP
│   │   └── event/         # RabbitMQ publisher
│   └── usecase/           # Business logic per operation
├── proto/
│   ├── auth.proto
│   └── gen/auth/          # gRPC stubs (JSON codec, replace with make proto)
├── migrations/            # golang-migrate SQL files
├── configs/config.yaml
├── docker/Dockerfile
├── docker-compose.yml
├── k8s/                   # Kubernetes manifests
└── Makefile
```

---

## Kubernetes Deployment

```bash
kubectl apply -f k8s/secret.yaml      # Edit secrets first!
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml     # Requires nginx ingress controller
```

---

## Security

- Passwords: **Argon2id** hashing
- Access tokens: **HS256 JWT**, 15 min expiry
- Refresh tokens: rotated on every use, stored as SHA-256 hash
- OTP codes: 6-digit, 15 min TTL
- Rate limiting: Redis-backed per IP
- Soft deletes: users are never hard-deleted by default
- Email enumeration prevention on forgot-password
