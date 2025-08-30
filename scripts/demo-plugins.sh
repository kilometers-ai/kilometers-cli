#!/bin/bash

# Demonstration script showing plugin behavior with different subscription tiers

echo "🎯 Kilometers CLI Plugin Architecture Demo"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "📋 Scenario 1: Free User (No API Key)"
echo "-------------------------------------"
echo -e "${YELLOW}Config:${NC} No API key set"
echo -e "${YELLOW}Expected:${NC} Only basic monitoring works"
echo ""
echo "Command: km monitor --server -- npx -y @modelcontextprotocol/server-github"
echo -e "${GREEN}Result:${NC}"
echo "  ✅ Basic monitoring: Active"
echo "  ❌ Console logging: Disabled (no feature)"
echo "  ❌ API logging: Disabled (no feature)"
echo ""

echo "📋 Scenario 2: Free User (With API Key)"
echo "---------------------------------------"
echo -e "${YELLOW}Config:${NC} API key = km_free_123456"
echo -e "${YELLOW}API Response:${NC}"
cat << EOF
{
  "tier": "free",
  "features": ["basic_monitoring", "console_logging"]
}
EOF
echo ""
echo -e "${GREEN}Result:${NC}"
echo "  ✅ Basic monitoring: Active"
echo "  ✅ Console logging: Active"
echo "  ❌ API logging: Disabled (requires Pro)"
echo ""

echo "📋 Scenario 3: Pro User"
echo "-----------------------"
echo -e "${YELLOW}Config:${NC} API key = km_pro_789012"
echo -e "${YELLOW}API Response:${NC}"
cat << EOF
{
  "tier": "pro",
  "features": [
    "basic_monitoring",
    "console_logging",
    "api_logging",
    "advanced_filters",
    "ml_analytics"
  ]
}
EOF
echo ""
echo -e "${GREEN}Result:${NC}"
echo "  ✅ Basic monitoring: Active"
echo "  ✅ Console logging: Active"
echo "  ✅ API logging: Active (Pro feature)"
echo "  ✨ API Logger Plugin initialized (Pro feature)"
echo "  📊 Sending events to Kilometers API..."
echo ""

echo "📋 Scenario 4: Subscription Downgrade (Pro → Free)"
echo "--------------------------------------------------"
echo -e "${YELLOW}Initial:${NC} Pro user with all features active"
echo -e "${YELLOW}After 5 minutes:${NC} API returns free tier"
echo -e "${YELLOW}API Response:${NC}"
cat << EOF
{
  "tier": "free",
  "features": ["basic_monitoring", "console_logging"]
}
EOF
echo ""
echo -e "${RED}Result:${NC}"
echo "  ⚠️  Some features are no longer available in your subscription:"
echo "    - api_logging"
echo "    - advanced_filters"
echo "    - ml_analytics"
echo ""
echo "  Upgrade your subscription to regain access."
echo ""
echo -e "${GREEN}Behavior:${NC}"
echo "  ✅ Basic monitoring: Still active"
echo "  ✅ Console logging: Still active"
echo "  ❌ API logging: Silently disabled"
echo "  • No crashes or errors"
echo "  • Premium plugins stop executing"
echo "  • No data sent to API"
echo ""

echo "📋 Scenario 5: API Unavailable"
echo "------------------------------"
echo -e "${YELLOW}Situation:${NC} API server is down or unreachable"
echo -e "${YELLOW}Cached subscription:${NC} Pro tier (from 2 hours ago)"
echo ""
echo -e "${GREEN}Result:${NC}"
echo "  ✅ Continues with cached features"
echo "  ✅ Pro features remain active"
echo "  ⚠️  Warning: Could not refresh features from API"
echo "  • Will retry in 5 minutes"
echo "  • Cache valid for 24 hours"
echo ""

echo "🔧 Technical Details"
echo "-------------------"
echo "• Free build size: ~15MB (no premium code)"
echo "• Premium build size: ~15MB (all plugins compiled in)"
echo "• Plugin overhead: <5ms per message"
echo "• Memory usage: <10MB for plugin system"
echo "• API calls: Max 1 per 5 minutes"
echo ""

echo "🔒 Security Features"
echo "-------------------"
echo "• Premium code only exists in premium binary"
echo "• Runtime validation on every message"
echo "• API-based feature control"
echo "• No error messages revealing features"
echo "• Graceful degradation on downgrade"
echo ""

echo "✅ Demo Complete!"
