# Release Checklist

Use this checklist when preparing a new MovieVault release.

## Version: _________

**Release Date:** _________

---

## Pre-Release

- [ ] All planned features/fixes are merged to main branch
- [ ] All tests pass (if available)
- [ ] Local testing complete
- [ ] Documentation is up to date

## Version Management

- [ ] Update `VERSION` file with new version number (e.g., `1.3.0`)
- [ ] Update `CHANGELOG.md` with:
  - [ ] New version header and date
  - [ ] All new features
  - [ ] All bug fixes
  - [ ] Any breaking changes
  - [ ] Migration notes (if needed)
- [ ] Update version in `CLAUDE.md` "Current Version" section (if changed)
- [ ] Update "Last Updated" date in `DOCKER_REGISTRY.md`

## Code Review

- [ ] Review `git diff main` for unintended changes
- [ ] Check for debug code, console.logs, or temporary fixes
- [ ] Verify no secrets or API keys in tracked files
- [ ] Confirm `.gitignore` is protecting sensitive files

## Local Testing

- [ ] **Build Go scanner:**
  ```bash
  go build -o scanner cmd/scanner/main.go
  ```
- [ ] **Run scanner on test library:**
  ```bash
  ./scanner --verbose --dry-run
  ```
- [ ] **Build Docker image locally:**
  ```bash
  docker build -t test-movievault .
  ```
- [ ] **Test Docker image:**
  ```bash
  docker run --rm -v /path/to/test-movies:/movies:ro \
    -e TMDB_API_KEY=$TMDB_API_KEY \
    test-movievault
  ```
- [ ] **Test Astro website:**
  ```bash
  cd website && npm run build && npm run preview
  ```
- [ ] Verify website displays correctly in browser

## Commit and Tag

- [ ] Commit version changes:
  ```bash
  git add VERSION CHANGELOG.md CLAUDE.md DOCKER_REGISTRY.md
  git commit -m "Release v1.X.Y"
  ```
- [ ] Push to GitHub:
  ```bash
  git push origin main
  ```

## Docker Publishing

**Note:** If using multi-machine workflow (Mac → Linux PC), complete the steps above on Mac, then switch to Linux PC for publishing.

### On Publishing Machine (Mac or Linux PC)

- [ ] If using Linux PC, pull latest changes:
  ```bash
  cd /path/to/movievault
  git pull origin main
  ```
- [ ] Authenticate with GitHub Container Registry (if not already):
  ```bash
  ./scripts/docker-login-ghcr.sh
  ```
- [ ] Publish image (dry-run first):
  ```bash
  ./scripts/docker-publish.sh v1.X.Y --dry-run
  ```
- [ ] Review dry-run output for correctness
- [ ] Publish image for real:
  ```bash
  ./scripts/docker-publish.sh v1.X.Y
  ```
- [ ] Wait for build to complete (~5-10 minutes for multi-platform)
- [ ] Verify no build errors in output

## Verification

- [ ] Check image appears on GitHub Packages:
  - URL: `https://github.com/YOUR_USERNAME?tab=packages`
  - Package: `movievault`
  - Tags: `v1.X.Y`, `v1.X`, `v1`, `latest`
- [ ] Verify multi-architecture support:
  ```bash
  docker manifest inspect ghcr.io/YOUR_USERNAME/movievault:v1.X.Y | grep architecture
  ```
  - Should show: `amd64` and `arm64`
- [ ] Test pull from registry:
  ```bash
  # Remove local image first
  docker rmi ghcr.io/YOUR_USERNAME/movievault:v1.X.Y

  # Pull from registry
  docker pull ghcr.io/YOUR_USERNAME/movievault:v1.X.Y
  ```
- [ ] Test pulled image works:
  ```bash
  docker run --rm ghcr.io/YOUR_USERNAME/movievault:v1.X.Y scanner --version
  ```

## GitHub Release

- [ ] Create GitHub Release:
  - URL: `https://github.com/YOUR_USERNAME/movievault/releases/new`
  - Tag: `v1.X.Y`
  - Target: `main` branch
  - Release title: `v1.X.Y - [Feature Name]`
  - Description: Copy from `CHANGELOG.md`
- [ ] Mark as latest release (checkbox)
- [ ] Publish release

## Post-Release

- [ ] Announce release (if applicable):
  - [ ] Update README.md badges (if version shown)
  - [ ] Post in discussions/community channels
  - [ ] Update any external documentation links
- [ ] Monitor for issues:
  - [ ] Check GitHub Issues for new bug reports
  - [ ] Test on different platforms if available
  - [ ] Verify downloads work for end users

## Rollback Plan (If Needed)

If critical issues are found after release:

- [ ] Document the issue clearly
- [ ] Decide: patch release (v1.X.Y+1) or rollback
- [ ] If rollback needed:
  - [ ] Don't delete Docker images (users may depend on them)
  - [ ] Update `latest` tag to previous version:
    ```bash
    docker pull ghcr.io/USER/movievault:v1.X.Y-1
    docker tag ghcr.io/USER/movievault:v1.X.Y-1 ghcr.io/USER/movievault:latest
    docker push ghcr.io/USER/movievault:latest
    ```
  - [ ] Mark GitHub release as "pre-release"
  - [ ] Create new issue documenting the problem

---

## Notes

**Typical Timeline:**
- Pre-release checks: 15-30 minutes
- Docker build and push: 5-10 minutes
- Verification: 5-10 minutes
- GitHub Release: 5 minutes
- **Total:** ~30-60 minutes

**Common Issues:**
- Buildx not available: Install Docker Desktop or buildx plugin
- Authentication failed: Re-run `./scripts/docker-login-ghcr.sh`
- Build failed: Check Dockerfile syntax and test locally first
- Rate limit: Wait or authenticate (increases GitHub rate limits)

**Version Numbering:**
- **Patch** (1.2.0 → 1.2.1): Bug fixes only
- **Minor** (1.2.0 → 1.3.0): New features, backward-compatible
- **Major** (1.2.0 → 2.0.0): Breaking changes

**Multi-Platform Builds:**
- Builds for both `linux/amd64` and `linux/arm64`
- Takes longer than single-platform builds
- Enables support for Intel PCs, AMD servers, Apple Silicon, and Raspberry Pi

---

**Last Updated:** January 30, 2026
**Template Version:** 1.0
