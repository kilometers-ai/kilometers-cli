#!/bin/bash
set -e

echo "🚀 Running comprehensive integration tests for km CLI..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if binary exists
if [ ! -f "./target/debug/km" ] && [ ! -f "./target/release/km" ]; then
    print_error "km binary not found. Please build the project first."
    exit 1
fi

# Use release binary if available, otherwise debug
if [ -f "./target/release/km" ]; then
    KM_BINARY="./target/release/km"
    print_status "Using release binary: $KM_BINARY"
else
    KM_BINARY="./target/debug/km"
    print_status "Using debug binary: $KM_BINARY"
fi

print_status "Testing binary functionality..."

# Test 1: Version check
print_status "Test 1: Version check"
if $KM_BINARY --version; then
    print_status "✅ Version check passed"
else
    print_error "❌ Version check failed"
    exit 1
fi

# Test 2: Help command
print_status "Test 2: Help command"
if $KM_BINARY --help > /dev/null; then
    print_status "✅ Help command passed"
else
    print_error "❌ Help command failed"
    exit 1
fi

# Test 3: Init command (dry run)
print_status "Test 3: Init command validation"
if $KM_BINARY init --help > /dev/null; then
    print_status "✅ Init command help passed"
else
    print_error "❌ Init command help failed"
    exit 1
fi

# Test 4: Monitor command validation
print_status "Test 4: Monitor command validation"
if $KM_BINARY monitor --help > /dev/null; then
    print_status "✅ Monitor command help passed"
else
    print_error "❌ Monitor command help failed"
    exit 1
fi

# Test 5: Clear-logs command validation
print_status "Test 5: Clear-logs command validation"
if $KM_BINARY clear-logs --help > /dev/null; then
    print_status "✅ Clear-logs command help passed"
else
    print_error "❌ Clear-logs command help failed"
    exit 1
fi

# Test 6: Configuration directory creation
print_status "Test 6: Configuration handling"
# Test without actually creating config to avoid side effects
if $KM_BINARY init --help | grep -q "api-key"; then
    print_status "✅ Configuration parameter validation passed"
else
    print_error "❌ Configuration parameter validation failed"
    exit 1
fi

# Test 7: Run unit tests to ensure integration
print_status "Test 7: Running unit tests"
if cargo test --lib --verbose; then
    print_status "✅ Unit tests passed"
else
    print_error "❌ Unit tests failed"
    exit 1
fi

# Test 8: Run integration tests
print_status "Test 8: Running integration tests"
if cargo test --test '*' --verbose; then
    print_status "✅ Integration tests passed"
else
    print_error "❌ Integration tests failed"
    exit 1
fi

# Test 9: Binary size check (ensure it's reasonable)
print_status "Test 9: Binary size check"
BINARY_SIZE=$(stat -c%s "$KM_BINARY" 2>/dev/null || stat -f%z "$KM_BINARY" 2>/dev/null || echo "0")
BINARY_SIZE_MB=$((BINARY_SIZE / 1024 / 1024))
if [ $BINARY_SIZE_MB -lt 50 ]; then
    print_status "✅ Binary size check passed ($BINARY_SIZE_MB MB)"
else
    print_warning "⚠️  Binary size is large ($BINARY_SIZE_MB MB) but acceptable"
fi

# Test 10: Ensure no debug symbols in release build
if [ "$KM_BINARY" = "./target/release/km" ]; then
    print_status "Test 10: Release build validation"
    if file "$KM_BINARY" | grep -q "not stripped"; then
        print_warning "⚠️  Release binary contains debug symbols"
    else
        print_status "✅ Release binary is properly stripped"
    fi
fi

print_status "🎉 All integration tests completed successfully!"
print_status "Binary is ready for deployment: $KM_BINARY"