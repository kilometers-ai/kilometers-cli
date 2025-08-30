#!/bin/bash

# Setup script for Git hooks
# Run this script to configure pre-commit hooks for the project

set -e

echo "🔧 Setting up Git hooks..."

# Create scripts directory if it doesn't exist
mkdir -p scripts

# Configure Git to use custom hooks directory
git config core.hooksPath .githooks

echo "✅ Git hooks configured!"
echo ""
echo "The following hooks are now active:"
echo "  - pre-commit: Automatic Go formatting and static analysis"
echo ""
echo "To test the pre-commit hook manually:"
echo "  ./.githooks/pre-commit"
echo ""
echo "To disable hooks temporarily:"
echo "  git commit --no-verify"