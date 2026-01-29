# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MovieVault is a movie library scanner that discovers video files, fetches metadata from TMDB API (or Jellyfin .nfo files), generates MDX files, and builds a static Astro website for browsing a personal movie collection.

**Tech Stack:**
- **Backend:** Go 1.25.5
- **Frontend:** Astro 5.2.0 with MDX
- **Metadata:** TMDB API + Jellyfin NFO support
- **Deployment:** Docker or native

**Current Version:** 1.1.0 (with NFO support)

## Build & Run Commands

### Go Scanner

```bash
# Build the scanner
go build -o scanner cmd/scanner/main.go

# Run with default config
./scanner

# Common flags
./scanner --verbose              # Detailed logging
./scanner --force-refresh        # Re-fetch all metadata from TMDB
./scanner --no-build            # Skip Astro build step
./scanner --dry-run             # Preview without changes
./scanner --config /path/to/config.yaml  # Custom config
```

### Astro Website

```bash
cd website

# Development server (http://localhost:4321)
npm run dev

# Production build
npm run build

# Preview production build
npm run preview
```

### Docker

```bash
# Build and start
docker-compose up -d

# View logs
docker-compose logs -f

# Manual scan
docker exec filmscanner scanner

# Rebuild
docker-compose build && docker-compose up -d
```

## Core Architecture

### Data Flow Pipeline

```
Video Files → Scanner → Metadata Fetcher → MDX Writer → Astro Site
                            ↓
                     NFO Parser (priority)
                            ↓
                     TMDB API (fallback)
```

### Package Structure

```
internal/
├── config/          # YAML config loading, validation
├── scanner/         # File discovery, title extraction, slug generation
├── metadata/        # Metadata sources
│   ├── tmdb.go      # TMDB API client
│   ├── nfo/         # Jellyfin NFO parser (v1.1.0)
│   │   ├── types.go    # XML structs
│   │   └── parser.go   # Parse, convert, merge logic
│   └── types.go     # Shared TMDB types
└── writer/          # MDX generation
    ├── models.go    # Movie struct (canonical)
    └── mdx.go       # YAML frontmatter + markdown body
```

### Key Architectural Patterns

#### 1. Metadata Priority System (NFO Support)

**Hierarchy:** NFO → TMDB → Error

The scanner implements a 3-tier metadata strategy in `cmd/scanner/main.go`:

```go
if cfg.Options.UseNFO {
    // 1. Try NFO first
    movie, err = nfoParser.GetMovieFromNFO(file.Path)

    if err != nil {
        // 2. Fall back to TMDB
        movie, err = tmdbClient.GetFullMovieData(file.Title, file.Year)
        metadataSource = "TMDB"
    } else {
        metadataSource = "NFO"

        // 3. Merge if NFO incomplete
        if movie.Title == "" || movie.ReleaseYear == 0 {
            tmdbMovie, _ := tmdbClient.GetFullMovieData(...)
            movie = mergeMovieData(movie, tmdbMovie)  // NFO fields take priority
            metadataSource = "NFO+TMDB"
        }
    }
}
```

**Critical:** NFO fields always take priority in merges. TMDB only fills gaps.

#### 2. NFO File Discovery

`internal/metadata/nfo/parser.go` searches in priority order:

1. `{filename}.nfo` - Same basename as video (e.g., `The Matrix (1999).nfo`)
2. `movie.nfo` - Jellyfin/Kodi standard in same directory

#### 3. Title Extraction Pipeline

`internal/scanner/patterns.go` contains regex patterns that:

1. Extract year from filename: `(2023)`, `[2023]`, `.2023.`
2. Remove quality markers: `1080p`, `BluRay`, `x264`, `HEVC`
3. Remove release groups: `-GROUP`, `[GROUP]`
4. Clean up: Convert `.` to spaces, trim

**Known Issue:** "The.Matrix.1999.1080p.BluRay.mkv" → "The Matrix p" (includes "p" from "1080p")
See ROADMAP.md section 4.1 for planned fix.

#### 4. Slug Generation

`internal/scanner/scanner.go` generates URL-safe slugs:

```go
slug = sanitizeFilename(title)
if year > 0 {
    slug = slug + "-" + strconv.Itoa(year)
}
```

