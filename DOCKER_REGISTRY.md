# Docker Registry Guide

This guide explains how to publish and use MovieVault Docker images from GitHub Container Registry (ghcr.io).

## Table of Contents

1. [Overview](#overview)
2. [Multi-Machine Publishing Workflow](#multi-machine-publishing-workflow)
3. [Prerequisites](#prerequisites)
4. [Authentication](#authentication)
5. [Publishing Images](#publishing-images)
6. [Using Published Images](#using-published-images)
7. [Tagging Strategy](#tagging-strategy)
8. [Multi-Architecture Support](#multi-architecture-support)
9. [Troubleshooting](#troubleshooting)
10. [Best Practices](#best-practices)

---

## Overview

GitHub Container Registry (ghcr.io) is a free Docker image hosting service integrated with GitHub. Publishing MovieVault to GHCR allows you to:

- **Share images** across machines without rebuilding
- **Deploy faster** by pulling pre-built images
- **Support multiple architectures** (amd64, arm64)
- **Version images** with semantic tags
- **Keep images private or public** based on your needs

**Registry URL format:**
```
ghcr.io/YOUR_GITHUB_USERNAME/movievault:TAG
```

**Example:**
```
ghcr.io/marco/movievault:v1.2.0
ghcr.io/marco/movievault:latest
```

---

## Multi-Machine Publishing Workflow

### Overview

MovieVault's publishing scripts require Docker Buildx for multi-platform builds (ARM64 + AMD64). If your development machine doesn't have Buildx available (e.g., Docker 29.1.4 on Mac without buildx plugin), you can publish from another machine that does (e.g., a Linux PC).

**This is the recommended approach for publishing multi-platform images when Buildx is not available on your primary development machine.**

### When to Use This Workflow

Use this workflow if:
- Your development machine lacks Docker Buildx
- You have access to another machine with Buildx (Linux PC, Linux server, etc.)
- You want multi-platform images (ARM64 + AMD64) without installing additional tools on your Mac

### The Workflow

The workflow separates development from publishing:

1. **Mac (Development):** Write code, test locally, commit, push to git
2. **Linux PC (Publishing):** Pull changes, run publish scripts
3. **From Anywhere:** Pull and use published images

### Step-by-Step Guide

#### Initial Setup on Linux PC

**Do this once:**

1. **Clone the repository:**
   ```bash
   # Clone from GitHub
   git clone https://github.com/YOUR_USERNAME/movievault.git
   cd movievault
   ```

2. **Verify Buildx is available:**
   ```bash
   docker buildx version
   # Should show: github.com/docker/buildx vX.Y.Z
   ```

3. **Authenticate with GitHub Container Registry:**
   ```bash
   ./scripts/docker-login-ghcr.sh
   ```

   You'll need:
   - Your GitHub username
   - A Personal Access Token (PAT) with `write:packages` scope
   - See [Authentication](#authentication) section for creating a PAT

#### Publishing a New Version

**For each release:**

1. **On Mac (Development):**

   Make your changes, test locally, then push:
   ```bash
   # Update version
   echo "1.3.0" > VERSION

   # Update changelog
   nano CHANGELOG.md

   # Commit and push
   git add VERSION CHANGELOG.md
   git commit -m "Release v1.3.0"
   git push origin main
   ```

2. **On Linux PC (Publishing):**

   Pull the changes and publish:
   ```bash
   # Pull latest changes
   cd /path/to/movievault
   git pull origin main

   # Optional: Test with dry-run
   ./scripts/docker-publish.sh v1.3.0 --dry-run

   # Publish for real
   ./scripts/docker-publish.sh v1.3.0
   ```

3. **Verify (From Anywhere):**

   ```bash
   # Check on GitHub
   # Visit: https://github.com/YOUR_USERNAME?tab=packages

   # Test pull
   docker pull ghcr.io/YOUR_USERNAME/movievault:v1.3.0

   # Verify multi-architecture
   docker manifest inspect ghcr.io/YOUR_USERNAME/movievault:v1.3.0 | grep architecture
   # Should show both: "architecture": "amd64" and "architecture": "arm64"
   ```

### Quick Reference Commands

**On Mac (every time you make changes):**
```bash
git add .
git commit -m "Your changes"
git push origin main
```

**On Linux PC (every time you want to publish):**
```bash
cd /path/to/movievault
git pull
./scripts/docker-publish.sh v1.X.Y
```

### Optional: Helper Script for Linux PC

You can create a simple wrapper script on your Linux PC to streamline the workflow:

**File: `publish-from-linux.sh`** (create this on Linux PC)
```bash
#!/bin/bash
# Helper script to pull latest changes and publish
# Usage: ./publish-from-linux.sh v1.3.0

echo "Pulling latest changes from git..."
git pull

echo ""
echo "Publishing Docker images..."
./scripts/docker-publish.sh "$@"
```

Make it executable:
```bash
chmod +x publish-from-linux.sh
```

Then use it:
```bash
./publish-from-linux.sh v1.3.0
```

### Benefits of This Approach

✅ **No additional setup on Mac** - Continue developing normally
✅ **Multi-platform images** - Works on Intel, AMD, ARM64, Raspberry Pi
✅ **Scripts work as-is** - No modifications needed
✅ **Clean separation** - Development on Mac, publishing on Linux PC
✅ **One-time setup** - Configure Linux PC once, use forever

### Alternative: Install Buildx on Mac

If you prefer to publish from your Mac, you can install Docker Buildx manually:

```bash
# Create directory
mkdir -p ~/.docker/cli-plugins

# Download buildx (macOS ARM64 - for M1/M2/M3 Macs)
curl -Lo ~/.docker/cli-plugins/docker-buildx \
  https://github.com/docker/buildx/releases/latest/download/buildx-v0.12.0.darwin-arm64

# Make executable
chmod +x ~/.docker/cli-plugins/docker-buildx

# Verify
docker buildx version

# Create builder
docker buildx create --name movievault-builder --use

# Now you can publish from Mac
./scripts/docker-publish.sh v1.3.0
```

**Note:** The multi-machine workflow is simpler if you already have a Linux PC with Buildx.

---

## Prerequisites

### 1. GitHub Account
You need a GitHub account with access to this repository.

### 2. Docker Installation
Ensure Docker is installed and running:
```bash
docker --version
# Should show: Docker version 20.x or higher

docker buildx version
# Required for multi-platform builds
```

### 3. GitHub Personal Access Token (PAT)

You need a PAT with the `write:packages` scope to publish images.

**Creating a PAT:**

1. Go to: https://github.com/settings/tokens/new
2. Fill in the form:
   - **Note:** `Docker GHCR Access` (or any descriptive name)
   - **Expiration:** Choose duration (90 days, 1 year, or no expiration)
   - **Scopes:** Check `write:packages` (this includes `read:packages`)
3. Click **Generate token**
4. **Copy the token immediately** (you won't see it again!)

**Token storage:**
- Store it securely (password manager recommended)
- Never commit it to git
- Docker will store it in your credential store after login

---

## Authentication

### Quick Start: Use the Helper Script

```bash
# Run the authentication script
./scripts/docker-login-ghcr.sh
```

The script will:
1. Prompt for your GitHub username
2. Prompt for your PAT (hidden input)
3. Authenticate with ghcr.io
4. Confirm successful login

**Example:**
```
$ ./scripts/docker-login-ghcr.sh
GitHub Container Registry Authentication
========================================

GitHub username: marco
GitHub PAT: ••••••••••••••••••

Attempting login to ghcr.io...

✓ Successfully authenticated with ghcr.io
```

### Manual Authentication (Alternative)

If you prefer not to use the script:

```bash
# Read token from file (recommended for security)
cat ~/my-github-token.txt | docker login ghcr.io -u YOUR_USERNAME --password-stdin

# Or interactively
docker login ghcr.io -u YOUR_USERNAME
# Enter PAT when prompted for password
```

### Verify Authentication

```bash
docker info | grep ghcr.io
# Should show your ghcr.io registry in the list
```

---

## Publishing Images

### Quick Start: Use the Publishing Script

```bash
# Publish with version tag
./scripts/docker-publish.sh v1.2.0

# Dry run to preview (no actual build/push)
./scripts/docker-publish.sh v1.2.0 --dry-run
```

The script will:
1. Validate version format
2. Detect your GitHub username from git remote
3. Create multi-platform builds (amd64 + arm64)
4. Tag with multiple versions (see [Tagging Strategy](#tagging-strategy))
5. Push to ghcr.io
6. Provide next steps

**Example output:**
```
MovieVault Docker Publisher
============================

Version:     v1.2.0
Major:       v1
Major.Minor: v1.2
Registry:    ghcr.io
Repository:  marco/movievault
Dry Run:     false

✓ Authenticated
✓ Using existing builder: movievault-builder

Building multi-platform image...
Platforms: linux/amd64, linux/arm64

Tags:
  - ghcr.io/marco/movievault:v1.2.0
  - ghcr.io/marco/movievault:v1.2
  - ghcr.io/marco/movievault:v1
  - ghcr.io/marco/movievault:latest

[... build output ...]

✓ Successfully published images!
```

### Manual Publishing (Alternative)

If you prefer to publish manually:

```bash
# 1. Create/use buildx builder
docker buildx create --name movievault-builder --use

# 2. Build and push
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ghcr.io/YOUR_USERNAME/movievault:v1.2.0 \
  -t ghcr.io/YOUR_USERNAME/movievault:latest \
  --push \
  .
```

---

## Using Published Images

### Pull Images

```bash
# Pull latest version
docker pull ghcr.io/YOUR_USERNAME/movievault:latest

# Pull specific version
docker pull ghcr.io/YOUR_USERNAME/movievault:v1.2.0

# Pull specific major version (e.g., latest v1.x.x)
docker pull ghcr.io/YOUR_USERNAME/movievault:v1
```

### Run with Docker

```bash
docker run -d \
  --name movievault \
  -p 8080:80 \
  -v /path/to/movies:/movies:ro \
  -v ./data:/data \
  -e TMDB_API_KEY=your_key_here \
  ghcr.io/YOUR_USERNAME/movievault:latest
```

### Use in docker-compose.yml

Update your `docker-compose.yml` to use the published image:

```yaml
services:
  scanner:
    # Option 1: Build locally (current default)
    # build: .

    # Option 2: Use published image
    image: ghcr.io/YOUR_USERNAME/movievault:latest
    # OR specific version:
    # image: ghcr.io/YOUR_USERNAME/movievault:v1.2.0

    ports:
      - "8080:80"
    volumes:
      - /path/to/movies:/movies:ro
      - ./data:/data
      - ./config/config.docker.yaml:/config/config.yaml:ro
    environment:
      - TMDB_API_KEY=${TMDB_API_KEY}
      - AUTO_SCAN=true
```

Then run:
```bash
docker-compose pull  # Pull latest image
docker-compose up -d
```

### Public vs Private Images

By default, images published to GHCR are **private** (only you can access them).

**To make an image public:**
1. Go to: https://github.com/YOUR_USERNAME?tab=packages
2. Find the `movievault` package
3. Click **Package settings**
4. Scroll to **Danger Zone**
5. Click **Change visibility** → **Public**

Public images can be pulled without authentication by anyone.

---

## Tagging Strategy

MovieVault uses semantic versioning with multiple tags per release:

### Version Format: vMAJOR.MINOR.PATCH

Example: `v1.2.0`
- **MAJOR:** Breaking changes (v1 → v2)
- **MINOR:** New features, backward-compatible (v1.2 → v1.3)
- **PATCH:** Bug fixes, backward-compatible (v1.2.0 → v1.2.1)

### Tags Created Per Release

When you publish `v1.2.0`, the script creates **4 tags**:

| Tag | Description | Example | Updates When |
|-----|-------------|---------|--------------|
| `v1.2.0` | Exact version | `ghcr.io/USER/movievault:v1.2.0` | Never (immutable) |
| `v1.2` | Latest patch in v1.2.x | `ghcr.io/USER/movievault:v1.2` | v1.2.1 released |
| `v1` | Latest minor in v1.x.x | `ghcr.io/USER/movievault:v1` | v1.3.0 released |
| `latest` | Latest stable release | `ghcr.io/USER/movievault:latest` | Any release |

### Choosing the Right Tag

**For production:**
- Use **exact version** for reproducibility: `v1.2.0`
- Use **major version** for auto-updates: `v1`

**For development:**
- Use `latest` for cutting-edge features

**Examples:**
```yaml
# Pin to exact version (recommended for production)
image: ghcr.io/marco/movievault:v1.2.0

# Auto-update to latest v1.x patch/minor (flexible)
image: ghcr.io/marco/movievault:v1

# Always use latest stable (bleeding edge)
image: ghcr.io/marco/movievault:latest
```

---

## Multi-Architecture Support

MovieVault images support multiple CPU architectures:

| Architecture | Platform | Common Devices |
|--------------|----------|----------------|
| **amd64** | `linux/amd64` | Intel/AMD servers, most PCs |
| **arm64** | `linux/arm64` | Apple Silicon Macs (M1/M2/M3), Raspberry Pi 4+ |

### How It Works

Docker automatically pulls the correct architecture for your system:

```bash
# On Intel Mac or Linux PC
docker pull ghcr.io/marco/movievault:latest
# → Pulls linux/amd64 image

# On Apple Silicon Mac (M1/M2/M3)
docker pull ghcr.io/marco/movievault:latest
# → Pulls linux/arm64 image
```

### Verify Architecture

```bash
# Check manifest (shows all architectures)
docker manifest inspect ghcr.io/marco/movievault:latest | grep architecture
# Output:
#   "architecture": "amd64",
#   "architecture": "arm64",

# Check local image
docker image inspect ghcr.io/marco/movievault:latest | grep Architecture
# Output: "Architecture": "arm64"  (or "amd64")
```

### Building for Specific Architecture

If you need to build for a specific architecture only:

```bash
# Build only amd64
docker buildx build --platform linux/amd64 \
  -t ghcr.io/USER/movievault:v1.2.0-amd64 \
  --push .

# Build only arm64
docker buildx build --platform linux/arm64 \
  -t ghcr.io/USER/movievault:v1.2.0-arm64 \
  --push .
```

---

## Troubleshooting

### Authentication Failed

**Error:**
```
Error response from daemon: Get "https://ghcr.io/v2/": denied: denied
```

**Solutions:**
1. Check PAT has `write:packages` scope
2. Verify PAT hasn't expired
3. Re-authenticate: `./scripts/docker-login-ghcr.sh`
4. Try manual login: `docker login ghcr.io -u YOUR_USERNAME`

### Image Not Found

**Error:**
```
Error response from daemon: manifest for ghcr.io/USER/movievault:v1.2.0 not found
```

**Solutions:**
1. Verify image was published: Check https://github.com/USER?tab=packages
2. Check if image is private (requires authentication to pull)
3. Verify tag name is correct (case-sensitive)
4. Try `docker pull ghcr.io/USER/movievault:latest`

### Buildx Not Available

**Error:**
```
docker: 'buildx' is not a docker command.
```

**Solutions:**
1. Update Docker to latest version (20.10+)
2. Install Docker Desktop (includes buildx)
3. On Linux, install buildx plugin:
   ```bash
   mkdir -p ~/.docker/cli-plugins
   curl -Lo ~/.docker/cli-plugins/docker-buildx \
     https://github.com/docker/buildx/releases/download/v0.10.0/buildx-v0.10.0.linux-amd64
   chmod +x ~/.docker/cli-plugins/docker-buildx
   ```

### Build Failed for arm64

**Error:**
```
ERROR: failed to solve: process "/bin/sh -c ..." did not complete successfully: exit code: 1
```

**Solutions:**
1. Ensure QEMU is installed (Docker Desktop includes it)
2. On Linux, install QEMU:
   ```bash
   docker run --privileged --rm tonistiigi/binfmt --install all
   ```
3. Try building amd64 only if arm64 not needed:
   ```bash
   docker buildx build --platform linux/amd64 ...
   ```

### Rate Limit Exceeded

**Error:**
```
You have reached your pull rate limit
```

**Solutions:**
1. Authenticate with GitHub (even for public images, auth increases limits)
2. Wait 6 hours for rate limit reset
3. Use GitHub Actions with GITHUB_TOKEN (higher limits)

### Permission Denied When Pushing

**Error:**
```
denied: permission_denied: write_package
```

**Solutions:**
1. Verify PAT has `write:packages` scope (not just `read:packages`)
2. Ensure you own the repository or have push access
3. Check repository name matches: `ghcr.io/YOUR_USERNAME/movievault`

---

## Best Practices

### Version Management

**1. Update VERSION file before publishing:**
```bash
echo "1.3.0" > VERSION
git add VERSION
git commit -m "Bump version to 1.3.0"
```

**2. Use semantic versioning:**
- **Patch** (1.2.0 → 1.2.1): Bug fixes only
- **Minor** (1.2.0 → 1.3.0): New features, backward-compatible
- **Major** (1.2.0 → 2.0.0): Breaking changes

**3. Update CHANGELOG.md:**
Document what changed in each release.

### Publishing Workflow

**Recommended release process:**

1. **Make changes** and test locally
2. **Update VERSION** file
3. **Update CHANGELOG.md** with changes
4. **Commit changes:**
   ```bash
   git add VERSION CHANGELOG.md
   git commit -m "Release v1.3.0"
   git push
   ```
5. **Publish image:**

   **Option A - Single Machine (if Buildx available):**
   ```bash
   ./scripts/docker-publish.sh v1.3.0
   ```

   **Option B - Linux PC (multi-machine workflow):**
   ```bash
   # On Linux PC
   cd /path/to/movievault
   ./scripts/publish-from-linux.sh v1.3.0
   ```

6. **Verify on GitHub:**
   - Check packages: https://github.com/YOUR_USERNAME?tab=packages
   - Test pull: `docker pull ghcr.io/YOUR_USERNAME/movievault:v1.3.0`
7. **Create GitHub Release:**
   - Go to: https://github.com/YOUR_USERNAME/movievault/releases/new
   - Tag: `v1.3.0`
   - Title: `v1.3.0 - Feature Name`
   - Description: Copy from CHANGELOG.md

### Security Best Practices

**1. Never commit secrets:**
- PAT tokens (already in .gitignore)
- TMDB API keys (use .env, already in .gitignore)

**2. Use environment variables for tokens:**
```bash
# Good: Read from environment
export GITHUB_TOKEN=$(cat ~/.github-token)
echo "$GITHUB_TOKEN" | docker login ghcr.io -u marco --password-stdin

# Bad: Hardcode in scripts (DON'T DO THIS)
docker login ghcr.io -u marco -p ghp_xxxxxxxxxxxx
```

**3. Set token expiration:**
- Use 90 days or 1 year expiration (not "no expiration")
- Rotate tokens periodically

**4. Use read-only volumes in production:**
```yaml
volumes:
  - /path/to/movies:/movies:ro  # :ro = read-only
```

### Image Management

**1. Clean up old images locally:**
```bash
# Remove old local images
docker image prune -a

# Remove specific old versions
docker rmi ghcr.io/marco/movievault:v1.1.0
```

**2. Keep old versions in registry:**
- Don't delete old tags from GHCR
- Users may depend on specific versions
- Storage is free for public images

**3. Monitor image size:**
```bash
# Check image size
docker images ghcr.io/marco/movievault

# Current MovieVault size: ~150-200 MB (optimized)
```

### Automation (Future Enhancement)

For frequent releases, consider automating with GitHub Actions:

**.github/workflows/docker-publish.yml** (example, not included):
```yaml
name: Publish Docker Image

on:
  push:
    tags:
      - 'v*'

jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: docker/setup-buildx-action@v2
      - uses: docker/login-action@v2
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/build-push-action@v4
        with:
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            ghcr.io/${{ github.repository }}:latest
            ghcr.io/${{ github.repository }}:${{ github.ref_name }}
```

This auto-publishes when you push a git tag.

---

## Additional Resources

- **GitHub Container Registry Docs:** https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry
- **Docker Buildx Docs:** https://docs.docker.com/buildx/working-with-buildx/
- **Semantic Versioning:** https://semver.org/
- **MovieVault Documentation:** See README.md, CLAUDE.md

---

## Quick Reference

### Common Commands

```bash
# Authenticate
./scripts/docker-login-ghcr.sh

# Publish new version
./scripts/docker-publish.sh v1.2.0

# Publish with dry-run
./scripts/docker-publish.sh v1.2.0 --dry-run

# Pull latest image
docker pull ghcr.io/YOUR_USERNAME/movievault:latest

# Pull specific version
docker pull ghcr.io/YOUR_USERNAME/movievault:v1.2.0

# Run container
docker run -d -p 8080:80 \
  -v /movies:/movies:ro \
  -e TMDB_API_KEY=key \
  ghcr.io/YOUR_USERNAME/movievault:latest

# Check local images
docker images ghcr.io/YOUR_USERNAME/movievault

# View image manifest
docker manifest inspect ghcr.io/YOUR_USERNAME/movievault:latest
```

### URL Templates

```
Personal Access Token: https://github.com/settings/tokens/new
Your Packages:         https://github.com/YOUR_USERNAME?tab=packages
Package Settings:      https://github.com/YOUR_USERNAME/movievault/pkgs/container/movievault
Create Release:        https://github.com/YOUR_USERNAME/movievault/releases/new
```

---

**Last Updated:** February 4, 2026
**MovieVault Version:** 1.3.0
