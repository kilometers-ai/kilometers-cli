#!/bin/bash

# Script to run the plugin install integration test with mock server
set -e

echo "🚀 Running Plugin Install Integration Test"
echo "=========================================="

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to clean up background processes
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    if [[ -n $MOCK_API_PID ]]; then
        echo "Stopping mock API server (PID: $MOCK_API_PID)"
        kill $MOCK_API_PID 2>/dev/null || true
        wait $MOCK_API_PID 2>/dev/null || true
    fi
}

# Set up trap for cleanup
trap cleanup EXIT INT TERM

# Start the mock API server
echo -e "${BLUE}Starting mock API server...${NC}"
cd test/mock-api
go run main.go &
MOCK_API_PID=$!
cd ../..

# Wait for the server to start
echo "Waiting for mock API server to be ready..."
for i in {1..10}; do
    if curl -s http://localhost:5194/health >/dev/null 2>&1; then
        echo -e "${GREEN}Mock API server is ready!${NC}"
        break
    fi
    
    if [ $i -eq 10 ]; then
        echo -e "${RED}Mock API server failed to start${NC}"
        exit 1
    fi
    
    sleep 1
done

# Run the integration tests
echo -e "\n${BLUE}Running plugin install integration tests...${NC}"
export TEST_WITH_MOCK_SERVER=true
go test -v ./test/integration/plugin_install_test.go

echo -e "\n${GREEN}✅ All tests completed successfully!${NC}"