Slugs are used for:
- MDX filenames: `{slug}.mdx`
- Cover images: `{slug}.jpg`, `{slug}-backdrop.jpg`
- URL routing: `/movies/{slug}`

#### 5. Smart Scanning (Incremental)

**Default behavior:** Skip files that already have MDX

```go
shouldScan := !mdxFileExists(slug)
```

**Force refresh:** `--force-refresh` flag re-processes all files

#### 6. MDX File Structure

`internal/writer/mdx.go` generates:

```yaml
---
title: Movie Title
slug: movie-slug-2023
description: Plot summary
coverImage: /covers/slug.jpg
backdropImage: /covers/slug-backdrop.jpg
rating: 8.5
releaseYear: 2023
runtime: 120
genres: [Action, Thriller]
director: Director Name
cast: [Actor 1, Actor 2, ...]
tmdbId: 12345
imdbId: tt1234567
scannedAt: 2026-01-27T12:00:00Z
---

# Movie Title (2023)

## Synopsis
Plot summary...

## Details
- Rating, runtime, etc.
```

## Configuration System

**Version:** v1.2 - Dual-config pattern (as of January 29, 2026)

MovieVault uses **two configuration files** for different deployment scenarios:

### Configuration Files

| File | Purpose | API Key | Used By |
|------|---------|---------|---------|
| `config/config.yaml` | Local development | Hardcoded | `./scanner` |
| `config/config.docker.yaml` | Docker deployment | Env var | Docker containers |
| `.env` | Docker secrets | Real keys | Docker Compose |

### Local Development Config (`config/config.yaml`)

```yaml
tmdb:
  api_key: "your_actual_key_here"  # Hardcoded for local use
  language: "en-US"

scanner:
  directories: ["/Users/you/Movies"]  # Local paths
  extensions: [".mkv", ".mp4", ...]

output:
  mdx_dir: "./website/src/content/movies"
  covers_dir: "./website/public/covers"
  website_dir: "./website"
  auto_build: true

options:
  rate_limit_delay: 250
  download_covers: true
  download_backdrops: true
  use_nfo: true            # Enable NFO parsing (v1.1.0)
  nfo_fallback_tmdb: true  # Merge TMDB if NFO incomplete
```

**Security:** Gitignored (`/config/config.yaml` in .gitignore line 2)

### Docker Deployment Config (`config/config.docker.yaml`)

```yaml
tmdb:
  api_key: "${TMDB_API_KEY}"  # Environment variable placeholder
  language: "${TMDB_LANGUAGE:-en-US}"

scanner:
  directories: ["/movies"]  # Container paths
  extensions: [".mkv", ".mp4", ...]

output:
  mdx_dir: "/data/movies"     # Mapped to ./data/movies
  covers_dir: "/data/covers"  # Mapped to ./data/covers
  auto_build: true

options:
  rate_limit_delay: 250
  download_covers: true
  download_backdrops: true
  use_nfo: true
  nfo_fallback_tmdb: true
```

**Security:** Safe to commit (no secrets, uses env vars)

### Environment Variables (`.env`)

```bash
TMDB_API_KEY=your_actual_key_here
AUTO_SCAN=true
SCAN_INTERVAL=3600
WEB_PORT=8080
```

**Security:** Gitignored (`.env` in .gitignore line 3)

### How Docker Config Works

1. Docker Compose reads `.env` file automatically
2. Mounts `config.docker.yaml` as `/config/config.yaml` inside container
3. Go app reads config file and expands `${TMDB_API_KEY}` from environment
4. No secrets in git-tracked files

**docker-compose.yml excerpt:**
```yaml
volumes:
  - ./config/config.docker.yaml:/config/config.yaml:ro  # Note: .docker variant
environment:
  - TMDB_API_KEY=${TMDB_API_KEY}  # From .env file
```

### Why Two Configs?

**Problem:** Can't commit API keys to git, but config files need them.

**Solution:**
- **Local:** Simple hardcoded config (fast iteration, gitignored)
- **Docker:** Environment variables (secure, shareable, 12-factor compliant)

**Validation:** `internal/config/config.go` validates required fields on load.

**See Also:** `CONFIG.md` for complete configuration documentation

## Critical Implementation Details

### NFO Parser (v1.1.0)

