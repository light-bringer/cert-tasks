.PHONY: help build run test test-coverage test-race lint fmt clean install-deps

# Variables
BINARY_NAME=api
BINARY_PATH=bin/$(BINARY_NAME)
CMD_PATH=./cmd/api
GO=go
GOTEST=$(GO) test
GOVET=$(GO) vet
GOFMT=gofmt

# Default target
.DEFAULT_GOAL := help

help: ## Display this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	@$(GO) build -o $(BINARY_PATH) $(CMD_PATH)
	@echo "Build complete: $(BINARY_PATH)"

run: build ## Build and run the application
	@echo "Starting server..."
	@./$(BINARY_PATH)

run-dev: ## Run the application without building (using go run)
	@echo "Running in development mode..."
	@$(GO) run $(CMD_PATH)

test: ## Run unit tests
	@echo "Running unit tests..."
	@$(GOTEST) -v ./cmd/... ./internal/...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@$(GOTEST) -coverprofile=coverage.out ./cmd/... ./internal/...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	@$(GOTEST) -race ./cmd/... ./internal/...

test-integration: ## Run integration tests (requires server running on localhost:8080)
	@echo "Running integration tests..."
	@$(GOTEST) -v ./test/...

test-integration-standalone: build ## Build, run server, execute integration tests, then stop server
	@echo "Starting server in background..."
	@./$(BINARY_PATH) & echo $$! > .server.pid
	@sleep 2
	@echo "Running integration tests..."
	@$(GOTEST) -v ./test/... || (kill `cat .server.pid` 2>/dev/null; rm -f .server.pid; exit 1)
	@echo "Stopping server..."
	@kill `cat .server.pid` 2>/dev/null || true
	@rm -f .server.pid
	@echo "Integration tests complete"

test-all: test test-integration-standalone ## Run all tests (unit + integration)

test-short: ## Run tests in short mode
	@$(GOTEST) -short ./...

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@$(GOTEST) -bench=. -benchmem ./...

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: brew install golangci-lint" && exit 1)
	@golangci-lint run

fmt: ## Format Go code
	@echo "Formatting code..."
	@$(GOFMT) -w .
	@$(GO) mod tidy
	@echo "Code formatted"

vet: ## Run go vet
	@echo "Running go vet..."
	@$(GOVET) ./...

check: fmt vet lint test ## Run all checks (format, vet, lint, unit tests)

clean: ## Remove build artifacts and test outputs
	@echo "Cleaning..."
	@rm -rf $(BINARY_PATH) bin/ coverage.* *.prof *.out
	@echo "Clean complete"

install-deps: ## Install/update dependencies
	@echo "Installing dependencies..."
	@$(GO) mod download
	@$(GO) mod tidy
	@$(GO) mod verify
	@echo "Dependencies installed"

deps-update: ## Update all dependencies
	@echo "Updating dependencies..."
	@$(GO) get -u ./...
	@$(GO) mod tidy
	@echo "Dependencies updated"

test-api: ## Run Python API test script (requires requests library) - DEPRECATED: use test-integration
	@echo "Running Python API tests (deprecated - use 'make test-integration' instead)..."
	@which python3 > /dev/null || (echo "python3 not installed" && exit 1)
	@python3 test_api.py

run-test-api: ## Start server and run API tests
	@echo "Starting server in background..."
	@./$(BINARY_PATH) & echo $$! > .server.pid
	@sleep 2
	@$(MAKE) test-api
	@kill `cat .server.pid` && rm .server.pid

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@$(GO) install golang.org/x/tools/cmd/goimports@latest
	@$(GO) install honnef.co/go/tools/cmd/staticcheck@latest
	@$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "Tools installed"

security: ## Run security vulnerability check
	@echo "Running security check..."
	@which govulncheck > /dev/null || (echo "Installing govulncheck..." && $(GO) install golang.org/x/vuln/cmd/govulncheck@latest)
	@govulncheck ./...

docker-build: ## Build Docker image (if Dockerfile exists)
	@echo "Building Docker image..."
	@docker build -t cert-tasks:latest .

docker-run: ## Run Docker container (if Dockerfile exists)
	@echo "Running Docker container..."
	@docker run -p 8080:8080 cert-tasks:latest

docker-compose-up: ## Start services using Docker Compose
	@echo "Starting Docker Compose services..."
	@docker-compose up

docker-compose-down: ## Stop and remove Docker Compose services
	@echo "Stopping Docker Compose services..."
	@docker-compose down

docker-compose-logs: ## Follow Docker Compose logs
	@docker-compose logs -f api

all: clean install-deps fmt vet test build ## Clean, install deps, format, vet, unit tests, and build
