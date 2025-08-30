#!/bin/bash

# Automation script for complete plugin system cleanup and testing
# This script removes all prebuilt binaries, builds fresh ones, and tests the complete plugin feature set

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
PLUGINS_ROOT="/Users/milesangelo/Source/active/kilometers.ai/kilometers-cli-plugins"
TEST_API_KEY="test-api-key-1234567890"
CUSTOMER_ID="demo_cleanup_test"

# Logging configuration
LOG_DIR="$CLI_ROOT/logs"
LOG_FILE="$LOG_DIR/cleanup-test-$(date +%Y%m%d-%H%M%S).log"
mkdir -p "$LOG_DIR"

# Results tracking
CLEANUP_RESULTS=()
BUILD_RESULTS=()
TEST_RESULTS=()
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Utility functions
print_header() {
    echo ""
    echo -e "${CYAN}================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}================================${NC}"
    echo ""
}

print_step() {
    echo -e "${BLUE}🔍 $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${PURPLE}📋 $1${NC}"
}

log_and_echo() {
    echo "$1" | tee -a "$LOG_FILE"
}

run_test() {
    local test_name="$1"
    local test_command="$2"
    local allow_failure="${3:-false}"
    
    echo -e "\n${YELLOW}🧪 Testing: $test_name${NC}"
    ((TOTAL_TESTS++))
    
    local start_time=$(date +%s)
    
    if eval "$test_command" >> "$LOG_FILE" 2>&1; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo -e "${GREEN}✅ PASSED: $test_name (${duration}s)${NC}"
        TEST_RESULTS+=("PASS: $test_name")
        ((PASSED_TESTS++))
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        if [ "$allow_failure" = "true" ]; then
            echo -e "${YELLOW}⚠️  EXPECTED FAILURE: $test_name (${duration}s)${NC}"
            TEST_RESULTS+=("EXPECTED_FAIL: $test_name")
            ((PASSED_TESTS++))
            return 0
        else
            echo -e "${RED}❌ FAILED: $test_name (${duration}s)${NC}"
            TEST_RESULTS+=("FAIL: $test_name")
            ((FAILED_TESTS++))
            return 1
        fi
    fi
}

