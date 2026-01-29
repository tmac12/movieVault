# Configuration Guide

**Last Updated:** January 29, 2026

This document explains MovieVault's configuration system, which uses different config files for local development vs Docker deployment.

---

## Configuration Files Overview

MovieVault uses a **dual-config pattern** for flexibility and security:

| File | Purpose | API Key Storage | Use Case |
|------|---------|----------------|----------|
| `config/config.yaml` | Local development | Hardcoded in file | Running `./scanner` directly |
| `config/config.docker.yaml` | Docker deployment | Environment variable | Production, Docker containers |
| `.env` | Environment secrets | Real API keys | Docker Compose reads this |
| `config/config.example.yaml` | Template | Placeholder | Copy to create config.yaml |
| `.env.example` | Template | Placeholder | Copy to create .env |

---

## Setup Instructions

### For Local Development (Native Go)

**Step 1:** Create your config file
```bash
cp config/config.example.yaml config/config.yaml
```

**Step 2:** Edit `config/config.yaml` and add your TMDB API key
```yaml
tmdb:
  api_key: "your_actual_api_key_here"  # ← Put your real key here
```

**Step 3:** Update movie directories
```yaml
scanner:
  directories:
    - "/Users/you/Movies"  # ← Your actual movie paths
```

**Step 4:** Run the scanner
```bash
./scanner
```

**Security:** `config/config.yaml` is gitignored and will NOT be committed.

---

### For Docker Deployment

**Step 1:** Create your .env file
```bash
cp .env.example .env
```

**Step 2:** Edit `.env` and add your TMDB API key
```bash
TMDB_API_KEY=your_actual_api_key_here  # ← Put your real key here
AUTO_SCAN=true
SCAN_INTERVAL=3600
WEB_PORT=8080
```

**Step 3:** Update `docker-compose.yml` with your movie paths
```yaml
volumes:
  - /path/to/your/movies:/movies:ro  # ← Your actual movie directory
```

**Step 4:** Start Docker
```bash
docker-compose up -d
```

**How it works:**
1. Docker Compose reads `.env` file (line: `TMDB_API_KEY=${TMDB_API_KEY}`)
2. Passes it as environment variable to container
3. Container mounts `config.docker.yaml` as `/config/config.yaml`
4. Go app reads `${TMDB_API_KEY}` placeholder and replaces with env var value

**Security:** `.env` is gitignored and will NOT be committed.

---

## Why Two Config Files?

### The Problem

Git repositories shouldn't contain secrets (API keys, passwords). But config files need those secrets to work. How do you share code without sharing secrets?

### The Solution: Dual-Config Pattern

**Local Development (`config.yaml`):**
- ✅ Simple: API key directly in file
- ✅ Fast: No environment setup needed
- ✅ Flexible: Easy to test different settings
- ❌ Risk: Could accidentally commit (protected by .gitignore)

**Docker Deployment (`config.docker.yaml`):**
- ✅ Secure: No secrets in config files
- ✅ Shareable: Can commit to git safely
- ✅ Production-ready: Follows 12-factor app principles
- ✅ Flexible: Different keys per environment (dev/staging/prod)
- ❌ Setup: Requires .env file creation

---

## Configuration Reference

### TMDB Settings

```yaml
tmdb:
  api_key: "YOUR_KEY"        # Required: TMDB API key
  language: "en-US"          # Optional: Metadata language
```

**Supported languages:** `en-US`, `it-IT`, `es-ES`, `fr-FR`, `de-DE`, `ja-JP`, etc.

Get API key: https://www.themoviedb.org/settings/api

---

### Scanner Settings

```yaml
scanner:
  directories:               # Movie directories to scan
    - "/path/to/movies"
    - "/another/path"
  
  extensions:                # Video file types
    - ".mp4"
    - ".mkv"
    - ".avi"
    - ".mov"
    - ".m4v"
    - ".webm"
    - ".flv"
    - ".wmv"
  
  exclude_dirs:              # Skip these directories
    - "extras"
    - "bonus"
    - "featurettes"
    - "behind the scenes"
    - "deleted scenes"
    - "interviews"
    - "trailers"
    - "samples"
    - "subs"
    - "subtitles"
```

**Directory paths:**
- **Local:** Use absolute paths like `/Users/marco/Movies`
- **Docker:** Use container paths like `/movies` (mapped in docker-compose.yml)

**Exclude patterns:** Case-insensitive, partial match anywhere in path

---

### Output Settings

```yaml
output:
  mdx_dir: "./website/src/content/movies"    # MDX file output
  covers_dir: "./website/public/covers"      # Cover image output
  website_dir: "./website"                   # Astro website location
  auto_build: true                           # Run npm run build after scan
  cleanup_missing: false                     # Delete MDX for removed movies
```

