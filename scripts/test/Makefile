# Makefile for install script testing

.PHONY: help test test-all test-platform clean setup build-images start-server stop-server logs

# Default target
help:
	@echo "Install Script Testing"
	@echo "======================"
	@echo ""
	@echo "Usage:"
	@echo "  make test              # Run tests on all platforms"
	@echo "  make test-platform PLATFORM=ubuntu  # Test specific platform"
	@echo "  make test-mode MODE=timeout         # Test with specific mode"
	@echo "  make setup             # Setup test environment"
	@echo "  make build-images      # Build all Docker images"
	@echo "  make start-server      # Start mock server"
	@echo "  make stop-server       # Stop mock server"
	@echo "  make clean             # Clean up test environment"
	@echo "  make logs              # Show test logs"
	@echo ""
	@echo "Available platforms: ubuntu, alpine, debian, fedora"
	@echo "Available modes: normal, timeout, rate_limit, server_error,"
	@echo "                malformed_json, corrupted_binary, missing_binary"

# Variables
PLATFORM ?=
MODE ?= normal
DOCKER_COMPOSE = docker-compose -f docker-compose.yml

# Setup test environment
setup:
	@echo "Setting up test environment..."
	mkdir -p results
	$(DOCKER_COMPOSE) build mock-server
	@echo "Test environment ready!"

# Build all Docker images
build-images: setup
	@echo "Building test images..."
	$(DOCKER_COMPOSE) build
	@echo "All images built!"

# Run tests on all platforms
test: setup
	@echo "Running tests on all platforms..."
	./test-install-docker.sh --mode $(MODE) --verbose

# Run tests on all platforms using Docker Compose
test-compose: build-images
	@echo "Running tests with Docker Compose..."
	TEST_MODE=$(MODE) $(DOCKER_COMPOSE) --profile test up --abort-on-container-exit --remove-orphans

# Test specific platform
test-platform: setup
ifndef PLATFORM
	@echo "Error: PLATFORM not specified"
	@echo "Usage: make test-platform PLATFORM=ubuntu"
	@exit 1
endif
	@echo "Testing platform: $(PLATFORM)"
	./test-install-docker.sh --platform $(PLATFORM) --mode $(MODE) --verbose

# Start mock server only
start-server:
	@echo "Starting mock server..."
	TEST_MODE=$(MODE) $(DOCKER_COMPOSE) up -d mock-server
	@echo "Mock server started on http://localhost:8080"

# Stop mock server
stop-server:
	@echo "Stopping mock server..."
	$(DOCKER_COMPOSE) stop mock-server
	$(DOCKER_COMPOSE) rm -f mock-server

# Show logs
logs:
	@echo "=== Test Results ==="
	@if [ -d "results" ]; then \
		for log in results/*.log; do \
			if [ -f "$$log" ]; then \
				echo ""; \
				echo "--- $$(basename $$log) ---"; \
				tail -20 "$$log"; \
			fi; \
		done; \
	else \
		echo "No test results found. Run 'make test' first."; \
	fi

# Clean up test environment
clean:
	@echo "Cleaning up test environment..."
	$(DOCKER_COMPOSE) down --volumes --remove-orphans
	docker system prune -f --filter "label=com.docker.compose.project=test"
	rm -rf results/*.log
	@echo "Clean up complete!"

# Clean everything including images
clean-all: clean
	@echo "Removing all test images..."
	-docker rmi km-test-ubuntu km-test-alpine km-test-debian km-test-fedora km-mock-server 2>/dev/null
	@echo "All images removed!"

# Interactive development
dev:
	@echo "Starting development environment..."
	$(DOCKER_COMPOSE) up -d mock-server
	@echo "Mock server available at http://localhost:8080"
	@echo "Use 'make test-platform PLATFORM=ubuntu' to run tests"

# Show server status
status:
	@echo "Service status:"
	$(DOCKER_COMPOSE) ps
