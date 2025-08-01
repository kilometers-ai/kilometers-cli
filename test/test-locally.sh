#!/bin/bash

# Kilometers CLI Plugin Architecture - Local Testing Guide

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}📦 Kilometers CLI Plugin Architecture - Local Testing Guide${NC}"
echo "========================================================"
echo ""

# Function to wait for user
wait_for_user() {
    echo -e "${YELLOW}Press Enter to continue...${NC}"
    read
}

# Function to run command with description
run_command() {
    local description=$1
    local command=$2
    echo -e "${GREEN}➜${NC} $description"
    echo -e "${YELLOW}  \$ $command${NC}"
    echo ""
}

echo -e "${BLUE}## Prerequisites${NC}"
echo -e "1. Go 1.21+ installed"
echo -e "2. Access to both repositories:"
echo -e "   - /projects/kilometers.ai/kilometers-cli (public)"
echo -e "   - /projects/kilometers.ai/kilometers-cli-plugins (private)"
echo ""
wait_for_user

echo -e "${BLUE}## Step 1: Start the Mock API Server${NC}"
echo -e "The mock API simulates the Kilometers API for testing"
echo ""
run_command "Terminal 1 - Start mock API server" "cd /projects/kilometers.ai/kilometers-cli/test/mock-api && go run main.go"
echo -e "${YELLOW}Keep this terminal open. The API should be running on http://localhost:5194${NC}"
echo ""
echo -e "${GREEN}Expected output:${NC}"
cat << 'EOF'
🚀 Mock Kilometers API Server
=============================
Listening on http://localhost:5194

Test API Keys:
  - km_free_123456     (Free tier)
  - km_pro_789012      (Pro tier)
  - km_ent_345678      (Enterprise tier)
  - km_downgrade_test  (Simulates downgrade)
EOF
echo ""
wait_for_user

echo -e "${BLUE}## Step 2: Build the CLI Versions${NC}"
echo -e "We'll build both free and premium versions to test the difference"
echo ""

echo -e "${GREEN}Terminal 2 - Build the CLI:${NC}"
echo ""
run_command "Navigate to CLI directory" "cd /projects/kilometers.ai/kilometers-cli"
echo ""
run_command "Build FREE version (no premium plugins)" "go build -o build/km-free cmd/main.go"
echo ""
run_command "Build PREMIUM version (with premium plugins)" "go build -tags premium -o build/km cmd/main.go"
echo ""
run_command "Check build sizes (should be similar ~15MB)" "ls -lh build/"
echo ""
wait_for_user

echo -e "${BLUE}## Step 3: Test Scenario 1 - Free User (No API Key)${NC}"
echo ""
run_command "Clean config to start fresh" "rm -f ~/.config/kilometers/config.json ~/.config/kilometers/subscription.json"
echo ""
run_command "Run monitor with FREE build" "./build/km-free monitor --server -- echo 'test message'"
echo ""
echo -e "${GREEN}Expected:${NC} Basic monitoring works, but no logging features"
echo ""
wait_for_user

echo -e "${BLUE}## Step 4: Test Scenario 2 - Free User with API Key${NC}"
echo ""
run_command "Set free tier API key" "./build/km auth login --api-key km_free_123456"
echo ""
echo -e "${GREEN}Expected output:${NC}"
cat << 'EOF'
✅ API key configured successfully
🔄 Fetching subscription features...
📋 Subscription: Free tier
   Features: basic_monitoring, console_logging
EOF
echo ""
run_command "Check status" "./build/km auth status"
echo ""
run_command "Run monitor - console logging should be active" "./build/km monitor --server -- echo 'test with free key'"
echo ""
wait_for_user

echo -e "${BLUE}## Step 5: Test Scenario 3 - Pro User${NC}"
echo ""
run_command "Set Pro tier API key" "./build/km auth login --api-key km_pro_789012"
echo ""
echo -e "${GREEN}Expected output:${NC}"
cat << 'EOF'
✅ API key configured successfully
🔄 Fetching subscription features...
📋 Subscription: Pro tier
   Features: basic_monitoring, console_logging, api_logging, advanced_filters
EOF
echo ""
run_command "Run monitor with PREMIUM build" "./build/km monitor --server -- echo 'test with pro key'"
echo ""
echo -e "${GREEN}Expected:${NC}"
echo "✨ API Logger Plugin initialized (Pro feature)"
echo "Check Terminal 1 (API server) - you should see:"
echo "[API] POST /api/events/batch - API Key: km_pro...9012, Events: X"
echo ""
wait_for_user

echo -e "${BLUE}## Step 6: Test Scenario 4 - Subscription Downgrade${NC}"
echo ""
run_command "Set downgrade test key" "./build/km auth login --api-key km_downgrade_test"
echo ""
run_command "First run - Pro features active" "./build/km monitor --server -- echo 'first run'"
echo -e "${GREEN}Expected:${NC} API Logger active, events sent to API"
echo ""
run_command "Second run - Simulates downgrade" "./build/km monitor --server -- echo 'second run'"
echo ""
echo -e "${GREEN}Expected:${NC}"
cat << 'EOF'
⚠️  Some features are no longer available in your subscription:
  - api_logging

Upgrade your subscription to regain access.
EOF
echo "API Logger no longer sends events"
echo ""
wait_for_user

echo -e "${BLUE}## Step 7: Test Free Build Can't Access Premium Features${NC}"
echo ""
run_command "Try Pro key with FREE build" "./build/km-free auth login --api-key km_pro_789012"
echo ""
run_command "Run monitor with free build" "./build/km-free monitor --server -- echo 'pro key but free build'"
echo ""
echo -e "${GREEN}Expected:${NC} No API Logger plugin (not compiled in free build)"
echo "API server shows feature fetch but no event batches"
echo ""
wait_for_user

echo -e "${BLUE}## Step 8: Test Debug Mode${NC}"
echo ""
run_command "Enable debug mode" "export KILOMETERS_DEBUG=true"
echo ""
run_command "Run with debug enabled" "./build/km monitor --server -- echo 'debug test'"
echo ""
echo -e "${GREEN}Expected:${NC} More verbose output from plugins"
echo ""
wait_for_user

echo -e "${BLUE}## Step 9: Clean Up${NC}"
echo ""
run_command "Remove API key" "./build/km auth logout"
echo ""
run_command "Stop mock API server" "Press Ctrl+C in Terminal 1"
echo ""

echo -e "${BLUE}## Summary${NC}"
echo ""
echo "✅ Free build: Only includes basic features"
echo "✅ Premium build: Includes all plugins, activated by API"
echo "✅ Feature control: API determines active features"
echo "✅ Graceful degradation: Features disable without errors"
echo "✅ Security: Premium code not in free build"
echo ""

echo -e "${BLUE}## Troubleshooting${NC}"
echo ""
echo "1. ${YELLOW}Can't build premium version?${NC}"
echo "   - Ensure /projects/kilometers.ai/kilometers-cli-plugins exists"
echo "   - Check go.mod has correct replace directive"
echo ""
echo "2. ${YELLOW}API connection errors?${NC}"
echo "   - Ensure mock API is running on port 5194"
echo "   - Check firewall/network settings"
echo ""
echo "3. ${YELLOW}Features not activating?${NC}"
echo "   - Check ~/.config/kilometers/subscription.json"
echo "   - Try removing cache and re-authenticating"
echo ""

echo -e "${GREEN}✅ Testing guide complete!${NC}"
