#!/bin/bash
# Main test harness for testing install scripts with Docker
# Usage: ./test-install-docker.sh [platform] [test-mode]

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_RESULTS_DIR="$SCRIPT_DIR/results"

# Available platforms for testing
PLATFORMS=(
    "ubuntu-amd64:ubuntu:22.04"
    "alpine:alpine:latest"
    "debian:debian:bookworm-slim"
    "fedora:fedora:latest"
)

# Available test modes
TEST_MODES=(
    "normal"
    "timeout"
    "rate_limit"
    "server_error"
    "malformed_json"
    "corrupted_binary"
    "missing_binary"
)

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Test install scripts using Docker containers"
    echo ""
    echo "Options:"
    echo "  -p, --platform PLATFORM    Test specific platform (default: all)"
    echo "  -m, --mode MODE            Test mode (default: normal)"
    echo "  -c, --cleanup              Clean up containers and images after testing"
    echo "  -v, --verbose              Verbose output"
    echo "  -h, --help                 Show this help message"
    echo ""
    echo "Available platforms:"
    for platform in "${PLATFORMS[@]}"; do
        name=$(echo "$platform" | cut -d: -f1)
        base=$(echo "$platform" | cut -d: -f2-)
        echo "  $name ($base)"
    done
    echo ""
    echo "Available test modes:"
    for mode in "${TEST_MODES[@]}"; do
        echo "  $mode"
    done
}

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

# Parse command line arguments
PLATFORM=""
TEST_MODE="normal"
CLEANUP=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--platform)
            PLATFORM="$2"
            shift 2
            ;;
        -m|--mode)
            TEST_MODE="$2"
            shift 2
            ;;
        -c|--cleanup)
            CLEANUP=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate test mode
if [[ ! " ${TEST_MODES[@]} " =~ " ${TEST_MODE} " ]]; then
    log_error "Invalid test mode: $TEST_MODE"
    echo "Available modes: ${TEST_MODES[*]}"
    exit 1
fi

# Setup test environment
setup_test_env() {
    log "Setting up test environment..."

    # Create results directory
    mkdir -p "$TEST_RESULTS_DIR"

    # Create Docker network for testing
    if ! docker network ls | grep -q "km-test-network"; then
        if docker network create km-test-network 2>/dev/null; then
            log "Created Docker network: km-test-network"
        else
            log "Docker network km-test-network already exists or creation failed, continuing..."
        fi
    else
        log "Docker network km-test-network already exists"
    fi
}

# Start mock server
start_mock_server() {
    local mode="$1"
    log "Starting mock server in mode: $mode"

    # Stop existing mock server if running
    docker stop km-mock-server 2>/dev/null || true
    docker rm km-mock-server 2>/dev/null || true

    # Build mock server image if needed
    if ! docker images | grep -q "km-mock-server"; then
        log "Building mock server image..."
        docker build -t km-mock-server -f - "$SCRIPT_DIR/mock-server" << 'EOF'
FROM python:3.11-slim
WORKDIR /app
COPY server.py .
EXPOSE 8080
CMD ["python", "server.py", "--port", "8080"]
EOF
    fi

    # Start mock server
    docker run -d \
        --name km-mock-server \
        --network km-test-network \
        -p ${MOCK_SERVER_PORT:-8088}:8080 \
        -v "$SCRIPT_DIR/data:/app/data:ro" \
        km-mock-server \
        python server.py --port 8080 --mode "$mode"

    # Wait for server to be ready
    log "Waiting for mock server to be ready..."
    for i in {1..30}; do
        if curl -s "http://localhost:${MOCK_SERVER_PORT:-8088}/repos/kilometers-ai/kilometers-cli/releases/latest" >/dev/null 2>&1; then
            log_success "Mock server is ready"
            return 0
        fi
        sleep 1
    done

    log_error "Mock server failed to start"
    return 1
}

# Build Docker image for testing
build_test_image() {
    local platform="$1"
    local dockerfile="$2"
    local base_image="$3"

    log "Building test image for $platform..."

    # Update Dockerfile to use correct base image
    sed "s|FROM .*|FROM $base_image|" "$dockerfile" > "$dockerfile.tmp"

    if docker build \
        -f "$dockerfile.tmp" \
        -t "km-test-$platform" \
        "$SCRIPT_DIR/docker"; then

        rm "$dockerfile.tmp"
        log_success "Built test image: km-test-$platform"
        return 0
    else
        rm "$dockerfile.tmp"
        log_error "Failed to build Docker image for $platform"
        return 1
    fi
}