**Path types:**
- Relative paths (`./website`) - relative to scanner working directory
- Absolute paths (`/data/movies`) - used in Docker

**Docker paths:**
```yaml
output:
  mdx_dir: "/data/movies"      # Mapped to ./data/movies on host
  covers_dir: "/data/covers"   # Mapped to ./data/covers on host
```

---

### Options

```yaml
options:
  rate_limit_delay: 250        # Milliseconds between TMDB requests
  download_covers: true        # Download poster images
  download_backdrops: true     # Download backdrop images
  use_nfo: true                # Parse Jellyfin .nfo files
  nfo_fallback_tmdb: true      # Use TMDB if .nfo missing/incomplete
```

**Rate limiting:**
- Default: 250ms (4 requests/second)
- Lower: Faster scans, risk of TMDB throttling
- Higher: Slower scans, safer for large libraries

**NFO support:**
- Reads Jellyfin/Kodi `.nfo` files first
- Falls back to TMDB if NFO missing or incomplete
- Merges data (NFO fields take priority)

---

## Environment Variables (.env)

**Docker-specific environment file:**

```bash
# TMDB API Key (required)
TMDB_API_KEY=your_api_key_here

# Auto-scan on container startup (optional)
AUTO_SCAN=true

# Scan interval in seconds (optional)
# 3600 = 1 hour, 86400 = 1 day
SCAN_INTERVAL=3600

# Web server port (optional)
WEB_PORT=8080
```

**How Docker Compose uses this:**
1. Reads `.env` file automatically
2. Substitutes `${VARIABLE}` placeholders in docker-compose.yml
3. Passes to container as environment variables

---

## Security Best Practices

### ✅ DO

1. **Use .gitignore properly**
   ```bash
   # These files are already gitignored:
   /config/config.yaml
   .env
   ```

2. **Use example files for sharing**
   ```bash
   # Commit these (they're safe):
   config/config.example.yaml
   config/config.docker.yaml
   .env.example
   ```

3. **Use environment variables in production**
   - Docker: Use `.env` file
   - Cloud: Use platform secrets (AWS Secrets Manager, etc.)
   - CI/CD: Use GitHub Secrets, GitLab CI variables

4. **Rotate keys if exposed**
   - Generate new TMDB API key immediately
   - Delete old key from TMDB account
   - Update local config/env files

### ❌ DON'T

1. **Never commit secrets**
   ```bash
   # BAD - Don't do this:
   git add config/config.yaml
   git add .env
   ```

2. **Never share config files directly**
   - Use example templates instead
   - Share via secure channels if absolutely necessary

3. **Never hardcode keys in Docker configs**
   ```yaml
   # BAD - Don't do this:
   tmdb:
     api_key: "hardcoded_key_here"
   ```

4. **Never disable .gitignore for config files**

---

## Troubleshooting

### "API key not found" error

**Symptom:** Scanner fails with "TMDB API key is required"

**Solution:**

For local:
```bash
# Check config exists and has key
cat config/config.yaml | grep api_key
# Should show: api_key: "your_key..."
```

For Docker:
```bash
# Check .env exists and has key
cat .env | grep TMDB_API_KEY
# Should show: TMDB_API_KEY=your_key...

# Check Docker container sees the variable
docker exec movievault env | grep TMDB_API_KEY
# Should show: TMDB_API_KEY=your_key...
```

---

### Docker uses wrong config file

**Symptom:** Docker ignores .env file, uses hardcoded key from config.yaml

**Cause:** `docker-compose.yml` mounts wrong config file

**Solution:**
```yaml
# Correct configuration:
volumes:
  - ./config/config.docker.yaml:/config/config.yaml:ro
  # NOT: ./config/config.yaml

# Restart after fixing:
docker-compose down
docker-compose up -d
```

---

### Changes to .env not taking effect

**Symptom:** Updated TMDB_API_KEY in .env but scanner uses old key

**Solution:**
```bash
# Restart Docker Compose (reloads .env)
docker-compose down
docker-compose up -d

# Verify new key loaded
docker exec movievault env | grep TMDB_API_KEY
```

---

### Config file format errors

**Symptom:** "yaml: unmarshal errors" or "cannot parse config"

**Solution:**
```bash
# Validate YAML syntax
yamllint config/config.yaml

# Check for common issues:
# - Incorrect indentation (use 2 spaces)
# - Missing quotes around strings with special chars
# - Incorrect path format (use absolute or relative)
```

---

## Migration Guide

### Upgrading from Single Config

**Old setup (before v1.2):**
- One `config.yaml` for everything
- No .env file
- API key hardcoded in config

**New setup (v1.2+):**
- Two configs: local + Docker
- `.env` for Docker secrets
- Environment variable substitution

**Migration steps:**

1. **Backup your current config**
   ```bash
   cp config/config.yaml config/config.yaml.backup
   ```