**XML Structure:** `internal/metadata/nfo/types.go`

Supports Jellyfin NFO format:
- Title, plot, rating, year, premiered, runtime
- Multiple genres, directors
- Actors (extracts top 5)
- TMDB ID, IMDb ID
- Images (parsed but not yet downloaded - uses TMDB images)

**Conversion Logic:** `parser.go:ConvertToMovie()`

- Joins multiple directors with ", "
- Extracts top 5 cast members
- Parses year from `<premiered>` if `<year>` empty
- Sets ScannedAt timestamp

**Error Handling:** All NFO errors are non-fatal
- Malformed XML → logs error, falls back to TMDB
- Missing NFO → silently falls back to TMDB
- Incomplete data → merges with TMDB

### TMDB Client

`internal/metadata/tmdb.go` implements:

- `SearchMovie(title, year)` - Search API with year filter
- `GetMovieDetails(id)` - Fetch full details by TMDB ID
- `GetFullMovieData(title, year)` - Search + fetch combined
- `DownloadImage(path, dest, type)` - Download poster/backdrop

**Rate Limiting:** Sleeps for `rate_limit_delay` ms after each request.

**Image Sizes:**
- Posters: `w500` (500px width)
- Backdrops: `w1280` (1280px width)

### Astro Integration

`website/src/content/movies/` contains generated MDX files.

**Content Collections:** Astro auto-discovers and validates MDX frontmatter.

**Key Pages:**
- `src/pages/index.astro` - Library grid with search/filter
- `src/pages/movies/[slug].astro` - Movie detail page
- `src/components/MovieCard.astro` - Grid item component

## Common Development Patterns

### Adding a New Metadata Source

1. Create package under `internal/metadata/{source}/`
2. Implement parser that returns `*writer.Movie`
3. Add priority logic in `cmd/scanner/main.go`
4. Add merge function if partial data possible
5. Update config with enable flag

Example: See NFO implementation in ENHANCEMENT_NFO_SUPPORT.md

### Modifying Movie Struct

`internal/writer/models.go` is the canonical Movie struct.

**Steps:**
1. Add field to `Movie` struct with yaml tag
2. Update `mdx.go:GenerateMDX()` to include in frontmatter
3. Update TMDB client to populate field
4. Update NFO parser if applicable
5. Update Astro components to display field

### Extending Title Extraction

`internal/scanner/patterns.go` contains cleanup patterns.

**Add pattern:**
1. Define regex in `patterns` slice
2. Test with edge cases
3. Update tests (when created)

## Important Gotchas

### 1. Slug Collisions

If two movies have same title+year, slug collision occurs. Currently overwrites.

**Planned Fix:** See ROADMAP.md section 8.3 (Duplicate Detection)

### 2. TMDB Rate Limits

Default 250ms delay between calls. Large libraries (500+ movies):
- First scan: ~2 minutes
- With NFO: ~15 seconds (60-80% API reduction)

### 3. NFO Image URLs

NFO `<thumb>` and `<fanart>` URLs are parsed but not downloaded. Still uses TMDB images.

**Reason:** NFO URLs often local paths or private servers.
**Planned:** See ROADMAP.md section 1.1

### 4. Auto-Build Timing

`auto_build: true` runs `npm run build` after scan. Large sites (500+ movies):
- Build time: 30-60 seconds
- Use `--no-build` for faster iteration

### 5. Config Path Resolution

`~` expands to home directory. Use absolute paths or `./` for relative.

### 6. File Permissions

Scanner needs read access to movie directories. Docker: ensure volume mounts are readable.

## Testing Strategy

**Current:** Manual testing only. No unit tests.

**Test Scenarios (from ENHANCEMENT_NFO_SUPPORT.md):**
1. Complete NFO → verify NFO data used
2. Missing NFO → verify TMDB fallback
3. Incomplete NFO → verify merge behavior
4. Malformed XML → verify graceful fallback
5. Both filename.nfo and movie.nfo → verify priority
6. NFO disabled → verify TMDB-only mode

**Planned:** See ROADMAP.md for test infrastructure plans

## Performance Characteristics

**Scan Performance (500 movie library):**

