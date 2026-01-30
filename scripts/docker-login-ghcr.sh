#!/bin/bash

# docker-login-ghcr.sh
# Helper script to authenticate with GitHub Container Registry (ghcr.io)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}GitHub Container Registry Authentication${NC}"
echo "========================================"
echo ""

# Check if already logged in
if docker info 2>/dev/null | grep -q "ghcr.io"; then
    echo -e "${GREEN}✓ Already logged in to ghcr.io${NC}"
    echo ""
    read -p "Re-authenticate? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 0
    fi
fi

echo "To authenticate with GitHub Container Registry, you need:"
echo "  1. Your GitHub username"
echo "  2. A Personal Access Token (PAT) with 'write:packages' scope"
echo ""
echo -e "${YELLOW}Don't have a PAT?${NC}"
echo "Create one at: https://github.com/settings/tokens/new"
echo "  - Note: 'Docker GHCR Access'"
echo "  - Scopes: Select 'write:packages' (includes read:packages)"
echo "  - Generate token and copy it (you won't see it again!)"
echo ""

# Prompt for username
read -p "GitHub username: " GITHUB_USER

if [ -z "$GITHUB_USER" ]; then
    echo -e "${RED}✗ Username cannot be empty${NC}"
    exit 1
fi

# Prompt for token (hidden input)
echo -n "GitHub PAT: "
read -s GITHUB_TOKEN
echo ""

if [ -z "$GITHUB_TOKEN" ]; then
    echo -e "${RED}✗ Token cannot be empty${NC}"
    exit 1
fi

# Attempt login
echo ""
echo "Attempting login to ghcr.io..."

if echo "$GITHUB_TOKEN" | docker login ghcr.io -u "$GITHUB_USER" --password-stdin; then
    echo ""
    echo -e "${GREEN}✓ Successfully authenticated with ghcr.io${NC}"
    echo ""
    echo "You can now:"
    echo "  - Push images: docker push ghcr.io/$GITHUB_USER/movievault:latest"
    echo "  - Pull images: docker pull ghcr.io/$GITHUB_USER/movievault:latest"
    echo "  - Use publish script: ./scripts/docker-publish.sh v1.2.0"
    echo ""
    echo -e "${YELLOW}Note:${NC} Your credentials are stored securely by Docker."
    exit 0
else
    echo ""
    echo -e "${RED}✗ Authentication failed${NC}"
    echo ""
    echo "Common issues:"
    echo "  - Wrong username or token"
    echo "  - Token doesn't have 'write:packages' scope"
    echo "  - Token has expired"
    echo "  - Network connectivity issues"
    echo ""
    echo "Please verify your credentials and try again."
    exit 1
fi
