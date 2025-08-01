#!/bin/bash

# Kilometers CLI Deployment Script
# Deploys to various package managers and distribution channels

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
APP_NAME="kilometers-cli"
BINARY_NAME="km"
VERSION=${VERSION:-"dev"}
DIST_DIR="dist"
REPO_URL="https://github.com/kilometers-ai/kilometers-cli"
DOWNLOAD_URL_BASE="https://github.com/kilometers-ai/kilometers-cli/releases/download/v${VERSION}"

echo -e "${BLUE}🚀 Deploying Kilometers CLI v${VERSION}${NC}"
echo -e "${BLUE}======================================${NC}"
echo

# Check if distribution files exist
if [ ! -d "$DIST_DIR" ]; then
    echo -e "${RED}❌ Distribution directory not found. Run build script first.${NC}"
    exit 1
fi

# 1. Generate Homebrew Formula
echo -e "${YELLOW}🍺 Generating Homebrew formula...${NC}"
mkdir -p deploy/homebrew

# Calculate SHA256 for macOS binary
MACOS_AMD64_FILE=$(ls $DIST_DIR/*darwin_amd64.tar.gz | head -1)
MACOS_ARM64_FILE=$(ls $DIST_DIR/*darwin_arm64.tar.gz | head -1)

if [ -f "$MACOS_AMD64_FILE" ] && [ -f "$MACOS_ARM64_FILE" ]; then
    MACOS_AMD64_SHA=$(shasum -a 256 "$MACOS_AMD64_FILE" | cut -d ' ' -f 1)
    MACOS_ARM64_SHA=$(shasum -a 256 "$MACOS_ARM64_FILE" | cut -d ' ' -f 1)
    
    cat > deploy/homebrew/kilometers-cli.rb << EOF
class KilometersCli < Formula
  desc "MCP server monitoring and analysis tool with advanced plugin system"
  homepage "$REPO_URL"
  version "$VERSION"
  license "MIT"

  if Hardware::CPU.arm?
    url "${DOWNLOAD_URL_BASE}/$(basename $MACOS_ARM64_FILE)"
    sha256 "$MACOS_ARM64_SHA"
  else
    url "${DOWNLOAD_URL_BASE}/$(basename $MACOS_AMD64_FILE)"
    sha256 "$MACOS_AMD64_SHA"
  end

  def install
    bin.install "$BINARY_NAME"
    
    # Generate shell completions
    generate_completions_from_executable(bin/"$BINARY_NAME", "completion")
  end

  test do
    system "#{bin}/$BINARY_NAME", "version"
    assert_match "$VERSION", shell_output("#{bin}/$BINARY_NAME version")
  end
end
EOF
    echo -e "${GREEN}✅ Homebrew formula generated: deploy/homebrew/kilometers-cli.rb${NC}"
else
    echo -e "${YELLOW}⚠️  macOS binaries not found, skipping Homebrew formula${NC}"
fi

# 2. Generate Scoop Manifest (Windows)
echo -e "${YELLOW}🪣 Generating Scoop manifest...${NC}"
mkdir -p deploy/scoop

WINDOWS_FILE=$(ls $DIST_DIR/*windows_amd64.zip | head -1)
if [ -f "$WINDOWS_FILE" ]; then
    WINDOWS_SHA=$(shasum -a 256 "$WINDOWS_FILE" | cut -d ' ' -f 1)
    
    cat > deploy/scoop/kilometers-cli.json << EOF
{
    "version": "$VERSION",
    "description": "MCP server monitoring and analysis tool with advanced plugin system",
    "homepage": "$REPO_URL",
    "license": "MIT",
    "url": "${DOWNLOAD_URL_BASE}/$(basename $WINDOWS_FILE)",
    "hash": "$WINDOWS_SHA",
    "extract_dir": "$(basename $WINDOWS_FILE .zip)",
    "bin": "$BINARY_NAME.exe",
    "checkver": {
        "github": "$REPO_URL"
    },
    "autoupdate": {
        "url": "https://github.com/kilometers-ai/kilometers-cli/releases/download/v\$version/kilometers-cli_\$version_windows_amd64.zip",
        "extract_dir": "kilometers-cli_\$version_windows_amd64"
    }
}
EOF
    echo -e "${GREEN}✅ Scoop manifest generated: deploy/scoop/kilometers-cli.json${NC}"
else
    echo -e "${YELLOW}⚠️  Windows binary not found, skipping Scoop manifest${NC}"
fi

# 3. Generate Debian Package Info
echo -e "${YELLOW}📦 Generating Debian package info...${NC}"
mkdir -p deploy/debian

LINUX_AMD64_FILE=$(ls $DIST_DIR/*linux_amd64.tar.gz | head -1)
if [ -f "$LINUX_AMD64_FILE" ]; then
    LINUX_AMD64_SHA=$(shasum -a 256 "$LINUX_AMD64_FILE" | cut -d ' ' -f 1)
    
    cat > deploy/debian/control << EOF
Package: $APP_NAME
Version: $VERSION
Section: utils
Priority: optional
Architecture: amd64
Depends: libc6 (>= 2.17)
Maintainer: Kilometers.ai <support@kilometers.ai>
Description: MCP server monitoring and analysis tool
 Kilometers CLI provides comprehensive monitoring and analysis capabilities
 for Model Context Protocol (MCP) servers, featuring:
 .
 * Real-time MCP message monitoring and logging
 * Advanced filtering and custom rule engines (Pro tier)
 * AI-powered security analysis and threat detection (Pro tier)
 * Machine learning analytics and insights (Pro tier)
 * Compliance reporting and audit trails (Enterprise tier)
 * Team collaboration and shared configurations (Enterprise tier)
 .
 The tool acts as a transparent proxy between MCP clients and servers,
 providing complete visibility into message flows without disrupting
 communication.
Homepage: https://kilometers.ai
EOF

    cat > deploy/debian/install-instructions.md << 'EOF'
# Debian/Ubuntu Installation

## Method 1: Download and Install Manually

```bash
# Download the binary
wget https://github.com/kilometers-ai/kilometers-cli/releases/download/v{VERSION}/kilometers-cli_{VERSION}_linux_amd64.tar.gz

# Extract
tar -xzf kilometers-cli_{VERSION}_linux_amd64.tar.gz

# Install
sudo mv kilometers-cli_{VERSION}_linux_amd64/km /usr/local/bin/
sudo chmod +x /usr/local/bin/km

# Verify installation
km version
```

## Method 2: Using Package Manager (Coming Soon)

We're working on official APT repository support. For now, use the manual method above.
EOF

    echo -e "${GREEN}✅ Debian package info generated: deploy/debian/${NC}"
else
    echo -e "${YELLOW}⚠️  Linux binary not found, skipping Debian package${NC}"
fi

# 4. Generate RPM Package Info
echo -e "${YELLOW}📦 Generating RPM package info...${NC}"
mkdir -p deploy/rpm

if [ -f "$LINUX_AMD64_FILE" ]; then
    cat > deploy/rpm/kilometers-cli.spec << EOF
Name:           $APP_NAME
Version:        $VERSION
Release:        1%{?dist}
Summary:        MCP server monitoring and analysis tool

License:        MIT
URL:            $REPO_URL
Source0:        ${DOWNLOAD_URL_BASE}/$(basename $LINUX_AMD64_FILE)

BuildArch:      x86_64
Requires:       glibc

%description
Kilometers CLI provides comprehensive monitoring and analysis capabilities
for Model Context Protocol (MCP) servers, featuring real-time monitoring,
advanced filtering, security analysis, and compliance reporting.

%prep
%setup -q -n $(basename $LINUX_AMD64_FILE .tar.gz)

%install
mkdir -p %{buildroot}%{_bindir}
cp $BINARY_NAME %{buildroot}%{_bindir}/

%files
%{_bindir}/$BINARY_NAME

%changelog
* $(date '+%a %b %d %Y') Kilometers.ai <support@kilometers.ai> - $VERSION-1
- Version $VERSION release
EOF

    cat > deploy/rpm/install-instructions.md << 'EOF'
# RHEL/CentOS/Fedora Installation

## Method 1: Download and Install Manually

```bash
# Download the binary
curl -L -o kilometers-cli.tar.gz https://github.com/kilometers-ai/kilometers-cli/releases/download/v{VERSION}/kilometers-cli_{VERSION}_linux_amd64.tar.gz

# Extract
tar -xzf kilometers-cli.tar.gz

# Install
sudo mv kilometers-cli_{VERSION}_linux_amd64/km /usr/local/bin/
sudo chmod +x /usr/local/bin/km

# Verify installation
km version
```

## Method 2: Using Package Manager (Coming Soon)

We're working on official YUM/DNF repository support. For now, use the manual method above.
EOF

    echo -e "${GREEN}✅ RPM package info generated: deploy/rpm/${NC}"
fi

# 5. Generate Docker Instructions
echo -e "${YELLOW}🐳 Generating Docker deployment...${NC}"
mkdir -p deploy/docker

cat > deploy/docker/Dockerfile << 'EOF'
# Multi-stage build for Kilometers CLI
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o km ./cmd/

FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/km .

# Make it executable
RUN chmod +x ./km

# Create directories for configuration
RUN mkdir -p ~/.config/kilometers

ENTRYPOINT ["./km"]
EOF

cat > deploy/docker/docker-compose.yml << 'EOF'
version: '3.8'

services:
  kilometers-cli:
    build: .
    container_name: kilometers-cli
    volumes:
      - ~/.config/kilometers:/root/.config/kilometers
      - /var/run/docker.sock:/var/run/docker.sock  # For monitoring Docker-based MCP servers
    stdin_open: true
    tty: true
    network_mode: host
    command: ["monitor", "--server", "--", "your-mcp-server-command"]
EOF

cat > deploy/docker/README.md << 'EOF'
# Docker Deployment

## Using Pre-built Image (Coming Soon)

```bash
docker run -it kilometers/cli:latest version
```

## Building Locally

```bash
# Build the image
docker build -t kilometers-cli .

# Run with interactive mode
docker run -it \
  -v ~/.config/kilometers:/root/.config/kilometers \
  kilometers-cli version

# Monitor an MCP server
docker run -it \
  -v ~/.config/kilometers:/root/.config/kilometers \
  --network host \
  kilometers-cli monitor --server -- npx -y @modelcontextprotocol/server-github
```

## Docker Compose

```bash
# Edit docker-compose.yml to specify your MCP server command
docker-compose up
```
EOF

echo -e "${GREEN}✅ Docker deployment generated: deploy/docker/${NC}"

# 6. Generate GitHub Actions for Auto-deployment
echo -e "${YELLOW}⚙️  Generating GitHub Actions...${NC}"
mkdir -p deploy/github-actions

cat > deploy/github-actions/release.yml << 'EOF'
name: Release Kilometers CLI

on:
  push:
    tags:
      - 'v*'

jobs:
  build-and-release:
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
      
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
        
    - name: Run tests
      run: go test -v ./...
      
    - name: Build binaries
      run: ./scripts/build-with-plugins.sh
      env:
        VERSION: ${{ github.ref_name }}
        
    - name: Create Release
      uses: softprops/action-gh-release@v1
      with:
        files: dist/*
        draft: false
        prerelease: false
        generate_release_notes: true
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        
    - name: Update Homebrew Formula
      uses: mislav/bump-homebrew-formula-action@v2
      with:
        formula-name: kilometers-cli
        formula-path: Formula/kilometers-cli.rb
        homebrew-tap: kilometers-ai/homebrew-tap
        download-url: https://github.com/kilometers-ai/kilometers-cli/releases/download/${{ github.ref_name }}/kilometers-cli_${{ github.ref_name }}_darwin_amd64.tar.gz
        commit-message: |
          Update kilometers-cli to ${{ github.ref_name }}
          
          Auto-generated by GitHub Actions
      env:
        COMMITTER_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
EOF

echo -e "${GREEN}✅ GitHub Actions generated: deploy/github-actions/${NC}"

# 7. Generate Installation Scripts
echo -e "${YELLOW}📜 Generating installation scripts...${NC}"
mkdir -p deploy/scripts

cat > deploy/scripts/install.sh << 'EOF'
#!/bin/bash

# Kilometers CLI Installation Script
# This script automatically detects your platform and installs the appropriate binary

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
REPO="kilometers-ai/kilometers-cli"
BINARY_NAME="km"
INSTALL_DIR="/usr/local/bin"

echo -e "${BLUE}🚀 Kilometers CLI Installer${NC}"
echo -e "${BLUE}============================${NC}"

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo -e "${RED}❌ Unsupported architecture: $ARCH${NC}"; exit 1 ;;
esac

case $OS in
    linux) PLATFORM="linux" ;;
    darwin) PLATFORM="darwin" ;;
    *) echo -e "${RED}❌ Unsupported OS: $OS${NC}"; exit 1 ;;
esac

echo "Detected platform: ${PLATFORM}/${ARCH}"

# Get latest release
echo -e "${YELLOW}📡 Fetching latest release...${NC}"
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_RELEASE" ]; then
    echo -e "${RED}❌ Failed to get latest release${NC}"
    exit 1
fi

echo "Latest version: $LATEST_RELEASE"

# Download binary
FILENAME="kilometers-cli_${LATEST_RELEASE}_${PLATFORM}_${ARCH}"
if [ "$PLATFORM" = "linux" ] || [ "$PLATFORM" = "darwin" ]; then
    FILENAME="${FILENAME}.tar.gz"
else
    FILENAME="${FILENAME}.zip"
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_RELEASE/$FILENAME"

echo -e "${YELLOW}⬇️  Downloading: $DOWNLOAD_URL${NC}"
curl -L -o "/tmp/$FILENAME" "$DOWNLOAD_URL"

# Extract and install
echo -e "${YELLOW}📦 Installing...${NC}"
cd /tmp

if [ "$PLATFORM" = "linux" ] || [ "$PLATFORM" = "darwin" ]; then
    tar -xzf "$FILENAME"
    EXTRACT_DIR="kilometers-cli_${LATEST_RELEASE}_${PLATFORM}_${ARCH}"
else
    unzip "$FILENAME"
    EXTRACT_DIR="kilometers-cli_${LATEST_RELEASE}_${PLATFORM}_${ARCH}"
fi

# Install binary
if [ -w "$INSTALL_DIR" ]; then
    cp "$EXTRACT_DIR/$BINARY_NAME" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
else
    echo "Installing to $INSTALL_DIR (requires sudo)..."
    sudo cp "$EXTRACT_DIR/$BINARY_NAME" "$INSTALL_DIR/"
    sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
fi

# Cleanup
rm -rf "/tmp/$FILENAME" "/tmp/$EXTRACT_DIR"

# Verify installation
echo -e "${YELLOW}✅ Verifying installation...${NC}"
if command -v $BINARY_NAME >/dev/null 2>&1; then
    VERSION_OUTPUT=$($BINARY_NAME version)
    echo -e "${GREEN}✅ Successfully installed!${NC}"
    echo -e "${GREEN}$VERSION_OUTPUT${NC}"
    echo
    echo -e "${BLUE}🚀 Quick Start:${NC}"
    echo "  1. Check subscription: $BINARY_NAME auth status"
    echo "  2. Login (if you have a license): $BINARY_NAME auth login --license-key YOUR_KEY"
    echo "  3. List plugins: $BINARY_NAME plugins list"
    echo "  4. Start monitoring: $BINARY_NAME monitor --server -- your-mcp-server-command"
    echo
    echo -e "${BLUE}📚 Documentation: https://kilometers.ai/docs${NC}"
else
    echo -e "${RED}❌ Installation failed - binary not found in PATH${NC}"
    exit 1
fi
EOF

chmod +x deploy/scripts/install.sh

echo -e "${GREEN}✅ Installation script generated: deploy/scripts/install.sh${NC}"

# 8. Generate Documentation
echo -e "${YELLOW}📚 Generating deployment documentation...${NC}"

cat > deploy/DEPLOYMENT.md << 'EOF'
# Kilometers CLI Deployment Guide

This directory contains all the necessary files and configurations for deploying Kilometers CLI across various platforms and package managers.

## Package Managers

### Homebrew (macOS)
- Formula: `homebrew/kilometers-cli.rb`
- Submit to: https://github.com/kilometers-ai/homebrew-tap

### Scoop (Windows)
- Manifest: `scoop/kilometers-cli.json`
- Submit to: https://github.com/ScoopInstaller/Main

### Debian/Ubuntu
- Package info: `debian/control`
- Manual installation: `debian/install-instructions.md`

### RHEL/CentOS/Fedora
- Spec file: `rpm/kilometers-cli.spec`
- Manual installation: `rpm/install-instructions.md`

## Container Deployment

### Docker
- Dockerfile: `docker/Dockerfile`
- Docker Compose: `docker/docker-compose.yml`
- Instructions: `docker/README.md`

## Automated Deployment

### GitHub Actions
- Release workflow: `github-actions/release.yml`
- Automatically creates releases and updates package managers

## Installation Scripts

### Universal Installer
- Script: `scripts/install.sh`
- Usage: `curl -sSL https://install.kilometers.ai | bash`

## Deployment Checklist

1. **Build Release**
   ```bash
   ./scripts/build-with-plugins.sh
   ```

2. **Test Binaries**
   ```bash
   ./build/km_linux_amd64 version
   ./build/km_darwin_amd64 version
   ./build/km_windows_amd64.exe version
   ```

3. **Create GitHub Release**
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

4. **Update Package Managers**
   - Homebrew: Submit PR to homebrew-tap
   - Scoop: Submit PR to main bucket
   - APT/YUM: Update repository

5. **Update Documentation**
   - Installation instructions
   - Feature documentation
   - Plugin system guide

## Support Channels

- Documentation: https://kilometers.ai/docs
- Issues: https://github.com/kilometers-ai/kilometers-cli/issues
- Support: support@kilometers.ai
EOF

echo -e "${GREEN}✅ Deployment documentation generated: deploy/DEPLOYMENT.md${NC}"

# Summary
echo
echo -e "${BLUE}📊 Deployment Summary${NC}"
echo -e "${BLUE}===================${NC}"
echo "Generated deployment assets for:"
echo "  ✅ Homebrew (macOS)"
echo "  ✅ Scoop (Windows)"
echo "  ✅ Debian/Ubuntu packages"
echo "  ✅ RHEL/CentOS/Fedora packages"
echo "  ✅ Docker containers"
echo "  ✅ GitHub Actions automation"
echo "  ✅ Universal installation script"
echo
echo "All files are in the deploy/ directory."
echo
echo -e "${YELLOW}🚀 Next Steps:${NC}"
echo "1. Review generated files in deploy/"
echo "2. Test installation scripts"
echo "3. Create GitHub release"
echo "4. Submit to package managers"
echo

echo -e "${GREEN}🎉 Deployment assets generated successfully!${NC}"