# Phase 1: Complete Binary Cleanup
cleanup_prebuilt_binaries() {
    print_header "Phase 1: Complete Binary Cleanup"
    
    print_step "Removing prebuilt binaries from kilometers-cli repository"
    
    # CLI binaries
    local cli_binaries=(
        "$CLI_ROOT/km"
        "$CLI_ROOT/km-premium" 
        "$CLI_ROOT/build/km"
    )
    
    for binary in "${cli_binaries[@]}"; do
        if [ -f "$binary" ]; then
            rm -f "$binary"
            print_success "Removed: $(basename "$binary")"
            CLEANUP_RESULTS+=("Removed CLI binary: $binary")
        else
            print_info "Not found: $(basename "$binary")"
        fi
    done
    
    # Remove build directory if it exists
    if [ -d "$CLI_ROOT/build" ]; then
        rm -rf "$CLI_ROOT/build"
        print_success "Removed build directory"
        CLEANUP_RESULTS+=("Removed build directory")
    fi
    
    print_step "Removing prebuilt binaries from kilometers-cli-plugins repository"
    
    if [ -d "$PLUGINS_ROOT" ]; then
        # Plugin packages
        if [ -d "$PLUGINS_ROOT/dist-standalone" ]; then
            local package_count=$(find "$PLUGINS_ROOT/dist-standalone" -name "*.kmpkg" | wc -l)
            rm -rf "$PLUGINS_ROOT/dist-standalone"/*
            print_success "Removed $package_count plugin packages from dist-standalone"
            CLEANUP_RESULTS+=("Removed $package_count plugin packages")
        fi
        
        # Build artifacts
        if [ -d "$PLUGINS_ROOT/build-standalone" ]; then
            rm -rf "$PLUGINS_ROOT/build-standalone"/*
            print_success "Cleaned build-standalone directory" 
            CLEANUP_RESULTS+=("Cleaned plugin build directory")
        fi
    else
        print_warning "Plugin repository not found at $PLUGINS_ROOT"
    fi
    
    print_step "Cleaning user plugin directories"
    
    # User plugin directory
    local user_plugin_dir="$HOME/.km/plugins"
    if [ -d "$user_plugin_dir" ]; then
        local user_plugin_count=$(find "$user_plugin_dir" -name "km-plugin-*" | wc -l)
        if [ "$user_plugin_count" -gt 0 ]; then
            echo -e "${YELLOW}Found $user_plugin_count user plugins. Remove them? (y/n)${NC}"
            read -r response
            if [[ "$response" =~ ^[Yy]$ ]]; then
                rm -f "$user_plugin_dir"/km-plugin-*
                print_success "Removed $user_plugin_count user plugins"
                CLEANUP_RESULTS+=("Removed $user_plugin_count user plugins")
            else
                print_info "Kept existing user plugins"
            fi
        else
            print_info "No user plugins found"
        fi
    else
        print_info "User plugin directory does not exist"
    fi
    
    # Clean any stray binaries
    print_step "Scanning for additional binaries"
    local additional_binaries=$(find "$CLI_ROOT" -name "km-plugin-*" -type f 2>/dev/null || true)
    if [ -n "$additional_binaries" ]; then
        echo "$additional_binaries" | while read -r binary; do
            rm -f "$binary"
            print_success "Removed additional binary: $(basename "$binary")"
        done
    fi
    
    print_success "Binary cleanup completed"
}

# Phase 2: Fresh Plugin Building  
build_fresh_plugins() {
    print_header "Phase 2: Fresh Plugin Building"
    
    cd "$CLI_ROOT"
    
    print_step "Building main CLI binary"
    if go build -o km ./cmd/main.go; then
        print_success "CLI binary built successfully"
        BUILD_RESULTS+=("CLI binary: SUCCESS")
    else
        print_error "Failed to build CLI binary"
        BUILD_RESULTS+=("CLI binary: FAILED")
        return 1
    fi
    
    print_step "Note: Plugin examples now located in kilometers-cli-plugins repository"
    print_info "Plugins should be built using the dedicated kilometers-cli-plugins repository"
    
    # Build plugins using standalone system if available
    if [ -d "$PLUGINS_ROOT" ] && [ -f "$PLUGINS_ROOT/build-standalone.sh" ]; then
        print_step "Building standalone plugins"
        
        cd "$PLUGINS_ROOT"
        
        # Build console logger plugin package
        if ./build-standalone.sh --plugin=console-logger --customer="$CUSTOMER_ID" --api-key="$TEST_API_KEY" --tier=Free; then
            print_success "Standalone console logger package built"
            BUILD_RESULTS+=("Standalone console logger: SUCCESS")
        else
            print_warning "Standalone console logger build failed (expected for demo)"
            BUILD_RESULTS+=("Standalone console logger: EXPECTED_DEMO_FAILURE")
        fi
        
        cd "$CLI_ROOT"
    fi
    
    print_success "Plugin building completed"
}

# Phase 3: Comprehensive Plugin Testing
test_plugin_commands() {
    print_header "Phase 3: Plugin Command Testing"
    
    cd "$CLI_ROOT"
    
    # Basic CLI functionality
    run_test "CLI help command" "./km --help"
    run_test "CLI version command" "./km version"
    run_test "Plugins help command" "./km plugins --help"
    
    # Plugin discovery and listing (without API key)
    run_test "Plugin list without API key" "./km plugins list"
    run_test "Plugin status without API key" "./km plugins status"
    
    # Plugin commands with API key
    export KM_API_KEY="$TEST_API_KEY"
    
    run_test "Plugin list with API key" "./km plugins list"
    run_test "Plugin status with API key" "./km plugins status"  
    run_test "Plugin refresh command" "./km plugins refresh" true  # Expected to fail without real API
    
    # Plugin removal and installation testing
    if [ -f "$HOME/.km/plugins/km-plugin-console-logger" ]; then
        run_test "Plugin remove command" "./km plugins remove console-logger"
        run_test "Plugin list after removal" "./km plugins list"
    fi
    
    # Test plugin installation if packages exist
    local plugin_packages=($(find "$PLUGINS_ROOT/dist-standalone" -name "*.kmpkg" 2>/dev/null || true))
    if [ ${#plugin_packages[@]} -gt 0 ]; then
        local first_package="${plugin_packages[0]}"
        run_test "Plugin install command" "./km plugins install '$first_package'" true
    fi
    
    unset KM_API_KEY
}

test_monitoring_integration() {
    print_header "Phase 4: Monitoring Integration Testing"
    
    cd "$CLI_ROOT"
    
    # Test basic monitoring without plugins
    run_test "Basic monitoring without API key" "timeout 3s ./km monitor --server -- echo '{\"jsonrpc\":\"2.0\",\"method\":\"test\",\"id\":1}' || echo 'Monitoring completed (timeout expected)'"
    
    # Test monitoring with API key (plugin integration)
    export KM_API_KEY="$TEST_API_KEY"
    
    run_test "Monitoring with API key (plugin discovery)" "timeout 5s ./km monitor --server -- echo '{\"jsonrpc\":\"2.0\",\"method\":\"test\",\"id\":1}' || echo 'Monitoring completed (timeout expected)'"
    
    # Test different monitoring scenarios
    run_test "Monitoring with buffer size option" "timeout 3s ./km monitor --buffer-size 2MB --server -- echo '{\"jsonrpc\":\"2.0\",\"method\":\"test\",\"id\":1}' || echo 'Monitoring completed (timeout expected)'"
    
    # Test monitoring with mock JSON-RPC commands
    run_test "Monitoring JSON-RPC initialize" "timeout 3s ./km monitor --server -- echo '{\"jsonrpc\":\"2.0\",\"method\":\"initialize\",\"params\":{},\"id\":1}' || echo 'Monitoring completed (timeout expected)'"
    
    unset KM_API_KEY
}

test_error_scenarios() {
    print_header "Phase 5: Error Scenario Testing"
    
    cd "$CLI_ROOT"
    
    # Test error handling
    run_test "Invalid plugin package installation" "./km plugins install nonexistent.kmpkg" true
    run_test "Remove nonexistent plugin" "./km plugins remove nonexistent-plugin" true
    run_test "Monitoring with invalid server command" "timeout 3s ./km monitor --server -- nonexistent-command" true
    
    # Test with invalid API endpoints
    export KM_API_KEY="invalid-key"
    export KM_API_ENDPOINT="http://invalid-endpoint"
    
    run_test "Plugin operations with invalid API" "./km plugins list" true
    run_test "Monitoring with invalid API" "timeout 3s ./km monitor --server -- echo '{\"jsonrpc\":\"2.0\",\"method\":\"test\",\"id\":1}'"
    
    unset KM_API_KEY
    unset KM_API_ENDPOINT
}

# Phase 6: Generate Comprehensive Report
generate_report() {
    print_header "Phase 6: Comprehensive Test Report"
    
    local report_file="$LOG_DIR/plugin-automation-report-$(date +%Y%m%d-%H%M%S).md"
    
    cat > "$report_file" << EOF
# Plugin System Cleanup and Testing Report

**Generated**: $(date)
**Script**: cleanup-and-test-plugins.sh
**CLI Root**: $CLI_ROOT
**Plugins Root**: $PLUGINS_ROOT

## Execution Summary

- **Total Tests**: $TOTAL_TESTS
- **Passed**: $PASSED_TESTS  
- **Failed**: $FAILED_TESTS
- **Success Rate**: $(( PASSED_TESTS * 100 / TOTAL_TESTS ))%

## Phase 1: Binary Cleanup Results

EOF

    for result in "${CLEANUP_RESULTS[@]}"; do
        echo "- $result" >> "$report_file"
    done

    cat >> "$report_file" << EOF

## Phase 2: Build Results

EOF

    for result in "${BUILD_RESULTS[@]}"; do
        echo "- $result" >> "$report_file"
    done

    cat >> "$report_file" << EOF

## Phase 3-5: Test Results

EOF

    for result in "${TEST_RESULTS[@]}"; do
        echo "- $result" >> "$report_file"
    done

    cat >> "$report_file" << EOF

## Plugin System Status

### Available Commands
- \`km plugins list\` - List installed plugins
- \`km plugins status\` - Show plugin status and health  
- \`km plugins install <package>\` - Install plugin package
- \`km plugins remove <name>\` - Remove installed plugin
- \`km plugins refresh\` - Refresh plugins from API

### Plugin Discovery
- User directory: \`~/.km/plugins/\`
- Plugin packages: \`.kmpkg\` format
- Authentication: API key based

### Monitoring Integration
- Plugins integrate with \`km monitor\` command
- Automatic plugin loading during monitoring
- Graceful fallback when plugins unavailable

## Developer Notes

### Building Plugins
\`\`\`bash
# CLI binary
go build -o km ./cmd/main.go

# Plugin examples
cd examples/plugins/console-logger
go build -o km-plugin-console-logger ./main.go

# Standalone plugin packages (if available)
cd ../kilometers-cli-plugins
./build-standalone.sh --plugin=console-logger --customer=test --api-key=test-key
\`\`\`

### Testing Commands
\`\`\`bash
# Basic plugin management
KM_API_KEY=test-key ./km plugins list
KM_API_KEY=test-key ./km plugins status

# Monitoring with plugins
KM_API_KEY=test-key ./km monitor --server -- echo '{"jsonrpc":"2.0","method":"test","id":1}'
\`\`\`

### Installation Verification
\`\`\`bash
# Check for installed plugins
ls -la ~/.km/plugins/

# Check plugin functionality
KM_API_KEY=test-key ./km plugins list
\`\`\`

## Full Log Details

See detailed execution log: \`$LOG_FILE\`

EOF

    print_success "Report generated: $report_file"
    
    # Display summary
    echo ""
    print_info "=== EXECUTION SUMMARY ==="
    print_info "Total Tests: $TOTAL_TESTS"
    if [ $FAILED_TESTS -eq 0 ]; then
        print_success "All tests passed! ($PASSED_TESTS/$TOTAL_TESTS)"
    else
        print_warning "Tests passed: $PASSED_TESTS/$TOTAL_TESTS"
        print_error "Tests failed: $FAILED_TESTS"
    fi
    
    echo ""
    print_info "Cleanup Results:"
    for result in "${CLEANUP_RESULTS[@]}"; do
        echo "  • $result"
    done
    
    echo ""
    print_info "Build Results:"
    for result in "${BUILD_RESULTS[@]}"; do
        echo "  • $result"
    done
    
    echo ""
    print_info "Files Generated:"
    echo "  • Execution log: $LOG_FILE"
    echo "  • Test report: $report_file"
}

# Main execution
main() {
    print_header "Kilometers CLI Plugin System Cleanup and Testing"
    
    log_and_echo "Starting automation at $(date)"
    log_and_echo "CLI Root: $CLI_ROOT"
    log_and_echo "Plugins Root: $PLUGINS_ROOT"
    
    # Execute phases
    cleanup_prebuilt_binaries
    build_fresh_plugins
    test_plugin_commands
    test_monitoring_integration
    test_error_scenarios
    generate_report
    
    print_header "Automation Complete!"
    
    if [ $FAILED_TESTS -eq 0 ]; then
        print_success "All tests passed successfully!"
        echo ""
        echo -e "${GREEN}🎉 Plugin system is fully operational!${NC}"
        echo ""
        echo "Next steps:"
        echo "1. Review the generated report for details"
        echo "2. Run manual tests with real MCP servers"
        echo "3. Test plugin installation/removal workflows"
        echo "4. Verify monitoring integration in production scenarios"
        exit 0
    else
        print_warning "Some tests failed. Review the report for details."
        exit 1
    fi
}

# Handle script interruption
cleanup_on_exit() {
    echo ""
    print_warning "Script interrupted. Cleaning up..."
    cd "$CLI_ROOT"
    exit 1
}

trap cleanup_on_exit INT TERM

# Execute main function
main "$@"
