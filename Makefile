.PHONY: dev build test migrate clippy fmt docker-up docker-down logs clean

# ============================================================================
# Development
# ============================================================================

dev: ## Start all services with Docker Compose
	docker compose up --build

dev-detached: ## Start all services in background
	docker compose up --build -d

dev-backend: ## Start backend only (local)
	cd backend && cargo run

dev-frontend: ## Start frontend only (local)
	cd frontend && npm run dev

# ============================================================================
# Build
# ============================================================================

build: build-backend build-frontend ## Build all

build-backend: ## Build backend (release)
	cd backend && cargo build --release

build-frontend: ## Build frontend
	cd frontend && npm run build

# ============================================================================
# Test
# ============================================================================

test: test-backend test-frontend ## Run all tests

test-backend: ## Run backend tests
	cd backend && cargo test

test-frontend: ## Run frontend tests
	cd frontend && npm test

# ============================================================================
# Rust Tools
# ============================================================================

clippy: ## Run Clippy linter
	cd backend && cargo clippy -- -D warnings

fmt: ## Format Rust code
	cd backend && cargo fmt

fmt-check: ## Check Rust formatting
	cd backend && cargo fmt -- --check

# ============================================================================
# Database
# ============================================================================

migrate: ## Run database migrations
	cd backend && sqlx migrate run

migrate-revert: ## Revert last migration
	cd backend && sqlx migrate revert

db-shell: ## Open PostgreSQL shell
	docker compose exec postgres psql -U lokatik -d lokatik

db-reset: ## Reset database (drop + recreate + migrate)
	docker compose exec postgres psql -U lokatik -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	cd backend && sqlx migrate run

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
	cd backend && cargo clean
	rm -rf frontend/dist frontend/node_modules

install: ## Install all dependencies
	cd backend && cargo fetch
	cd frontend && npm install

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
