#!/bin/bash

# End-to-End Plugin Manifest and Download Test Script
# Tests the complete plugin lifecycle: manifest retrieval, JWT exchange, plugin installation
# Prerequisites: Docker environment must be running (use docker-compose-up.sh first)

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TIMESTAMP=$(date '+%Y%m%d-%H%M%S')
LOG_DIR="$REPO_ROOT/logs"
LOG_FILE="$LOG_DIR/plugin-e2e-test-$TIMESTAMP.log"
TEST_DATA_DIR="$REPO_ROOT/test-data"

# Test configuration
API_ENDPOINT="${KM_API_ENDPOINT:-http://localhost:5194}"
TEST_EMAIL="test-$(date +%s)@gmail.com"
TEST_PASSWORD="TestPassword123!"
CLI_BINARY="$REPO_ROOT/km"
TEMP_DIR=""

# Test state tracking
TESTS_PASSED=0
TESTS_FAILED=0
TEST_API_KEY=""
TEST_JWT=""
CUSTOMER_ID=""

# Ensure required directories exist
mkdir -p "$LOG_DIR" "$TEST_DATA_DIR"

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

# Test result tracking
test_passed() {
    local test_name="$1"
    ((TESTS_PASSED++))
    log_success "✅ PASS: $test_name"
}

test_failed() {
    local test_name="$1"
    local error_msg="$2"
    ((TESTS_FAILED++))
    log_error "❌ FAIL: $test_name - $error_msg"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -e, --endpoint URL    API endpoint (default: $API_ENDPOINT)"
    echo "  -v, --verbose         Enable verbose output"
    echo "  --skip-build         Skip building CLI binary"
    echo "  --cleanup           Clean up test data on exit"
    echo "  -h, --help           Show this help"
    echo ""
    echo "Prerequisites:"
    echo "  - Docker environment must be running"
    echo "  - Use ./docker-compose-up.sh to start the environment first"
    echo ""
    echo "Example:"
    echo "  ./docker-compose-up.sh shared"
    echo "  ./test-plugin-e2e.sh"
}

# Cleanup function
cleanup() {
    local exit_code=$?
    
    if [[ -n "$TEMP_DIR" && -d "$TEMP_DIR" ]]; then
        log_info "Cleaning up temporary directory: $TEMP_DIR"
        rm -rf "$TEMP_DIR"
    fi
    
    # Clean up test environment variables
    unset KM_API_KEY KM_API_ENDPOINT KM_DEBUG
    
    log_info "Test cleanup completed"
    
    if [[ $exit_code -ne 0 ]]; then
        log_error "Script exited with error code: $exit_code"
    fi
    
    exit $exit_code
}

# Setup cleanup trap
trap cleanup EXIT INT TERM

# Function to check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if curl is available
    if ! command -v curl >/dev/null 2>&1; then
        log_error "curl is required but not installed"
        exit 1
    fi
    
    # Check if jq is available
    if ! command -v jq >/dev/null 2>&1; then
        log_error "jq is required but not installed"
        log_info "Install with: brew install jq (macOS) or apt-get install jq (Linux)"
        exit 1
    fi
    
    # Check API endpoint connectivity
    log_info "Checking API endpoint connectivity: $API_ENDPOINT"
    if ! curl -s --connect-timeout 5 "$API_ENDPOINT/health" >/dev/null; then
        log_error "Cannot connect to API endpoint: $API_ENDPOINT"
        log_error "Make sure the Docker environment is running:"
        log_error "  ./scripts/tests/docker-compose-up.sh shared"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Function to build CLI binary
build_cli_binary() {
    log_info "Building CLI binary..."
    
    cd "$REPO_ROOT"
    
    # Build with version info
    local version="e2e-test-$TIMESTAMP"
    local commit=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    local build_date=$(date -u +%Y-%m-%d)
    
    local ldflags="-X main.version=$version -X main.commit=$commit -X main.date=$build_date"
    
    if go build -ldflags="$ldflags" -o "$CLI_BINARY" ./cmd/main.go; then
        log_success "CLI binary built successfully: $CLI_BINARY"
        
        # Verify binary works
        if "$CLI_BINARY" --version >/dev/null 2>&1; then
            log_success "CLI binary verification passed"
        else
            log_error "CLI binary verification failed"
            return 1
        fi
    else
        log_error "Failed to build CLI binary"
        return 1
    fi
}

# Function to test API health
test_api_health() {
    log_info "Testing API health..."
    
    local response
    if response=$(curl -s -w "%{http_code}" "$API_ENDPOINT/health"); then
        local http_code="${response: -3}"
        local body="${response%???}"
        
        if [[ "$http_code" == "200" ]]; then
            test_passed "API health check"
            log_info "API health response: $body"
        else
            test_failed "API health check" "HTTP $http_code: $body"
            return 1
        fi
    else
        test_failed "API health check" "Failed to connect to API"
        return 1
    fi
}

