# Changelog

All notable changes to this project will be documented in this file.

---

## [1.4.4] - 2026-02-20

### Added

- **Multi-library filtering:** New `LibraryFilter` component displays filter chips when movies span multiple source directories. Only renders when more than one library is detected. Clicking a chip hides movies from other libraries; legacy movies without `sourceDir` remain always visible. (`website/src/components/LibraryFilter.astro`)
- **Year filter:** New `YearFilter` component lets users filter movies by release year with interactive chips. (`website/src/components/YearFilter.astro`)
- **Source directory tracking:** Scanner now records `sourceDir` (the configured root directory) for each scanned file. Propagated through the MDX pipeline as an optional frontmatter field. Enables multi-library UI features. (`internal/scanner/scanner.go`, `cmd/scanner/scan.go`, `internal/writer/models.go`, `website/src/content/config.ts`)
- **Library badge on movie cards:** Movie cards display the source library name as a small badge when `sourceDir` is present, using the last path segment as a human-readable label. (`website/src/components/MovieCard.astro`)
- **Multi-platform Docker builds:** Dockerfile updated to use BuildKit's `TARGETOS`/`TARGETARCH` for cross-compilation. Supports `linux/amd64` and `linux/arm64`.
- **Docker content sync (`syncBuildToNginx`):** New entrypoint logic copies the Astro build output to the nginx serve directory with correct file permissions and triggers a live nginx reload. Handles website build based on whether scheduled scanning is active. (`docker/entrypoint.sh`)

### Changed

- **TMDB image retrieval:** Poster and backdrop images are now sourced from the full movie details endpoint rather than search results, improving image quality and availability. (`internal/metadata/tmdb.go`)
- **Docker configuration restructured:** Persistent data (MDX files, covers, cache) and Docker config now live under `configServer/` directory for cleaner separation from source. `docker-compose.yml` updated to reflect new mount paths.
- **Composable filter CSS:** Added `.movie-card.library-hidden` rule to `global.css` so genre, year, and library filters coexist without style conflicts.

### Fixed

- **Docker entrypoint scheduling:** Entrypoint script correctly detects `SCHEDULE_ENABLED` and runs the scanner in background mode when scheduling is active; falls back to a one-shot scan otherwise.

---

## [1.4.0] - 2026-02-13

### Added

- **Concurrent file processing:** Worker pool architecture for parallel scanning with configurable concurrency. New config option: `scanner.concurrent_workers` (default: 5, range: 1-20). New CLI flag: `--workers N` to override config. Delivers ~5x performance improvement for large libraries.
- **Thread-safe slug deduplication:** `SlugGuard` mechanism prevents slug collisions when multiple workers process files with identical title+year combinations. First worker to claim a slug wins; subsequent duplicates are skipped.
- **Atomic progress tracking:** Worker pool reports real-time progress via atomic counters, enabling accurate status updates during concurrent operations.
- **Context-aware cancellation:** Workers respect context cancellation (SIGINT/SIGTERM), ensuring graceful shutdown mid-scan without orphaned goroutines.
- **Scheduled scanning:** Interval-based periodic scanning at configurable intervals. Scanner runs continuously, triggering scans every N minutes. Optionally runs immediately on startup (default: true). New config keys: `scanner.schedule_enabled` (default: false), `scanner.schedule_interval` (default: 60 minutes), `scanner.schedule_on_startup` (default: true). New CLI flags: `--schedule`, `--schedule-interval N`.
- **Overlap prevention:** Atomic lock-free mechanism prevents concurrent scheduled scans from overlapping. If a scan exceeds the interval, subsequent triggers are skipped with warnings suggesting interval adjustment.
- **Docker scheduled mode:** Native support for scheduled scanning in Docker deployments via environment variables. Set `SCHEDULE_ENABLED=true` and `SCHEDULE_INTERVAL=60` (minutes) in `.env`. Scanner runs as background service alongside nginx.
- **Coexistence with watch mode:** Watch mode and scheduled scanning can run simultaneously (watch = immediate, schedule = periodic validation). Managed via `sync.WaitGroup` with shared context for graceful shutdown.
- **Scan extraction:** Core scan logic extracted into reusable `runScan()` function in `cmd/scanner/scan.go`. Enables consistent behavior across manual scans, scheduled scans, and future enhancements. Returns `ScanResults` struct with counts, duration, and errors.

### New files

- `internal/scanner/pool.go` — Worker pool implementation with `ProcessFilesConcurrently` and `SlugGuard`
- `internal/scanner/pool_test.go` — Comprehensive test suite (8 tests covering concurrency, slug guard, cancellation, edge cases)
- `cmd/scanner/scan.go` — Extracted scan logic with `runScan()` function and `ScanResults` struct (~250 lines)
- `cmd/scanner/scheduler.go` — Scheduler implementation with `startScheduler()` and overlap prevention (~90 lines)

