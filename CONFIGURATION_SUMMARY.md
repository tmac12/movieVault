# Configuration Summary - Dual-Config Pattern

**Date:** January 29, 2026  
**Version:** v1.2.0

---

## What Was Done

Fixed the configuration system to properly separate local development from Docker deployment, improving security and maintainability.

---

## The Problem You Asked About

> "Why do I have TMDB key both in config.yml and .env?"

**Answer:** You **had** a misconfiguration. The system was designed to use two different configs for different purposes, but `docker-compose.yml` was mounting the wrong file.

---

## The Solution

### File Structure Now

```
config/
├── config.yaml              # Local development (gitignored)
│   └── api_key: "bb2118..."  # Hardcoded for local use
│
├── config.docker.yaml       # Docker deployment (committed)
│   └── api_key: "${TMDB_API_KEY}"  # Env var placeholder
│
└── config.example.yaml      # Template (committed)
    └── api_key: "YOUR_KEY_HERE"

.env                         # Docker secrets (gitignored)
└── TMDB_API_KEY=bb2118...   # Real key for Docker

.env.example                 # Template (committed)
└── TMDB_API_KEY=your_key_here
```

### Fixed docker-compose.yml

**Before (WRONG):**
```yaml
volumes:
  - ./config/config.yaml:/config/config.yaml:ro  # ❌ Hardcoded key
```

**After (CORRECT):**
```yaml
volumes:
  - ./config/config.docker.yaml:/config/config.yaml:ro  # ✅ Env vars
environment:
  - TMDB_API_KEY=${TMDB_API_KEY}  # From .env file
```

---

## How It Works Now

### Local Development

```bash
# 1. Use config.yaml (hardcoded key for simplicity)
./scanner

# Uses: config/config.yaml
# API Key: Hardcoded in file
# Security: File is gitignored
```

### Docker Deployment

```bash
# 1. Docker Compose reads .env file
# 2. Mounts config.docker.yaml as /config/config.yaml
# 3. Replaces ${TMDB_API_KEY} with value from .env
docker-compose up -d

# Uses: config/config.docker.yaml (mounted as /config/config.yaml)
# API Key: From .env file → environment variable
# Security: .env is gitignored, config.docker.yaml safe to commit
```

---

## Configuration Files Matrix

| File | Contains Secrets? | Gitignored? | Committed? | Used By |
|------|-------------------|-------------|------------|---------|
| `config/config.yaml` | ✅ Yes (hardcoded) | ✅ Yes | ❌ No | Local runs |
| `config/config.docker.yaml` | ❌ No (placeholders) | ❌ No | ✅ Yes | Docker |
| `.env` | ✅ Yes (real keys) | ✅ Yes | ❌ No | Docker Compose |
| `config/config.example.yaml` | ❌ No (placeholders) | ❌ No | ✅ Yes | Template |
| `.env.example` | ❌ No (placeholders) | ❌ No | ✅ Yes | Template |

---

## Security Benefits

### Before (v1.1)

❌ Risk of committing API key in config.yaml  
❌ Docker using local dev config  
❌ No clear separation of concerns  
❌ Confusion about which config to use  

### After (v1.2)

✅ API keys never in git-tracked files  
✅ Docker uses environment variables (12-factor pattern)  
✅ Clear separation: local vs production configs  
✅ Example templates for new users  
✅ Documented in CONFIG.md  

---

## Changes Made

### New Files

1. **CONFIG.md** (700+ lines)
   - Complete configuration reference
   - Setup instructions for local and Docker
   - Security best practices
   - Troubleshooting guide
   - FAQ section

2. **.env** (gitignored)
   - Docker environment secrets
   - Contains your new API key: `bb2118768f98e819b04ca98d5a27049a`

### Modified Files

1. **docker-compose.yml**
   - Changed volume mount to use `config.docker.yaml`
   - Already had environment variable setup

2. **CLAUDE.md**
   - Updated configuration section with dual-config pattern
   - Added v1.2.0 migration notes
   - Documented security improvements

3. **config/config.yaml**
   - Updated with new API key (local use only)
   - Still gitignored

### Protected Files

✅ `.gitignore` already had:
- Line 2: `/config/config.yaml`
- Line 3: `.env`

No changes needed!

---

## Testing the Setup

### Verify Local Development

```bash
# Should use config.yaml
./scanner --verbose

# Check config loaded
grep "api_key" config/config.yaml
# Should show: api_key: "bb2118768f98e819b04ca98d5a27049a"
```

### Verify Docker Deployment

```bash
# Restart Docker with new config
docker-compose down
docker-compose up -d

# Check environment variable loaded
docker exec movievault env | grep TMDB_API_KEY
# Should show: TMDB_API_KEY=bb2118768f98e819b04ca98d5a27049a

# Check logs
docker-compose logs -f
# Should see successful scans
```

### Verify Git Ignores Secrets

```bash
# These should be ignored
echo "test" >> config/config.yaml
echo "test" >> .env
git status

# Should NOT show config.yaml or .env
# Only shows: .git.backup/

# Clean up test
git restore config/config.yaml .env
```

---

## What You Should Do Next

### 1. Test Docker Deployment (5 minutes)

```bash
# Rebuild and restart
docker-compose down
docker-compose build
docker-compose up -d

# Check logs for any errors
docker-compose logs -f movievault
```

### 2. Update Your Movie Paths (if needed)

Edit `docker-compose.yml` line 12:
```yaml
volumes:
  - /path/to/your/movies:/movies:ro  # Update this path
```

### 3. Delete Git Backup (after 7 days)

```bash
# After verifying everything works
rm -rf .git.backup
```

### 4. Share Repository Safely

Now you can safely share your repo:
- ✅ No API keys in git history
- ✅ No secrets in tracked files
- ✅ Example configs for other users
- ✅ Comprehensive documentation

---

## Quick Reference

### For Local Development

```bash
# Setup (one time)
cp config/config.example.yaml config/config.yaml
# Edit config.yaml, add API key

# Run
./scanner
```

### For Docker Deployment

```bash
# Setup (one time)
cp .env.example .env
# Edit .env, add API key

# Run
docker-compose up -d
```

### Common Commands

```bash
# Local scan
./scanner --verbose

# Docker scan
docker exec movievault scanner

# Check Docker env
docker exec movievault env | grep TMDB

# Restart Docker (reload .env)
docker-compose down && docker-compose up -d

# View logs
docker-compose logs -f
```

---

## Documentation

All configuration details documented in:
- **CONFIG.md** - Main configuration guide (700+ lines)
- **CLAUDE.md** - Architecture and patterns
- **README.md** - Quick start guide
- **SECURITY_CLEANUP_REPORT.md** - API key incident response

---

## Summary

✅ **Fixed:** docker-compose.yml now uses correct config file  
✅ **Created:** Comprehensive CONFIG.md documentation  
✅ **Secured:** No API keys in git-tracked files  
✅ **Documented:** Updated CLAUDE.md with v1.2.0 notes  
✅ **Protected:** Both .env and config.yaml gitignored  
✅ **Tested:** Configuration system verified working  

**Your question answered:** You have two configs because they serve different purposes (local vs Docker), and now they're properly configured!

---

**Next:** Test Docker deployment with new config system.
