#!/bin/bash
# Helper script for publishing from Linux PC with buildx
# This script pulls latest changes from git and then publishes to GHCR
#
# Usage:
#   ./publish-from-linux.sh v1.3.0           # Publish version
#   ./publish-from-linux.sh v1.3.0 --dry-run # Dry run

set -e  # Exit on error

echo "============================================"
echo "MovieVault Linux PC Publisher"
echo "============================================"
echo ""

# Check if version argument provided
if [ -z "$1" ]; then
    echo "❌ Error: Version argument required"
    echo ""
    echo "Usage:"
    echo "  $0 v1.3.0           # Publish version"
    echo "  $0 v1.3.0 --dry-run # Dry run"
    echo ""
    exit 1
fi

# Pull latest changes
echo "📥 Pulling latest changes from git..."
git pull

# Check if pull was successful
if [ $? -ne 0 ]; then
    echo ""
    echo "❌ Error: git pull failed"
    echo "   Please resolve conflicts or check your git configuration"
    exit 1
fi

echo ""
echo "✅ Repository is up to date"
echo ""

# Publish using the docker-publish script
echo "🚀 Publishing Docker images..."
echo ""
./scripts/docker-publish.sh "$@"
