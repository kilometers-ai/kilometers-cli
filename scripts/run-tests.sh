#!/bin/bash

# Test runner script with fail-fast behavior and non-interactive mode
# This script ensures tests don't hang on stdin and fail immediately when issues are found

set -euo pipefail  # Exit on error, undefined vars, and pipe failures

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_TIMEOUT=${TEST_TIMEOUT:-120}  # Default timeout of 2 minutes
VERBOSE=${VERBOSE:-false}
RACE_DETECTION=${RACE_DETECTION:-true}
COVERAGE=${COVERAGE:-false}

# Test result tracking
TESTS_PASSED=0
TESTS_FAILED=0
START_TIME=$(date +%s)

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

print_error() {
    print_status "$RED" "❌ $1"
}

print_success() {
    print_status "$GREEN" "✅ $1"
}

print_info() {
    print_status "$BLUE" "ℹ️  $1"
}

print_warning() {
    print_status "$YELLOW" "⚠️  $1"
}

# Function to run a test suite with timeout and proper error handling
run_test_suite() {
    local test_name="$1"
    local test_path="$2"
    local extra_flags="${3:-}"
    
    print_info "Running $test_name tests..."
    
    # Build test command with non-interactive flags
    local cmd="go test"
    
    # Add timeout
    cmd="$cmd -timeout ${TEST_TIMEOUT}s"
    
    # Add race detection if enabled
    if [[ "$RACE_DETECTION" == "true" ]]; then
        cmd="$cmd -race"
    fi
    
    # Add coverage if enabled
    if [[ "$COVERAGE" == "true" ]]; then
        local coverage_file="coverage-$(echo "$test_name" | tr ' ' '_' | tr '/' '_').out"
        cmd="$cmd -coverprofile=$coverage_file -covermode=atomic"
    fi
    
    # Add verbose if enabled
    if [[ "$VERBOSE" == "true" ]]; then
        cmd="$cmd -v"
    fi
    
    # Add extra flags
    if [[ -n "$extra_flags" ]]; then
        cmd="$cmd $extra_flags"
    fi
    
    # Add test path
    cmd="$cmd $test_path"
    
    # Run with timeout protection and non-interactive stdin
    if timeout "${TEST_TIMEOUT}s" bash -c "$cmd < /dev/null"; then
        print_success "$test_name tests passed"
        ((TESTS_PASSED++))
        return 0
    else
        local exit_code=$?
        if [[ $exit_code -eq 124 ]]; then
            print_error "$test_name tests timed out after ${TEST_TIMEOUT}s"
        else
            print_error "$test_name tests failed with exit code $exit_code"
        fi
        ((TESTS_FAILED++))
        return 1
    fi
}

# Function to check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    # Check Go installation
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check Go version
    local go_version=$(go version | cut -d' ' -f3)
    print_info "Using Go version: $go_version"
    
    # Verify go.mod exists
    if [[ ! -f "go.mod" ]]; then
        print_error "go.mod not found. Are you in the project root?"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Function to setup test environment
setup_test_environment() {
    print_info "Setting up test environment..."
    
    # Ensure we're in non-interactive mode
    export CI=true
    export TERM=dumb
    export NO_COLOR=1
    
    # Disable any interactive prompts
    export DEBIAN_FRONTEND=noninteractive
    export KM_TEST_MODE=true
    export KM_DEBUG=false
    
    # Clean up any previous test artifacts
    find . -name "*.test" -type f -delete 2>/dev/null || true
    find . -name "coverage*.out" -type f -delete 2>/dev/null || true
    
    print_success "Test environment setup complete"
}

# Function to run go mod commands
verify_dependencies() {
    print_info "Verifying dependencies..."
    
    # Download dependencies with timeout
    if ! timeout 60s go mod download; then
        print_error "Failed to download dependencies"
        exit 1
    fi
    
    # Verify dependencies
    if ! go mod verify; then
        print_error "Dependency verification failed"
        exit 1
    fi
    
    print_success "Dependencies verified"
}

# Function to run static analysis
run_static_analysis() {
    print_info "Running static analysis..."
    
    # Run go vet
    if ! go vet ./...; then
        print_error "go vet failed"
        return 1
    fi
    
    # Run go fmt check
    local fmt_output
    fmt_output=$(go fmt ./... 2>&1) || true
    local fmt_issues=$(echo "$fmt_output" | wc -l)
    if [[ $fmt_issues -gt 1 ]]; then  # wc -l always returns at least 1 for empty input
        print_error "Code formatting issues found. Run 'go fmt ./...' to fix."
        return 1
    fi
    
    print_success "Static analysis passed"
}

# Function to display final results
display_results() {
    local end_time=$(date +%s)
    local duration=$((end_time - START_TIME))
    
    echo ""
    print_info "Test Results Summary"
    echo "===================="
    echo "Total duration: ${duration}s"
    echo "Suites passed: $TESTS_PASSED"
    echo "Suites failed: $TESTS_FAILED"
    echo ""
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        print_error "Some tests failed. Deployment blocked."
        return 1
    else
        print_success "All tests passed! Ready for deployment."
        return 0
    fi
}

