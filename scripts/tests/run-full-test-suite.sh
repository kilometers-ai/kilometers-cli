#!/bin/bash

# Full Test Suite Runner for Kilometers CLI
# Runs complete test suite including unit tests, integration tests, and E2E tests

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TIMESTAMP=$(date '+%Y%m%d-%H%M%S')
LOG_DIR="$REPO_ROOT/logs"
LOG_FILE="$LOG_DIR/full-test-suite-$TIMESTAMP.log"

# Test configuration
DEFAULT_ENVIRONMENT="shared"
CLEANUP_AFTER="true"
SKIP_E2E="false"
SKIP_INTEGRATION="false"
SKIP_UNIT="false"
VERBOSE="false"

# Test results tracking
UNIT_TESTS_PASSED=0
INTEGRATION_TESTS_PASSED=0
E2E_TESTS_PASSED=0
TOTAL_FAILURES=0

# Ensure log directory exists
mkdir -p "$LOG_DIR"

# Logging functions
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] [$level] $message" | tee -a "$LOG_FILE"
}

log_info() { log "INFO" "$@"; }
log_warn() { log "WARN" "$@"; }
log_error() { log "ERROR" "$@"; }
log_success() { log "SUCCESS" "$@"; }

# Function to show usage
show_usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -e, --environment ENV    Docker environment to use (default: $DEFAULT_ENVIRONMENT)"
    echo "  --skip-unit             Skip unit tests"
    echo "  --skip-integration      Skip integration tests"
    echo "  --skip-e2e              Skip end-to-end tests"
    echo "  --no-cleanup            Don't clean up Docker environment after tests"
    echo "  -v, --verbose           Enable verbose output"
    echo "  -h, --help              Show this help"
    echo ""
    echo "Environments:"
    echo "  shared - Shared API environment (recommended)"
    echo "  dev    - Development environment"
    echo "  test   - Test environment"
    echo ""
    echo "Examples:"
    echo "  $0                      # Run full test suite with shared environment"
    echo "  $0 --skip-e2e          # Run unit and integration tests only"
    echo "  $0 -e dev --verbose    # Run with dev environment and verbose output"
}

# Function to run unit tests
run_unit_tests() {
    log_info "=========================================="
    log_info "RUNNING UNIT TESTS"
    log_info "=========================================="
    
    cd "$REPO_ROOT"
    
    log_info "Running Go unit tests..."
    if "$SCRIPT_DIR/run-tests.sh" --coverage 2>&1 | tee -a "$LOG_FILE"; then
        UNIT_TESTS_PASSED=1
        log_success "Unit tests passed"
    else
        log_error "Unit tests failed"
        ((TOTAL_FAILURES++))
    fi
}

# Function to run integration tests
run_integration_tests() {
    log_info "=========================================="
    log_info "RUNNING INTEGRATION TESTS"
    log_info "=========================================="
    
    log_info "Running plugin integration tests..."
    if "$SCRIPT_DIR/test-plugin-integration.sh" 2>&1 | tee -a "$LOG_FILE"; then
        log_success "Plugin integration tests passed"
    else
        log_error "Plugin integration tests failed"
        ((TOTAL_FAILURES++))
    fi
    
    log_info "Running plugin provisioning tests..."
    if "$SCRIPT_DIR/test-plugin-provisioning.sh" 2>&1 | tee -a "$LOG_FILE"; then
        log_success "Plugin provisioning tests passed"
    else
        log_error "Plugin provisioning tests failed"
        ((TOTAL_FAILURES++))
    fi
    
    log_info "Running MCP monitoring tests..."
    if "$SCRIPT_DIR/test-mcp-monitoring.sh" 2>&1 | tee -a "$LOG_FILE"; then
        log_success "MCP monitoring tests passed"
        INTEGRATION_TESTS_PASSED=1
    else
        log_error "MCP monitoring tests failed"
        ((TOTAL_FAILURES++))
    fi
}

