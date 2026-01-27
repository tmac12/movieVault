#!/bin/bash

# MovieVault Update Script
# This script runs the scanner and rebuilds the Astro website

set -e

echo "========================================="
echo "MovieVault Update Script"
echo "========================================="

# Change to script directory
cd "$(dirname "$0")/.."

# Check if scanner binary exists
if [ ! -f "./scanner" ]; then
    echo "Scanner binary not found. Building..."
    go build -o scanner cmd/scanner/main.go
fi

# Run scanner
echo ""
echo "Step 1: Scanning for movies..."
echo "-----------------------------------------"
./scanner

# Check if website directory exists
if [ ! -d "./website" ]; then
    echo "Error: website directory not found"
    exit 1
fi

# Check if node_modules exists
if [ ! -d "./website/node_modules" ]; then
    echo ""
    echo "Step 2: Installing npm dependencies..."
    echo "-----------------------------------------"
    cd website
    npm install
    cd ..
fi

# Build Astro site
echo ""
echo "Step 3: Building website..."
echo "-----------------------------------------"
cd website
npm run build

echo ""
echo "========================================="
echo "Update completed successfully!"
echo "========================================="
echo ""
echo "Static files are in: website/dist/"
echo ""
echo "To preview the site:"
echo "  cd website && npm run preview"
echo ""
echo "Or serve with your web server:"
echo "  nginx, apache, caddy, etc."
