.PHONY: run build docker-up docker-down migrate-up migrate-down proto lint test

# ── Dev ───────────────────────────────────────────────────────────────────────
run:
	go run ./cmd/server

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/auth-service ./cmd/server

# ── Docker ────────────────────────────────────────────────────────────────────
docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f auth

# ── Database ──────────────────────────────────────────────────────────────────
migrate-up:
	docker compose run --rm migrate

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

# ── Kubernetes ───────────────────────────────────────────────────────────────
k8s-deploy:
	@echo "Applying secrets and config..."
	kubectl apply -f k8s/secret.yaml
	kubectl apply -f k8s/configmap.yaml
	kubectl apply -f k8s/migration-configmap.yaml
	@echo "Running migrations..."
	kubectl apply -f k8s/migration-job.yaml
	kubectl wait --for=condition=complete job/auth-migrate --timeout=120s
	@echo "Deploying service..."
	kubectl apply -f k8s/deployment.yaml
	kubectl apply -f k8s/service.yaml
	kubectl apply -f k8s/ingress.yaml
	kubectl rollout status deployment/auth-service

k8s-delete:
	kubectl delete -f k8s/ --ignore-not-found

k8s-logs:
	kubectl logs -l app=auth-service -f

# ── Proto (run after installing protoc) ───────────────────────────────────────
proto:
	@which protoc > /dev/null 2>&1 || (echo "Install protoc: https://grpc.io/docs/protoc-installation/" && exit 1)
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/auth.proto
	@echo "Proto generated. Remove the JSON codec in proto/gen/auth/auth.go after verifying."

# ── Quality ───────────────────────────────────────────────────────────────────
lint:
	golangci-lint run ./...

test:
	go test -race -cover ./...

tidy:
	go mod tidy
