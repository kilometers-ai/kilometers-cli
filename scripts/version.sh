#!/bin/bash
# Version management script for Kilometers CLI

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Generate date-based version
generate_version() {
    # Format: vYYYY.M.D.BUILD
    # Using non-zero-padded month and day for cleaner versions
    DATE=$(date -u +%Y.%-m.%-d)
    
    # Get build number for today (count of tags with today's date + 1)
    BUILD_NUM=$(git tag -l "v${DATE}.*" 2>/dev/null | wc -l | xargs)
    BUILD_NUM=$((BUILD_NUM + 1))
    
    VERSION="v${DATE}.${BUILD_NUM}"
    echo "$VERSION"
}

# Update Cargo.toml with semantic version
update_cargo_version() {
    local version=$1
    # Remove 'v' prefix and convert to semantic version for Cargo.toml
    # v2025.9.8.1 -> 2025.9.801 (concatenate day and build)
    local cargo_version=$(echo "$version" | sed 's/^v//' | awk -F. '{printf "%s.%s.%s%02d", $1, $2, $3, $4}')
    
    echo -e "${YELLOW}Updating Cargo.toml version to ${cargo_version}...${NC}"
    
    # Update version in Cargo.toml
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        sed -i '' "s/^version = \".*\"/version = \"${cargo_version}\"/" Cargo.toml
    else
        # Linux
        sed -i "s/^version = \".*\"/version = \"${cargo_version}\"/" Cargo.toml
    fi
    
    # Update Cargo.lock
    cargo update -p km
}

# Create git tag
create_tag() {
    local version=$1
    local message=${2:-"Release $version"}
    
    echo -e "${YELLOW}Creating tag ${version}...${NC}"
    git tag -a "$version" -m "$message"
    echo -e "${GREEN}Tag $version created successfully!${NC}"
}

# Push tag to remote
push_tag() {
    local version=$1
    
    echo -e "${YELLOW}Pushing tag ${version} to remote...${NC}"
    git push origin "$version"
    echo -e "${GREEN}Tag pushed successfully!${NC}"
}

# List recent versions
list_versions() {
    echo -e "${YELLOW}Recent versions:${NC}"
    git tag -l "v*" | sort -V | tail -10
}

# Main command handling
case "${1:-}" in
    generate)
        VERSION=$(generate_version)
        echo -e "${GREEN}Generated version: ${VERSION}${NC}"
        ;;
    
    create)
        VERSION=$(generate_version)
        
        # Check if tag already exists
        if git rev-parse "$VERSION" >/dev/null 2>&1; then
            echo -e "${RED}Error: Tag $VERSION already exists${NC}"
            exit 1
        fi
        
        update_cargo_version "$VERSION"
        
        # Commit Cargo.toml changes
        git add Cargo.toml Cargo.lock
        git commit -m "chore: bump version to $VERSION"
        
        create_tag "$VERSION"
        
        echo -e "${GREEN}Version $VERSION created successfully!${NC}"
        echo -e "${YELLOW}Run '$0 push' to push the tag to remote${NC}"
        ;;
    
    push)
        # Get the latest tag
        VERSION=$(git describe --tags --abbrev=0 2>/dev/null)
        if [ -z "$VERSION" ]; then
            echo -e "${RED}Error: No tags found${NC}"
            exit 1
        fi
        
        push_tag "$VERSION"
        ;;
    
    list)
        list_versions
        ;;
    
    current)
        VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "No version tags")
        echo -e "${GREEN}Current version: ${VERSION}${NC}"
        ;;
    
    *)
        echo "Kilometers CLI Version Management"
        echo ""
        echo "Usage: $0 <command>"
        echo ""
        echo "Commands:"
        echo "  generate    Generate a new version number"
        echo "  create      Create a new version (updates Cargo.toml, commits, and tags)"
        echo "  push        Push the latest tag to remote"
        echo "  list        List recent version tags"
        echo "  current     Show current version"
        echo ""
        echo "Version format: vYYYY.M.D.BUILD"
        echo "Example: v2025.9.8.1"
        ;;
esac