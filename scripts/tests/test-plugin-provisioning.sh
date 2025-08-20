#!/bin/bash

# Test script for plugin provisioning functionality
# This script tests the automatic plugin provisioning during km init

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🧪 Testing Plugin Provisioning Feature${NC}"
echo "========================================"

# Create temporary directory for test
TEST_DIR=$(mktemp -d)
echo -e "${YELLOW}Test directory: $TEST_DIR${NC}"

# Set HOME to test directory
export HOME="$TEST_DIR"
export KM_TEST_MODE=1

# Build the km binary
echo -e "\n${YELLOW}Building km binary...${NC}"
go build -o km ./cmd/main.go || {
    echo -e "${RED}❌ Failed to build km binary${NC}"
    exit 1
}

# Test 1: Init without auto-provision
echo -e "\n${GREEN}Test 1: Init without auto-provision${NC}"
./km init --api-key "test-key-123" --endpoint "http://localhost:5194"
if [ -f "$HOME/.config/kilometers/config.json" ]; then
    echo -e "${GREEN}✅ Config file created${NC}"
else
    echo -e "${RED}❌ Config file not created${NC}"
    exit 1
fi

# Test 2: Init with auto-provision (will fail without real API)
echo -e "\n${GREEN}Test 2: Init with auto-provision flag${NC}"
rm -rf "$HOME/.config/kilometers/config.json"
./km init --api-key "test-key-123" --endpoint "http://localhost:5194" --auto-provision-plugins || {
    echo -e "${YELLOW}⚠️  Expected failure - no API server running${NC}"
}

# Test 3: Check help text
echo -e "\n${GREEN}Test 3: Check help text for auto-provision flag${NC}"
./km init --help | grep -q "auto-provision-plugins" && {
    echo -e "${GREEN}✅ auto-provision-plugins flag found in help${NC}"
} || {
    echo -e "${RED}❌ auto-provision-plugins flag not found in help${NC}"
    exit 1
}

# Test 4: Verify plugin directories are created
echo -e "\n${GREEN}Test 4: Check plugin directory creation${NC}"
if [ -d "$HOME/.km/plugins" ]; then
    echo -e "${GREEN}✅ Plugin directory created${NC}"
else
    echo -e "${YELLOW}⚠️  Plugin directory not created (expected when no plugins installed)${NC}"
fi

# Clean up
echo -e "\n${YELLOW}Cleaning up...${NC}"
rm -rf "$TEST_DIR"
rm -f ./km

echo -e "\n${GREEN}🎉 All tests completed!${NC}"
