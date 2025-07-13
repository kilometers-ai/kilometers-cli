# Kilometers CLI - Makefile
# Provides convenient commands for development and testing

.PHONY: help test test-verbose test-coverage test-fast test-integration clean build install lint

# Default target
help: ## Show this help message
	@echo "Kilometers CLI - Available Commands:"
	@echo "===================================="
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Test commands
test: ## Run all tests with default settings
	@./scripts/run-tests.sh

test-verbose: ## Run all tests with verbose output
	@./scripts/run-tests.sh --verbose

test-coverage: ## Run all tests with coverage report
	@./scripts/run-tests.sh --coverage --verbose

test-fast: ## Run tests without race detection (faster)
	@./scripts/run-tests.sh --no-race

test-integration: ## Run only integration tests
	@./scripts/run-tests.sh --verbose -t 180 ./integration_test/...

test-unit: ## Run only unit tests (excludes integration tests)
	@go test -timeout 60s -race ./internal/...

# Development commands
clean: ## Clean build artifacts and test files
	@echo "🧹 Cleaning up..."
	@go clean -testcache
	@rm -f coverage*.out coverage*.html
	@find . -name "*.test" -type f -delete
	@echo "✅ Cleanup complete"

build: ## Build the CLI binary
	@echo "🔨 Building km CLI..."
	@go build -o km cmd/main.go
	@echo "✅ Build complete: ./km"

install: build ## Build and install the CLI to GOPATH/bin
	@echo "📦 Installing km CLI..."
	@go install ./cmd
	@echo "✅ Install complete"

lint: ## Run code linting and formatting checks
	@echo "🔍 Running linters..."
	@go vet ./...
	@go fmt ./...
	@echo "✅ Linting complete"

# CI/CD commands
ci-test: ## Run tests as they would run in CI
	@echo "🚀 Running CI tests..."
	@export CI=true && ./scripts/run-tests.sh --coverage

# Quick verification before commit
pre-commit: lint test-fast ## Run quick checks before committing
	@echo "🎯 Pre-commit checks passed!"

# Full verification before deployment
pre-deploy: lint test-coverage ## Run comprehensive checks before deployment
	@echo "🚀 Ready for deployment!"

# Development setup
dev-setup: ## Set up development environment
	@echo "⚙️  Setting up development environment..."
	@go mod download
	@go mod verify
	@chmod +x scripts/*.sh
	@echo "✅ Development environment ready"

# Show test status
test-status: ## Show current test results without running tests
	@echo "📊 Test Status:"
	@echo "==============="
	@if [ -f coverage.out ]; then \
		echo "📈 Coverage: $$(go tool cover -func=coverage.out | grep total | awk '{print $$3}')"; \
	else \
		echo "📈 Coverage: Not available (run 'make test-coverage')"; \
	fi
	@echo "🕒 Last test run: $$(stat -f '%Sm' coverage.out 2>/dev/null || echo 'Never')" 