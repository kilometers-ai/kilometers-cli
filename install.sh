#!/bin/bash
set -e

# Kilometers CLI installer for Unix-like systems (Linux, macOS)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="km"
DIST_HOST="https://get.kilometers.ai"
INSTALL_DIR="$HOME/.local/bin"

# Functions
print_info() {
    printf "${GREEN}[INFO]${NC} %s\n" "$1"
}

print_warn() {
    printf "${YELLOW}[WARN]${NC} %s\n" "$1"
}

print_error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1"
    exit 1
}

# Detect OS and architecture
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case "$os" in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        *)
            print_error "Unsupported operating system: $os"
            ;;
    esac

    case "$arch" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $arch"
            ;;
    esac

    PLATFORM="${OS}-${ARCH}"
}

# Get the latest release version
get_latest_version() {
    print_info "Fetching latest release information..."

    local version_url="${DIST_HOST}/latest-version"

    if command -v curl >/dev/null 2>&1; then
        VERSION=$(curl -sL "$version_url")
    elif command -v wget >/dev/null 2>&1; then
        VERSION=$(wget -qO- "$version_url")
    else
        print_error "Neither curl nor wget is available. Please install one of them."
    fi

    if [ -z "$VERSION" ]; then
        print_warn "Failed to get latest version from ${DIST_HOST}. Falling back to GitHub."
        # Fallback to GitHub API if distribution server is down or not yet populated
        GITHUB_REPO="kilometers-ai/kilometers-cli"
        if command -v curl >/dev/null 2>&1; then
            VERSION=$(curl -sL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | \
                     grep '"tag_name":' | \
                     sed -E 's/.*"([^"]+)".*/\1/')
        elif command -v wget >/dev/null 2>&1; then
            VERSION=$(wget -qO- "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | \
                     grep '"tag_name":' | \
                     sed -E 's/.*"([^"]+)".*/\1/')
        fi
    fi

    if [ -z "$VERSION" ]; then
        print_error "Failed to get latest version"
    fi

    # Ensure version starts with v
    if [[ ! "$VERSION" =~ ^v ]]; then
        VERSION="v$VERSION"
    fi

    print_info "Latest version: $VERSION"
}

# Download and extract binary
download_binary() {
    local filename="${BINARY_NAME}-${PLATFORM}.tar.gz"
    # Try distribution URL first
    local url="${DIST_HOST}/releases/${VERSION}/${filename}"
    local temp_dir=$(mktemp -d)

    print_info "Downloading $filename from $url..."

    local download_success=false

    if command -v curl >/dev/null 2>&1; then
        if curl -sL --fail "$url" -o "$temp_dir/$filename"; then
            download_success=true
        fi
    elif command -v wget >/dev/null 2>&1; then
        if wget -q "$url" -O "$temp_dir/$filename"; then
            download_success=true
        fi
    fi

    # Fallback to GitHub Releases if CDN fails
    if [ "$download_success" = false ]; then
        print_warn "Download from ${DIST_HOST} failed. Falling back to GitHub Releases."
        GITHUB_REPO="kilometers-ai/kilometers-cli"
        url="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${filename}"
        print_info "Downloading from $url..."

        if command -v curl >/dev/null 2>&1; then
            curl -sL "$url" -o "$temp_dir/$filename"
        elif command -v wget >/dev/null 2>&1; then
            wget -q "$url" -O "$temp_dir/$filename"
        fi
    fi

    if [ ! -f "$temp_dir/$filename" ] || [ ! -s "$temp_dir/$filename" ]; then
        print_error "Failed to download $filename"
    fi

    print_info "Extracting binary..."
    cd "$temp_dir"
    tar -xzf "$filename"

    if [ ! -f "$BINARY_NAME" ]; then
        print_error "Binary not found in archive"
    fi

    # Create install directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"

    # Install binary
    print_info "Installing to $INSTALL_DIR/$BINARY_NAME..."
    cp "$BINARY_NAME" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"

    # Cleanup
    cd - >/dev/null
    rm -rf "$temp_dir"
}

# Check if binary is in PATH
check_path() {
    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        print_warn "$INSTALL_DIR is not in your PATH"
        print_warn "Add this line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        printf "${YELLOW}export PATH=\"\$PATH:%s\"${NC}\n" "$INSTALL_DIR"
        print_warn "Then restart your shell or run: source ~/.bashrc (or your shell config file)"
    fi
}

# Verify installation
verify_installation() {
    if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        print_info "Installation successful!"
        print_info "Binary location: $INSTALL_DIR/$BINARY_NAME"

        # Try to run the binary
        if "$INSTALL_DIR/$BINARY_NAME" --help >/dev/null 2>&1; then
            print_info "Binary is working correctly"
        else
            print_warn "Binary may not be working correctly"
        fi

        check_path

        printf "\n${GREEN}Next steps:${NC}\n"
        printf "1. Ensure $INSTALL_DIR is in your PATH (see warning above if needed)\n"
        printf "2. Run: ${YELLOW}km init${NC} to configure your API key\n"
        printf "3. Start monitoring: ${YELLOW}km monitor -- <your-mcp-command>${NC}\n"
    else
        print_error "Installation failed: binary not found at $INSTALL_DIR/$BINARY_NAME"
    fi
}

# Main execution
main() {
    print_info "Installing Kilometers CLI..."

    detect_platform
    print_info "Detected platform: $PLATFORM"

    get_latest_version
    download_binary
    verify_installation
}

# Allow specifying version via environment variable
if [ -n "$KM_VERSION" ]; then
    VERSION="$KM_VERSION"
    print_info "Using specified version: $VERSION"
    detect_platform
    download_binary
    verify_installation
else
    main
fi