# Function to run E2E tests
run_e2e_tests() {
    log_info "=========================================="
    log_info "RUNNING END-TO-END TESTS"
    log_info "=========================================="
    
    local e2e_args=""
    if [[ "$VERBOSE" == "true" ]]; then
        e2e_args="--verbose"
    fi
    
    log_info "Running E2E plugin tests..."
    if "$SCRIPT_DIR/test-plugin-e2e.sh" $e2e_args 2>&1 | tee -a "$LOG_FILE"; then
        E2E_TESTS_PASSED=1
        log_success "E2E tests passed"
    else
        log_error "E2E tests failed"
        ((TOTAL_FAILURES++))
    fi
}

# Function to setup test environment
setup_environment() {
    local environment="$1"
    
    log_info "=========================================="
    log_info "SETTING UP TEST ENVIRONMENT"
    log_info "=========================================="
    log_info "Environment: $environment"
    
    # Start Docker environment
    if "$SCRIPT_DIR/docker-compose-up.sh" "$environment" 2>&1 | tee -a "$LOG_FILE"; then
        log_success "Test environment started successfully"
        
        # Wait a bit for services to stabilize
        log_info "Waiting for services to stabilize..."
        sleep 10
        
        # Verify API health
        local api_endpoint="http://localhost:5194"
        local max_attempts=30
        local attempt=1
        
        while [ $attempt -le $max_attempts ]; do
            if curl -s --connect-timeout 5 "$api_endpoint/health" >/dev/null 2>&1; then
                log_success "API is healthy and ready for testing"
                return 0
            fi
            
            log_info "Waiting for API to be ready (attempt $attempt/$max_attempts)..."
            sleep 2
            ((attempt++))
        done
        
        log_warn "API health check timeout, proceeding anyway..."
        return 0
    else
        log_error "Failed to start test environment"
        return 1
    fi
}

# Function to cleanup environment
cleanup_environment() {
    local environment="$1"
    local force_cleanup="$2"
    
    if [[ "$CLEANUP_AFTER" == "true" || "$force_cleanup" == "true" ]]; then
        log_info "=========================================="
        log_info "CLEANING UP TEST ENVIRONMENT"
        log_info "=========================================="
        
        if "$SCRIPT_DIR/docker-compose-down.sh" "$environment" --force 2>&1 | tee -a "$LOG_FILE"; then
            log_success "Test environment cleaned up successfully"
        else
            log_warn "Failed to clean up test environment"
        fi
    else
        log_info "Skipping environment cleanup (use --no-cleanup)"
    fi
}