# Function to cleanup on exit
cleanup() {
    local exit_code=$?
    
    # Temporarily disable strict error checking for cleanup
    set +e
    
    print_info "Cleaning up test artifacts..."
    
    # Kill any background processes that might be hanging
    pkill -f "mock.*server" 2>/dev/null || true
    pkill -f "go test" 2>/dev/null || true
    
    # Clean up temporary files
    find . -name "*.test" -type f -delete 2>/dev/null || true
    
    # Don't report cleanup as error - only report actual test failures
    # The exit code will speak for itself
    
    # Preserve the original exit code without modification
    exit $exit_code
}

# Set trap for cleanup
trap cleanup EXIT INT TERM

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS] [TEST_PATHS...]"
    echo ""
    echo "Options:"
    echo "  -v, --verbose          Enable verbose test output"
    echo "  -c, --coverage         Enable test coverage collection"
    echo "  --no-race             Disable race condition detection"
    echo "  -t, --timeout SECONDS Set test timeout (default: 120)"
    echo "  -h, --help            Show this help message"
    echo ""
    echo "Arguments:"
    echo "  TEST_PATHS            Optional test paths (e.g., ./internal/core/session/...)"
    echo "                        If not provided, runs all default test suites"
    echo ""
    echo "Environment variables:"
    echo "  TEST_TIMEOUT          Test timeout in seconds (default: 120)"
    echo "  VERBOSE               Enable verbose output (true/false)"
    echo "  RACE_DETECTION        Enable race detection (true/false)"
    echo "  COVERAGE              Enable coverage collection (true/false)"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Run all tests"
    echo "  $0 --verbose                         # Run all tests with verbose output"
    echo "  $0 --coverage ./internal/core/...    # Run core tests with coverage"
    echo "  $0 -t 60 ./integration_test/...      # Run integration tests with 60s timeout"
}

# Parse command line arguments
parse_args() {
    CUSTOM_TEST_PATHS=()
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -c|--coverage)
                COVERAGE=true
                shift
                ;;
            --no-race)
                RACE_DETECTION=false
                shift
                ;;
            -t|--timeout)
                TEST_TIMEOUT="$2"
                shift 2
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            -*)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
            *)
                # Positional argument - treat as test path
                CUSTOM_TEST_PATHS+=("$1")
                shift
                ;;
        esac
    done
}

# Main function
main() {
    parse_args "$@"
    
    print_info "Starting comprehensive test suite"
    echo "=================================="
    echo "Configuration:"
    echo "  Timeout: ${TEST_TIMEOUT}s"
    echo "  Verbose: $VERBOSE"
    echo "  Race detection: $RACE_DETECTION"
    echo "  Coverage: $COVERAGE"
    echo ""
    
    # Run all checks and tests
    check_prerequisites
    setup_test_environment
    verify_dependencies
    
    # Run static analysis
    if ! run_static_analysis; then
        print_error "Static analysis failed"
        exit 1
    fi
    
    # Run test suites
    if [[ ${#CUSTOM_TEST_PATHS[@]} -gt 0 ]]; then
        # Run custom test paths provided by user
        for test_path in "${CUSTOM_TEST_PATHS[@]}"; do
            local test_name=$(basename "$test_path")
            run_test_suite "$test_name" "$test_path"
        done
    else
        # Run default test suites in order of increasing complexity
        run_test_suite "Core Event" "./internal/core/event/..."
        run_test_suite "Core Filtering" "./internal/core/filtering/..."
        run_test_suite "Core Session" "./internal/core/session/..."
        run_test_suite "Core Risk" "./internal/core/risk/..."
        run_test_suite "Infrastructure" "./internal/infrastructure/..."
        run_test_suite "Application Services" "./internal/application/..."
        run_test_suite "CLI Interface" "./internal/interfaces/..."
        # Run integration tests with shorter timeout to prevent hanging
        ORIGINAL_TIMEOUT=$TEST_TIMEOUT
        TEST_TIMEOUT=30
        run_test_suite "Integration Tests" "./integration_test/..." "-count=1"
        TEST_TIMEOUT=$ORIGINAL_TIMEOUT
    fi
    
    # Generate coverage report if enabled
    if [[ "$COVERAGE" == "true" ]]; then
        print_info "Generating coverage report..."
        
        # Temporarily disable strict error checking for coverage generation
        set +e
        
        echo "mode: atomic" > coverage.out
        
        # Find and combine coverage files, handling the case where none exist
        local coverage_files_found=false
        if find . -name "coverage-*.out" -type f 2>/dev/null | head -1 | grep -q . 2>/dev/null; then
            coverage_files_found=true
            find . -name "coverage-*.out" -type f -exec tail -n +2 {} \; >> coverage.out 2>/dev/null
        fi
        
        # Generate coverage report only if we have coverage data
        if [[ "$coverage_files_found" == "true" ]] && [[ -s coverage.out ]]; then
            go tool cover -html=coverage.out -o coverage.html 2>/dev/null || true
            local total_coverage
            total_coverage=$(go tool cover -func=coverage.out 2>/dev/null | grep total | awk '{print $3}' 2>/dev/null || echo "N/A")
            print_info "Total test coverage: $total_coverage"
        else
            print_warning "No coverage data collected"
        fi
        
        # Re-enable strict error checking
        set -e
    fi
    
    # Display results and exit with appropriate code
    if display_results; then
        exit 0
    else
        exit 1
    fi
}

# Run main function with all arguments
main "$@" 