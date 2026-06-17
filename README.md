# Auth Service

Centralized authentication and authorization microservice.
Other services authenticate users via **REST** (external) and **gRPC** (internal).

---

## Table of Contents

- [Status](#status)
- [How Docker & Kubernetes Work Together](#how-docker--kubernetes-work-together)
- [Prerequisites](#prerequisites)
- [Run Locally](#run-locally)
- [Verify It Works](#verify-it-works)
- [Environment Variables](#environment-variables)
- [REST API](#rest-api)
- [gRPC API](#grpc-api-internal)
- [Events](#events-rabbitmq)
- [Database Migrations](#database-migrations)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Swagger UI](#swagger-ui)
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

## How Docker & Kubernetes Work Together

### The Core Mental Model

```
Your Code (Go app)
     ↓
Docker  — packages your code + its dependencies into a portable "box" (image/container)
     ↓
Kubernetes — runs and manages those boxes at scale in production
```

Docker and Kubernetes are not competitors. **Docker builds the thing; Kubernetes runs and manages many copies of it.**

---

### What Docker Does in This Project

**Problem Docker solves:** this service needs four things to run — PostgreSQL, Redis, RabbitMQ, and the Go app itself. Installing all four on your machine is messy, version-specific, and won't match production. Docker wraps each one in an isolated container that works the same everywhere.

#### Dockerfile (`docker/Dockerfile`)

A two-stage recipe:

| Stage | Base image | What it does |
|---|---|---|
| `builder` | `golang:1.24-alpine` | Downloads deps, compiles the Go binary |
| `runtime` | `scratch` (empty) | Copies only the final binary — no OS, no shell |

The result is a ~10 MB image instead of a ~300 MB one.

#### docker-compose.yml

Defines your entire local environment in one file. All services share a private network (`auth-net`) so they can reach each other **by container name** — that's why `DATABASE_URL` uses `@postgres:5432` instead of `@localhost:5432`.

| Service | Role | Port |
|---|---|---|
| `postgres` | Database | 5432 |
| `redis` | Token blacklist + rate limiting | 6379 |
| `rabbitmq` | Event bus | 5672 / 15672 (UI) |
| `auth` | Your Go service | 8080 (HTTP), 50051 (gRPC) |
| `migrate` | One-time DB setup job | — |

`depends_on` + `healthcheck` ensures the Go app doesn't start until Postgres is actually accepting connections, not just started.

---

### What Kubernetes Does

**Problem Kubernetes solves:** Docker Compose runs on one machine. In production you need:

- **Multiple copies** running so one crash doesn't take everything down
- **Automatic restarts** when a container dies
- **Rolling updates** — deploy a new version with zero downtime
- **Health checking** — stop sending traffic to a broken container
- **Secret management** — keep passwords out of your code

Kubernetes (K8s) manages all of this. You write YAML describing *what you want*, and K8s makes it happen and keeps it that way.

#### K8s files (`k8s/`)

| File | What it does |
|---|---|
| `deployment.yaml` | "Run 2 copies of the container. If one crashes, restart it. Update one at a time." |
| `service.yaml` | "Give those 2 copies a single stable address so other services can reach them." |
| `ingress.yaml` | "Route external HTTP traffic into the service." |
| `secret.yaml` | "Store passwords (DB URL, JWT secrets) securely — inject them as env vars." |
| `configmap.yaml` | "Store `config.yaml` and mount it into the container." |
| `migration-job.yaml` | "Run DB migrations once before the app starts, then stop." |

Key settings in `deployment.yaml`:

- `replicas: 2` — K8s always keeps 2 copies alive; if one dies it spins up a replacement.
- `livenessProbe` — K8s calls `GET /health`. If it fails repeatedly, the container is restarted.
- `readinessProbe` — K8s calls `GET /health`. Until it passes, no traffic is sent to that pod.
- `rollingUpdate: maxUnavailable: 0` — during a deploy, the old copy stays up until the new one is healthy.

---

### How They Fit Together

```
Write code
    ↓
docker build        ← Dockerfile turns your code into an image
    ↓
docker push         ← Image goes to a registry (Docker Hub, ECR, GCR, etc.)
    ↓
kubectl apply       ← K8s pulls the image from the registry, runs it, watches it
```

`docker-compose.yml` is for **local development only**. `k8s/` is for **production**. Both use the same Docker image — that is the link between them.

---

### Running Locally (Windows — step by step)

**1. Install Docker Desktop** — [docs.docker.com/desktop/install/windows-install](https://docs.docker.com/desktop/install/windows-install/). Make sure it is running (check the system tray icon).

**2. Create your `.env` file**

```powershell
copy .env.example .env
```

Open `.env` and set at minimum:

```
JWT_ACCESS_SECRET=some-random-string-at-least-32-chars
JWT_REFRESH_SECRET=another-random-string-at-least-32-chars
```

**3. Start infrastructure**

```powershell
docker compose up -d postgres redis rabbitmq
```

Starts the three backing services in the background. Your Go app runs directly on your machine, which makes debugging fast.

**4. Run database migrations** (once, or after adding new migration files)

```powershell
docker compose run --rm migrate
```

**5. Start the Go app**

```powershell
got mod tidy
go run ./cmd/server
```

**6. Verify**

```powershell
curl http://localhost:8080/health
```

> **Alternative — run everything in Docker** (no Go installation needed, but slower to iterate):
> ```powershell
> docker compose up -d --build
> ```

---

### When You're Ready for Kubernetes

You'll need `kubectl` and a cluster. For local Kubernetes try [minikube](https://minikube.sigs.k8s.io) or [kind](https://kind.sigs.k8s.io). The deploy flow is:

```bash
# 1. Build and push the image to a registry
docker build -f docker/Dockerfile -t your-registry/auth-service:1.0.0 .
docker push your-registry/auth-service:1.0.0

# 2. Update the image name in k8s/deployment.yaml
# 3. Fill in real secrets in k8s/secret.yaml

# 4. Deploy (migrations run first, then the service)
make k8s-deploy
```

`make k8s-deploy` applies in order: secrets → configmaps → migration Job (waits for it to finish) → Deployment → Service → Ingress.

## Troubleshooting: Ingress `/health` returns 404

Symptom: public requests to `https://<host>/health` return HTTP 404, while `kubectl port-forward svc/auth-service 8080:8080` and `curl http://localhost:8080/health` return `{"status":"ok"}`.

Root cause: an NGINX Ingress annotation `nginx.ingress.kubernetes.io/rewrite-target: /` rewrites incoming paths (for example `/health`) to `/`, which the app does not handle and returns 404.

Quick fixes (run on the machine with `kubectl` configured):

```bash
# Apply the edited ingress that preserves original paths
kubectl apply -f k8s/ingress.yaml

# Or remove the annotation in-place and force controller reload
kubectl annotate ingress auth-service nginx.ingress.kubernetes.io/rewrite-target- --overwrite
kubectl rollout restart deployment/ingress-nginx-controller -n ingress-nginx

# Verify the Ingress no longer rewrites and controller reloaded
kubectl describe ingress auth-service
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller --tail=200

# Test public endpoint
curl -kv https://auth.kossti.com/health

# Port-forward sanity check (keeps a local connection open)
kubectl port-forward svc/auth-service 8080:8080
curl http://127.0.0.1:8080/health
```

What to watch for:
- `kubectl describe ingress` should not show the `rewrite-target` annotation.
- Ingress controller logs should show a successful reload and no fatal errors.
- App logs should show `GET "/health"` (not `GET "/"`) and return 200.

Commit the fix to track it in the repo:

```bash
git add k8s/ingress.yaml
git commit -m "Ingress: preserve paths, add /health"
git push
```

---

### Tool Responsibility Summary

| Tool | Environment | Responsibility |
|---|---|---|
| `Dockerfile` | Everywhere | Defines how to package your app into an image |
| `docker-compose.yml` | Local dev | Runs all services together on your machine |
| `k8s/*.yaml` | Production | Tells Kubernetes how to run, scale, and watch your service |

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
}

## Monitoring

This section summarizes quick runtime checks, metrics, and recommended monitoring stack components.

- Quick cluster/runtime checks:

```bash
kubectl get ns
kubectl get pods --all-namespaces
kubectl get deploy --all-namespaces
kubectl get svc --all-namespaces
kubectl get ingress --all-namespaces
```

- Quick checks for `auth-service`:

```bash
kubectl get pods -l app=auth-service -o wide
kubectl logs -l app=auth-service --tail=200
kubectl describe deploy auth-service
kubectl port-forward svc/auth-service 8080:8080
curl http://127.0.0.1:8080/health
```

- Fetch metrics from the app (Prometheus exposition at `/metrics`):

```bash
kubectl port-forward svc/auth-service 8080:8080
curl http://127.0.0.1:8080/metrics | head
```

- Short-term monitoring (Prometheus + Grafana):

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
helm install monitoring prometheus-community/kube-prometheus-stack -n monitoring --create-namespace
```

After installation, create a `ServiceMonitor` or PodMonitor for `auth-service` so Prometheus scrapes `/metrics`.

- Logs: deploy Loki + Promtail and view logs from Grafana.

- Tracing: deploy Jaeger or Tempo and an OpenTelemetry Collector. `auth-service` supports OTel (see `internal/observability`).

- Alerts & Dashboards: add PrometheusRule manifests for errors, high latency, and pod restarts; import Go/Gin dashboards into Grafana.

If you want, I can: add a `ServiceMonitor` manifest for `auth-service`, create a sample Grafana dashboard, or generate alert rules next.
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

### Deploying to a VPS (Contabo) — auth.kossti.com

Production deployment on a single Contabo VPS using k3s (lightweight Kubernetes) with automatic HTTPS via Let's Encrypt.

**Target:** `https://auth.kossti.com` → VPS `13.140.158.119`

#### 1. DNS

Add an A record at your domain registrar:

```
A   auth.kossti.com   13.140.158.119   TTL 300
```

Verify with `nslookup auth.kossti.com` before proceeding.

#### 2. SSH into the VPS and install dependencies

```bash
ssh root@13.140.158.119

apt update && apt install -y docker.io git curl
systemctl enable docker && systemctl start docker
```

#### 3. Install k3s (without built-in Traefik)

```bash
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--disable=traefik" sh -

export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
echo 'export KUBECONFIG=/etc/rancher/k3s/k3s.yaml' >> ~/.bashrc

kubectl get nodes   # should show Ready
```

#### 4. Install Nginx Ingress Controller

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.10.1/deploy/static/provider/cloud/deploy.yaml

kubectl rollout status deployment/ingress-nginx-controller -n ingress-nginx
```

#### 5. Install cert-manager

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.15.0/cert-manager.yaml

kubectl rollout status deployment/cert-manager -n cert-manager
```

#### 6. Create persistent storage directory

```bash
# Only host directory needed — for PostgreSQL data
mkdir -p /opt/k8s-data/postgres
```

#### 7. Clone the repo and build the image

```bash
cd /opt
git clone https://github.com/monijaman/auth_service.git
cd auth_service

# Build the Docker image
docker build -f docker/Dockerfile -t auth-service:latest .

# Import into k3s containerd (k3s does not use the Docker daemon)
docker save auth-service:latest | k3s ctr images import -

# Verify
k3s ctr images ls | grep auth-service
```

#### 8. Create the dependencies manifest
Create `k8s/deps.yaml` with PostgreSQL, Redis, and RabbitMQ deployments and their ClusterIP services. Each service name (`postgres-svc`, `redis-svc`, `rabbitmq-svc`) must match the values in `k8s/secret.yaml` so the auth service can reach them.

The repository already includes a ready-to-use file: [auth_service/k8s/deps.yaml](auth_service/k8s/deps.yaml#L1-L200). Key notes:

- PostgreSQL uses a `hostPath` on the VPS: `/opt/k8s-data/postgres` for persistence.
- RabbitMQ exposes both `5672` (AMQP) and `15672` (management UI) on the ClusterIP service.
- Redis uses the standard port `6379`.

Apply and verify the dependencies before continuing with secrets and migrations:

```bash
# From the repo root
kubectl apply -f k8s/deps.yaml
kubectl rollout status deployment/postgres --timeout=120s
# List all three using a set-based selector (preferred)
kubectl get pods -l 'app in (postgres,redis,rabbitmq)'

# Or list each separately:
kubectl get pods -l app=postgres
kubectl get pods -l app=redis
kubectl get pods -l app=rabbitmq
```

If you need to inspect or edit connection strings, compare with `k8s/secret.yaml.example` and then create `k8s/secret.yaml` with your real values.

#### 9. Fill in real secrets

```bash
# Generate two strong random secrets
openssl rand -hex 32   # use output for JWT_ACCESS_SECRET
openssl rand -hex 32   # use output for JWT_REFRESH_SECRET

cp k8s/secret.yaml.example k8s/secret.yaml
nano k8s/secret.yaml   # paste real values
```

`k8s/secret.yaml` is in `.gitignore` — never commit it.

#### 10. Apply everything in order

```bash
# Dependencies
kubectl apply -f k8s/deps.yaml
kubectl rollout status deployment/postgres

# Config and secrets
kubectl apply -f k8s/secret.yaml
kubectl apply -f k8s/configmap.yaml

# Database migrations (runs once, then stops)
kubectl apply -f k8s/migration-configmap.yaml
kubectl apply -f k8s/migration-job.yaml
kubectl wait --for=condition=complete job/auth-migrate --timeout=120s

# Auth service
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/deployment.yaml

# TLS issuer + Ingress (apply after DNS is live)
kubectl apply -f k8s/cluster-issuer.yaml
kubectl apply -f k8s/ingress.yaml
```

#### 11. Verify

```bash
kubectl get pods                          # all should show Running
kubectl get ingress                       # should show the VPS IP
kubectl describe certificate auth-kossti-tls   # wait for Ready: True
curl https://auth.kossti.com/health       # {"status":"ok"}
```

#### What lives where on the host

| Path | Purpose |
|---|---|
| `/opt/auth_service/` | Cloned repo and k8s manifests |
| `/opt/k8s-data/postgres/` | PostgreSQL data (persisted across pod restarts) |
| Everything else | Runs inside containers — no files on host |

> No `/var/www` directory is needed. Nginx runs as a pod inside Kubernetes and routes external traffic to your service via the Ingress resource.

---

## Swagger UI

Interactive API docs — try every endpoint directly from the browser.

```bash
docker compose up -d swagger-ui
```

Then open: **`http://localhost:8081`**

The spec lives at [docs/swagger.yaml](docs/swagger.yaml).

**Typical test flow:**

1. `POST /auth/register` — create a user
2. `POST /auth/login` — copy the `access_token` from the response
3. Click **Authorize** (lock icon, top right) → paste the token
4. All protected endpoints will now send `Authorization: Bearer ...` automatically

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
