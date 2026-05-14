.PHONY: dev build test migrate migrate-down migrate-force fmt tidy docker-up docker-down logs clean install help

APP_NAME := api
EXAMPLE  := ./examples/saas-b2b

# ============================================================================
# Development
# ============================================================================

dev: ## Start all services with Docker Compose
	docker compose up --build

dev-detached: ## Start all services in background
	docker compose up --build -d

run: ## Run the saas-b2b example locally
	go run $(EXAMPLE)

# ============================================================================
# Build / Test
# ============================================================================

build: ## Build the saas-b2b example binary
	go build -o bin/$(APP_NAME) $(EXAMPLE)

test: ## Run all Go tests
	go test -v ./...

fmt: ## Format Go code
	go fmt ./...

tidy: ## Tidy go modules
	go mod tidy

# ============================================================================
# Database
# ============================================================================

DB_URL         ?= $(shell grep '^DATABASE_URL' .env | cut -d '=' -f2-)
MIGRATIONS_DIR := migrations
MIGRATE        := $(shell go env GOPATH)/bin/migrate

migrate: ## Apply all pending UP migrations
	@echo "→ Applying migrations from $(MIGRATIONS_DIR)..."
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

migrate-down: ## Roll back the last migration step
	@echo "→ Rolling back last migration..."
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down 1

migrate-force: ## Force a specific migration version (usage: make migrate-force VERSION=<n>)
	@echo "→ Forcing migration version $(VERSION)..."
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" force $(VERSION)

db-shell: ## Open PostgreSQL shell
	docker compose exec postgres psql -U $${APP_NAME:-myapp} -d $${APP_NAME:-myapp}

db-reset: ## Reset database (drop + recreate + migrate)
	docker compose exec postgres psql -U $${APP_NAME:-myapp} -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	$(MAKE) migrate

# ============================================================================
# Docker
# ============================================================================

docker-up: ## Start Docker services
	docker compose up -d

docker-down: ## Stop Docker services
	docker compose down

docker-down-volumes: ## Stop Docker services and remove volumes
	docker compose down -v

docker-build: ## Build Docker images
	docker compose build

logs: ## Show all logs
	docker compose logs -f

logs-backend: ## Show backend logs
	docker compose logs -f backend

# ============================================================================
# Utilities
# ============================================================================

clean: ## Clean build artifacts
	rm -rf bin/

install: tidy ## Install all dependencies

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
