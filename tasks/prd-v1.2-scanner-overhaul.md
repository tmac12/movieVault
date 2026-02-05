# PRD: v1.2 Scanner Overhaul

## Introduction

MovieVault v1.2 focuses on scanner accuracy, reliability, and quality-of-life improvements. This release improves metadata accuracy through direct TMDB lookups and better title extraction, while adding medium-priority features like watch folders, duplicate detection, and local caching. All changes maintain backward compatibility with existing configurations and output formats.

**Version:** 1.2.0
**Codename:** Scanner Overhaul
**Previous Version:** 1.1.0 (NFO Support)

## Goals

- Improve metadata accuracy by using TMDB IDs from NFO files for direct lookups (eliminates search ambiguity)
- Fix title extraction bugs (e.g., "The Matrix p" issue) with comprehensive pattern matching
- Add watch folder support for automatic scanning of new files
- Detect and report duplicate movies in library
- Reduce API usage and improve resilience with local caching and smart retry logic
- Download images from NFO files when available (reduce TMDB dependency)
- Maintain full backward compatibility with v1.1.x configurations

## User Stories

### Phase 1: Accuracy Improvements (Must-Have)

#### US-001: Direct TMDB lookup via NFO ID
**Description:** As a user with NFO files containing TMDB IDs, I want the scanner to use direct API lookups instead of title searches so that I get 100% accurate metadata matches.

**Acceptance Criteria:**
- [ ] Add `GetMovieByID(tmdbID int)` method to TMDB client
- [ ] When NFO contains `<tmdbid>`, use direct lookup instead of search
- [ ] Fall back to title/year search only if TMDB ID missing or invalid
- [ ] Log which method was used (direct vs search) in verbose mode
- [ ] Handle invalid/deleted TMDB IDs gracefully (fall back to search)
- [ ] Typecheck passes

#### US-002: Better title extraction - quality markers
**Description:** As a user, I want quality markers (1080p, BluRay, x264, etc.) properly removed from filenames so that title extraction is accurate.

**Acceptance Criteria:**
- [ ] Remove resolution markers: 1080p, 720p, 2160p, 4K, 480p
- [ ] Remove source markers: BluRay, BRRip, WEB-DL, WEBRip, HDRip, DVDRip, HDTV
- [ ] Remove codec markers: x264, x265, HEVC, H.264, H.265, AVC, XviD, DivX
- [ ] Remove audio markers: AAC, AC3, DTS, DTS-HD, TrueHD, FLAC, MP3
- [ ] Patterns are case-insensitive
- [ ] "The.Matrix.1999.1080p.BluRay.x264.mkv" extracts as "The Matrix" (1999)
- [ ] Typecheck passes

#### US-003: Better title extraction - release groups
**Description:** As a user, I want release group tags removed from filenames so that titles are clean.

**Acceptance Criteria:**
- [ ] Remove bracketed groups: [YTS], [YIFY], [RARBG], [EVO], etc.
- [ ] Remove hyphenated suffixes: -SPARKS, -GECKOS, -FGT, etc.
- [ ] Handle groups at start or end of filename
- [ ] Preserve legitimate brackets in titles (e.g., "[REC]" the movie)
- [ ] "Inception.2010.1080p.BluRay-SPARKS.mkv" extracts as "Inception" (2010)
- [ ] Typecheck passes

#### US-004: Better title extraction - edge cases
**Description:** As a user, I want edge cases like movies starting with years handled correctly.

**Acceptance Criteria:**
- [ ] Handle year-starting titles: "2001.A.Space.Odyssey.1968" → "2001: A Space Odyssey" (1968)
- [ ] Handle multi-part titles: "The.Lord.of.the.Rings.The.Fellowship.2001" → correct title
- [ ] Handle extended editions: "Movie.2020.Extended.Cut" → "Movie" (2020), not "Movie Extended Cut"
- [ ] Handle director's cuts: "Movie.2020.Directors.Cut" → "Movie" (2020)
- [ ] Preserve intentional dots in titles when possible
- [ ] Typecheck passes

#### US-005: Title extraction test mode
**Description:** As a developer, I want a test mode to verify title extraction without running full scans.