### Changed

- **Scanner architecture:** Refactored main scan loop in `cmd/scanner/main.go` to use worker pool when `concurrent_workers > 1`. Sequential processing still available as fallback.
- **Config validation:** Added validation for `scanner.concurrent_workers` with warnings for values > 20 (potential TMDB rate limit issues).
- **Main loop refactoring:** `cmd/scanner/main.go` simplified to use extracted `runScan()` function. Goroutine management added for watch and schedule modes with unified signal handling.
- **Config validation:** Added validation for schedule settings with warnings for intervals < 5 minutes (high CPU/API usage). Info log when both watch and schedule are enabled.
- **Docker entrypoint:** Updated `docker/entrypoint.sh` to detect `SCHEDULE_ENABLED` and run scanner in background mode. Backward compatible with legacy `AUTO_SCAN` variable.
- **Environment variables:** Added `SCHEDULE_ENABLED` and `SCHEDULE_INTERVAL` to `.env` and `docker-compose.yml`. Legacy `AUTO_SCAN` and `SCAN_INTERVAL` remain for backward compatibility.

### Technical details

- **Thread safety:** Scheduler uses `atomic.Bool` with `CompareAndSwap` for lock-free overlap detection. All existing components (worker pool, TMDB client, cache, MDX writer) are already thread-safe.
- **Incremental scans:** Scheduled scans process only new files (skip existing MDX). Users can manually run `--force-refresh` for full rescans.
- **Graceful shutdown:** Context cancellation propagates to both watch and schedule goroutines. Signal handling ensures clean exit with WaitGroup synchronization.

---

## [1.3.2] - 2026-02-05

### Fixed

- **NFO `<art>` block parsing:** Poster and fanart images defined inside `<art><poster>` / `<art><fanart>` (standard Jellyfin/Kodi format) were silently ignored. The parser now reads these elements and uses them as fallback when `<thumb>`-based extraction returns empty (`internal/metadata/nfo/types.go`, `internal/metadata/nfo/parser.go`).
- **Local image paths in NFO files:** `DownloadImageFromURL` previously rejected any path that wasn't `http://` or `https://`, causing local poster/fanart paths from NFO files to fail silently and produce no poster. Local filesystem paths are now copied directly to the covers directory. TMDB fallback still applies if the local file does not exist (`internal/metadata/tmdb.go`).

---

## [1.3.1] - 2026-02-04

### Changed

- **Docker build — GHCR access verification:** Added a pre-checkout step to the GitHub Actions workflow (`docker-build.yml`) that verifies the `movievault` package is linked to the repository before attempting a push. The step checks the HTTP status of the package endpoint and fails early with an actionable error message and a direct link to the GitHub package settings page if access would be denied.

---

## [1.3.0] - 2026-02-03

### Added

- **Watch mode:** Continuous directory monitoring via `fsnotify` with a configurable debounce delay (default 30 s). Rename and delete events cancel pending timers. Graceful shutdown on SIGINT/SIGTERM. New flag: `--watch`. New config keys: `scanner.watch_mode`, `scanner.watch_debounce`, `scanner.watch_recursive`.
- **Duplicate detection:** `--find-duplicates` groups movies by TMDB ID (or title + year as fallback) and scores each copy using `resolution_rank × 10 + source_rank`. The highest-scoring copy in each group is marked as recommended. `--detailed` shows the full quality breakdown per copy.
- **SQLite metadata cache:** Caches TMDB API responses in a local SQLite database with configurable TTL. New flag: `--cache-stats` displays hit/miss summary. New config keys: `cache.enabled`, `cache.path`, `cache.ttl_days`.
- **Title extraction improvements:** Recognition of audio markers (DTS-HD, Atmos, FLAC), release groups (`-SPARKS`, `[YTS]`), edition markers (Director's Cut, IMAX), and year-starting titles (e.g. "2001: A Space Odyssey"). New flag: `--test-parser` for interactive extraction testing.
- **NFO image downloads:** Extract and download poster/backdrop from NFO `<thumb>` and `<fanart>` elements. Falls back to TMDB on failure. Controlled by the `options.nfo_download_images` config key.
- **Structured logging:** Verbose mode (`--verbose`) uses consistent structured fields across all operations (`file`, `nfo_status`, `method`, `source`, etc.).
- **Config validation:** Startup validation for retry, cache, NFO, and watch settings with clear error messages.

### New files

- `internal/scanner/watcher.go` — watch mode implementation
- `internal/scanner/duplicates.go` — duplicate detection and quality scoring
- `internal/scanner/patterns_test.go` — unit tests for title extraction
- `internal/metadata/cache/cache.go` — cache interface and stats
- `internal/metadata/cache/sqlite.go` — SQLite cache implementation
