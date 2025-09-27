#!/bin/bash
set -e

# Script to create test binaries for installation testing
# These are minimal mock binaries that simulate the real km binary

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Creating test binaries for installation testing..."

# Create a simple mock km binary
create_mock_binary() {
    local arch="$1"
    local os="$2"
    local filename="$3"
    local dir="${SCRIPT_DIR}/binaries"

    mkdir -p "$dir"

    cat > "$dir/km" << 'EOF'
#!/bin/bash
# Mock km binary for testing
echo "Kilometers CLI v2024.1.1 (test build)"
echo "Architecture: $(uname -m)"
echo "OS: $(uname -s)"
echo "This is a test binary created for installation script validation"

case "$1" in
    --version)
        echo "km v2024.1.1-test"
        ;;
    --help)
        echo "Usage: km [COMMAND]"
        echo ""
        echo "Commands:"
        echo "  init        Initialize configuration"
        echo "  monitor     Monitor MCP requests"
        echo "  --version   Show version"
        echo "  --help      Show this help"
        ;;
    init)
        echo "Initializing km configuration..."
        echo "This is a test binary - no actual configuration will be created"
        ;;
    *)
        echo "Unknown command: $1"
        echo "Run 'km --help' for usage information"
        exit 1
        ;;
esac
EOF

    chmod +x "$dir/km"

    # Create tar.gz archive
    cd "$dir"
    tar czf "../$filename" km
    cd - > /dev/null

    echo "Created $filename ($(du -h "$SCRIPT_DIR/$filename" | cut -f1))"
}

# Create test binaries for different platforms
# Linux variants
create_mock_binary "x86_64" "linux" "km-x86_64-unknown-linux-gnu.tar.gz"
create_mock_binary "aarch64" "linux" "km-aarch64-unknown-linux-gnu.tar.gz"

# macOS variants
create_mock_binary "x86_64" "darwin" "km-x86_64-apple-darwin.tar.gz"
create_mock_binary "aarch64" "darwin" "km-aarch64-apple-darwin.tar.gz"

# Legacy naming for compatibility with existing install scripts
create_mock_binary "x86_64" "linux" "km-linux-amd64.tar.gz"
create_mock_binary "aarch64" "linux" "km-linux-arm64.tar.gz"
create_mock_binary "x86_64" "darwin" "km-darwin-amd64.tar.gz"
create_mock_binary "aarch64" "darwin" "km-darwin-arm64.tar.gz"

# Create a corrupted archive for error testing
echo "Creating corrupted archive for error testing..."
echo "This is not a valid tar.gz file" > "${SCRIPT_DIR}/km-corrupted.tar.gz"

# Clean up temporary binary directory
rm -rf "$SCRIPT_DIR/binaries"

echo "Test binaries created successfully!"
echo "Files created:"
ls -la "$SCRIPT_DIR"/*.tar.gz