# Function to generate final report
generate_final_report() {
    local total_test_suites=0
    local passed_test_suites=0
    
    echo ""
    echo "================================================================"
    echo "              FULL TEST SUITE REPORT"
    echo "================================================================"
    echo "Timestamp: $(date)"
    echo "Environment: $DEFAULT_ENVIRONMENT"
    echo "Log File: $LOG_FILE"
    echo ""
    
    # Count test suites
    if [[ "$SKIP_UNIT" == "false" ]]; then
        ((total_test_suites++))
        if [[ "$UNIT_TESTS_PASSED" == "1" ]]; then
            ((passed_test_suites++))
            echo "✅ Unit Tests: PASSED"
        else
            echo "❌ Unit Tests: FAILED"
        fi
    fi
    
    if [[ "$SKIP_INTEGRATION" == "false" ]]; then
        ((total_test_suites++))
        if [[ "$INTEGRATION_TESTS_PASSED" == "1" ]]; then
            ((passed_test_suites++))
            echo "✅ Integration Tests: PASSED"
        else
            echo "❌ Integration Tests: FAILED"
        fi
    fi
    
    if [[ "$SKIP_E2E" == "false" ]]; then
        ((total_test_suites++))
        if [[ "$E2E_TESTS_PASSED" == "1" ]]; then
            ((passed_test_suites++))
            echo "✅ End-to-End Tests: PASSED"
        else
            echo "❌ End-to-End Tests: FAILED"
        fi
    fi
    
    echo ""
    echo "SUMMARY:"
    echo "  Test Suites Run: $total_test_suites"
    echo "  Test Suites Passed: $passed_test_suites"
    echo "  Test Suites Failed: $((total_test_suites - passed_test_suites))"
    echo "  Total Failures: $TOTAL_FAILURES"
    echo ""
    
    if [[ $TOTAL_FAILURES -eq 0 ]]; then
        echo "🎉 ALL TEST SUITES PASSED!"
        log_success "Full test suite completed successfully"
    else
        echo "⚠️  SOME TESTS FAILED"
        log_error "Full test suite completed with failures"
        echo ""
        echo "Check the log file for detailed error information:"
        echo "  $LOG_FILE"
    fi
    
    echo "================================================================"
    
    # Save report to file
    local report_file="$LOG_DIR/full-test-suite-report-$TIMESTAMP.txt"
    {
        echo "Full Test Suite Report"
        echo "====================="
        echo "Timestamp: $(date)"
        echo "Environment: $DEFAULT_ENVIRONMENT"
        echo "Test Suites Run: $total_test_suites"
        echo "Test Suites Passed: $passed_test_suites"
        echo "Total Failures: $TOTAL_FAILURES"
        echo ""
        echo "Test Suite Details:"
        if [[ "$SKIP_UNIT" == "false" ]]; then
            echo "  Unit Tests: $([ "$UNIT_TESTS_PASSED" == "1" ] && echo "PASSED" || echo "FAILED")"
        fi
        if [[ "$SKIP_INTEGRATION" == "false" ]]; then
            echo "  Integration Tests: $([ "$INTEGRATION_TESTS_PASSED" == "1" ] && echo "PASSED" || echo "FAILED")"
        fi
        if [[ "$SKIP_E2E" == "false" ]]; then
            echo "  E2E Tests: $([ "$E2E_TESTS_PASSED" == "1" ] && echo "PASSED" || echo "FAILED")"
        fi
        echo ""
        echo "Detailed logs: $LOG_FILE"
    } > "$report_file"
    
    log_info "Full test suite report saved to: $report_file"
}

# Cleanup function
cleanup() {
    local exit_code=$?
    
    # Always try to cleanup environment on exit
    cleanup_environment "$DEFAULT_ENVIRONMENT" "true"
    
    if [[ $exit_code -ne 0 ]]; then
        log_error "Test suite exited with error code: $exit_code"
    fi
    
    exit $exit_code
}

# Setup cleanup trap
trap cleanup EXIT INT TERM

# Main execution
main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -e|--environment)
                DEFAULT_ENVIRONMENT="$2"
                shift 2
                ;;
            --skip-unit)
                SKIP_UNIT="true"
                shift
                ;;
            --skip-integration)
                SKIP_INTEGRATION="true"
                shift
                ;;
            --skip-e2e)
                SKIP_E2E="true"
                shift
                ;;
            --no-cleanup)
                CLEANUP_AFTER="false"
                shift
                ;;
            -v|--verbose)
                VERBOSE="true"
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    log_info "Full Test Suite Starting"
    log_info "Configuration:"
    log_info "  Environment: $DEFAULT_ENVIRONMENT"
    log_info "  Skip Unit: $SKIP_UNIT"
    log_info "  Skip Integration: $SKIP_INTEGRATION"
    log_info "  Skip E2E: $SKIP_E2E"
    log_info "  Cleanup After: $CLEANUP_AFTER"
    log_info "  Verbose: $VERBOSE"
    
    # Setup test environment
    if ! setup_environment "$DEFAULT_ENVIRONMENT"; then
        log_error "Failed to setup test environment"
        exit 1
    fi
    
    # Run test suites
    if [[ "$SKIP_UNIT" == "false" ]]; then
        run_unit_tests
    fi
    
    if [[ "$SKIP_INTEGRATION" == "false" ]]; then
        run_integration_tests
    fi
    
    if [[ "$SKIP_E2E" == "false" ]]; then
        run_e2e_tests
    fi
    
    # Generate final report
    generate_final_report
    
    # Return appropriate exit code
    if [[ $TOTAL_FAILURES -eq 0 ]]; then
        return 0
    else
        return 1
    fi
}

# Execute main function
main "$@"