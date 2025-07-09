#!/bin/sh
# Kilometers.ai CLI Installer
# 
# Installation:
#   curl -sSL https://get.kilometers.ai/install.sh | sh
#
# Downloads binaries from Kilometers.ai CDN for maximum reliability

set -e

# Configuration
BINARY_NAME="km"
GITHUB_REPO="kilometers-ai/kilometers-cli"
INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
info() {
    printf "${GREEN}[INFO]${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}[WARN]${NC} %s\n" "$1"
}

error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1" >&2
    exit 1
}

success() {
    printf "${BLUE}[SUCCESS]${NC} %s\n" "$1"
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $OS in
        linux) OS="linux" ;;
        darwin) OS="darwin" ;;
        mingw*|msys*|cygwin*) OS="windows" ;;
        *) error "Unsupported operating system: $OS" ;;
    esac
    
    case $ARCH in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac
    
    PLATFORM="${OS}-${ARCH}"
    info "Detected platform: $PLATFORM"
}

# Check if we need sudo
check_permissions() {
    if [ -w "$INSTALL_DIR" ]; then
        SUDO=""
        info "Installing to $INSTALL_DIR (no sudo required)"
    else
        if command -v sudo >/dev/null 2>&1; then
            info "Installing to $INSTALL_DIR (requires sudo)"
            SUDO="sudo"
        else
            error "Cannot write to $INSTALL_DIR and sudo is not available"
        fi
    fi
}

# Download the binary from GitHub Releases (most reliable)
download_binary() {
    BINARY_FILE="km-${PLATFORM}"
    if [ "$OS" = "windows" ]; then
        BINARY_FILE="${BINARY_FILE}.exe"
    fi
    
    TEMP_FILE="/tmp/km-download-$$"
    
    # Download from CDN only
    CDN_URL="https://get.kilometers.ai/releases/latest/${BINARY_FILE}"
    
    info "Downloading Kilometers CLI..."
    info "Source: ${CDN_URL}"
    
    if command -v curl >/dev/null 2>&1; then
        if curl -L --fail -o "$TEMP_FILE" "$CDN_URL" 2>/dev/null; then
            success "Download completed successfully"
        else
            error "Download failed. Please check your internet connection or try again later."
        fi
    elif command -v wget >/dev/null 2>&1; then
        if wget -O "$TEMP_FILE" "$CDN_URL" 2>/dev/null; then
            success "Download completed successfully"
        else
            error "Download failed. Please check your internet connection or try again later."
        fi
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
    
    # Verify download
    if [ ! -f "$TEMP_FILE" ] || [ ! -s "$TEMP_FILE" ]; then
        error "Downloaded file is missing or empty"
    fi
}

# Install the binary
install_binary() {
    info "Installing Kilometers CLI..."
    
    chmod +x "$TEMP_FILE"
    $SUDO mv "$TEMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
    
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        success "Installation completed successfully!"
    else
        warn "Binary installed but not found in PATH"
        warn "You may need to add $INSTALL_DIR to your PATH"
    fi
}

# Post-installation setup and info
post_install() {
    echo ""
    success "🎉 Kilometers CLI installed successfully!"
    echo ""
    
    # Show version if available
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        VERSION=$("$BINARY_NAME" --version 2>/dev/null || echo "unknown")
        info "Installed version: $VERSION"
        echo ""
    fi
    
    # Show quick start guide
    printf "${BLUE}Quick Start:${NC}\n"
    echo "  1. Set your API key:"
    echo "     export KILOMETERS_API_KEY=\"your-api-key-here\""
    echo ""
    echo "  2. Wrap your MCP server:"
    echo "     km npx @modelcontextprotocol/server-github"
    echo ""
    echo "  3. View debug logs:"
    echo "     export KM_DEBUG=true"
    echo ""
    
    printf "${BLUE}Resources:${NC}\n"
    echo "  • Dashboard: https://app.kilometers.ai"
    echo "  • Documentation: https://docs.kilometers.ai"
    echo "  • Support: support@kilometers.ai"
    echo ""
    
    printf "${GREEN}✓ Ready to monitor your AI tools!${NC}\n"
}

# Main installation flow
main() {
    echo ""
    printf "${BLUE}Kilometers.ai CLI Installer${NC}\n"
    echo "==============================="
    echo ""
    
    case "${1:-}" in
        --help)
            echo "Usage: $0 [--help]"
            echo ""
            echo "Install Kilometers.ai CLI:"
            echo "  curl -sSL https://get.kilometers.ai/install.sh | sh"
            echo ""
            echo "Downloads binaries from Kilometers.ai CDN for reliability."
            exit 0
            ;;
    esac
    
    detect_platform
    check_permissions
    download_binary
    install_binary
    post_install
}

# Run main function
main "$@" 