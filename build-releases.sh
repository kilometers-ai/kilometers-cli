#!/bin/bash
set -e

# Build script for Kilometers CLI releases
VERSION="0.1.0"
BINARY_NAME="km"

echo "Building Kilometers CLI v$VERSION for multiple platforms..."

# Create releases directory
mkdir -p releases

# Build for different platforms
PLATFORMS=(
    "linux/amd64"
    "linux/arm64" 
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r -a array <<< "$platform"
    GOOS="${array[0]}"
    GOARCH="${array[1]}"
    
    output_name="$BINARY_NAME-$GOOS-$GOARCH"
    if [ "$GOOS" = "windows" ]; then
        output_name+=".exe"
    fi
    
    echo "Building for $GOOS/$GOARCH..."
    env GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags="-X main.Version=$VERSION" -o "releases/$output_name" .
    
    echo "Built: releases/$output_name"
done

echo ""
echo "Build complete! Generated binaries:"
ls -la releases/

echo ""
echo "To upload to Azure Storage:"
echo "az storage blob upload-batch --source releases --destination releases/latest --account-name STORAGE_ACCOUNT" 