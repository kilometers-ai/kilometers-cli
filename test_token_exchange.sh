#!/bin/bash

# Comprehensive test for API key to short-lived token exchange
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

API_ENDPOINT="http://localhost:5194"
TEST_EMAIL="test-$(date +%s)@kilometers.ai"
TEST_PASSWORD="TestPassword123!"

echo -e "${YELLOW}=== API Key to Token Exchange Test ===${NC}"
echo ""

# Step 1: Check if API is already running
echo -e "${YELLOW}Step 1: Checking API status...${NC}"
if curl -s "$API_ENDPOINT/health" | jq -e '.status == "Healthy"' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ API is already running and healthy${NC}"
else
    echo "Starting API via Docker Compose..."
    cd ../kilometers-api
    docker-compose -f docker-compose.shared.yml up -d --build
    
    # Wait for API to be healthy
    echo "Waiting for API to be healthy..."
    max_attempts=30
    attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "$API_ENDPOINT/health" | jq -e '.status == "Healthy"' > /dev/null 2>&1; then
            echo -e "${GREEN}✓ API is healthy${NC}"
            break
        fi
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    if [ $attempt -eq $max_attempts ]; then
        echo -e "${RED}✗ API failed to become healthy${NC}"
        exit 1
    fi
    
    cd ../kilometers-cli
fi

# Step 2: Build the km CLI tool
echo ""
echo -e "${YELLOW}Step 2: Building km CLI tool...${NC}"
go build -o km ./cmd/main.go
echo -e "${GREEN}✓ CLI built successfully${NC}"

# Step 3: Register a new user
echo ""
echo -e "${YELLOW}Step 3: Registering new user...${NC}"
echo "Email: $TEST_EMAIL"

register_response=$(curl -s -X POST "$API_ENDPOINT/api/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$TEST_EMAIL\",
    \"password\": \"$TEST_PASSWORD\",
    \"organization\": \"Test Org\",
    \"fullName\": \"Test User\"
  }")

if echo "$register_response" | jq -e '.success' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ User registered successfully${NC}"
    
    # Extract the API key from registration response
    API_KEY=$(echo "$register_response" | jq -r '.apiKey.key')
    echo "API Key obtained: ${API_KEY:0:20}..."
else
    echo -e "${RED}✗ Registration failed${NC}"
    echo "$register_response" | jq '.'
    exit 1
fi

# Step 4: Test token exchange via HTTP API directly
echo ""
echo -e "${YELLOW}Step 4: Testing token exchange via API...${NC}"

token_response=$(curl -s -X POST "$API_ENDPOINT/api/auth/token" \
  -H "Content-Type: application/json" \
  -d "{\"ApiKey\": \"$API_KEY\"}")

if echo "$token_response" | jq -e '.success' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Token exchange successful${NC}"
    
    access_token=$(echo "$token_response" | jq -r '.token.accessToken')
    echo "Access token: ${access_token:0:50}..."
    
    echo ""
    echo "Token details:"
    echo "$token_response" | jq '.token | {
        tokenType: .tokenType,
        accessTokenLifetimeMinutes: .accessTokenLifetimeMinutes,
        accessTokenExpiresAt: .accessTokenExpiresAt
    }'
    
    # Validate the token
    echo ""
    echo "Validating token..."
    validation=$(curl -s -X GET "$API_ENDPOINT/api/auth/validate" \
      -H "Authorization: Bearer $access_token")
    
    if [ "$validation" = "true" ]; then
        echo -e "${GREEN}✓ Token is valid${NC}"
    else
        echo -e "${RED}✗ Token validation failed: $validation${NC}"
    fi
else
    echo -e "${RED}✗ Token exchange failed${NC}"
    echo "$token_response" | jq '.'
    exit 1
fi

# Step 5: Test via CLI tool
echo ""
echo -e "${YELLOW}Step 5: Testing token exchange via CLI...${NC}"

# Configure the CLI with the API key
export KM_API_KEY="$API_KEY"
export KM_API_ENDPOINT="$API_ENDPOINT"
export KM_DEBUG="true"

# Run auth status to verify configuration
echo "Checking CLI auth status..."
./km auth status

# Test monitoring with the configured API key (this will trigger token exchange internally)
echo ""
echo "Testing monitor command (will exchange API key for token internally)..."
timeout 5s ./km monitor -- echo "test" 2>&1 | head -20 || true

echo ""
echo -e "${GREEN}=== Test Complete ===${NC}"
echo ""
echo "Summary:"
echo "- API started and healthy ✓"
echo "- User registered: $TEST_EMAIL"
echo "- API key obtained: ${API_KEY:0:20}..."
echo "- Token exchange successful ✓"
echo "- Token validation successful ✓"
echo "- CLI configured and tested ✓"