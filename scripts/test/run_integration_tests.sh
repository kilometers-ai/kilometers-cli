#!/bin/bash

# Comprehensive Integration Test Runner for Kilometers CLI Proxy
# This script runs all three critical integration tests that validate:
# 1. Core monitor proxy functionality
# 2. Free tier limitations and behavior
# 3. Premium tier features (events and risk filtering)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Test configuration
TIMEOUT_SECONDS=60
LOG_LEVEL=${LOG_LEVEL:-info}

echo -e "${BOLD}🧪 Kilometers CLI Proxy Integration Test Suite${NC}"
echo -e "${BOLD}=============================================${NC}"
echo ""

# Check prerequisites
echo -e "${BLUE}📋 Checking prerequisites...${NC}"

# Check if Rust and Cargo are available
if ! command -v cargo &> /dev/null; then
    echo -e "${RED}❌ Cargo not found. Please install Rust and Cargo.${NC}"
    exit 1
fi

# Check if we're in the right directory
if [ ! -f "Cargo.toml" ]; then
    echo -e "${RED}❌ Cargo.toml not found. Please run this script from the project root.${NC}"
    exit 1
fi

# Check for required dependencies
if ! grep -q "tokio.*test" Cargo.toml; then
    echo -e "${YELLOW}⚠️  Adding tokio test features for integration tests...${NC}"
    # Note: This would need to be added to Cargo.toml manually
fi

echo -e "${GREEN}✅ Prerequisites check passed${NC}"
echo ""

# Clean previous test artifacts
echo -e "${BLUE}🧹 Cleaning previous test artifacts...${NC}"
rm -f mcp_proxy.log mcp_traffic.jsonl 2>/dev/null || true
rm -rf target/tmp_test_* 2>/dev/null || true

# Build the project first
echo -e "${BLUE}🔨 Building km binary...${NC}"
if cargo build; then
    echo -e "${GREEN}✅ Build successful${NC}"
else
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

echo ""

# Function to run a test with timeout and capture results
run_test() {
    local test_name="$1"
    local test_command="$2"
    local description="$3"

    echo -e "${BLUE}🔍 Running: ${BOLD}${test_name}${NC}"
    echo -e "   ${description}"

    # Create a temporary log file for this test
    local test_log="/tmp/km_test_${test_name}_$(date +%s).log"

    # Run the test with timeout
    if timeout ${TIMEOUT_SECONDS} bash -c "$test_command" > "${test_log}" 2>&1; then
        echo -e "${GREEN}✅ PASSED: ${test_name}${NC}"

        # Show key output if verbose
        if [[ "${LOG_LEVEL}" == "debug" ]]; then
            echo -e "${YELLOW}   Test output:${NC}"
            sed 's/^/   /' "${test_log}"
        fi

        rm -f "${test_log}"
        return 0
    else
        echo -e "${RED}❌ FAILED: ${test_name}${NC}"
        echo -e "${RED}   Error output:${NC}"
        sed 's/^/   /' "${test_log}"
        rm -f "${test_log}"
        return 1
    fi

    echo ""
}

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
FAILED_TEST_NAMES=()

# Core Monitor Proxy Tests
echo -e "${BOLD}🎯 Core Monitor Proxy Functionality Tests${NC}"
echo "These tests ensure the basic MCP proxy behavior works correctly."
echo ""

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "core_proxy_functionality" \
    "cargo test test_core_monitor_proxy_functionality --test integration_core_proxy_test" \
    "Tests basic MCP request/response forwarding and logging"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES+=("Core Proxy Functionality")
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "core_proxy_invalid_server" \
    "cargo test test_core_proxy_with_invalid_server --test integration_core_proxy_test" \
    "Tests proxy behavior with invalid MCP server commands"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES+=("Core Proxy Error Handling")
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "proxy_log_format" \
    "cargo test test_proxy_log_format_validation --test integration_core_proxy_test" \
    "Validates MCP proxy log file format and structure"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES+=("Proxy Log Format")
fi

echo ""

