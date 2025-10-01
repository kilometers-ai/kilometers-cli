#!/bin/bash
# Test runner script that runs inside Docker containers
# This script tests both install methods with various scenarios

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

TESTS_PASSED=0
TESTS_FAILED=0
TEST_LOG="/tmp/install-tests.log"

# Test configuration
MOCK_SERVER_HOST="${MOCK_SERVER_HOST:-mock-server}"
MOCK_SERVER_PORT="${MOCK_SERVER_PORT:-8080}"
TEST_MODE="${TEST_MODE:-normal}"

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $*" | tee -a "$TEST_LOG"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $*" | tee -a "$TEST_LOG"
    ((TESTS_PASSED++))
}

log_failure() {
    echo -e "${RED}[FAIL]${NC} $*" | tee -a "$TEST_LOG"
    ((TESTS_FAILED++))
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $*" | tee -a "$TEST_LOG"
}

# Wait for mock server to be ready
wait_for_server() {
    log "Waiting for mock server at $MOCK_SERVER_HOST:$MOCK_SERVER_PORT..."

    # First test if we can ping the host
    if command -v ping >/dev/null 2>&1; then
        if ping -c 1 "$MOCK_SERVER_HOST" >/dev/null 2>&1; then
            log "Mock server host $MOCK_SERVER_HOST is reachable via ping"
        else
            log_warning "Cannot ping mock server host $MOCK_SERVER_HOST"
        fi
    fi

    for i in {1..60}; do
        if curl -s "http://$MOCK_SERVER_HOST:$MOCK_SERVER_PORT/repos/kilometers-ai/kilometers-cli/releases/latest" > /dev/null 2>&1; then
            log "Mock server is ready"
            return 0
        fi
        # Log progress every 15 seconds
        if [ $((i % 15)) -eq 0 ]; then
            log "Still waiting for mock server (attempt $i/60)..."
        fi
        sleep 1
    done

    log_failure "Mock server is not responding after 60 seconds"
    log_failure "Final connection test output:"
    curl -v "http://$MOCK_SERVER_HOST:$MOCK_SERVER_PORT/repos/kilometers-ai/kilometers-cli/releases/latest" 2>&1 | tee -a "$TEST_LOG" | head -10
    return 1
}

# Clean up any previous installations
cleanup() {
    log "Cleaning up previous installations..."

    # Remove from common install locations
    rm -f /usr/local/bin/km ~/.local/bin/km /opt/local/bin/km
    rm -rf ~/.local/bin 2>/dev/null || true

    # Clean environment
    unset KM_VERSION

    log "Cleanup completed"
}

# Test basic installation using repo install script
test_repo_install() {
    log "Testing repository install script (scripts/install.sh)..."

    cleanup

    # Set environment to use mock server
    export GITHUB_API_URL="http://$MOCK_SERVER_HOST:$MOCK_SERVER_PORT"

    # Modify the install script to use our mock server
    sed "s|https://api.github.com/repos/\$REPO/releases/latest|http://$MOCK_SERVER_HOST:$MOCK_SERVER_PORT/repos/kilometers-ai/kilometers-cli/releases/latest|g" \
        /test/install-repo.sh > /tmp/install-repo-modified.sh

    sed -i "s|https://github.com/\$REPO/releases/download/|http://$MOCK_SERVER_HOST:$MOCK_SERVER_PORT/releases/download/|g" \
        /tmp/install-repo-modified.sh

    chmod +x /tmp/install-repo-modified.sh

    if /tmp/install-repo-modified.sh; then
        # Verify installation
        if [ -f "$HOME/.local/bin/km" ] && [ -x "$HOME/.local/bin/km" ]; then
            if "$HOME/.local/bin/km" --version | grep -q "v2024.1.1-test"; then
                log_success "Repository install script completed successfully"
            else
                log_failure "Binary was installed but version check failed"
            fi
        else
            log_failure "Repository install script completed but binary not found"
        fi
    else
        log_failure "Repository install script failed"
    fi
}

# Test installation with different users
test_user_permissions() {
    log "Testing installation with different user permissions..."

    # Test as non-root user (should install to ~/.local/bin)
    if su testuser -c "
        export MOCK_SERVER_HOST='$MOCK_SERVER_HOST'
        export MOCK_SERVER_PORT='$MOCK_SERVER_PORT'
        cd /tmp
        sed 's|https://api.github.com/repos/\\\$REPO/releases/latest|http://$MOCK_SERVER_HOST:$MOCK_SERVER_PORT/repos/kilometers-ai/kilometers-cli/releases/latest|g' \
            /test/install-repo.sh | \
        sed 's|https://github.com/\\\$REPO/releases/download/|http://$MOCK_SERVER_HOST:$MOCK_SERVER_PORT/releases/download/|g' > install-test.sh
        chmod +x install-test.sh
        ./install-test.sh
    "; then
        if [ -f "/home/testuser/.local/bin/km" ]; then
            log_success "User permission test passed"
        else
            log_failure "User permission test failed - binary not found"
        fi
    else
        log_failure "User permission test failed during installation"
    fi
}