# Function to register a test customer
register_test_customer() {
    log_info "Registering test customer..."
    
    local registration_data=$(cat <<EOF
{
    "email": "$TEST_EMAIL",
    "password": "$TEST_PASSWORD",
    "firstName": "Test",
    "lastName": "User",
    "tier": "Pro"
}
EOF
)
    
    log_info "Registration payload: $registration_data"
    
    local response
    if response=$(curl -s -w "%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$registration_data" \
        "$API_ENDPOINT/api/auth/register"); then
        
        local http_code="${response: -3}"
        local body="${response%???}"
        
        log_info "Registration response code: $http_code"
        log_info "Registration response body: $body"
        
        if [[ "$http_code" == "200" || "$http_code" == "201" ]]; then
            # Extract API key and customer ID from response
            TEST_API_KEY=$(echo "$body" | jq -r '.apiKey.key // .api_key // empty')
            CUSTOMER_ID=$(echo "$body" | jq -r '.customer.id // .customerId // .customer_id // .id // empty')
            
            if [[ -n "$TEST_API_KEY" && "$TEST_API_KEY" != "null" ]]; then
                test_passed "Customer registration"
                log_success "API Key: ${TEST_API_KEY:0:10}..."
                log_info "Customer ID: $CUSTOMER_ID"
            else
                test_failed "Customer registration" "No API key in response"
                return 1
            fi
        else
            test_failed "Customer registration" "HTTP $http_code: $body"
            return 1
        fi
    else
        test_failed "Customer registration" "Failed to make registration request"
        return 1
    fi
}

# Function to test JWT token exchange
test_jwt_exchange() {
    log_info "Testing JWT token exchange..."
    
    if [[ -z "$TEST_API_KEY" ]]; then
        test_failed "JWT exchange" "No API key available"
        return 1
    fi
    
    local response
    if response=$(curl -s -w "%{http_code}" \
        -H "X-API-Key: $TEST_API_KEY" \
        "$API_ENDPOINT/api/auth/token"); then
        
        local http_code="${response: -3}"
        local body="${response%???}"
        
        log_info "JWT exchange response code: $http_code"
        
        if [[ "$http_code" == "200" ]]; then
            TEST_JWT=$(echo "$body" | jq -r '.token // .access_token // empty')
            
            if [[ -n "$TEST_JWT" && "$TEST_JWT" != "null" ]]; then
                test_passed "JWT token exchange"
                log_success "JWT Token: ${TEST_JWT:0:20}..."
                
                # Verify JWT structure (should have 3 parts separated by dots)
                local jwt_parts=$(echo "$TEST_JWT" | tr '.' '\n' | wc -l)
                if [[ "$jwt_parts" -eq 3 ]]; then
                    test_passed "JWT format validation"
                else
                    test_failed "JWT format validation" "JWT has $jwt_parts parts, expected 3"
                fi
            else
                test_failed "JWT token exchange" "No JWT token in response"
                return 1
            fi
        else
            test_failed "JWT token exchange" "HTTP $http_code: $body"
            return 1
        fi
    else
        test_failed "JWT token exchange" "Failed to make token request"
        return 1
    fi
}

# Function to test CLI authentication
test_cli_authentication() {
    log_info "Testing CLI authentication..."
    
    # Set environment variables for CLI
    export KM_API_KEY="$TEST_API_KEY"
    export KM_API_ENDPOINT="$API_ENDPOINT"
    export KM_DEBUG="true"
    
    log_info "Set KM_API_KEY: ${TEST_API_KEY:0:10}..."
    log_info "Set KM_API_ENDPOINT: $API_ENDPOINT"
    
    # Test auth status command
    local output
    if output=$("$CLI_BINARY" auth status 2>&1); then
        test_passed "CLI auth status command"
        log_info "CLI auth status output:"
        echo "$output" | while IFS= read -r line; do
            log_info "  $line"
        done
        
        # Verify output contains expected information
        if echo "$output" | grep -q "API Key:" && echo "$output" | grep -q "API Endpoint:"; then
            test_passed "CLI auth status content validation"
        else
            test_failed "CLI auth status content validation" "Missing expected fields in output"
        fi
    else
        test_failed "CLI auth status command" "Command failed: $output"
        return 1
    fi
}

