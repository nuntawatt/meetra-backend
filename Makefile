# WeGo Makefile
# Usage: make <target>

BINARY     := wego
BUILD_DIR  := ./build
CMD        := ./cmd/server
MIGRATIONS := ./migrations

.PHONY: all build run dev test test-unit test-integration \
        migrate-up migrate-down migrate-drop \
        docker-up docker-down docker-logs \
        seed lint tidy help

# ——— Default ——————————————————————————————————————————————————————————————————

all: build

# ——— Build ————————————————————————————————————————————————————————————————————

build:
	@echo "==> Building binary..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY) $(CMD)
	@echo "==> Binary: $(BUILD_DIR)/$(BINARY)"

# ——— Run ——————————————————————————————————————————————————————————————————————

run: build
	@echo "==> Running $(BINARY)..."
	@$(BUILD_DIR)/$(BINARY)

# Live-reload dev mode (requires github.com/air-verse/air)
dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air

# ——— Testing ——————————————————————————————————————————————————————————————————

test:
	@echo "==> Running all tests..."
	go test -v -race -count=1 ./...

test-unit:
	@echo "==> Running unit tests..."
	go test -v -race -count=1 ./tests/unit/...

test-integration:
	@echo "==> Running integration tests..."
	go test -v -race -count=1 ./tests/integration/...

test-coverage:
	@echo "==> Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "==> Report: coverage.html"

# ——— Database ————————————————————————————————————————————————————————————————

# Requires golang-migrate CLI: brew install golang-migrate
DB_URL ?= postgres://postgres:postgres@localhost:5432/wego?sslmode=disable

migrate-up:
	@echo "==> Applying migrations..."
	migrate -path $(MIGRATIONS) -database "$(DB_URL)" up

migrate-down:
	@echo "==> Rolling back last migration..."
	migrate -path $(MIGRATIONS) -database "$(DB_URL)" down 1

migrate-drop:
	@echo "==> Dropping all migrations (DESTRUCTIVE!)..."
	migrate -path $(MIGRATIONS) -database "$(DB_URL)" drop -f

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir $(MIGRATIONS) -seq $$name

seed:
	@echo "==> Seeding database..."
	psql "$(DB_URL)" -f scripts/seed.sql

# ——— Docker ——————————————————————————————————————————————————————————————————

docker-up:
	@echo "==> Starting services..."
	docker compose up -d --build

docker-down:
	@echo "==> Stopping services..."
	docker compose down

docker-logs:
	docker compose logs -f app

docker-clean:
	docker compose down -v --remove-orphans

# ——— Code quality ————————————————————————————————————————————————————————————

tidy:
	@echo "==> Tidying modules..."
	go mod tidy

lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w . 2>/dev/null || true

vet:
	go vet ./...

# ——— Help ————————————————————————————————————————————————————————————————————

help:
	@echo ""
	@echo "WeGo — Available make targets:"
	@echo ""
	@echo "  make build           Build the binary"
	@echo "  make run             Build & run"
	@echo "  make dev             Live-reload dev mode (requires air)"
	@echo "  make test            Run all tests"
	@echo "  make test-unit       Run unit tests only"
	@echo "  make test-coverage   Run tests + generate HTML coverage report"
	@echo "  make migrate-up      Apply all pending migrations"
	@echo "  make migrate-down    Roll back last migration"
	@echo "  make seed            Seed development data"
	@echo "  make docker-up       Start all services via docker compose"
	@echo "  make docker-down     Stop all services"
	@echo "  make lint            Run golangci-lint"
	@echo "  make tidy            go mod tidy"
	@echo ""
