.PHONY: help dev build test clean docker-up docker-down migrate-up migrate-down

help: ## Show this help message
	@echo 'Immobilier SaaS - Makefile Commands'
	@echo ''
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ==============================================================================
# Development
# ==============================================================================

setup: ## Initial project setup
	@echo "Setting up Immobilier SaaS..."
	@echo "Creating environment files..."
	@cp services/api/.env.example services/api/.env 2>/dev/null || echo "API .env already exists"
	@cp services/web/.env.example services/web/.env 2>/dev/null || echo "Web .env already exists"
	@echo ""
	@echo "Generating JWT keys..."
	@openssl genrsa -out services/api/private.pem 2048
	@openssl rsa -in services/api/private.pem -pubout -out services/api/public.pem
	@echo ""
	@echo "✅ Setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Add JWT keys to services/api/.env (see services/api/.env.example)"
	@echo "2. Run 'make docker-up' to start all services"
	@echo "3. Run 'make migrate-up' to initialize the database"

dev: ## Start all services in development mode with hot reload
	@docker compose up

dev-detached: ## Start all services in detached mode
	@docker compose up -d

dev-api: ## Start only API service
	@cd services/api && make dev

dev-web: ## Start only Web service
	@cd services/web && npm run dev

# ==============================================================================
# Docker Commands
# ==============================================================================

docker-up: ## Start all services with Docker Compose
	@docker compose up -d
	@echo ""
	@echo "✅ All services started!"
	@echo ""
	@echo "Services available at:"
	@echo "  API:           http://localhost:8080"
	@echo "  Web:           http://localhost:3000"
	@echo "  Swagger:       http://localhost:8080/swagger/index.html"
	@echo "  PostgreSQL:    localhost:5432"
	@echo "  Redis:         localhost:6379"
	@echo "  MinIO Console: http://localhost:9001 (admin/minioadmin)"
	@echo "  MailHog:       http://localhost:8025"
	@echo "  Grafana:       http://localhost:3001 (admin/admin)"
	@echo ""
	@echo "Run 'make migrate-up' to initialize the database"
	@echo "Run 'make logs' to view logs"

docker-down: ## Stop all services
	@docker compose down

docker-down-volumes: ## Stop all services and remove volumes
	@docker compose down -v

docker-restart: ## Restart all services
	@docker compose restart

docker-build: ## Rebuild all Docker images
	@docker compose build

docker-logs: logs ## Show logs from all services

logs: ## Show logs (alias for docker-logs)
	@docker compose logs -f

logs-api: ## Show API logs only
	@docker compose logs -f api

logs-web: ## Show Web logs only
	@docker compose logs -f web

ps: ## Show running containers
	@docker compose ps

# ==============================================================================
# Database
# ==============================================================================

migrate-up: ## Run database migrations
	@docker compose exec api migrate -path /app/migrations -database "$${DATABASE_URL}" up
	@echo "✅ Migrations completed successfully"

migrate-down: ## Rollback last migration
	@docker compose exec api migrate -path /app/migrations -database "$${DATABASE_URL}" down 1

migrate-force: ## Force migration version (usage: make migrate-force VERSION=1)
	@docker compose exec api migrate -path /app/migrations -database "$${DATABASE_URL}" force $(VERSION)

seed: ## Seed database with sample data
	@docker compose exec api go run scripts/seed.go

db-shell: ## Open PostgreSQL shell
	@docker compose exec postgres psql -U postgres -d immobilier_saas

redis-cli: ## Open Redis CLI
	@docker compose exec redis redis-cli

# ==============================================================================
# Testing
# ==============================================================================

test: ## Run all tests
	@echo "Running API tests..."
	@cd services/api && make test
	@echo ""
	@echo "Running Web tests..."
	@cd services/web && npm test

test-api: ## Run API tests only
	@cd services/api && make test

test-web: ## Run Web tests only
	@cd services/web && npm test

test-coverage: ## Generate test coverage report
	@cd services/api && make test-coverage
	@cd services/web && npm run test:coverage

# ==============================================================================
# Linting & Formatting
# ==============================================================================

lint: ## Run linters for all services
	@echo "Linting API..."
	@cd services/api && make lint
	@echo ""
	@echo "Linting Web..."
	@cd services/web && npm run lint

lint-api: ## Lint API code
	@cd services/api && make lint

lint-web: ## Lint Web code
	@cd services/web && npm run lint

format: ## Format all code
	@echo "Formatting API..."
	@cd services/api && gofmt -w .
	@echo ""
	@echo "Formatting Web..."
	@cd services/web && npm run format

