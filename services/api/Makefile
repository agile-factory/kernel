.PHONY: help install dev build test lint swagger migrate-up migrate-down docker-up docker-down clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install dependencies
	go mod download
	go install github.com/cosmtrek/air@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/swaggo/swag/cmd/swag@latest

dev: ## Run development server with hot reload
	air -c .air.toml

build: ## Build production binary
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-w -s" \
		-o bin/api \
		./cmd/api/main.go

test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests and show coverage
	go tool cover -html=coverage.out

lint: ## Run linter
	golangci-lint run --timeout=5m

swagger: ## Generate Swagger documentation
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

migrate-create: ## Create new migration (usage: make migrate-create name=<migration_name>)
	migrate create -ext sql -dir migrations -seq $(name)

migrate-up: ## Run database migrations up
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down: ## Run database migrations down
	migrate -path migrations -database "$(DATABASE_URL)" down

migrate-force: ## Force migration version (usage: make migrate-force version=<version>)
	migrate -path migrations -database "$(DATABASE_URL)" force $(version)

docker-build: ## Build Docker image
	docker build -t immobilier-api:latest .

docker-up: ## Start all services with docker-compose
	cd ../.. && docker-compose up -d

docker-down: ## Stop all services
	cd ../.. && docker-compose down

docker-logs: ## Show logs from docker-compose
	cd ../.. && docker-compose logs -f api

clean: ## Clean build artifacts
	rm -rf bin/ tmp/ coverage.out

seed: ## Seed database with sample data
	go run scripts/seed.go
