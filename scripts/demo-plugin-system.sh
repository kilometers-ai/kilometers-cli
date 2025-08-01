#!/bin/bash

# Kilometers CLI Plugin System Demo
# This script demonstrates the tiered plugin system in action

echo "🎯 Kilometers CLI Plugin System Demo"
echo "===================================="
echo

# 1. Check current status (should be free tier)
echo "📊 1. Checking current status..."
km auth status
echo

# 2. List available features (should only show basic monitoring)
echo "🔧 2. Listing available features..."
km auth features
echo

# 3. Try to use premium features (should fail)
echo "❌ 3. Trying to use premium features (should fail)..."
km plugins list
echo

# 4. Login with Pro license
echo "🔑 4. Logging in with Pro license..."
km auth login --license-key "km_pro_demo123456789abcdef"
echo

# 5. Check status again (should show Pro tier)
echo "📊 5. Checking status after login..."
km auth status
echo

# 6. List available plugins (should show Pro tier plugins)
echo "🔌 6. Listing available plugins..."
km plugins list
echo

# 7. Configure advanced filters
echo "⚙️  7. Configuring advanced filters..."
km plugins config --plugin advanced-filters --command add-rule
echo

# 8. Start monitoring with plugins
echo "🔍 8. Starting monitoring with Pro features..."
echo "   Run: km monitor --server -- npx -y @modelcontextprotocol/server-github"
echo "   (This would show plugin integration in action)"
echo

# 9. Upgrade to Enterprise
echo "🚀 9. Upgrading to Enterprise..."
km auth login --license-key "km_enterprise_demo123456789abcdef"
echo

# 10. Show Enterprise features
echo "💼 10. Listing Enterprise features..."
km auth features
echo

# 11. Show all available plugins
echo "🔌 11. All available plugins (Enterprise)..."
km plugins list
echo

# 12. Generate compliance report
echo "📋 12. Generating compliance report..."
km plugins config --plugin compliance-reporting --command generate-report
echo

# 13. Show plugin status
echo "📈 13. Plugin system status..."
km plugins status
echo

echo "✅ Demo complete! The tiered plugin system allows:"
echo "   • Free tier: Basic monitoring only"
echo "   • Pro tier: Advanced filters, poison detection, ML analytics"
echo "   • Enterprise tier: All features + compliance, team collaboration"
echo
echo "💡 Key Benefits:"
echo "   • Zero latency (local feature validation)"
echo "   • Secure (cryptographic license validation)"
echo "   • Extensible (plugin architecture)"
echo "   • User-friendly (single binary, seamless UX)"
