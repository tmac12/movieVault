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

# Link generated content to Astro directories
echo "Linking generated content to Astro website..."
mkdir -p /app/website/src/content/movies
mkdir -p /app/website/public/covers

# Copy MDX files from /data/movies to Astro content directory
if [ -d "/data/movies" ]; then
  cp -rf /data/movies/* /app/website/src/content/movies/ 2>/dev/null || echo "No MDX files to copy yet"
fi

# Copy cover images from /data/covers to Astro public directory
if [ -d "/data/covers" ]; then
  cp -rf /data/covers/* /app/website/public/covers/ 2>/dev/null || echo "No cover images to copy yet"
fi

echo "Content synced: $(ls /app/website/src/content/movies/*.mdx 2>/dev/null | wc -l) movies found"

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
