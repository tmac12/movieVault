# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MovieVault is a movie library scanner that discovers video files, fetches metadata from TMDB API (or Jellyfin .nfo files), generates MDX files, and builds a static Astro website for browsing a personal movie collection.

**Tech Stack:**
- **Backend:** Go 1.25.5
- **Frontend:** Astro 5.2.0 with MDX
- **Metadata:** TMDB API + Jellyfin NFO support
- **Deployment:** Docker or native

**Current Version:** 1.3.0

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

# Watch mode
./scanner --watch               # Continuously monitor for new files

# Diagnostics
./scanner --test-parser "Movie.Name.2020.1080p.mkv"  # Test title extraction
./scanner --find-duplicates     # Report duplicate movies
./scanner --find-duplicates --detailed  # With quality scores
./scanner --cache-stats         # Show cache hit/miss stats
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
│   ├── watcher.go      # Watch mode (fsnotify-based)
│   └── duplicates.go   # Duplicate detection + quality scoring
├── metadata/        # Metadata sources
│   ├── tmdb.go      # TMDB API client
│   ├── nfo/         # Jellyfin NFO parser
│   │   ├── types.go    # XML structs
│   │   └── parser.go   # Parse, convert, merge, image URL extraction
│   ├── cache/       # SQLite TMDB response cache
│   │   ├── cache.go    # Cache interface + stats
│   │   └── sqlite.go   # SQLite implementation
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

**Note:** Title extraction handles quality markers, audio codecs, edition markers, release groups,
and year-starting titles. Use `--test-parser` for interactive testing.

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

#### 7. Watch Mode

`internal/scanner/watcher.go` uses `fsnotify` to monitor configured directories. A debounce
timer (default 30s) waits after each file event before processing, preventing partial-write
issues during large file copies. Rename and delete events cancel pending timers.

#### 8. Duplicate Detection

`internal/scanner/duplicates.go` groups movies by TMDB ID (or title+year as fallback) and
computes a quality score per copy: `resolution_rank × 10 + source_rank`. The highest-scoring
copy in each group is marked as recommended. Activated via `--find-duplicates`.

#### 9. SQLite Cache

`internal/metadata/cache/` caches TMDB API responses in a local SQLite database with
configurable TTL. Tracks hits, misses, and entry count. Stats are viewable with `--cache-stats`.

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
  watch_mode: false           # Enable continuous directory monitoring
  watch_debounce: 30          # Seconds to wait after file change
  watch_recursive: true       # Watch subdirectories

output:
  mdx_dir: "./website/src/content/movies"
  covers_dir: "./website/public/covers"
  website_dir: "./website"
  auto_build: true

options:
  rate_limit_delay: 250
  download_covers: true
  download_backdrops: true
  use_nfo: true            # Enable NFO parsing
  nfo_fallback_tmdb: true  # Merge TMDB if NFO incomplete
  nfo_download_images: false  # Try NFO image URLs before TMDB

retry:
  max_attempts: 3           # Retries for transient API errors
  initial_backoff_ms: 1000  # Doubles each retry

cache:
  enabled: true             # SQLite cache for TMDB responses
  path: "./data/cache.db"
  ttl_days: 30              # Cache entry expiry
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
  watch_mode: false
  watch_debounce: 30
  watch_recursive: true

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
  nfo_download_images: false

retry:
  max_attempts: 3
  initial_backoff_ms: 1000

cache:
  enabled: true
  path: "/data/cache.db"
  ttl_days: 30
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
- Images (poster/backdrop URLs extracted; downloaded when `nfo_download_images` enabled)

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

NFO `<thumb>` and `<fanart>` URLs are extracted and downloaded when `nfo_download_images: true`.
Falls back to TMDB images if NFO URLs fail or are absent.

**Note:** Only HTTP/HTTPS URLs are attempted. Local filesystem paths in NFO files are skipped.

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

### v1.3.0 - Watch Mode, Duplicate Detection & Cache (2026-02-03)

**New Features:**
- **Title Extraction (US-013–017):** Audio markers (DTS-HD, Atmos, FLAC), release groups (`-SPARKS`, `[YTS]`), edition markers (Director's Cut, IMAX), year-starting titles (e.g. "2001: A Space Odyssey"), and `--test-parser` CLI flag
- **NFO Image Downloads (US-018–020):** Extract and download poster/backdrop from NFO `<thumb>`/`<fanart>`. Falls back to TMDB. Controlled by `nfo_download_images`
- **Watch Mode (US-021–023):** Continuous directory monitoring via fsnotify with debounce, rename/delete event handling, graceful shutdown
- **Duplicate Detection (US-024–025):** `--find-duplicates` with quality-based scoring (resolution × 10 + source rank) and recommended-copy marking. `--detailed` shows full breakdown
- **Cache Statistics (US-026):** SQLite cache tracks hits, misses, hit rate. `--cache-stats` flag for summary
- **Structured Logging (US-027):** Verbose mode uses consistent structured fields (file, nfo_status, method, source, etc.)
- **Config Validation (US-028):** Validates retry, cache, NFO, and watch settings on startup with clear error messages

**New CLI Flags:** `--watch`, `--test-parser`, `--find-duplicates`, `--find-duplicates --detailed`, `--cache-stats`

**New Config Options:** `scanner.watch_mode`, `scanner.watch_debounce`, `scanner.watch_recursive`, `options.nfo_download_images`, `retry.*`, `cache.*`

**New Files:**
- `internal/scanner/watcher.go` — watch mode implementation
- `internal/scanner/duplicates.go` — duplicate detection + quality scoring
- `internal/scanner/patterns_test.go` — unit tests for title extraction
- `internal/metadata/cache/cache.go` — cache interface + stats
- `internal/metadata/cache/sqlite.go` — SQLite cache implementation

**Migration:** No breaking changes. All new features are opt-in or additive.

---

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
4. SQLite library database (High/Medium) - enables advanced features
5. Built-in web UI (High/High) - major usability improvement

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

### Multi-Platform Builds

**Supported Platforms:** linux/amd64, linux/arm64

**Tag Convention:** All release tags must start with `v` (e.g., `v1.4.0`, not `1.4.0`) to trigger automated GitHub Actions builds.

**Build Arguments:** Dockerfile uses Docker BuildKit's automatic `TARGETOS` and `TARGETARCH` build arguments for cross-compilation. The Go builder stage sets `GOOS` and `GOARCH` environment variables to build platform-specific binaries.

**Local Multi-Platform Build:**
```bash
# Build for both platforms
docker buildx build --platform linux/amd64,linux/arm64 -t movievault:latest .

# Build and load for local testing (single platform only)
docker buildx build --platform linux/amd64 -t movievault:amd64 --load .
```

**Note:** The scanner binary is cross-compiled during the build, so it cannot be executed on the build host for testing. The Dockerfile uses `file` command to verify the binary architecture instead.
