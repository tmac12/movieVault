#!/bin/bash

# docker-publish.sh
# Publish MovieVault Docker images to GitHub Container Registry
# Usage: ./scripts/docker-publish.sh <version> [--dry-run]
# Example: ./scripts/docker-publish.sh v1.2.0

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse arguments
VERSION=""
DRY_RUN=false

for arg in "$@"; do
    case $arg in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        *)
            VERSION=$arg
            shift
            ;;
    esac
done

# Validate version argument
if [ -z "$VERSION" ]; then
    echo -e "${RED}✗ Error: Version argument required${NC}"
    echo ""
    echo "Usage: $0 <version> [--dry-run]"
    echo ""
    echo "Examples:"
    echo "  $0 v1.2.0"
    echo "  $0 v1.2.0 --dry-run"
    echo ""
    exit 1
fi

# Validate version format (vX.Y.Z or X.Y.Z)
if ! [[ "$VERSION" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${RED}✗ Error: Invalid version format${NC}"
    echo ""
    echo "Version must be in format: vX.Y.Z or X.Y.Z"
    echo "Examples: v1.2.0, 1.2.0"
    echo ""
    exit 1
fi

# Ensure version starts with 'v'
if [[ ! "$VERSION" =~ ^v ]]; then
    VERSION="v$VERSION"
fi

# Extract major and major.minor versions for tagging
MAJOR_VERSION=$(echo "$VERSION" | sed -E 's/v([0-9]+)\.[0-9]+\.[0-9]+/v\1/')
MAJOR_MINOR_VERSION=$(echo "$VERSION" | sed -E 's/v([0-9]+\.[0-9]+)\.[0-9]+/v\1/')

# Detect GitHub username from git remote
GITHUB_USER=$(git remote get-url origin 2>/dev/null | sed -E 's#.*github\.com[:/]([^/]+)/.*#\1#')

if [ -z "$GITHUB_USER" ]; then
    echo -e "${RED}✗ Error: Could not detect GitHub username from git remote${NC}"
    echo ""
    echo "Please ensure you're in a git repository with a GitHub remote configured."
    echo "Or set manually: export GITHUB_USER=your-username"
    exit 1
fi

# Image names
IMAGE_NAME="movievault"
REGISTRY="ghcr.io"
FULL_IMAGE_NAME="${REGISTRY}/${GITHUB_USER}/${IMAGE_NAME}"

echo -e "${BLUE}MovieVault Docker Publisher${NC}"
echo "============================"
echo ""
echo "Version:     ${VERSION}"
echo "Major:       ${MAJOR_VERSION}"
echo "Major.Minor: ${MAJOR_MINOR_VERSION}"
echo "Registry:    ${REGISTRY}"
echo "Repository:  ${GITHUB_USER}/${IMAGE_NAME}"
echo "Dry Run:     ${DRY_RUN}"
echo ""

# Check if buildx is available
if ! docker buildx version &>/dev/null; then
    echo -e "${RED}✗ Error: docker buildx not available${NC}"
    echo ""
    echo "Docker Buildx is required for multi-platform builds."
    echo "Please ensure you have Docker Desktop or install buildx."
    exit 1
fi

# Check authentication
echo -e "${YELLOW}Checking authentication...${NC}"
if ! docker info 2>/dev/null | grep -q "${REGISTRY}"; then
    echo -e "${RED}✗ Not authenticated with ${REGISTRY}${NC}"
    echo ""
    echo "Please run: ./scripts/docker-login-ghcr.sh"
    exit 1
fi
echo -e "${GREEN}✓ Authenticated${NC}"
echo ""

# Create/use buildx builder
echo -e "${YELLOW}Setting up buildx builder...${NC}"
BUILDER_NAME="movievault-builder"

if ! docker buildx inspect "$BUILDER_NAME" &>/dev/null; then
    if [ "$DRY_RUN" = true ]; then
        echo "[DRY RUN] Would create builder: $BUILDER_NAME"
    else
        docker buildx create --name "$BUILDER_NAME" --use
        echo -e "${GREEN}✓ Created builder: $BUILDER_NAME${NC}"
    fi
else
    if [ "$DRY_RUN" = true ]; then
        echo "[DRY RUN] Would use existing builder: $BUILDER_NAME"
    else
        docker buildx use "$BUILDER_NAME"
        echo -e "${GREEN}✓ Using existing builder: $BUILDER_NAME${NC}"
    fi
fi
echo ""

# Build and push
echo -e "${YELLOW}Building multi-platform image...${NC}"
echo "Platforms: linux/amd64, linux/arm64"
echo ""
echo "Tags:"
echo "  - ${FULL_IMAGE_NAME}:${VERSION}"
echo "  - ${FULL_IMAGE_NAME}:${MAJOR_MINOR_VERSION}"
echo "  - ${FULL_IMAGE_NAME}:${MAJOR_VERSION}"
echo "  - ${FULL_IMAGE_NAME}:latest"
echo ""

if [ "$DRY_RUN" = true ]; then
    echo -e "${BLUE}[DRY RUN] Would execute:${NC}"
    echo "docker buildx build \\"
    echo "  --platform linux/amd64,linux/arm64 \\"
    echo "  -t ${FULL_IMAGE_NAME}:${VERSION} \\"
    echo "  -t ${FULL_IMAGE_NAME}:${MAJOR_MINOR_VERSION} \\"
    echo "  -t ${FULL_IMAGE_NAME}:${MAJOR_VERSION} \\"
    echo "  -t ${FULL_IMAGE_NAME}:latest \\"
    echo "  --push \\"
    echo "  ."
    echo ""
    echo -e "${GREEN}✓ Dry run complete. No images were built or pushed.${NC}"
    exit 0
fi

# Actual build and push
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    -t "${FULL_IMAGE_NAME}:${VERSION}" \
    -t "${FULL_IMAGE_NAME}:${MAJOR_MINOR_VERSION}" \
    -t "${FULL_IMAGE_NAME}:${MAJOR_VERSION}" \
    -t "${FULL_IMAGE_NAME}:latest" \
    --push \
    .

echo ""
echo -e "${GREEN}✓ Successfully published images!${NC}"
echo ""
echo "Published tags:"
echo "  ${FULL_IMAGE_NAME}:${VERSION}"
echo "  ${FULL_IMAGE_NAME}:${MAJOR_MINOR_VERSION}"
echo "  ${FULL_IMAGE_NAME}:${MAJOR_VERSION}"
echo "  ${FULL_IMAGE_NAME}:latest"
echo ""
echo "Pull with:"
echo "  docker pull ${FULL_IMAGE_NAME}:${VERSION}"
echo "  docker pull ${FULL_IMAGE_NAME}:latest"
echo ""
echo "View on GitHub:"
echo "  https://github.com/${GITHUB_USER}?tab=packages"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "  1. Verify image on GitHub Packages"
echo "  2. Test pull: docker pull ${FULL_IMAGE_NAME}:${VERSION}"
echo "  3. Update documentation if needed"
echo "  4. Create GitHub Release at: https://github.com/${GITHUB_USER}/${IMAGE_NAME}/releases/new"
echo ""
