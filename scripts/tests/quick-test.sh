#!/bin/bash

# Quick Test Runner for Daily Development
# Runs essential tests quickly for rapid feedback during development

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    local color="$1"
    local message="$2"
    echo -e "${color}${message}${NC}"
}

print_info() { print_status "$BLUE" "ℹ️  $1"; }
print_success() { print_status "$GREEN" "✅ $1"; }
print_warning() { print_status "$YELLOW" "⚠️  $1"; }
print_error() { print_status "$RED" "❌ $1"; }

# Function to show usage
show_usage() {
    echo "Quick Test Runner for Kilometers CLI"
    echo ""
    echo "Usage: $0 [test-type]"
    echo ""
    echo "Test Types:"
    echo "  build      - Quick build verification"
    echo "  unit       - Run unit tests only"
    echo "  lint       - Run linting and format checks"
    echo "  api        - Test API connectivity (requires Docker environment)"
    echo "  plugins    - Quick plugin system test"
    echo "  all        - Run all quick tests (default)"
    echo ""
    echo "Examples:"
    echo "  $0              # Run all quick tests"
    echo "  $0 build        # Just verify build works"
    echo "  $0 unit         # Run unit tests only"
    echo "  $0 api          # Test API connectivity"
}

# Function to check if Docker environment is running
check_docker_env() {
    if curl -s --connect-timeout 2 http://localhost:5194/health >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function to test build
test_build() {
    print_info "Testing build..."
    cd "$REPO_ROOT"
    
    if go build -o km-test ./cmd/main.go; then
        print_success "Build test passed"
        rm -f km-test
        return 0
    else
        print_error "Build test failed"
        return 1
    fi
}

# Function to run unit tests
test_unit() {
    print_info "Running unit tests..."
    cd "$REPO_ROOT"
    
    if go test -short ./internal/... -timeout 30s; then
        print_success "Unit tests passed"
        return 0
    else
        print_error "Unit tests failed"
        return 1
    fi
}

# Function to run linting
test_lint() {
    print_info "Running lint checks..."
    cd "$REPO_ROOT"
    
    # Check if gofmt is needed
    if ! gofmt -l . | grep -v "^$"; then
        print_success "Go format check passed"
    else
        print_warning "Some files need formatting (run: go fmt ./...)"
    fi
    
    # Check if go vet passes
    if go vet ./...; then
        print_success "Go vet passed"
        return 0
    else
        print_error "Go vet failed"
        return 1
    fi
}

# Function to test API connectivity
test_api() {
    print_info "Testing API connectivity..."
    
    if check_docker_env; then
        print_success "API is accessible at http://localhost:5194"
        
        # Test health endpoint
        local health_response
        if health_response=$(curl -s http://localhost:5194/health); then
            print_success "API health check passed"
            print_info "Response: $health_response"
        else
            print_warning "API health check failed"
        fi
        return 0
    else
        print_warning "API not accessible - Docker environment may not be running"
        print_info "To start: ./scripts/tests/docker-compose-up.sh shared"
        return 1
    fi
}

# Function to test plugin system
test_plugins() {
    print_info "Testing plugin system..."
    cd "$REPO_ROOT"
    
    # Test plugin interfaces compile
    if go build ./internal/plugins/...; then
        print_success "Plugin system compiles"
    else
        print_error "Plugin system compilation failed"
        return 1
    fi
    
    # Test plugin discovery
    if go test -short ./internal/plugins/... -timeout 15s; then
        print_success "Plugin tests passed"
        return 0
    else
        print_error "Plugin tests failed"
        return 1
    fi
}

# Function to run all quick tests
test_all() {
    local failed_tests=0
    
    print_info "Running all quick tests..."
    echo ""
    
    test_build || ((failed_tests++))
    echo ""
    
    test_unit || ((failed_tests++))
    echo ""
    
    test_lint || ((failed_tests++))
    echo ""
    
    test_plugins || ((failed_tests++))
    echo ""
    
    # API test is optional
    test_api
    echo ""
    
    # Summary
    if [[ $failed_tests -eq 0 ]]; then
        print_success "All quick tests passed! 🎉"
        return 0
    else
        print_error "$failed_tests test(s) failed"
        return 1
    fi
}

# Main execution
main() {
    local test_type="${1:-all}"
    
    case "$test_type" in
        build)
            test_build
            ;;
        unit)
            test_unit
            ;;
        lint)
            test_lint
            ;;
        api)
            test_api
            ;;
        plugins)
            test_plugins
            ;;
        all)
            test_all
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown test type: $test_type"
            show_usage
            exit 1
            ;;
    esac
}

# Check prerequisites
if ! command -v go >/dev/null 2>&1; then
    print_error "Go is required but not installed"
    exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
    print_error "curl is required but not installed"
    exit 1
fi

# Execute main function
main "$@"