# ==============================================================================
# Build
# ==============================================================================

build: ## Build production binaries
	@echo "Building API..."
	@cd services/api && make build
	@echo ""
	@echo "Building Web..."
	@cd services/web && npm run build

build-api: ## Build API production binary
	@cd services/api && make build

build-web: ## Build Web production bundle
	@cd services/web && npm run build

# ==============================================================================
# Swagger Documentation
# ==============================================================================

swagger: ## Generate Swagger documentation
	@cd services/api && make swagger
	@echo "✅ Swagger docs generated at services/api/docs/"
	@echo "View at: http://localhost:8080/swagger/index.html"

# ==============================================================================
# Cleanup
# ==============================================================================

clean: ## Clean all build artifacts
	@echo "Cleaning API..."
	@cd services/api && make clean
	@echo ""
	@echo "Cleaning Web..."
	@cd services/web && rm -rf .next node_modules
	@echo ""
	@echo "✅ Cleanup complete"

clean-all: clean docker-down-volumes ## Clean everything including Docker volumes

# ==============================================================================
# Deployment
# ==============================================================================

deploy-staging: ## Deploy to staging environment
	@echo "Deploying to staging..."
	@cd services/api && flyctl deploy --app immobilier-api-staging
	@cd services/web && flyctl deploy --app immobilier-web-staging

deploy-prod: ## Deploy to production environment
	@echo "⚠️  Deploying to production..."
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		cd services/api && flyctl deploy --app immobilier-api-prod; \
		cd ../web && flyctl deploy --app immobilier-web-prod; \
	fi

# ==============================================================================
# Infrastructure
# ==============================================================================

terraform-init: ## Initialize Terraform
	@cd infrastructure/terraform/environments/production && terraform init

terraform-plan: ## Plan Terraform changes
	@cd infrastructure/terraform/environments/production && terraform plan

terraform-apply: ## Apply Terraform changes
	@cd infrastructure/terraform/environments/production && terraform apply

# ==============================================================================
# Utilities
# ==============================================================================

check-env: ## Check environment configuration
	@echo "Checking environment configuration..."
	@echo ""
	@echo "API .env:"
	@test -f services/api/.env && echo "✅ Exists" || echo "❌ Missing"
	@echo ""
	@echo "Web .env:"
	@test -f services/web/.env && echo "✅ Exists" || echo "❌ Missing"
	@echo ""
	@echo "JWT Keys:"
	@test -f services/api/private.pem && echo "✅ Private key exists" || echo "❌ Private key missing"
	@test -f services/api/public.pem && echo "✅ Public key exists" || echo "❌ Public key missing"

health: ## Check health of all services
	@echo "Checking service health..."
	@echo ""
	@curl -f http://localhost:8080/health && echo "✅ API is healthy" || echo "❌ API is down"
	@echo ""
	@curl -f http://localhost:3000/ > /dev/null 2>&1 && echo "✅ Web is healthy" || echo "❌ Web is down"

backup-db: ## Backup PostgreSQL database
	@mkdir -p backups
	@docker compose exec -T postgres pg_dump -U postgres immobilier_saas > backups/backup_$$(date +%Y%m%d_%H%M%S).sql
	@echo "✅ Database backed up to backups/"

restore-db: ## Restore PostgreSQL database (usage: make restore-db FILE=backups/backup_20240115_120000.sql)
	@test -f $(FILE) || (echo "❌ File not found: $(FILE)" && exit 1)
	@docker compose exec -T postgres psql -U postgres immobilier_saas < $(FILE)
	@echo "✅ Database restored from $(FILE)"

install: ## Install all dependencies
	@echo "Installing API dependencies..."
	@cd services/api && go mod download
	@echo ""
	@echo "Installing Web dependencies..."
	@cd services/web && npm install
	@echo ""
	@echo "✅ All dependencies installed"

update: ## Update all dependencies
	@echo "Updating API dependencies..."
	@cd services/api && go get -u ./...
	@cd services/api && go mod tidy
	@echo ""
	@echo "Updating Web dependencies..."
	@cd services/web && npm update
	@echo ""
	@echo "✅ All dependencies updated"

version: ## Show version information
	@echo "Immobilier SaaS Platform"
	@echo "Version: 1.0.0"
	@echo ""
	@echo "Go version:"
	@go version
	@echo ""
	@echo "Node version:"
	@node --version
	@echo ""
	@echo "Docker version:"
	@docker --version
	@echo ""
	@echo "Docker Compose version:"
	@docker compose --version

.DEFAULT_GOAL := help