# Function to test plugin manifest retrieval
test_plugin_manifest() {
    log_info "Testing plugin manifest retrieval..."
    
    # Test manifest endpoint directly via API
    local response
    if response=$(curl -s -w "%{http_code}" \
        -H "X-API-Key: $TEST_API_KEY" \
        -H "Authorization: Bearer $TEST_JWT" \
        "$API_ENDPOINT/api/plugins/manifest"); then
        
        local http_code="${response: -3}"
        local body="${response%???}"
        
        log_info "Manifest response code: $http_code"
        
        if [[ "$http_code" == "200" ]]; then
            test_passed "Plugin manifest API call"
            
            # Parse and validate manifest structure
            local plugins_count=$(echo "$body" | jq '.plugins | length')
            if [[ "$plugins_count" -gt 0 ]]; then
                test_passed "Plugin manifest contains plugins"
                log_info "Found $plugins_count plugins in manifest"
                
                # Show available plugins
                log_info "Available plugins:"
                echo "$body" | jq -r '.plugins[] | "  - \(.name) v\(.version) (\(.tier))"' | while IFS= read -r line; do
                    log_info "$line"
                done
                
                # Save manifest for CLI testing
                echo "$body" > "$TEST_DATA_DIR/manifest-$TIMESTAMP.json"
                log_info "Saved manifest to: $TEST_DATA_DIR/manifest-$TIMESTAMP.json"
            else
                test_failed "Plugin manifest validation" "No plugins found in manifest"
                return 1
            fi
        else
            test_failed "Plugin manifest API call" "HTTP $http_code: $body"
            return 1
        fi
    else
        test_failed "Plugin manifest API call" "Failed to make manifest request"
        return 1
    fi
}

# Function to test CLI plugin commands
test_cli_plugin_commands() {
    log_info "Testing CLI plugin commands..."
    
    # Test plugins list command
    log_info "Testing 'km plugins list' command..."
    local list_output
    if list_output=$("$CLI_BINARY" plugins list 2>&1); then
        test_passed "CLI plugins list command"
        log_info "Plugins list output:"
        echo "$list_output" | while IFS= read -r line; do
            log_info "  $line"
        done
    else
        test_failed "CLI plugins list command" "Command failed: $list_output"
        # Continue with other tests even if this fails
    fi
    
    # Get first available plugin for installation test
    local first_plugin
    if first_plugin=$(curl -s -H "X-API-Key: $TEST_API_KEY" "$API_ENDPOINT/api/plugins/manifest" | jq -r '.plugins[0].name // empty'); then
        if [[ -n "$first_plugin" && "$first_plugin" != "null" ]]; then
            log_info "Testing plugin installation with: $first_plugin"
            test_plugin_installation "$first_plugin"
        else
            log_warn "No plugins available for installation testing"
        fi
    else
        log_warn "Could not determine available plugins for installation test"
    fi
}

# Function to test plugin installation
test_plugin_installation() {
    local plugin_name="$1"
    log_info "Testing plugin installation: $plugin_name"
    
    # Test plugin install command
    local install_output
    if install_output=$("$CLI_BINARY" plugins install "$plugin_name" 2>&1); then
        test_passed "CLI plugin install command ($plugin_name)"
        log_info "Plugin install output:"
        echo "$install_output" | while IFS= read -r line; do
            log_info "  $line"
        done
        
        # Verify plugin was installed
        if echo "$install_output" | grep -q "Successfully installed"; then
            test_passed "Plugin installation verification ($plugin_name)"
            
            # Test plugin removal
            test_plugin_removal "$plugin_name"
        else
            test_failed "Plugin installation verification ($plugin_name)" "Success message not found"
        fi
    else
        test_failed "CLI plugin install command ($plugin_name)" "Command failed: $install_output"
    fi
}

# Function to test plugin removal
test_plugin_removal() {
    local plugin_name="$1"
    log_info "Testing plugin removal: $plugin_name"
    
    local remove_output
    if remove_output=$("$CLI_BINARY" plugins remove "$plugin_name" 2>&1); then
        test_passed "CLI plugin remove command ($plugin_name)"
        log_info "Plugin remove output:"
        echo "$remove_output" | while IFS= read -r line; do
            log_info "  $line"
        done
        
        # Verify plugin was removed
        if echo "$remove_output" | grep -q "Successfully removed"; then
            test_passed "Plugin removal verification ($plugin_name)"
        else
            test_failed "Plugin removal verification ($plugin_name)" "Success message not found"
        fi
    else
        test_failed "CLI plugin remove command ($plugin_name)" "Command failed: $remove_output"
    fi
}

