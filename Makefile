.PHONY: help run build test clean migrate-up migrate-down install

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install dependencies
	go mod download
	go mod verify

run: ## Run the application
	go run cmd/server/main.go

build: ## Build the application
	go build -o bin/server cmd/server/main.go

test: ## Run tests
	go test -v ./...

test-cover: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out

migrate-up: ## Run database migrations
	@echo "Running migrations..."
	psql -U $(DB_USER) -d $(DB_NAME) -f migrations/000_master_schema.sql

migrate-down: ## Rollback database migrations (manual)
	@echo "Please manually rollback migrations"

dev: ## Run with live reload (requires air)
	air

docker-build: ## Build Docker image
	docker build -t splitter:latest .

docker-run: ## Run Docker container
	docker run -p 8080:8080 --env-file .env splitter:latest

lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	go fmt ./...
	goimports -w .

tidy: ## Tidy dependencies
	go mod tidy