**Acceptance Criteria:**
- [ ] Add `--test-parser` flag that accepts filenames as arguments
- [ ] Output extracted title, year, and matched patterns for each filename
- [ ] Support reading filenames from stdin for batch testing
- [ ] Exit with non-zero code if any extraction fails
- [ ] Example: `./scanner --test-parser "The.Matrix.1999.1080p.mkv"` outputs title/year
- [ ] Typecheck passes

### Phase 2: Scanner Features (Medium Priority)

#### US-006: NFO image download support
**Description:** As a user with custom artwork in NFO files, I want images downloaded from NFO URLs so that I can use my preferred artwork instead of TMDB defaults.

**Acceptance Criteria:**
- [ ] Parse `<thumb>` element for poster URL
- [ ] Parse `<fanart><thumb>` elements for backdrop URLs
- [ ] Validate URLs are HTTP/HTTPS (skip local paths like `/config/...`)
- [ ] Download NFO images with retry logic
- [ ] Fall back to TMDB images if NFO download fails
- [ ] Add config option: `options.nfo_download_images: bool` (default: false)
- [ ] Log image source in verbose mode (NFO vs TMDB)
- [ ] Typecheck passes

#### US-007: Watch folder - basic implementation
**Description:** As a user, I want new movies automatically scanned when added to my library folders so that my catalog stays up-to-date without manual intervention.

**Acceptance Criteria:**
- [ ] Add `--watch` flag to start in watch mode
- [ ] Monitor all configured scanner directories for new files
- [ ] Detect new video files matching configured extensions
- [ ] Process new files using existing scan pipeline
- [ ] Add debounce delay (default 30s) to wait for file copy completion
- [ ] Add config options: `scanner.watch_mode`, `scanner.watch_debounce`
- [ ] Typecheck passes

#### US-008: Watch folder - file events
**Description:** As a user, I want moved and renamed files handled correctly in watch mode.