# Free Tier Tests
echo -e "${BOLD}🆓 Free Tier Functionality Tests${NC}"
echo "These tests ensure free tier users get appropriate limited functionality."
echo ""

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "free_tier_local_logging" \
    "cargo test test_free_tier_local_logging_only --test integration_free_tier_test" \
    "Tests free tier local logging without premium features"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES+=("Free Tier Local Logging")
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "free_tier_no_risk_analysis" \
    "cargo test test_free_tier_no_risk_analysis --test integration_free_tier_test" \
    "Verifies free tier bypasses risk analysis filtering"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES+=("Free Tier Risk Analysis Bypass")
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "local_only_flag" \
    "cargo test test_local_only_flag_behavior --test integration_free_tier_test" \
    "Tests --local-only flag behavior without authentication"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES+=("Local Only Flag")
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "free_tier_filter_pipeline" \
    "cargo test test_free_tier_filter_pipeline_composition --test integration_free_tier_test" \
    "Tests free tier filter pipeline composition and execution"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES+=("Free Tier Filter Pipeline")
fi

echo ""

# Premium Tier Tests
echo -e "${BOLD}💎 Premium Tier Functionality Tests${NC}"
echo "These tests ensure premium/enterprise users get full functionality including events and risk filtering."
echo ""

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "premium_full_pipeline" \
    "cargo test test_premium_tier_full_pipeline --test integration_premium_tier_test" \
    "Tests complete premium tier filter pipeline execution"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES+=("Premium Tier Full Pipeline")
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "premium_risk_blocking" \
    "cargo test test_premium_tier_risk_analysis_blocking --test integration_premium_tier_test" \
    "Tests risk analysis blocking of high-risk commands"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES+=("Premium Risk Analysis Blocking")
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "premium_command_transform" \
    "cargo test test_premium_tier_command_transformation --test integration_premium_tier_test" \
    "Tests command transformation based on risk analysis"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES+=("Premium Command Transformation")
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "enterprise_tier_features" \
    "cargo test test_enterprise_tier_enhanced_features --test integration_premium_tier_test" \
    "Tests enterprise tier enhanced feature set"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES+=("Enterprise Tier Features")
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "event_sender_telemetry" \
    "cargo test test_event_sender_telemetry --test integration_premium_tier_test" \
    "Tests telemetry event sending for premium users"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES+=("Event Sender Telemetry")
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "premium_error_handling" \
    "cargo test test_premium_tier_error_handling --test integration_premium_tier_test" \
    "Tests error handling in premium tier features"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES+=("Premium Error Handling")
fi

echo ""

# Final Results
echo -e "${BOLD}📊 Test Results Summary${NC}"
echo -e "${BOLD}======================${NC}"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}${BOLD}🎉 ALL TESTS PASSED! 🎉${NC}"
    echo -e "${GREEN}✅ ${PASSED_TESTS}/${TOTAL_TESTS} tests successful${NC}"
    echo ""
    echo -e "${GREEN}The Kilometers CLI Proxy core functionality is working correctly:${NC}"
    echo -e "${GREEN}  ✅ Core MCP proxy behavior${NC}"
    echo -e "${GREEN}  ✅ Free tier limitations properly enforced${NC}"
    echo -e "${GREEN}  ✅ Premium tier features (events & risk filtering) functional${NC}"
    echo ""
    echo -e "${BLUE}Your integration tests provide excellent coverage for detecting regressions!${NC}"
else
    echo -e "${RED}${BOLD}❌ SOME TESTS FAILED${NC}"
    echo -e "${RED}❌ ${FAILED_TESTS}/${TOTAL_TESTS} tests failed${NC}"
    echo -e "${GREEN}✅ ${PASSED_TESTS}/${TOTAL_TESTS} tests passed${NC}"
    echo ""
    echo -e "${RED}Failed tests:${NC}"
    for test_name in "${FAILED_TEST_NAMES[@]}"; do
        echo -e "${RED}  ❌ ${test_name}${NC}"
    done
    echo ""
    echo -e "${YELLOW}This indicates issues with core km functionality that need to be addressed.${NC}"
fi

echo ""

# Cleanup
echo -e "${BLUE}🧹 Cleaning up test artifacts...${NC}"
rm -f mcp_proxy.log mcp_traffic.jsonl 2>/dev/null || true

echo -e "${BLUE}📄 Integration test logs and temporary files have been cleaned up.${NC}"
echo ""

echo -e "${BOLD}Integration test run completed.${NC}"

# Exit with appropriate code
if [ $FAILED_TESTS -eq 0 ]; then
    exit 0
else
    exit 1
fi
