#!/bin/sh
# Health check script for MovieVault container
# Verifies nginx is responding and MDX files exist

# Check nginx is responding
if ! curl -f http://localhost:80 >/dev/null 2>&1; then
  echo "ERROR: Nginx is not responding"
  exit 1
fi

# Check MDX files exist (at least one movie processed)
if [ -z "$(ls /app/website/src/content/movies/*.mdx 2>/dev/null)" ]; then
  echo "ERROR: No MDX files found"
  exit 1
fi

echo "Health check passed"
exit 0