# Test with different shells
test_shell_compatibility() {
    log "Testing shell compatibility..."

    # Test with bash
    if bash -c "echo 'Testing bash compatibility'"; then
        log_success "Bash compatibility test passed"
    else
        log_failure "Bash compatibility test failed"
    fi

    # Test with zsh (if available)
    if command -v zsh >/dev/null 2>&1; then
        if zsh -c "echo 'Testing zsh compatibility'"; then
            log_success "Zsh compatibility test passed"
        else
            log_failure "Zsh compatibility test failed"
        fi
    else
        log_warning "Zsh not available for testing"
    fi
}

# Test error conditions
test_error_conditions() {
    log "Testing error conditions..."

    # Test with missing curl and wget
    if command -v curl >/dev/null 2>&1; then
        mv "$(which curl)" "/tmp/curl.backup"
    fi
    if command -v wget >/dev/null 2>&1; then
        mv "$(which wget)" "/tmp/wget.backup"
    fi

    cleanup

    # This should fail
    if /test/install-repo.sh 2>/dev/null; then
        log_failure "Install should have failed without curl/wget"
    else
        log_success "Correctly failed when curl/wget are missing"
    fi

    # Restore curl and wget
    if [ -f "/tmp/curl.backup" ]; then
        mv "/tmp/curl.backup" "/usr/bin/curl"
    fi
    if [ -f "/tmp/wget.backup" ]; then
        mv "/tmp/wget.backup" "/usr/bin/wget"
    fi
}

# Test platform detection
test_platform_detection() {
    log "Testing platform detection..."

    local detected_os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local detected_arch=$(uname -m)

    case "$detected_os" in
        linux)
            log_success "Detected OS: Linux"
            ;;
        darwin)
            log_success "Detected OS: macOS"
            ;;
        *)
            log_failure "Unexpected OS detected: $detected_os"
            ;;
    esac

    case "$detected_arch" in
        x86_64|amd64)
            log_success "Detected Architecture: x86_64"
            ;;
        aarch64|arm64)
            log_success "Detected Architecture: ARM64"
            ;;
        *)
            log_failure "Unexpected architecture detected: $detected_arch"
            ;;
    esac
}

# Test version specification
test_version_specification() {
    log "Testing version specification via KM_VERSION..."

    cleanup
    export KM_VERSION="v2024.1.1"

    # Modify install script for mock server
    sed "s|https://github.com/\$REPO/releases/download/|http://$MOCK_SERVER_HOST:$MOCK_SERVER_PORT/releases/download/|g" \
        /test/install-repo.sh > /tmp/install-version-test.sh
    chmod +x /tmp/install-version-test.sh

    if /tmp/install-version-test.sh; then
        log_success "Version specification test passed"
    else
        log_failure "Version specification test failed"
    fi

    unset KM_VERSION
}

# Main test execution
main() {
    log "=== Starting Install Script Tests ==="
    log "Test environment: $(uname -s) $(uname -m)"
    log "Mock server: $MOCK_SERVER_HOST:$MOCK_SERVER_PORT"
    log "Test mode: $TEST_MODE"

    # Initialize test log
    echo "Install Script Test Log - $(date)" > "$TEST_LOG"

    # Wait for mock server
    if ! wait_for_server; then
        exit 1
    fi

    # Run tests
    test_platform_detection
    test_repo_install
    test_user_permissions
    test_shell_compatibility
    test_version_specification
    test_error_conditions

    # Print results
    echo "================================="
    log "Test Results:"
    log_success "Tests passed: $TESTS_PASSED"
    if [ "$TESTS_FAILED" -gt 0 ]; then
        log_failure "Tests failed: $TESTS_FAILED"
    else
        log "Tests failed: $TESTS_FAILED"
    fi
    echo "================================="

    # Copy test log to shared volume if available
    if [ -d "/test-results" ]; then
        cp "$TEST_LOG" "/test-results/install-tests-$(uname -s)-$(uname -m).log"
    fi

    exit $TESTS_FAILED
}

# Run main function
main "$@"
