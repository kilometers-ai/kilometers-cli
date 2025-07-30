#!/bin/bash

# Kilometers CLI Build Script with Plugin System
# This script builds the enhanced CLI with full plugin support

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
APP_NAME="km"
VERSION=${VERSION:-"dev"}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}
BUILD_DATE=$(date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_DIR="build"
DIST_DIR="dist"

# Build targets
PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
)

echo -e "${BLUE}🚀 Building Kilometers CLI with Plugin System${NC}"
echo -e "${BLUE}=============================================${NC}"
echo "Version: $VERSION"
echo "Commit: $COMMIT"
echo "Date: $BUILD_DATE"
echo

# Clean previous builds
echo -e "${YELLOW}🧹 Cleaning previous builds...${NC}"
rm -rf $BUILD_DIR $DIST_DIR
mkdir -p $BUILD_DIR $DIST_DIR

# Run tests first
echo -e "${YELLOW}🧪 Running tests...${NC}"
go test -v ./internal/core/domain/...
go test -v ./internal/infrastructure/plugins/...
echo -e "${GREEN}✅ Tests passed${NC}"
echo

# Download dependencies
echo -e "${YELLOW}📦 Downloading dependencies...${NC}"
go mod download
go mod tidy
echo -e "${GREEN}✅ Dependencies ready${NC}"
echo

# Generate build info
echo -e "${YELLOW}📝 Generating build info...${NC}"
cat > cmd/version.go << EOF
package main

// Build information injected at compile time
var (
    version = "$VERSION"
    commit  = "$COMMIT"
    date    = "$BUILD_DATE"
)
EOF

# Build for each platform
echo -e "${YELLOW}🔨 Building binaries...${NC}"
for platform in "${PLATFORMS[@]}"; do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    
    output_name="$APP_NAME"
    if [ $GOOS = "windows" ]; then
        output_name="$APP_NAME.exe"
    fi
    
    output_path="$BUILD_DIR/${APP_NAME}_${GOOS}_${GOARCH}"
    if [ $GOOS = "windows" ]; then
        output_path="$output_path.exe"
    fi
    
    echo -e "  Building for ${GOOS}/${GOARCH}..."
    
    env GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="-X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$BUILD_DATE -s -w" \
        -o "$output_path" \
        ./cmd/
    
    if [ $? -eq 0 ]; then
        echo -e "    ${GREEN}✅ Built: $output_path${NC}"
    else
        echo -e "    ${RED}❌ Failed to build for ${GOOS}/${GOARCH}${NC}"
        exit 1
    fi
done

echo -e "${GREEN}✅ All binaries built successfully${NC}"
echo

# Create distribution packages
echo -e "${YELLOW}📦 Creating distribution packages...${NC}"

# Create README for distribution
cat > $DIST_DIR/README.md << 'EOF'
# Kilometers CLI

## Installation

1. Download the appropriate binary for your platform
2. Make it executable: `chmod +x km` (Linux/macOS)
3. Move to PATH: `mv km /usr/local/bin/` (Linux/macOS)

## Quick Start

```bash
# Check version and subscription status
km version
km auth status

# Login with license key (Pro/Enterprise)
km auth login --license-key "your-license-key"

# List available plugins
km plugins list

# Start monitoring
km monitor --server -- npx -y @modelcontextprotocol/server-github
```

## Plugin System

The CLI includes a tiered plugin system:

- **Free Tier**: Basic MCP monitoring
- **Pro Tier**: Advanced filters, poison detection, ML analytics
- **Enterprise Tier**: All features + compliance reporting, team collaboration

For more information, visit: https://kilometers.ai/docs
EOF

# Package each binary
for platform in "${PLATFORMS[@]}"; do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    
    # Determine binary name
    binary_name="$APP_NAME"
    if [ $GOOS = "windows" ]; then
        binary_name="$APP_NAME.exe"
    fi
    
    source_binary="$BUILD_DIR/${APP_NAME}_${GOOS}_${GOARCH}"
    if [ $GOOS = "windows" ]; then
        source_binary="$source_binary.exe"
    fi
    
    # Create package directory
    package_name="${APP_NAME}_${VERSION}_${GOOS}_${GOARCH}"
    package_dir="$DIST_DIR/$package_name"
    mkdir -p "$package_dir"
    
    # Copy binary and README
    cp "$source_binary" "$package_dir/$binary_name"
    cp "$DIST_DIR/README.md" "$package_dir/"
    
    # Create package based on platform
    cd "$DIST_DIR"
    if [ $GOOS = "windows" ]; then
        # ZIP for Windows
        zip -r "${package_name}.zip" "$package_name/"
        echo -e "  ${GREEN}✅ Created: ${package_name}.zip${NC}"
    else
        # TAR.GZ for Unix-like systems
        tar -czf "${package_name}.tar.gz" "$package_name/"
        echo -e "  ${GREEN}✅ Created: ${package_name}.tar.gz${NC}"
    fi
    cd ..
    
    # Remove package directory (keep only archive)
    rm -rf "$package_dir"
done

echo -e "${GREEN}✅ Distribution packages created${NC}"
echo

# Generate checksums
echo -e "${YELLOW}🔐 Generating checksums...${NC}"
cd "$DIST_DIR"
sha256sum *.{zip,tar.gz} 2>/dev/null > checksums.txt || shasum -a 256 *.{zip,tar.gz} > checksums.txt
echo -e "${GREEN}✅ Checksums generated${NC}"
cd ..

# Build summary
echo -e "${BLUE}📊 Build Summary${NC}"
echo -e "${BLUE}================${NC}"
echo "Version: $VERSION"
echo "Commit: $COMMIT"
echo "Date: $BUILD_DATE"
echo "Platforms built:"
for platform in "${PLATFORMS[@]}"; do
    echo "  - $platform"
done
echo
echo "Artifacts:"
ls -la $DIST_DIR/
echo

# Plugin system features summary
echo -e "${BLUE}🔌 Plugin System Features${NC}"
echo -e "${BLUE}=========================${NC}"
echo "✅ Authentication & licensing system"
echo "✅ Tiered feature access (Free/Pro/Enterprise)"
echo "✅ Local validation (zero latency)"
echo "✅ Cryptographic license verification"
echo "✅ Plugin management CLI commands"
echo "✅ Configuration persistence"
echo "✅ Enhanced monitoring with plugin integration"
echo "✅ Security analysis and threat detection"
echo "✅ ML-powered analytics"
echo "✅ Compliance reporting (Enterprise)"
echo

# Next steps
echo -e "${YELLOW}🚀 Next Steps${NC}"
echo -e "${YELLOW}=============${NC}"
echo "1. Test the built binaries:"
echo "   ./build/km_$(go env GOOS)_$(go env GOARCH) version"
echo
echo "2. Test plugin system:"
echo "   ./scripts/demo-plugin-system.sh"
echo
echo "3. Upload to release:"
echo "   - Upload files from $DIST_DIR/ to GitHub releases"
echo "   - Update documentation with new version"
echo
echo "4. Deploy to package managers:"
echo "   - Homebrew (macOS)"
echo "   - Scoop (Windows)" 
echo "   - APT/YUM repositories (Linux)"
echo

echo -e "${GREEN}🎉 Build completed successfully!${NC}"
