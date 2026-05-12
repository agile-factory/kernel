.PHONY: dev build test migrate migrate-down migrate-force fmt docker-up docker-down logs clean

# ============================================================================
# Development
# ============================================================================

dev: ## Start all services with Docker Compose
	docker compose up --build

dev-detached: ## Start all services in background
	docker compose up --build -d

dev-backend: ## Start backend only (local)
	cd backend && make run

dev-frontend: ## Start frontend only (local)
	cd frontend && npm run dev

# ============================================================================
# Build
# ============================================================================

build: build-backend build-frontend ## Build all

build-backend: ## Build backend (release)
	cd backend && make build

build-frontend: ## Build frontend
	cd frontend && npm run build

# ============================================================================
# Test
# ============================================================================

test: test-backend test-frontend ## Run all tests

test-backend: ## Run backend tests
	cd backend && make test

test-frontend: ## Run frontend tests
	cd frontend && npm test

# ============================================================================
# Go Tools
# ============================================================================

fmt: ## Format Go code
	cd backend && make fmt

tidy: ## Tidy go modules
	cd backend && make tidy

# ============================================================================
# Database
# ============================================================================

DB_URL         ?= $(shell grep '^DATABASE_URL' .env | cut -d '=' -f2-)
MIGRATIONS_DIR := backend/migrations
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
	docker compose exec postgres psql -U lokatik -d lokatik

db-reset: ## Reset database (drop + recreate + migrate)
	docker compose exec postgres psql -U lokatik -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
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

logs-frontend: ## Show frontend logs
	docker compose logs -f frontend

# ============================================================================
# Utilities
# ============================================================================

clean: ## Clean build artifacts
	cd backend && make clean
	rm -rf frontend/dist frontend/node_modules

install: ## Install all dependencies
	cd backend && make tidy
	cd frontend && npm install

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
