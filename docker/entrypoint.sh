#!/bin/sh
set -e

echo "========================================="
echo "Starting MovieVault Container"
echo "========================================="

# Check if scheduled scanning is enabled (new mode in v1.5.0)
if [ "$SCHEDULE_ENABLED" = "true" ]; then
  echo "SCHEDULE_ENABLED=true, starting scanner in background (runs continuously)..."
  # Scanner will run continuously with scheduled scans every $SCHEDULE_INTERVAL minutes
  # It will perform an initial scan on startup if schedule_on_startup is true (default)
  /usr/local/bin/scanner --config /config/config.yaml &
  SCANNER_PID=$!
  echo "Scanner started in background (PID: $SCANNER_PID)"
  echo "Scheduled scans will run every ${SCHEDULE_INTERVAL:-60} minutes"
elif [ "$AUTO_SCAN" = "true" ]; then
  # Legacy mode: run scan once on startup, then exit
  echo "AUTO_SCAN enabled (legacy mode), running one-time movie scan..."
  if ! /usr/local/bin/scanner --config /config/config.yaml; then
    echo "ERROR: Scanner failed. Container will continue but data may be stale."
    >&2 echo "Scanner failed at $(date)"
  else
    echo "Initial scan completed successfully."
  fi
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
# Skip build if SCHEDULE_ENABLED=true (scanner handles all builds)
if [ ! -d "/app/website/dist" ] && [ "$SCHEDULE_ENABLED" != "true" ]; then
  echo "Building Astro website (legacy mode)..."
  cd /app/website
  if ! npm run build; then
    echo "ERROR: Astro build failed. Website may not be available."
    >&2 echo "Astro build failed at $(date)"
  fi
  cd /
elif [ "$SCHEDULE_ENABLED" = "true" ]; then
  echo "Skipping entrypoint build - scanner will build on startup (schedule_on_startup: true)"
fi

# Wait for scanner's initial build to complete (when SCHEDULE_ENABLED=true)
if [ "$SCHEDULE_ENABLED" = "true" ]; then
  echo "Waiting for scanner's initial build to complete..."
  WAIT_COUNT=0
  while [ ! -d "/app/website/dist" ] && [ $WAIT_COUNT -lt 60 ]; do
    sleep 2
    WAIT_COUNT=$((WAIT_COUNT + 1))
  done

  if [ -d "/app/website/dist" ]; then
    echo "Scanner build completed, syncing to nginx..."
    cp -r /app/website/dist/* /usr/share/nginx/html/ 2>/dev/null || true
  else
    echo "WARNING: Scanner build not ready after 120s, nginx will start with placeholder content"
    # Create minimal placeholder
    mkdir -p /usr/share/nginx/html
    echo "<h1>MovieVault is scanning your library...</h1><p>Refresh in a moment.</p>" > /usr/share/nginx/html/index.html
  fi
elif [ -d "/app/website/dist" ]; then
  # Legacy mode: copy built website to nginx directory
  echo "Copying website to nginx directory..."
  cp -r /app/website/dist/* /usr/share/nginx/html/
fi

# Start nginx in foreground
echo "Starting web server..."
echo "========================================="
exec nginx -g 'daemon off;'
