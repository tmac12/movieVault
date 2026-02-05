# Changelog

All notable changes to this project will be documented in this file.

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
