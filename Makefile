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
