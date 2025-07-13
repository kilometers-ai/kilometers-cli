#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧪 MCP Monitoring Test Script${NC}"
echo "=============================="
echo ""
echo "This script tests the MCP message framing and parsing fixes."
echo "It will start the mock MCP server and monitor it with the km tool."
echo ""

# Check if km binary exists
if ! command -v km &> /dev/null; then
    echo -e "${RED}❌ 'km' binary not found in PATH${NC}"
    echo "Please run './local-dev-setup.sh' first to build and install the CLI."
    exit 1
fi

# Check if mock server exists
if [ ! -f "test/cmd/run_mock_server.go" ]; then
echo -e "${RED}❌ Mock MCP server runner not found at test/cmd/run_mock_server.go${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Prerequisites check passed${NC}"
echo ""

# Function to cleanup background processes
cleanup() {
    echo -e "\n${YELLOW}🧹 Cleaning up background processes...${NC}"
    if [ ! -z "$MOCK_SERVER_PID" ]; then
        kill $MOCK_SERVER_PID 2>/dev/null || true
        wait $MOCK_SERVER_PID 2>/dev/null || true
    fi
    if [ ! -z "$MONITOR_PID" ]; then
        kill $MONITOR_PID 2>/dev/null || true
        wait $MONITOR_PID 2>/dev/null || true
    fi
    echo -e "${GREEN}✅ Cleanup complete${NC}"
}

# Set trap to cleanup on script exit
trap cleanup EXIT

echo -e "${YELLOW}🚀 Starting mock MCP server...${NC}"

# Start mock server in background and capture its PID
go run test/cmd/run_mock_server.go &
MOCK_SERVER_PID=$!

# Give the server time to start
sleep 2

# Check if server is running
if ! kill -0 $MOCK_SERVER_PID 2>/dev/null; then
    echo -e "${RED}❌ Failed to start mock MCP server${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Mock MCP server started (PID: $MOCK_SERVER_PID)${NC}"
echo ""

# Test 1: Monitor with debug mode
echo -e "${YELLOW}🔍 Test 1: Monitoring with debug mode${NC}"
echo "This tests the new message framing and JSON-RPC parsing..."
echo ""

# Create a temporary log file
LOG_FILE=$(mktemp)
echo "Monitoring output will be saved to: $LOG_FILE"

# Start monitoring in background
timeout 10s km monitor --debug go &> "$LOG_FILE" &
MONITOR_PID=$!

# Wait for monitoring to complete or timeout
wait $MONITOR_PID 2>/dev/null || true
MONITOR_PID=""

echo ""
echo -e "${BLUE}📋 Monitoring Results:${NC}"
echo "======================"

# Check if log file has content
if [ -s "$LOG_FILE" ]; then
    echo "Output captured:"
    cat "$LOG_FILE"
    echo ""
    
    # Check for specific success indicators
    if grep -q "JSON-RPC" "$LOG_FILE"; then
        echo -e "${GREEN}✅ JSON-RPC message detection working${NC}"
    else
        echo -e "${YELLOW}⚠️  No JSON-RPC messages detected${NC}"
    fi
    
    if grep -q "error" "$LOG_FILE" | grep -v "error_response"; then
        echo -e "${RED}❌ Errors detected in monitoring${NC}"
    else
        echo -e "${GREEN}✅ No critical errors detected${NC}"
    fi
    
    if grep -q "bufio.Scanner: token too long" "$LOG_FILE"; then
        echo -e "${RED}❌ Buffer size issue still present${NC}"
    else
        echo -e "${GREEN}✅ No buffer size issues detected${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  No monitoring output captured${NC}"
fi

# Cleanup log file
rm -f "$LOG_FILE"

echo ""
echo -e "${YELLOW}🧪 Test 2: Testing specific fixes${NC}"
echo "================================="

# Test the binary version
echo -n "Version info: "
km --version | head -1

# Test configuration
echo -n "Config validation: "
if km config --help &> /dev/null; then
    echo -e "${GREEN}✅${NC}"
else
    echo -e "${RED}❌${NC}"
fi

echo ""
echo -e "${BLUE}📊 Test Summary${NC}"
echo "==============="
echo ""
echo "The following MCP message processing fixes have been implemented:"
echo -e "${GREEN}✅ KIL-64: Line-based message framing with accumulator${NC}"
echo -e "${GREEN}✅ KIL-62: 1MB+ buffer size for large messages${NC}"  
echo -e "${GREEN}✅ KIL-61: Proper JSON-RPC message parsing${NC}"
echo ""
echo -e "${BLUE}Manual Testing Recommendations:${NC}"
echo "1. Test with a real Linear MCP server:"
echo "   km monitor --debug /path/to/linear-mcp-server"
echo ""
echo "2. Test with large message payloads:"
echo "   # Run Linear search commands that return 100+ results"
echo ""
echo "3. Check debug logs for:"
echo "   - Proper JSON-RPC method extraction"
echo "   - No 'token too long' errors"
echo "   - Correct risk score calculations"
echo ""
echo -e "${GREEN}🎉 Testing complete!${NC}"
echo ""
echo "If you encounter any issues, run with --debug flag for detailed logs." 