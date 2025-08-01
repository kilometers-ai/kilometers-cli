#!/bin/bash

# Build script for Kilometers CLI
# This script demonstrates building free vs premium versions

set -e

echo "🔨 Kilometers CLI Build Script"
echo "=============================="

# Determine build type from argument
BUILD_TYPE=${1:-free}

# Set up paths
CLI_DIR="/projects/kilometers.ai/kilometers-cli"
PLUGINS_DIR="/projects/kilometers.ai/kilometers-cli-plugins"
OUTPUT_DIR="${CLI_DIR}/build"

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Change to CLI directory
cd "${CLI_DIR}"

if [ "$BUILD_TYPE" == "premium" ]; then
    echo "🏆 Building PREMIUM version with all plugins..."
    
    # Check if we have access to private repo
    if [ ! -d "${PLUGINS_DIR}" ]; then
        echo "❌ Error: Private plugins repository not found at ${PLUGINS_DIR}"
        echo "   Premium build requires access to kilometers-cli-plugins repository"
        exit 1
    fi
    
    # For GitHub Actions or CI/CD, you would configure private repo access:
    # export GOPRIVATE=github.com/kilometers-ai/kilometers-cli-plugins
    # git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"
    
    # Build with premium tag
    echo "📦 Compiling with premium plugins..."
    go build -tags premium -o "${OUTPUT_DIR}/km" cmd/main.go
    
    if [ $? -eq 0 ]; then
        echo "✅ Premium build successful: ${OUTPUT_DIR}/km"
        echo "   Includes: All free features + API logging, advanced filters, etc."
    else
        echo "❌ Premium build failed"
        exit 1
    fi
    
else
    echo "🆓 Building FREE version with basic features only..."
    
    # Build without premium tag
    echo "📦 Compiling free version..."
    go build -o "${OUTPUT_DIR}/km-free" cmd/main.go
    
    if [ $? -eq 0 ]; then
        echo "✅ Free build successful: ${OUTPUT_DIR}/km-free"
        echo "   Includes: Basic monitoring, console logging"
    else
        echo "❌ Free build failed"
        exit 1
    fi
fi

echo ""
echo "📋 Build Summary:"
echo "=================="
ls -la "${OUTPUT_DIR}"

echo ""
echo "🚀 To test the build:"
echo "   ${OUTPUT_DIR}/km monitor --server -- npx -y @modelcontextprotocol/server-github"
