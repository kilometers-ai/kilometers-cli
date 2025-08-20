#!/bin/bash

# Demo Test Workflow
# Demonstrates the complete testing workflow for new developers

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Print functions
print_header() {
    echo ""
    echo -e "${CYAN}================================================================${NC}"
    echo -e "${CYAN} $1${NC}"
    echo -e "${CYAN}================================================================${NC}"
    echo ""
}

print_step() {
    echo -e "${BLUE}📋 Step $1: $2${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to wait for user input
wait_for_user() {
    echo ""
    read -p "Press Enter to continue..." -r
    echo ""
}

# Function to run command with logging
run_command() {
    local description="$1"
    local command="$2"
    local wait_after="${3:-true}"
    
    echo -e "${BLUE}Running: $description${NC}"
    echo -e "${CYAN}Command: $command${NC}"
    echo ""
    
    if eval "$command"; then
        print_success "$description completed successfully"
    else
        print_error "$description failed"
        echo ""
        echo "You can continue with the demo or exit to investigate the issue."
        read -p "Continue? (y/n): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    if [[ "$wait_after" == "true" ]]; then
        wait_for_user
    fi
}

# Main demo workflow
main() {
    print_header "KILOMETERS CLI TEST WORKFLOW DEMO"
    
    echo "This demo will guide you through the complete testing workflow."
    echo "It will:"
    echo "  1. Run quick development tests"
    echo "  2. Start a Docker environment"
    echo "  3. Run end-to-end tests"
    echo "  4. Clean up the environment"
    echo ""
    echo "Prerequisites:"
    echo "  - Go 1.21+ installed"
    echo "  - Docker installed and running"
    echo "  - curl and jq installed"
    echo ""
    
    wait_for_user
    
    # Step 1: Quick tests
    print_header "STEP 1: QUICK DEVELOPMENT TESTS"
    print_step "1" "Running quick development tests"
    print_info "These tests run quickly and give immediate feedback during development"
    
    run_command "Quick build test" "$SCRIPT_DIR/quick-test.sh build"
    run_command "Quick unit tests" "$SCRIPT_DIR/quick-test.sh unit" 
    run_command "Quick lint check" "$SCRIPT_DIR/quick-test.sh lint"
    
    # Step 2: Docker environment
    print_header "STEP 2: DOCKER ENVIRONMENT SETUP"
    print_step "2" "Starting Docker environment for integration testing"
    print_info "We'll use the 'shared' environment which provides the full API"
    
    run_command "Start Docker environment" "$SCRIPT_DIR/docker-compose-up.sh shared"
    
    # Step 3: API connectivity test
    print_header "STEP 3: API CONNECTIVITY TEST"
    print_step "3" "Testing API connectivity"
    print_info "Verifying that our API is accessible and healthy"
    
    run_command "Test API connectivity" "$SCRIPT_DIR/quick-test.sh api"
    
    # Step 4: End-to-End tests
    print_header "STEP 4: END-TO-END TESTING"
    print_step "4" "Running comprehensive E2E tests"
    print_info "This tests the complete plugin manifest and download workflow"
    print_info "Including: customer registration, JWT exchange, plugin installation, etc."
    
    run_command "Run E2E tests" "$SCRIPT_DIR/test-plugin-e2e.sh --verbose"
    
    # Step 5: Integration tests
    print_header "STEP 5: INTEGRATION TESTING"
    print_step "5" "Running integration tests"
    print_info "Testing various integration points and subsystems"
    
    run_command "Plugin integration tests" "$SCRIPT_DIR/test-plugin-integration.sh"
    
    # Step 6: Cleanup
    print_header "STEP 6: ENVIRONMENT CLEANUP"
    print_step "6" "Cleaning up Docker environment"
    print_info "Stopping all containers and cleaning up resources"
    
    run_command "Stop Docker environment" "$SCRIPT_DIR/docker-compose-down.sh shared"
    
    # Final summary
    print_header "DEMO COMPLETED SUCCESSFULLY!"
    
    echo "🎉 Congratulations! You've successfully run through the complete test workflow."
    echo ""
    echo "Here's what you've learned:"
    echo ""
    echo "📋 QUICK TESTS (for daily development):"
    echo "   ./scripts/tests/quick-test.sh          # All quick tests"
    echo "   ./scripts/tests/quick-test.sh build    # Just build verification"
    echo "   ./scripts/tests/quick-test.sh unit     # Just unit tests"
    echo ""
    echo "🐳 DOCKER ENVIRONMENT MANAGEMENT:"
    echo "   ./scripts/tests/docker-compose-up.sh shared    # Start environment"
    echo "   ./scripts/tests/docker-compose-down.sh shared  # Stop environment"
    echo ""
    echo "🧪 COMPREHENSIVE TESTING:"
    echo "   ./scripts/tests/test-plugin-e2e.sh             # E2E plugin tests"
    echo "   ./scripts/tests/run-full-test-suite.sh         # Complete test suite"
    echo ""
    echo "📚 DAILY DEVELOPMENT WORKFLOW:"
    echo "   1. ./scripts/tests/quick-test.sh               # Quick feedback"
    echo "   2. ./scripts/tests/docker-compose-up.sh shared # Start environment"
    echo "   3. ./scripts/tests/test-plugin-e2e.sh          # Full E2E validation"
    echo "   4. ./scripts/tests/docker-compose-down.sh shared # Clean up"
    echo ""
    echo "For more information, see: scripts/tests/README.md"
    echo ""
    print_success "Happy testing! 🚀"
}

# Check prerequisites
check_prerequisites() {
    local missing_tools=()
    
    if ! command -v go >/dev/null 2>&1; then
        missing_tools+=("go")
    fi
    
    if ! command -v docker >/dev/null 2>&1; then
        missing_tools+=("docker")
    fi
    
    if ! command -v curl >/dev/null 2>&1; then
        missing_tools+=("curl")
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        missing_tools+=("jq")
    fi
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        print_error "Missing required tools: ${missing_tools[*]}"
        echo ""
        echo "Please install the missing tools:"
        echo "  - Go: https://golang.org/doc/install"
        echo "  - Docker: https://docs.docker.com/get-docker/"
        echo "  - curl: Usually pre-installed or via package manager"
        echo "  - jq: brew install jq (macOS) or apt-get install jq (Linux)"
        exit 1
    fi
}

# Check if Docker is running
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker is not running"
        echo ""
        echo "Please start Docker and try again."
        exit 1
    fi
}

# Show usage
show_usage() {
    echo "Demo Test Workflow for Kilometers CLI"
    echo ""
    echo "Usage: $0 [--help]"
    echo ""
    echo "This script demonstrates the complete testing workflow including:"
    echo "  - Quick development tests"
    echo "  - Docker environment management"
    echo "  - End-to-end testing"
    echo "  - Integration testing"
    echo "  - Environment cleanup"
    echo ""
    echo "The demo is interactive and will wait for your input at each step."
}

# Handle arguments
if [[ $# -gt 0 ]]; then
    case "$1" in
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
fi

# Run prerequisite checks
check_prerequisites
check_docker

# Run the demo
main