# Run tests on a platform
test_platform() {
    local platform_spec="$1"
    local test_mode="$2"

    local platform_name=$(echo "$platform_spec" | cut -d: -f1)
    local base_image=$(echo "$platform_spec" | cut -d: -f2-)

    log "Testing platform: $platform_name ($base_image)"

    # Build test image
    local dockerfile="$SCRIPT_DIR/docker/Dockerfile.$platform_name"
    if [ ! -f "$dockerfile" ]; then
        log_warning "Dockerfile not found: $dockerfile, using ubuntu fallback"
        # Use ubuntu as fallback for missing Dockerfiles
        dockerfile="$SCRIPT_DIR/docker/Dockerfile.ubuntu-amd64"
    fi

    log "Building test image with dockerfile: $dockerfile"
    if ! build_test_image "$platform_name" "$dockerfile" "$base_image"; then
        log_error "Failed to build test image for $platform_name"
        return 1
    fi

    # Run tests
    local container_name="km-test-$platform_name-$$"
    local test_log="$TEST_RESULTS_DIR/test-$platform_name-$test_mode.log"

    log "Running tests on $platform_name..."
    log "Container name: $container_name"
    log "Mock server host: km-mock-server"
    log "Mock server port: ${MOCK_SERVER_PORT:-8080}"
    log "Test mode: $test_mode"

    # Test network connectivity first
    if [ "${SKIP_MOCK_SERVER:-}" = "1" ]; then
        log "Testing connectivity to mock server..."
        if ! docker run --rm --network km-test-network alpine:latest \
            sh -c "ping -c 1 km-mock-server >/dev/null 2>&1"; then
            log_error "Cannot reach km-mock-server on km-test-network"
            return 1
        fi
        log "Mock server is reachable on network"
    fi

    if docker run --rm \
        --name "$container_name" \
        --network km-test-network \
        -e MOCK_SERVER_HOST=km-mock-server \
        -e MOCK_SERVER_PORT=${MOCK_SERVER_PORT:-8080} \
        -e TEST_MODE="$test_mode" \
        -v "$TEST_RESULTS_DIR:/test-results" \
        -v "$PROJECT_ROOT/install.sh:/test/install-local.sh:ro" \
        -v "$PROJECT_ROOT/scripts/install.sh:/test/install-repo.sh:ro" \
        "km-test-$platform_name" 2>&1 | tee "$test_log"; then

        log_success "Tests passed on $platform_name"
        return 0
    else
        log_error "Tests failed on $platform_name"
        log_error "Last 10 lines of test log:"
        if [ -f "$test_log" ]; then
            tail -10 "$test_log" | while read -r line; do
                log_error "  $line"
            done
        fi
        return 1
    fi
}

# Cleanup function
cleanup_test_env() {
    if [ "$CLEANUP" = true ]; then
        log "Cleaning up test environment..."

        # Stop and remove containers
        docker stop km-mock-server 2>/dev/null || true
        docker rm km-mock-server 2>/dev/null || true

        # Remove test images
        for platform in "${PLATFORMS[@]}"; do
            local platform_name=$(echo "$platform" | cut -d: -f1)
            docker rmi "km-test-$platform_name" 2>/dev/null || true
        done

        # Remove mock server image
        docker rmi km-mock-server 2>/dev/null || true

        # Remove network
        docker network rm km-test-network 2>/dev/null || true

        log_success "Cleanup completed"
    fi
}

# Main execution
main() {
    log "=== Starting Docker Install Script Tests ==="
    log "Test mode: $TEST_MODE"
    log "Platform filter: ${PLATFORM:-all}"

    # Setup
    setup_test_env

    # Start mock server only if not using external service container
    if [ "${SKIP_MOCK_SERVER:-}" != "1" ]; then
        start_mock_server "$TEST_MODE"
    else
        log "Using external mock server, skipping local startup"
    fi

    # Determine which platforms to test
    log "Determining platforms to test..."
    log "PLATFORM variable: '${PLATFORM:-}'"

    local platforms_to_test=()
    if [ -n "$PLATFORM" ]; then
        log "Testing specific platform: $PLATFORM"
        # Test specific platform
        local found=false
        for platform in "${PLATFORMS[@]}"; do
            local platform_name=$(echo "$platform" | cut -d: -f1)
            log "Checking platform: $platform_name"
            if [ "$platform_name" = "$PLATFORM" ]; then
                platforms_to_test+=("$platform")
                found=true
                log "Found matching platform: $platform"
                break
            fi
        done
        if [ "$found" = false ]; then
            log_error "Platform not found: $PLATFORM"
            log "Available platforms: $(printf '%s ' "${PLATFORMS[@]}")"
            exit 1
        fi
    else
        log "Testing all platforms"
        # Test all platforms
        platforms_to_test=("${PLATFORMS[@]}")
    fi

    log "Platforms to test: ${platforms_to_test[*]}"

    # Run tests
    log "Starting test execution phase..."
    local total_tests=0
    local passed_tests=0

    for platform in "${platforms_to_test[@]}"; do
        log "Processing platform: $platform"
        ((total_tests++))
        if test_platform "$platform" "$TEST_MODE"; then
            ((passed_tests++))
            log "Platform $platform passed"
        else
            log_error "Platform $platform failed"
        fi
    done

    log "Finished test execution phase"

    # Results summary
    log "=== Test Results Summary ==="
    log "Total platforms tested: $total_tests"
    log_success "Tests passed: $passed_tests"

    if [ $((total_tests - passed_tests)) -gt 0 ]; then
        log_error "Tests failed: $((total_tests - passed_tests))"
    else
        log "Tests failed: 0"
    fi

    log "Test logs saved to: $TEST_RESULTS_DIR"

    # Cleanup
    cleanup_test_env

    # Exit with appropriate code
    if [ "$passed_tests" -eq "$total_tests" ]; then
        log_success "All tests passed!"
        exit 0
    else
        log_error "Some tests failed!"
        exit 1
    fi
}

# Trap cleanup on exit
trap cleanup_test_env EXIT

# Run main function
main "$@"