2. **Create .env file**
   ```bash
   cp .env.example .env
   # Edit .env and add your API key
   ```

3. **Update docker-compose.yml**
   ```yaml
   # Change this line:
   - ./config/config.yaml:/config/config.yaml:ro
   # To this:
   - ./config/config.docker.yaml:/config/config.yaml:ro
   ```

4. **Test Docker deployment**
   ```bash
   docker-compose down
   docker-compose up -d
   docker-compose logs -f
   # Should see: "Using metadata source: TMDB" (no errors)
   ```

5. **Keep config.yaml for local use**
   - No changes needed
   - Still works with `./scanner`

---

## FAQ

### Q: Which config file does the scanner use?

**A:** It depends how you run it:

```bash
# Local run - uses config/config.yaml
./scanner

# Local run with custom config
./scanner --config /path/to/custom.yaml

# Docker - uses config.docker.yaml (mounted as /config/config.yaml)
docker exec movievault scanner
```

### Q: Can I use environment variables in config.yaml for local runs?

**A:** Not directly. The Go app doesn't expand `${VAR}` syntax in local configs. Options:

1. Use `config.docker.yaml` pattern and set env vars:
   ```bash
   export TMDB_API_KEY="your_key"
   ./scanner --config config/config.docker.yaml
   ```

2. Use shell substitution:
   ```bash
   envsubst < config/config.docker.yaml > /tmp/config.yaml
   ./scanner --config /tmp/config.yaml
   ```

3. Hardcode in `config.yaml` (simplest for local dev)

### Q: Can I commit config.docker.yaml?

**A:** Yes! It's designed to be committed. It contains no secrets:

```yaml
tmdb:
  api_key: "${TMDB_API_KEY}"  # ✅ Placeholder, safe to commit
```

### Q: What if I accidentally committed config.yaml with my API key?

**A:** Follow the security cleanup procedure in SECURITY_CLEANUP_REPORT.md:

1. Rotate your TMDB API key immediately
2. Use BFG Repo-Cleaner to purge git history
3. Force push cleaned history
4. Verify .gitignore is working

### Q: Do I need both .env and config.docker.yaml?

**A:** Yes, for Docker:
- `.env` - Contains actual secrets (not committed)
- `config.docker.yaml` - Contains structure/placeholders (committed)

Docker Compose reads `.env` and injects values into config.docker.yaml placeholders.

---

## Example Configurations

### Minimal Local Config

```yaml
tmdb:
  api_key: "your_key_here"

scanner:
  directories:
    - "/Users/you/Movies"
  extensions:
    - ".mp4"
    - ".mkv"

output:
  mdx_dir: "./website/src/content/movies"
  covers_dir: "./website/public/covers"
  auto_build: false

options:
  rate_limit_delay: 250
  download_covers: true
  download_backdrops: true
  use_nfo: true
  nfo_fallback_tmdb: true
```

### Production Docker Config

**config.docker.yaml:**
```yaml
tmdb:
  api_key: "${TMDB_API_KEY}"
  language: "${TMDB_LANGUAGE:-en-US}"

scanner:
  directories:
    - "/movies"
  extensions:
    - ".mp4"
    - ".mkv"
    - ".avi"
  exclude_dirs:
    - "extras"
    - "bonus"

output:
  mdx_dir: "/data/movies"
  covers_dir: "/data/covers"
  auto_build: true
  cleanup_missing: false

options:
  rate_limit_delay: 500  # Conservative for production
  download_covers: true
  download_backdrops: true
  use_nfo: true
  nfo_fallback_tmdb: true
```

**.env:**
```bash
TMDB_API_KEY=your_production_key_here
TMDB_LANGUAGE=en-US
AUTO_SCAN=true
SCAN_INTERVAL=3600
WEB_PORT=8080
```

### Multi-Language Setup

```yaml
tmdb:
  api_key: "${TMDB_API_KEY}"
  language: "it-IT"  # Italian metadata

options:
  use_nfo: false  # Disable NFO (prefer TMDB for translations)
  nfo_fallback_tmdb: true
```

---

## Related Documentation

- **README.md** - Quick start guide
- **CLAUDE.md** - Project architecture and patterns
- **SECURITY_CLEANUP_REPORT.md** - API key exposure incident response
- **.env.example** - Environment variable template
- **config/config.example.yaml** - Config file template
- **docker-compose.yml** - Docker orchestration

---

## Changelog

**v1.2 (January 29, 2026)**
- Added dual-config pattern (local + Docker)
- Added .env file support
- Fixed docker-compose.yml to use config.docker.yaml
- Added this documentation

**v1.1 (January 27, 2026)**
- Added NFO support options
- Added language configuration

**v1.0 (January 2026)**
- Initial configuration system
- Single config.yaml file