| Scenario | NFO Files | TMDB Calls | Time |
|----------|-----------|------------|------|
| First scan (no NFO) | 0 | 500 | ~6 min |
| First scan (80% NFO) | 400 | 100 | ~1.5 min |
| Incremental (no new) | N/A | 0 | ~5 sec |
| Force refresh (80% NFO) | 400 | 100 | ~1.5 min |

**Bottlenecks:**
- TMDB API calls: ~750ms each (incl. rate limit)
- NFO parse: ~5ms each
- Image downloads: ~300ms each
- Astro build: ~45 seconds

## Recent Changes & Migration Notes

### v1.2.0 - Dual-Config Pattern & Security Fixes (2026-01-29)

**Breaking Changes:** Docker deployment requires changes to docker-compose.yml

**New Files:**
- `CONFIG.md` - Comprehensive configuration documentation
- `.env` - Docker environment secrets (gitignored)
- `SECURITY_CLEANUP_REPORT.md` - API key exposure incident report

**Modified Files:**
- `docker-compose.yml` - Now mounts `config.docker.yaml` instead of `config.yaml`
- `CLAUDE.md` - Updated configuration documentation
- `.gitignore` - Already protected `.env` and `config/config.yaml`

**Config Migration:**

For Docker users, update `docker-compose.yml`:
```yaml
# OLD (v1.1):
- ./config/config.yaml:/config/config.yaml:ro

# NEW (v1.2):
- ./config/config.docker.yaml:/config/config.yaml:ro
```

Create `.env` file:
```bash
cp .env.example .env
# Edit .env and add your TMDB API key
```

**Behavior Changes:**
- Local runs (`./scanner`): No changes, still uses `config/config.yaml`
- Docker runs: Now uses `.env` for API key (more secure)
- Both config files coexist for different use cases

**Security Improvements:**
- API keys in Docker now via environment variables (12-factor pattern)
- No secrets in git-tracked files
- Separate configs prevent accidental key commits

See `CONFIG.md` for complete migration guide.

---

### v1.1.0 - NFO Support (2026-01-27)

**Breaking Changes:** None (opt-in feature)

**New Files:**
- `internal/metadata/nfo/types.go`
- `internal/metadata/nfo/parser.go`

**Modified Files:**
- `cmd/scanner/main.go` - Added NFO priority logic, merge function
- `internal/config/config.go` - Added `UseNFO`, `NFOFallbackTMDB` options
- `config/config.yaml` - Added NFO configuration

**Config Migration:**
```yaml
options:
  use_nfo: true              # Add this
  nfo_fallback_tmdb: true    # Add this
```

**Behavior:**
- Default: NFO enabled with TMDB fallback
- Disable: Set `use_nfo: false` for v1.0 behavior

See ENHANCEMENT_NFO_SUPPORT.md for full details.

## Future Roadmap

See ROADMAP.md for 60+ planned features across 12 categories.

**High Priority Next:**
1. Direct TMDB lookup via NFO ID (High/Low)
2. Concurrent scanning (High/Medium) - 5x speed improvement
3. Incremental scanning (High/Medium) - only changed files
4. Better title extraction (High/Medium) - fix "The Matrix p" bug
5. SQLite library database (High/Medium) - enables advanced features

## External Documentation

- **ENHANCEMENT_NFO_SUPPORT.md** - Deep dive on NFO implementation (600+ lines)
- **ROADMAP.md** - Future features with priority matrix (1100+ lines)
- **README.md** - User-facing setup and usage guide
- **config/config.yaml** - Configuration reference with comments

## TMDB API Notes

**Endpoints Used:**
- `/search/movie` - Title + year search
- `/movie/{id}` - Full movie details
- `/movie/{id}/credits` - Cast and crew
- Image base URL: `https://image.tmdb.org/t/p/{size}/{path}`

**API Key:** Get from https://www.themoviedb.org/settings/api

**Rate Limits:** No hard limit, but 250ms delay recommended.

## Docker Specifics

**Build Context:** Root directory (includes `cmd/`, `internal/`, `website/`)

**Volumes:**
- Movie directories: Read-only mounts
- `./data`: Persistent storage for MDX, images, config

**Environment:**
- `TMDB_API_KEY` - Injected into config via `${TMDB_API_KEY}`

**Entry Point:** Runs scanner on startup, then nginx serves Astro build.
