#!/bin/bash

# Kilometers CLI Private Plugin System Demo
# This script demonstrates how private plugin repositories work

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔐 Kilometers CLI Private Plugin System Demo${NC}"
echo -e "${BLUE}=============================================${NC}"
echo

# 1. Configure private registry
echo -e "${YELLOW}⚙️  1. Configuring private plugin registry...${NC}"
km plugins registry config --url "https://plugins.kilometers.ai" --enabled=true --auto-update=true
km plugins registry auth --token "km_registry_token_abc123def456"
echo

# 2. Check registry status
echo -e "${YELLOW}📡 2. Checking registry connection...${NC}"
km plugins registry status
echo

# 3. Search for private plugins
echo -e "${YELLOW}🔍 3. Searching for available plugins...${NC}"
km plugins search
echo

# 4. Search for specific plugins
echo -e "${YELLOW}🔍 4. Searching for security plugins...${NC}"
km plugins search security
echo

# 5. Install a private plugin
echo -e "${YELLOW}📦 5. Installing custom analytics plugin...${NC}"
km plugins install custom-analytics --version v2.1.0
echo

# 6. Install an enterprise plugin
echo -e "${YELLOW}📦 6. Installing enterprise compliance plugin...${NC}"
km plugins install enterprise-compliance
echo

# 7. List all installed plugins
echo -e "${YELLOW}📋 7. Listing installed plugins...${NC}"
km plugins list
echo

# 8. Configure a private plugin
echo -e "${YELLOW}⚙️  8. Configuring custom analytics plugin...${NC}"
km plugins configure custom-analytics --data '{
  "ml_model": "gpt-4-analytics",
  "confidence_threshold": 0.85,
  "batch_size": 100,
  "custom_rules": [
    {
      "name": "detect_anomalies",
      "pattern": "unusual_pattern_.*",
      "action": "alert"
    }
  ]
}'
echo

# 9. Start monitoring with private plugins
echo -e "${YELLOW}🔍 9. Starting monitoring with private plugins...${NC}"
echo "   Command: km monitor --server -- npx -y @modelcontextprotocol/server-github"
echo "   (This would show private plugin integration in action)"
echo
echo "   Sample output:"
echo "   [Monitor] Subscription: enterprise"
echo "   [Monitor] Active plugins: advanced-filters, custom-analytics, enterprise-compliance"
echo "   [Monitor] Private registry: Connected to https://plugins.kilometers.ai"
echo "   [Monitor] Starting monitoring with 6 active plugins..."
echo "   [CustomAnalytics] ML model loaded: gpt-4-analytics"
echo "   [Compliance] SOC2 audit trail enabled"
echo "   [Security] Advanced threat detection active"
echo

# 10. Update plugins
echo -e "${YELLOW}🔄 10. Checking for plugin updates...${NC}"
km plugins update --dry-run
echo

# 11. Update specific plugin
echo -e "${YELLOW}🔄 11. Updating custom analytics plugin...${NC}"
km plugins update custom-analytics
echo

# 12. Export plugin configuration
echo -e "${YELLOW}📤 12. Exporting plugin configurations...${NC}"
km plugins export custom-analytics --output custom-analytics-config.json
echo "Exported configuration:"
cat custom-analytics-config.json
echo

# 13. Uninstall a plugin
echo -e "${YELLOW}🗑️  13. Uninstalling plugin...${NC}"
km plugins uninstall custom-analytics --yes
echo

# 14. Show final plugin status
echo -e "${YELLOW}📊 14. Final plugin system status...${NC}"
km plugins status
echo

echo -e "${GREEN}✅ Private Plugin System Demo Complete!${NC}"
echo
echo -e "${BLUE}🔐 Private Plugin System Features Demonstrated:${NC}"
echo "  ✅ Private registry configuration and authentication"
echo "  ✅ Plugin discovery and search in private repositories"
echo "  ✅ Installation of plugins from private registry"
echo "  ✅ Version management and updates"
echo "  ✅ Configuration management for private plugins"
echo "  ✅ Integration with monitoring pipeline"
echo "  ✅ Secure access control based on subscription tiers"
echo "  ✅ Plugin lifecycle management (install/configure/uninstall)"
echo
echo -e "${YELLOW}💡 Business Benefits:${NC}"
echo "  🔒 Secure distribution of proprietary plugins"
echo "  💰 Additional revenue streams through private plugins"
echo "  🎯 Customer-specific customizations and integrations"
echo "  🏢 Enterprise-only features and compliance tools"
echo "  🔄 Controlled updates and version management"
echo "  📊 Usage analytics and plugin adoption tracking"
