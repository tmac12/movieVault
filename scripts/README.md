# Scripts Directory

This directory contains helper scripts for managing MovieVault.

## Available Scripts

### docker-login-ghcr.sh

Authenticates with GitHub Container Registry (ghcr.io).

**Usage:**
```bash
./scripts/docker-login-ghcr.sh
```

**What it does:**
- Prompts for GitHub username
- Prompts for GitHub Personal Access Token (PAT)
- Authenticates with ghcr.io
- Validates successful login
- Provides guidance for creating PAT

**Requirements:**
- GitHub account
- Personal Access Token with `write:packages` scope
- Docker installed

**When to use:**
- Before publishing images for the first time
- When authentication expires
- After creating a new PAT

---

### publish-from-linux.sh

Helper script for publishing from a Linux PC with Docker Buildx.

**Usage:**
```bash
# Publish version
./scripts/publish-from-linux.sh v1.2.0

# Dry run (preview without publishing)
./scripts/publish-from-linux.sh v1.2.0 --dry-run
```

**What it does:**
- Pulls latest changes from git (`git pull`)
- Validates git pull was successful
- Runs the `docker-publish.sh` script with your arguments

**When to use:**
- When publishing from a Linux PC (multi-machine workflow)
- Ensures you always publish the latest code
- Simplifies the workflow to a single command

**Requirements:**
- Linux PC with Docker Buildx
- Repository already cloned
- Authenticated with ghcr.io (run `docker-login-ghcr.sh` first)
- Git configured with access to GitHub remote

**Examples:**
```bash
# Quick publish after Mac development work
./scripts/publish-from-linux.sh v1.3.0

# Test first with dry-run
./scripts/publish-from-linux.sh v1.3.0 --dry-run
```

---

### docker-publish.sh

Publishes MovieVault Docker images to GitHub Container Registry with multi-platform support.

**Usage:**
```bash
# Publish version
./scripts/docker-publish.sh v1.2.0

# Dry run (preview without publishing)
./scripts/docker-publish.sh v1.2.0 --dry-run
```

**What it does:**
- Validates version format (vX.Y.Z)
- Detects GitHub username from git remote
- Creates multi-platform builds (amd64 + arm64)
- Tags with multiple versions:
  - Exact version: `v1.2.0`
  - Major.Minor: `v1.2`
  - Major: `v1`
  - Latest: `latest`
- Pushes all tags to ghcr.io
- Provides verification steps

**Requirements:**
- Authenticated with ghcr.io (run `docker-login-ghcr.sh` first)
- Docker Buildx installed
- Git repository with GitHub remote

**Parameters:**
- `<version>`: Version to publish (e.g., `v1.2.0` or `1.2.0`)
- `--dry-run`: Preview operations without building/pushing

**Examples:**
```bash
# First time publishing
./scripts/docker-login-ghcr.sh
./scripts/docker-publish.sh v1.0.0

# Publishing a new version
./scripts/docker-publish.sh v1.2.0

# Test before publishing
./scripts/docker-publish.sh v1.3.0 --dry-run
```

---

## Workflow Examples

### Single-Machine Workflow

Complete workflow when publishing from the same machine where you develop:

```bash
# 1. Update VERSION file
echo "1.3.0" > VERSION

# 2. Update CHANGELOG.md with changes
nano CHANGELOG.md

# 3. Commit changes
git add VERSION CHANGELOG.md
git commit -m "Release v1.3.0"
git push

# 4. Authenticate (if not already)
./scripts/docker-login-ghcr.sh

# 5. Test publish with dry-run
./scripts/docker-publish.sh v1.3.0 --dry-run

# 6. Publish for real
./scripts/docker-publish.sh v1.3.0

# 7. Verify on GitHub
# Visit: https://github.com/YOUR_USERNAME?tab=packages

# 8. Test pull
docker pull ghcr.io/YOUR_USERNAME/movievault:v1.3.0
```

### Multi-Machine Workflow (Mac → Linux PC)

**Recommended when:** Your development machine (Mac) lacks Docker Buildx, but you have a Linux PC with Buildx available.

**On Mac (Development):**
```bash
# 1. Make your changes, test locally

# 2. Update VERSION file
echo "1.3.0" > VERSION

# 3. Update CHANGELOG.md
nano CHANGELOG.md

# 4. Commit and push to GitHub
git add VERSION CHANGELOG.md
git commit -m "Release v1.3.0"
git push origin main
```

**On Linux PC (Publishing):**
```bash
# 5. Pull latest changes
cd /path/to/movievault
git pull origin main

# 6. Authenticate (first time only)
./scripts/docker-login-ghcr.sh

# 7. Test publish with dry-run
./scripts/docker-publish.sh v1.3.0 --dry-run

# 8. Publish for real
./scripts/docker-publish.sh v1.3.0

# 9. Verify multi-architecture support
docker manifest inspect ghcr.io/YOUR_USERNAME/movievault:v1.3.0 | grep architecture
# Should show: "architecture": "amd64" and "architecture": "arm64"
```

**From Anywhere (Verification):**
```bash
# 10. Pull and test the published image
docker pull ghcr.io/YOUR_USERNAME/movievault:v1.3.0
docker run --rm ghcr.io/YOUR_USERNAME/movievault:v1.3.0 scanner --version
```

### Linux PC One-Time Setup

If using the multi-machine workflow, set up your Linux PC once:

```bash
# Clone the repository
git clone https://github.com/YOUR_USERNAME/movievault.git
cd movievault

# Verify Buildx is available
docker buildx version

# Authenticate with GitHub Container Registry
./scripts/docker-login-ghcr.sh
# Enter your GitHub username and Personal Access Token (PAT)

# Optional: Create helper script for easier publishing
cat > publish-from-linux.sh << 'EOF'
#!/bin/bash
# Helper script to pull latest changes and publish
echo "Pulling latest changes from git..."
git pull

echo ""
echo "Publishing Docker images..."
./scripts/docker-publish.sh "$@"
EOF

chmod +x publish-from-linux.sh
```

Then future publishes are just:
```bash
cd /path/to/movievault
./publish-from-linux.sh v1.3.0
```

---

## Troubleshooting

### Authentication Failed

**Problem:** `docker login` fails

**Solution:**
```bash
# Re-authenticate
./scripts/docker-login-ghcr.sh

# Or manually
docker login ghcr.io -u YOUR_USERNAME
```

### Buildx Not Available

**Problem:** `docker: 'buildx' is not a docker command`

**Solutions:**
- Update Docker to latest version (20.10+)
- Install Docker Desktop (includes buildx)
- On Linux, install buildx plugin manually

### Build Failed

**Problem:** Build errors during publish

**Solutions:**
```bash
# Test build locally first
docker build -t test-movievault .

# Check Dockerfile syntax
docker build --dry-run .

# View detailed build output
docker build --progress=plain .
```

### Wrong Username Detected

**Problem:** Script detects wrong GitHub username

**Solution:**
```bash
# Check git remote
git remote get-url origin

# Manually set username (in publish script)
GITHUB_USER="correct-username" ./scripts/docker-publish.sh v1.2.0
```

---

## Additional Resources

- **DOCKER_REGISTRY.md** - Complete guide to Docker publishing
- **.github/RELEASE_CHECKLIST.md** - Release process checklist
- **README.md** - Main documentation

---

**Last Updated:** January 30, 2026