**Acceptance Criteria:**
- [ ] Detect file moves within watched directories
- [ ] Detect file renames and update metadata accordingly
- [ ] Detect file deletions (log warning, don't auto-delete MDX)
- [ ] Handle rapid successive events (debounce)
- [ ] Recursive watching of subdirectories (configurable)
- [ ] Add config option: `scanner.watch_recursive: bool` (default: true)
- [ ] Typecheck passes

#### US-009: Duplicate detection - basic
**Description:** As a user, I want to know if I have duplicate movies so I can clean up my library.

**Acceptance Criteria:**
- [ ] Add `--find-duplicates` flag
- [ ] Detect duplicates by TMDB ID match
- [ ] Detect duplicates by title + year match
- [ ] Output report listing all duplicate sets
- [ ] Show file paths for each duplicate
- [ ] Exit with count of duplicate sets found
- [ ] Typecheck passes

#### US-010: Duplicate detection - quality comparison
**Description:** As a user, I want to see which duplicate has better quality so I can keep the best version.

**Acceptance Criteria:**
- [ ] Extract resolution from filename (1080p > 720p > 480p)
- [ ] Extract source quality (BluRay > WEB-DL > HDRip > DVDRip)
- [ ] Mark recommended file to keep in report
- [ ] Add `--find-duplicates --detailed` for quality breakdown
- [ ] Typecheck passes

#### US-011: Local metadata cache
**Description:** As a user, I want TMDB responses cached locally so that re-scans are faster and work offline.

**Acceptance Criteria:**
- [ ] Create SQLite cache database at configurable path
- [ ] Cache TMDB search results with TTL (default 30 days)
- [ ] Cache TMDB movie details with TTL
- [ ] Check cache before making API calls
- [ ] Add config options: `cache.enabled`, `cache.path`, `cache.ttl_days`
- [ ] Add `--clear-cache` flag to purge cache
- [ ] Log cache hits/misses in verbose mode
- [ ] Typecheck passes

#### US-012: Smart retry logic
**Description:** As a user, I want failed API calls automatically retried so that transient errors don't require manual re-runs.

**Acceptance Criteria:**
- [ ] Retry on network timeouts (up to 3 attempts)
- [ ] Retry on HTTP 429 (rate limit) with exponential backoff
- [ ] Retry on HTTP 5xx (server errors) with backoff
- [ ] Do NOT retry on 401 (bad API key) or 404 (not found)
- [ ] Add config options: `retry.max_attempts`, `retry.initial_backoff_ms`
- [ ] Log retry attempts in verbose mode
- [ ] Typecheck passes

### Phase 3: Polish & Integration

#### US-013: Cache statistics
**Description:** As a user, I want to see cache effectiveness so I know if it's helping.

**Acceptance Criteria:**
- [ ] Track cache hits and misses during scan
- [ ] Display cache stats at end of scan (hits, misses, hit rate)
- [ ] Add `--cache-stats` flag to show cache summary without scanning
- [ ] Show cache size and entry count
- [ ] Typecheck passes

#### US-014: Verbose logging improvements
**Description:** As a user, I want clearer verbose output so I can understand what the scanner is doing.

**Acceptance Criteria:**
- [ ] Log metadata source for each movie (NFO, NFO+TMDB, TMDB, Cache)
- [ ] Log TMDB lookup method (direct ID vs search)
- [ ] Log image download source (NFO vs TMDB)
- [ ] Log retry attempts with backoff duration
- [ ] Log cache operations (hit/miss/store)
- [ ] Typecheck passes

#### US-015: Configuration validation
**Description:** As a user, I want invalid configurations caught at startup so I don't waste time on failed scans.

**Acceptance Criteria:**
- [ ] Validate new config options have correct types
- [ ] Warn if `nfo_download_images: true` but `use_nfo: false`
- [ ] Warn if `watch_mode: true` but no directories configured
- [ ] Validate cache path is writable
- [ ] Validate retry settings are positive integers
- [ ] Typecheck passes

## Functional Requirements

### Direct TMDB Lookup
- FR-1: The scanner must check NFO `<tmdbid>` field before performing title search
- FR-2: When TMDB ID is present and valid, use `/movie/{id}` endpoint directly
- FR-3: When TMDB ID is invalid (404 response), fall back to title/year search
- FR-4: When TMDB ID is missing, use existing title/year search behavior

### Title Extraction
- FR-5: Title extraction must remove all common quality markers (resolution, source, codec, audio)
- FR-6: Title extraction must remove release group tags in brackets or after hyphens
- FR-7: Title extraction must handle movies starting with 4-digit years (e.g., "2001")
- FR-8: Title extraction must handle edition markers (Extended, Director's Cut, etc.)
- FR-9: The `--test-parser` flag must output extracted components without performing scans

### NFO Image Download
- FR-10: When `nfo_download_images: true`, attempt to download images from NFO URLs first
- FR-11: Only download from HTTP/HTTPS URLs (skip local paths, FTP, etc.)
- FR-12: Fall back to TMDB images when NFO image download fails
- FR-13: Existing image download behavior unchanged when `nfo_download_images: false`

### Watch Folder
- FR-14: In watch mode, monitor all directories listed in `scanner.directories`
- FR-15: Process new video files matching `scanner.extensions` after debounce delay
- FR-16: Watch mode must run continuously until terminated (SIGINT/SIGTERM)
- FR-17: Watch mode must handle directory permission changes gracefully

### Duplicate Detection
- FR-18: `--find-duplicates` must scan library and report all duplicate movies
- FR-19: Duplicates identified by matching TMDB ID or matching title+year
- FR-20: Report must show all file paths for each duplicate set
- FR-21: Quality comparison must parse resolution and source from filenames

### Local Cache
- FR-22: Cache must store TMDB responses in SQLite database
- FR-23: Cache entries must expire after configurable TTL
- FR-24: `--clear-cache` must remove all entries from cache database
- FR-25: Cache must be bypassed when `--force-refresh` is used

### Retry Logic
- FR-26: Retryable errors (timeout, 429, 5xx) must be retried up to max_attempts
- FR-27: Backoff must double after each failed attempt (exponential)
- FR-28: Non-retryable errors (401, 404) must fail immediately
- FR-29: All retry attempts must be logged in verbose mode

## Non-Goals (Out of Scope)

- **Concurrent scanning** - Deferred to v1.3 (requires more complex rate limiting)
- **Incremental scanning** - Deferred to v1.3 (requires file hash tracking)
- **SQLite library database** - Deferred to v1.3 (larger architectural change)
- **Web UI** - Deferred to v2.0
- **Multi-format NFO support** (Kodi, Plex, Emby) - Future enhancement
- **NFO writing/generation** - Future enhancement
- **Automatic duplicate cleanup** - Only detection/reporting in this release
- **Watch folder on Windows** - Linux/macOS only for v1.2

## Design Considerations

### File Structure Changes
```
internal/
├── metadata/
│   ├── tmdb.go           # Add GetMovieByID()
│   ├── cache/            # NEW: Cache package
│   │   ├── cache.go      # Cache interface
│   │   └── sqlite.go     # SQLite implementation
│   └── nfo/
│       └── parser.go     # Add image URL extraction
├── scanner/
│   ├── patterns.go       # Enhanced title extraction
│   ├── watcher.go        # NEW: Watch folder implementation
│   └── duplicates.go     # NEW: Duplicate detection
└── retry/                # NEW: Retry logic package
    └── retry.go
```

### Configuration Additions
```yaml
# New options (all optional, backward compatible)
options:
  nfo_download_images: false    # Download images from NFO URLs

scanner:
  watch_mode: false             # Enable file watching
  watch_debounce: 30            # Seconds to wait before processing
  watch_recursive: true         # Watch subdirectories

cache:
  enabled: true                 # Enable local metadata cache
  path: "./data/cache.db"       # SQLite database path
  ttl_days: 30                  # Cache entry lifetime

retry:
  max_attempts: 3               # Maximum retry attempts
  initial_backoff_ms: 1000      # Initial backoff (doubles each retry)
```

### Backward Compatibility
- All new config options have sensible defaults
- Existing configs work without modification
- MDX output format unchanged
- Existing CLI flags unchanged
- Cache is opt-in (enabled by default but can be disabled)

## Technical Considerations

### Dependencies
- `github.com/fsnotify/fsnotify` - File system watching (watch folder)
- `github.com/mattn/go-sqlite3` - SQLite driver (cache)
- No new dependencies for title extraction or retry logic

### Performance Impact
- Direct TMDB lookup: Faster (1 API call vs 2 for search+details)
- Local cache: Significantly faster re-scans
- Watch folder: Minimal overhead when idle
- Title extraction: Negligible (regex processing)

### Error Handling
- NFO image download failures: Silent fallback to TMDB
- Cache database errors: Log warning, continue without cache
- Watch folder errors: Log error, continue watching
- Invalid TMDB ID: Fall back to search, log warning

### Testing Strategy
- Unit tests for title extraction patterns
- Unit tests for retry logic
- Integration tests for cache operations
- Manual testing for watch folder (OS-specific behavior)

## Success Metrics

- Title extraction accuracy: 95%+ on standard release naming conventions
- Direct TMDB lookup usage: 80%+ of movies with NFO files
- Cache hit rate: 90%+ on re-scans
- Zero breaking changes for existing users
- Watch folder responds to new files within debounce + 5 seconds

## Open Questions

1. Should watch folder also monitor for NFO file changes (re-scan when NFO updated)?
2. Should duplicate detection consider file size as a quality indicator?
3. Should cache be shared between Docker container runs (persist in mounted volume)?
4. What's the behavior when watch folder detects a file during an active scan?

---

## Implementation Order

**Phase 1 (Must-Have):**
1. US-001: Direct TMDB lookup via NFO ID
2. US-002-004: Better title extraction (all patterns)
3. US-005: Title extraction test mode

**Phase 2 (Medium Priority):**
4. US-012: Smart retry logic (foundation for other features)
5. US-011: Local metadata cache
6. US-006: NFO image download support
7. US-007-008: Watch folder
8. US-009-010: Duplicate detection

**Phase 3 (Polish):**
9. US-013: Cache statistics
10. US-014: Verbose logging improvements
11. US-015: Configuration validation

---

*PRD generated for MovieVault v1.2 Scanner Overhaul*
