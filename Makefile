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

benchmark: ## Run all benchmarks with memory tracking
	@echo "Running comprehensive benchmarks..."
	@mkdir -p benchmarks/results
	@go test -bench=. -benchmem -benchtime=10s ./benchmarks/... | tee benchmarks/results/$(shell date +%Y%m%d_%H%M%S).txt
	@echo "Benchmark results saved to benchmarks/results/"

benchmark-quick: ## Run quick benchmarks (shorter duration)
	@echo "Running quick benchmarks..."
	@go test -bench=. -benchmem -benchtime=1s ./benchmarks/...

benchmark-compare: ## Compare with baseline (usage: make benchmark-compare)
	@echo "Comparing with baseline..."
	@if [ -f benchmarks/results/baseline.txt ]; then \
		benchstat benchmarks/results/baseline.txt benchmarks/results/latest.txt; \
	else \
		echo "Error: baseline.txt not found. Run 'make benchmark-baseline' first."; \
	fi

benchmark-baseline: ## Set current benchmarks as baseline
	@echo "Setting baseline benchmarks..."
	@mkdir -p benchmarks/results
	@go test -bench=. -benchmem -benchtime=10s ./benchmarks/... > benchmarks/results/baseline.txt
	@echo "Baseline saved to benchmarks/results/baseline.txt"

benchmark-mem: ## Run benchmarks with memory profiling
	@echo "Running memory profiling..."
	@mkdir -p benchmarks/profiles
	@go test -bench=. -benchmem -memprofile=benchmarks/profiles/mem.prof -benchtime=10s ./benchmarks/...
	@echo "Memory profile saved. View with: go tool pprof benchmarks/profiles/mem.prof"

benchmark-cpu: ## Run benchmarks with CPU profiling
	@echo "Running CPU profiling..."
	@mkdir -p benchmarks/profiles
	@go test -bench=. -cpuprofile=benchmarks/profiles/cpu.prof -benchtime=10s ./benchmarks/...
	@echo "CPU profile saved. View with: go tool pprof benchmarks/profiles/cpu.prof"

benchmark-view-mem: ## View memory profile in browser
	@go tool pprof -http=:8080 benchmarks/profiles/mem.prof

benchmark-view-cpu: ## View CPU profile in browser
	@go tool pprof -http=:8080 benchmarks/profiles/cpu.prof

benchmark-ci: ## Run benchmarks for CI with multiple iterations
	@echo "Running CI benchmarks..."
	@mkdir -p benchmarks/results
	@go test -bench=. -benchmem -count=5 -benchtime=5s ./benchmarks/... > benchmarks/results/ci_$(shell git rev-parse --short HEAD 2>/dev/null || echo "local").txt
	@echo "CI benchmark results saved"

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
	@go install golang.org/x/perf/cmd/benchstat@latest
	@echo "All development tools installed successfully"

.DEFAULT_GOAL := help
