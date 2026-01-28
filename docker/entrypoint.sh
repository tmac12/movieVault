#!/bin/sh
set -e

echo "========================================="
echo "Starting MovieVault Container"
echo "========================================="

# Run initial scan if AUTO_SCAN is enabled
if [ "$AUTO_SCAN" = "true" ]; then
  echo "AUTO_SCAN enabled, running initial movie scan..."
  /usr/local/bin/scanner --config /config/config.yaml || echo "Warning: Scanner failed, continuing anyway..."
  echo "Initial scan completed."
fi

# Check if website is built
if [ ! -d "/app/website/dist" ]; then
  echo "Building Astro website..."
  cd /app/website
  npm run build || echo "Warning: Astro build failed"
  cd /
fi

# Copy built website to nginx directory
if [ -d "/app/website/dist" ]; then
  echo "Copying website to nginx directory..."
  cp -r /app/website/dist/* /usr/share/nginx/html/
fi

# Start nginx in foreground
echo "Starting web server..."
echo "========================================="
exec nginx -g 'daemon off;'
