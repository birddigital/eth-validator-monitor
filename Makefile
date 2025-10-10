.PHONY: help build test run clean migrate-up migrate-down docker-up docker-down lint

# Variables
BINARY_NAME=eth-validator-monitor
SERVER_BINARY=bin/server
CLI_BINARY=bin/cli
MIGRATION_DIR=migrations

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the application binaries
	@echo "Building server..."
	@go build -o $(SERVER_BINARY) ./cmd/server
	@echo "Building CLI..."
	@go build -o $(CLI_BINARY) ./cmd/cli

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage summary:"
	@go tool cover -func=coverage.out | grep total

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	@go test -v -short ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -tags=integration ./tests/integration/...

test-e2e: ## Run end-to-end tests
	@echo "Running E2E tests..."
	@go test -v -tags=e2e ./tests/e2e/...

test-coverage: ## Generate detailed coverage report
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
	@go tool cover -func=coverage.out

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

run: ## Run the server
	@echo "Running server..."
	@go run ./cmd/server

run-cli: ## Run the CLI
	@echo "Running CLI..."
	@go run ./cmd/cli

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

lint: ## Run linters
	@echo "Running linters..."
	@golangci-lint run
	@go vet ./...

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

migrate-up: ## Run database migrations up
	@echo "Running migrations up..."
	@migrate -path $(MIGRATION_DIR) -database "${DATABASE_URL}" up

migrate-down: ## Run database migrations down
	@echo "Running migrations down..."
	@migrate -path $(MIGRATION_DIR) -database "${DATABASE_URL}" down

migrate-create: ## Create a new migration (usage: make migrate-create NAME=migration_name)
	@echo "Creating migration: $(NAME)"
	@migrate create -ext sql -dir $(MIGRATION_DIR) -seq $(NAME)

docker-up: ## Start Docker containers
	@echo "Starting Docker containers..."
	@docker-compose up -d

docker-down: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	@docker-compose down

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):latest -f docker/Dockerfile .

generate: ## Generate code (GraphQL, mocks, etc.)
	@echo "Generating code..."
	@go generate ./...

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/99designs/gqlgen@latest
	@go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest

.DEFAULT_GOAL := help