# Function to test error handling
test_error_handling() {
    log_info "Testing error handling..."
    
    # Test with invalid API key
    log_info "Testing with invalid API key..."
    local old_api_key="$KM_API_KEY"
    export KM_API_KEY="invalid_key"
    
    local error_output
    if error_output=$("$CLI_BINARY" plugins list 2>&1); then
        # Command should fail with invalid key
        if echo "$error_output" | grep -qi "unauthorized\|invalid\|forbidden"; then
            test_passed "Invalid API key error handling"
        else
            test_failed "Invalid API key error handling" "Expected error not found: $error_output"
        fi
    else
        test_passed "Invalid API key error handling"
    fi
    
    # Restore valid API key
    export KM_API_KEY="$old_api_key"
    
    # Test with non-existent plugin
    log_info "Testing with non-existent plugin..."
    if error_output=$("$CLI_BINARY" plugins install "non-existent-plugin-$(date +%s)" 2>&1); then
        if echo "$error_output" | grep -qi "not found\|not available"; then
            test_passed "Non-existent plugin error handling"
        else
            test_failed "Non-existent plugin error handling" "Expected error not found: $error_output"
        fi
    else
        test_passed "Non-existent plugin error handling"
    fi
}

# Function to generate test report
generate_test_report() {
    local total_tests=$((TESTS_PASSED + TESTS_FAILED))
    local success_rate=0
    
    if [[ $total_tests -gt 0 ]]; then
        success_rate=$((TESTS_PASSED * 100 / total_tests))
    fi
    
    echo ""
    echo "================================================================"
    echo "              E2E PLUGIN TEST REPORT"
    echo "================================================================"
    echo "Timestamp: $(date)"
    echo "API Endpoint: $API_ENDPOINT"
    echo "Test Email: $TEST_EMAIL"
    echo "Log File: $LOG_FILE"
    echo ""
    echo "RESULTS:"
    echo "  Total Tests: $total_tests"
    echo "  Passed: $TESTS_PASSED"
    echo "  Failed: $TESTS_FAILED"
    echo "  Success Rate: $success_rate%"
    echo ""
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo "🎉 ALL TESTS PASSED!"
        log_success "E2E test suite completed successfully"
    else
        echo "⚠️  SOME TESTS FAILED"
        log_error "E2E test suite completed with failures"
        echo ""
        echo "Check the log file for detailed error information:"
        echo "  $LOG_FILE"
    fi
    
    echo "================================================================"
    
    # Save report to file
    local report_file="$LOG_DIR/test-report-$TIMESTAMP.txt"
    {
        echo "E2E Plugin Test Report"
        echo "====================="
        echo "Timestamp: $(date)"
        echo "API Endpoint: $API_ENDPOINT"
        echo "Total Tests: $total_tests"
        echo "Passed: $TESTS_PASSED"
        echo "Failed: $TESTS_FAILED"
        echo "Success Rate: $success_rate%"
        echo ""
        echo "Test Details:"
        echo "See log file: $LOG_FILE"
    } > "$report_file"
    
    log_info "Test report saved to: $report_file"
}

# Main test execution
run_tests() {
    log_info "Starting E2E Plugin Test Suite"
    log_info "API Endpoint: $API_ENDPOINT"
    log_info "Test Email: $TEST_EMAIL"
    
    # Create temporary directory for test files
    TEMP_DIR=$(mktemp -d -t "km-e2e-test-XXXXXX")
    log_info "Using temporary directory: $TEMP_DIR"
    
    # Run test sequence
    check_prerequisites
    
    if [[ "${SKIP_BUILD:-false}" != "true" ]]; then
        build_cli_binary
    fi
    
    test_api_health
    register_test_customer
    test_jwt_exchange
    test_cli_authentication
    test_plugin_manifest
    test_cli_plugin_commands
    test_error_handling
    
    # Generate final report
    generate_test_report
    
    # Return appropriate exit code
    if [[ $TESTS_FAILED -eq 0 ]]; then
        return 0
    else
        return 1
    fi
}

# Main execution
main() {
    local skip_build="false"
    local cleanup_on_exit="false"
    local verbose="false"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -e|--endpoint)
                API_ENDPOINT="$2"
                shift 2
                ;;
            -v|--verbose)
                verbose="true"
                shift
                ;;
            --skip-build)
                skip_build="true"
                shift
                ;;
            --cleanup)
                cleanup_on_exit="true"
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
    
    # Set global variables
    SKIP_BUILD="$skip_build"
    
    # Enable verbose logging if requested
    if [[ "$verbose" == "true" ]]; then
        set -x
    fi
    
    log_info "E2E Plugin Test Script Starting"
    log_info "Configuration:"
    log_info "  API Endpoint: $API_ENDPOINT"
    log_info "  Skip Build: $skip_build"
    log_info "  Cleanup: $cleanup_on_exit"
    log_info "  Verbose: $verbose"
    
    # Run the tests
    if run_tests; then
        log_success "E2E test suite completed successfully"
        exit 0
    else
        log_error "E2E test suite failed"
        exit 1
    fi
}

# Execute main function
main "$@"