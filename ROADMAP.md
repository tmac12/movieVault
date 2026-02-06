# MovieVault - Future Roadmap

**Last Updated:** February 6, 2026
**Current Version:** 1.3.2

This document outlines potential improvements, feature requests, and enhancement ideas for the MovieVault project. Items are categorized by area and prioritized based on impact, complexity, and user demand.

---

## Table of Contents

1. [NFO Enhancements](#nfo-enhancements)
2. [Metadata Sources](#metadata-sources)
3. [Performance Optimizations](#performance-optimizations)
4. [Scanner Improvements](#scanner-improvements)
5. [Output & Templates](#output--templates)
6. [Database & Caching](#database--caching)
7. [Web Interface](#web-interface)
8. [Library Management](#library-management)
9. [Media Analysis](#media-analysis)
10. [Integration & APIs](#integration--apis)
11. [Quality of Life](#quality-of-life)
12. [Advanced Features](#advanced-features)

---

## NFO Enhancements

### 1.1 NFO Image Download Support ✅ COMPLETED
**Priority:** Medium | **Complexity:** Medium | **Impact:** Medium | **Implemented:** US-018, US-019, US-020

**Description:**
Currently, NFO image URLs are parsed but not downloaded. Images still come from TMDB. Add support for downloading images referenced in .nfo files.

**Implementation:**
```go
// Parse NFO image URLs
if nfo.Fanart != nil && len(nfo.Fanart.Thumbs) > 0 {
    backdropURL := nfo.Fanart.Thumbs[0].URL
    if isValidURL(backdropURL) {
        downloadImage(backdropURL, backdropPath)
    }
}
```

**Challenges:**
- NFO URLs may be local paths (`/config/metadata/...`)
- Need URL validation (HTTP/HTTPS only)
- Fallback to TMDB if download fails
- Handle authentication for private servers

**Benefits:**
- Use user's custom artwork
- Support local/custom images
- Reduce TMDB dependency

**Tasks:**
- [x] Add URL validation function
- [x] Implement image download with retry
- [x] Add fallback to TMDB on failure
- [x] Support both local and remote URLs
- [x] Add config option: `nfo_download_images: bool`
- [x] Parse `<art><poster>`/`<art><fanart>` blocks (v1.3.2)
- [x] Support local filesystem image paths from NFO files (v1.3.2)

---

### 1.2 Direct TMDB Lookup via NFO ID ✅ COMPLETED
**Priority:** High | **Complexity:** Low | **Impact:** High

**Description:**
When .nfo contains `<tmdbid>`, use direct TMDB API lookup instead of searching by title/year. More accurate and faster.

**Current Behavior:**
```go
// Always searches by title
movie, err := tmdbClient.GetFullMovieData(file.Title, file.Year)
```

**Proposed Behavior:**
```go
if nfo.TMDBID > 0 {
    // Direct fetch by ID
    movie, err := tmdbClient.GetMovieByID(nfo.TMDBID)
} else {
    // Fall back to search
    movie, err := tmdbClient.GetFullMovieData(file.Title, file.Year)
}
```

**Benefits:**
- 100% accurate TMDB match
- Faster (skip search step)
- No ambiguity with similar titles
- Better for foreign films

**Tasks:**
- [x] Add `GetMovieByID(id int)` to TMDB client
- [x] Update NFO parser to use TMDB ID first
- [x] Add metrics tracking (ID vs search usage) — logged via structured logging as `method: "direct ID"` vs `"search"`
- [x] Handle invalid/deleted TMDB IDs — falls back to title search on 404

---

### 1.3 NFO Writing/Generation
**Priority:** Low | **Complexity:** Medium | **Impact:** Medium

**Description:**
Generate .nfo files from TMDB data for movies that don't have them. Bootstrap Jellyfin metadata.

**Use Cases:**
- New library without existing NFO files
- Backup TMDB data locally
- Pre-populate Jellyfin metadata
- Offline metadata access

**Implementation:**
```go
func WriteNFO(movie *writer.Movie, videoPath string) error {
    nfo := &NFOMovie{
        Title:     movie.Title,
        Plot:      movie.Description,
        Rating:    movie.Rating,
        Year:      movie.ReleaseYear,
        Premiered: movie.ReleaseDate,
        Runtime:   movie.Runtime,
        Genres:    movie.Genres,
        TMDBID:    movie.TMDBID,
        IMDbID:    movie.IMDbID,
        // ... map all fields
    }

    return writeXML(nfo, getNFOPath(videoPath))
}
```

**Configuration:**
```yaml
options:
  write_nfo: true           # Generate .nfo files
  nfo_overwrite: false      # Overwrite existing .nfo
  nfo_format: "jellyfin"    # jellyfin, kodi, plex
```

**Tasks:**
- [ ] Implement XML marshaling for NFO
- [ ] Add NFO template for different formats
- [ ] Respect existing .nfo files (don't overwrite)
- [ ] Add flag: `--generate-nfo`
- [ ] Support actor images and detailed metadata

---

### 1.4 Multi-Format NFO Support
**Priority:** Low | **Complexity:** High | **Impact:** Low

**Description:**
Support .nfo formats from Kodi, Plex, Emby in addition to Jellyfin.

**Different Schemas:**
```xml
<!-- Kodi -->
<movie>
    <ratings>
        <rating name="tmdb" max="10">8.5</rating>
    </ratings>
</movie>

<!-- Plex -->
<movie>
    <rating>8.5</rating>
</movie>

<!-- Jellyfin -->
<movie>
    <rating>8.5</rating>
</movie>
```

**Implementation:**
- Auto-detect format from XML structure
- Parse using format-specific logic
- Normalize to common Movie struct

**Tasks:**
- [ ] Research Kodi NFO schema
- [ ] Research Plex NFO schema
- [ ] Research Emby NFO schema
- [ ] Implement format detection
- [ ] Add format-specific parsers
- [ ] Test with real-world files

---

### 1.5 NFO Validation & Repair
**Priority:** Low | **Complexity:** Medium | **Impact:** Medium

**Description:**
Validate .nfo files against schema and offer repair/enrichment options.

**Features:**
- Detect missing required fields
- Validate TMDB/IMDb IDs
- Check image URLs
- Fix common XML errors
- Suggest improvements

**CLI Commands:**
```bash
# Validate all NFO files
./scanner --validate-nfo

# Repair malformed NFO files
./scanner --repair-nfo

# Enrich incomplete NFO files with TMDB data
./scanner --enrich-nfo
```

**Output:**
```
Validating NFO files...

The Matrix (1999).nfo: ✓ Valid
Inception.nfo: ⚠ Warning - Missing IMDb ID
Malformed.nfo: ❌ Error - Invalid XML on line 5

Summary: 1 valid, 1 warning, 1 error
```

**Tasks:**
- [ ] Define NFO validation rules
- [ ] Implement XML schema validation
- [ ] Add repair logic for common errors
- [ ] Create validation report
- [ ] Add `--validate-nfo` flag

---

## Metadata Sources

### 2.1 IMDb Integration
**Priority:** Medium | **Complexity:** Medium | **Impact:** High

**Description:**
Add IMDb as an alternative metadata source. Some users prefer IMDb ratings and data.

**API Options:**
- OMDb API (free with key, 1000 calls/day)
- IMDb datasets (downloadable, offline)
- Web scraping (fragile, against ToS)

**Recommended:** OMDb API

**Configuration:**
```yaml
imdb:
  api_key: "your_omdb_key"
  use_imdb_rating: true    # Prefer IMDb rating over TMDB
  fallback_order: ["nfo", "imdb", "tmdb"]
```

**Data Comparison:**
```go
type MetadataSource interface {
    GetMovie(title string, year int) (*Movie, error)
    GetMovieByID(id string) (*Movie, error)
}

// Implement for TMDB, IMDb, NFO
```

**Benefits:**
- More accurate ratings for some users
- Alternative when TMDB down
- Cross-reference data for accuracy
- IMDb IDs for linking

**Tasks:**
- [ ] Research OMDb API
- [ ] Implement IMDb client
- [ ] Add IMDb to fallback chain
- [ ] Add rating source preference
- [ ] Update MDX template to show source

---

### 2.2 Trakt.tv Integration
**Priority:** Low | **Complexity:** Medium | **Impact:** Medium

**Description:**
Integrate with Trakt.tv for watch history, ratings, and social features.

**Features:**
- Import watch history
- Sync user ratings
- Get trending/popular lists
- Show watched/unwatched status
- Export library to Trakt

**Configuration:**
```yaml
trakt:
  client_id: "your_client_id"
  client_secret: "your_secret"
  sync_watched: true
  sync_ratings: true
```

**MDX Output:**
```yaml
watchedAt: "2025-12-25T15:30:00Z"
userRating: 9.0
traktId: 12345
```

**Tasks:**
- [ ] Implement Trakt OAuth
- [ ] Add Trakt API client
- [ ] Sync watch history
- [ ] Import user ratings
- [ ] Add watched status to MDX

---

### 2.3 Local Metadata Cache ✅ COMPLETED
**Priority:** Medium | **Complexity:** Low | **Impact:** High | **Implemented:** v1.3.0 (see also §6.2)

**Description:**
Cache TMDB responses locally to avoid redundant API calls during re-scans.

**Implementation:**
```go
type MetadataCache struct {
    store map[string]*CachedMovie
    ttl   time.Duration
}

func (c *MetadataCache) Get(key string) (*Movie, bool) {
    cached, exists := c.store[key]
    if exists && !cached.IsExpired() {
        return cached.Movie, true
    }
    return nil, false
}
```

**Storage Options:**
- In-memory (fast, lost on restart)
- SQLite (persistent, queryable)
- JSON files (simple, portable)

**Recommended:** SQLite

**Benefits:**
- Instant re-scans
- Offline mode support
- Reduced API usage
- Historical metadata

**Tasks:**
- [x] Design cache schema
- [x] Implement SQLite backend
- [x] Add cache hit/miss metrics
- [x] Add cache invalidation (TTL-based)
- [x] Add `--clear-cache` flag

---

### 2.4 Multiple Language Support
**Priority:** Medium | **Complexity:** Medium | **Impact:** Medium

**Description:**
Fetch metadata in different languages from TMDB.

**Configuration:**
```yaml
tmdb:
  language: "en-US"        # Primary language
  fallback_languages:      # Fallback order
    - "en-US"
    - "ja-JP"
    - "fr-FR"
```

**Use Cases:**
- Foreign film libraries
- Multilingual users
- Original title preservation
- International cast names

**MDX Output:**
```yaml
title: "Spirited Away"
originalTitle: "千と千尋の神隠し"
language: "ja"
translations:
  en: "Spirited Away"
  ja: "千と千尋の神隠し"
  fr: "Le Voyage de Chihiro"
```

**Tasks:**
- [ ] Add language parameter to TMDB calls
- [ ] Fetch multiple language versions
- [ ] Store translations in MDX
- [ ] Update templates for i18n
- [ ] Add language auto-detection

---

### 2.5 Custom Metadata Sources
**Priority:** Low | **Complexity:** High | **Impact:** Low

**Description:**
Plugin system for custom metadata sources (private APIs, local databases, etc.).

**Architecture:**
```go
type MetadataProvider interface {
    Name() string
    Search(title string, year int) ([]*Movie, error)
    GetByID(id string) (*Movie, error)
    Priority() int
}

// Users can implement custom providers
type MyCustomProvider struct{}

func (p *MyCustomProvider) Search(...) {...}
```

**Configuration:**
```yaml
metadata_providers:
  - type: "nfo"
    priority: 1
  - type: "custom"
    priority: 2
    plugin: "./plugins/my_provider.so"
  - type: "tmdb"
    priority: 3
```

**Tasks:**
- [ ] Define provider interface
- [ ] Implement plugin loading
- [ ] Add provider registry
- [ ] Create example plugin
- [ ] Document plugin development

---

## Performance Optimizations

### 3.1 Concurrent Scanning
**Priority:** High | **Complexity:** Medium | **Impact:** High

**Description:**
Process multiple movies in parallel instead of sequentially.

**Current:** Sequential processing
```go
for _, file := range files {
    processMovie(file)  // One at a time
}
```

**Proposed:** Concurrent processing
```go
const workers = 5  // Configurable
semaphore := make(chan struct{}, workers)

for _, file := range files {
    semaphore <- struct{}{}  // Acquire
    go func(f FileInfo) {
        defer func() { <-semaphore }()  // Release
        processMovie(f)
    }(file)
}
```

**Configuration:**
```yaml
scanner:
  concurrent_scans: 5       # Parallel workers
  rate_limit_per_worker: 250  # Per-worker delay
```

**Benefits:**
- 5x faster with 5 workers (estimated)
- Better resource utilization
- Configurable parallelism

**Challenges:**
- TMDB rate limiting (need per-worker limits)
- File write conflicts
- Progress reporting
- Error handling complexity

**Tasks:**
- [ ] Implement worker pool
- [ ] Add rate limiting per worker
- [ ] Thread-safe progress reporting
- [ ] Concurrent MDX writing
- [ ] Add `--workers N` flag

---

### 3.2 Incremental Scanning
**Priority:** High | **Complexity:** Medium | **Impact:** High

**Description:**
Only scan files that changed since last run (modification time check).

**Implementation:**
```go
type ScanState struct {
    LastScan   time.Time
    FileHashes map[string]string  // path -> hash
}

func shouldScan(file FileInfo, state *ScanState) bool {
    // Check if file modified since last scan
    if file.ModTime.After(state.LastScan) {
        return true
    }

    // Check if NFO modified
    nfoPath, _ := findNFO(file.Path)
    nfoModTime, _ := getModTime(nfoPath)
    if nfoModTime.After(state.LastScan) {
        return true
    }

    return false
}
```

**Storage:**
```json
{
  "last_scan": "2026-01-27T12:00:00Z",
  "files": {
    "/path/to/movie.mkv": {
      "mod_time": "2025-12-01T10:00:00Z",
      "nfo_mod_time": "2026-01-15T08:00:00Z",
      "hash": "abc123..."
    }
  }
}
```

**Benefits:**
- Near-instant re-scans
- Only process changed files
- Better for scheduled scans

**Tasks:**
- [ ] Design scan state schema
- [ ] Track file modification times
- [ ] Track NFO modification times
- [ ] Implement state persistence
- [ ] Add `--incremental` flag

---

### 3.3 Smart Caching Strategy
**Priority:** Medium | **Complexity:** Medium | **Impact:** Medium

**Description:**
Multi-tier caching: memory → disk → API.

**Architecture:**
```
Request → Memory Cache (fast)
  ↓ miss
Disk Cache (SQLite)
  ↓ miss
API Call (TMDB/IMDb)
  ↓
Update Caches
```

**Implementation:**
```go
type CacheManager struct {
    memory    *sync.Map           // L1: In-memory
    disk      *sql.DB             // L2: SQLite
    ttl       map[string]time.Duration
}

func (c *CacheManager) Get(key string) (*Movie, error) {
    // Try L1
    if val, ok := c.memory.Load(key); ok {
        return val.(*Movie), nil
    }

    // Try L2
    if movie := c.loadFromDisk(key); movie != nil {
        c.memory.Store(key, movie)  // Promote to L1
        return movie, nil
    }

    // Cache miss
    return nil, ErrCacheMiss
}
```

**Configuration:**
```yaml
cache:
  memory_size: 100          # Max movies in memory
  disk_ttl: 2592000         # 30 days in seconds
  enable_preload: true      # Load cache at startup
```

**Tasks:**
- [ ] Implement two-tier cache
- [ ] Add LRU eviction for memory
- [ ] Implement cache preloading
- [ ] Add cache statistics
- [ ] Optimize cache key generation

---

### 3.4 Batch API Requests
**Priority:** Low | **Complexity:** Medium | **Impact:** Low

**Description:**
Batch multiple TMDB requests into single API calls where possible.

**TMDB Limitations:**
- Most endpoints don't support batching
- Would need request queueing and deduplication

**Alternative:** Request deduplication
```go
type RequestDeduplicator struct {
    pending map[string]chan *Movie
    mu      sync.Mutex
}

func (d *RequestDeduplicator) Get(title string, year int) (*Movie, error) {
    key := fmt.Sprintf("%s-%d", title, year)

    d.mu.Lock()
    if ch, exists := d.pending[key]; exists {
        d.mu.Unlock()
        return <-ch, nil  // Wait for in-flight request
    }

    ch := make(chan *Movie, 1)
    d.pending[key] = ch
    d.mu.Unlock()

    // Make request
    movie, err := tmdb.Get(title, year)
    ch <- movie

    d.mu.Lock()
    delete(d.pending, key)
    d.mu.Unlock()

    return movie, err
}
```

**Benefits:**
- Avoid duplicate requests
- Useful with concurrent scanning
- Reduces API usage

**Tasks:**
- [ ] Implement request deduplication
- [ ] Add metrics tracking
- [ ] Test with concurrent scans

---

### 3.5 Image Download Optimization
**Priority:** Medium | **Complexity:** Low | **Impact:** Medium

**Description:**
Optimize image downloads with better compression and formats.

**Current:** Download full-size JPG images

**Improvements:**
- Download appropriate size (no need for 4K posters)
- Use modern formats (WebP, AVIF)
- Lazy download (download on-demand, not during scan)
- Progressive images for faster loads

**Configuration:**
```yaml
images:
  poster_size: "w500"       # TMDB sizes: w92, w154, w185, w342, w500, w780, original
  backdrop_size: "w1280"    # w300, w780, w1280, original
  format: "jpg"             # jpg, webp, avif
  quality: 85               # 0-100
  lazy_download: false      # Download during scan or on-demand
```

**Benefits:**
- Smaller file sizes
- Faster downloads
- Better quality/size ratio
- Reduced bandwidth

**Tasks:**
- [ ] Add size parameter to image downloads
- [ ] Implement format conversion (WebP)
- [ ] Add quality settings
- [ ] Implement lazy loading option
- [ ] Add image optimization metrics

---

## Scanner Improvements

### 4.1 Better Title Extraction ✅ COMPLETED
**Priority:** High | **Complexity:** Medium | **Impact:** High | **Implemented:** US-013 through US-017

**Description:**
Improve filename parsing to handle more formats and edge cases.

**Current Issues:**
- "The.Matrix.1999.1080p.BluRay.mkv" → "The Matrix p" (❌ includes "p")
- Multi-word years: "2001.A.Space.Odyssey" → wrong parsing
- Special characters not handled well

**Improvements:**
- Better regex patterns
- Remove quality tags (1080p, BluRay, x264, etc.)
- Handle common patterns (YIFY, RARBG, etc.)
- Support ISO dates (2024-01-01)
- Handle TV show episodes (skip them)

**Enhanced Patterns:**
```go
var patterns = []struct {
    regex *regexp.Regexp
    clean func(string) string
}{
    // Remove quality markers
    {regexp.MustCompile(`(?i)(1080p|720p|2160p|4k|BluRay|WEB-DL|x264|x265|HEVC)`), removeQuality},

    // Extract year
    {regexp.MustCompile(`\((\d{4})\)`), extractYear},
    {regexp.MustCompile(`\.(\d{4})\.`), extractYear},

    // Remove release groups
    {regexp.MustCompile(`\[.*?\]`), removeGroups},
}
```

**Test Cases:**
```go
// Should extract correctly
"The.Matrix.1999.1080p.BluRay.x264.mkv" → "The Matrix" (1999)
"Inception (2010) [IMAX].mkv" → "Inception" (2010)
"2001.A.Space.Odyssey.1968.mkv" → "2001: A Space Odyssey" (1968)
"The.Lord.of.the.Rings.2001.Extended.mkv" → "The Lord of the Rings" (2001)
```

**Tasks:**
- [x] Research common filename patterns
- [x] Implement pattern library
- [x] Add quality tag removal
- [x] Add release group removal
- [x] Create comprehensive test suite
- [x] Add `--test-parser` debug mode

---

### 4.2 Watch Folder / Auto-Scan ✅ COMPLETED
**Priority:** Medium | **Complexity:** Medium | **Impact:** High | **Implemented:** US-021, US-022, US-023

**Description:**
Automatically scan when new files are added to watched directories.

**Implementation:**
```go
import "github.com/fsnotify/fsnotify"

func watchDirectories(dirs []string) {
    watcher, _ := fsnotify.NewWatcher()

    for _, dir := range dirs {
        watcher.Add(dir)
    }

    for {
        select {
        case event := <-watcher.Events:
            if event.Op&fsnotify.Create == fsnotify.Create {
                if isVideoFile(event.Name) {
                    processFile(event.Name)
                }
            }
        }
    }
}
```

**Configuration:**
```yaml
scanner:
  watch_mode: true          # Enable file watching
  watch_debounce: 30        # Wait 30s before processing
  watch_recursive: true     # Watch subdirectories
```

**CLI:**
```bash
# Start in watch mode
./scanner --watch

# Watch mode with custom debounce
./scanner --watch --debounce 60
```

**Use Cases:**
- Automated media server
- Download folder monitoring
- Continuous scanning

**Tasks:**
- [x] Add fsnotify dependency
- [x] Implement file watching
- [x] Add debouncing (wait for file to finish copying)
- [x] Handle file moves/renames
- [x] Add `--watch` flag
- [ ] Create systemd service example

---

### 4.3 Duplicate Detection ✅ COMPLETED
**Priority:** Medium | **Complexity:** Medium | **Impact:** Medium | **Implemented:** US-024, US-025

**Description:**
Detect and report duplicate movies in library.

**Detection Methods:**
1. **Exact match:** Same TMDB ID
2. **File hash:** Same content (different encoding)
3. **Filename similarity:** Fuzzy matching

**Implementation:**
```go
type DuplicateDetector struct {
    seenTMDBIDs map[int][]string      // TMDB ID → file paths
    seenHashes  map[string][]string   // Hash → file paths
}

func (d *DuplicateDetector) Check(movie *Movie) []Duplicate {
    var duplicates []Duplicate

    // Check TMDB ID
    if paths, exists := d.seenTMDBIDs[movie.TMDBID]; exists {
        duplicates = append(duplicates, Duplicate{
            Type:     "exact",
            Original: paths[0],
            Duplicate: movie.FilePath,
        })
    }

    return duplicates
}
```

**Report:**
```
Duplicate Movies Found:

The Matrix (1999)
  ✓ /movies/The.Matrix.1999.1080p.mkv
  ✗ /movies/The.Matrix.1999.720p.mkv (duplicate)

Inception (2010)
  ✓ /movies/Inception/movie.mkv
  ✗ /downloads/Inception.2010.mkv (duplicate)

Total: 2 duplicates found
```

**Actions:**
```bash
# Report duplicates only
./scanner --find-duplicates

# Interactive cleanup
./scanner --find-duplicates --interactive

# Auto-delete lower quality
./scanner --find-duplicates --auto-cleanup
```

**Tasks:**
- [x] Implement duplicate detection
- [ ] Add file hashing
- [x] Create duplicate report
- [ ] Add interactive cleanup
- [x] Add quality comparison logic

---

### 4.4 Custom Scan Profiles
**Priority:** Low | **Complexity:** Low | **Impact:** Medium

**Description:**
Pre-configured scan profiles for different use cases.

**Profiles:**
```yaml
profiles:
  quick:
    use_nfo: true
    nfo_fallback_tmdb: false
    download_covers: false
    download_backdrops: false

  complete:
    use_nfo: true
    nfo_fallback_tmdb: true
    download_covers: true
    download_backdrops: true
    download_actors: true

  tmdb_only:
    use_nfo: false
    download_covers: true
    download_backdrops: true
```

**CLI:**
```bash
# Use profile
./scanner --profile quick

# List available profiles
./scanner --list-profiles

# Create custom profile
./scanner --save-profile my_profile
```

**Tasks:**
- [ ] Define default profiles
- [ ] Implement profile system
- [ ] Add profile CLI flags
- [ ] Allow custom profiles
- [ ] Document profile options

---

### 4.5 Smart Retry Logic ✅ COMPLETED
**Priority:** Medium | **Complexity:** Low | **Impact:** Medium | **Implemented:** US-028

**Description:**
Automatically retry failed operations with exponential backoff.

**Implementation:**
```go
func retryWithBackoff(operation func() error, maxRetries int) error {
    backoff := time.Second

    for i := 0; i < maxRetries; i++ {
        err := operation()
        if err == nil {
            return nil
        }

        if isRetryable(err) {
            time.Sleep(backoff)
            backoff *= 2
        } else {
            return err  // Don't retry non-retryable errors
        }
    }

    return fmt.Errorf("max retries exceeded")
}
```

**Retryable Errors:**
- Network timeouts
- TMDB rate limits (429)
- Temporary server errors (5xx)

**Non-Retryable:**
- Invalid API key (401)
- Not found (404)
- Malformed XML

**Configuration:**
```yaml
retry:
  max_attempts: 3
  initial_backoff: 1000     # 1 second
  max_backoff: 30000        # 30 seconds
  retryable_codes: [429, 500, 502, 503, 504]
```

**Tasks:**
- [x] Implement retry logic
- [x] Add exponential backoff
- [x] Classify retryable errors
- [ ] Add retry metrics
- [x] Log retry attempts

---

## Output & Templates

### 5.1 Custom MDX Templates
**Priority:** Medium | **Complexity:** Low | **Impact:** Medium

**Description:**
Allow users to customize MDX output format with templates.

**Current:** Hard-coded template in `mdx.go`

**Proposed:**
```yaml
output:
  template_file: "./templates/custom.mdx.tmpl"
```

**Template Example:**
```go
// templates/custom.mdx.tmpl
---
title: {{.Title}}
year: {{.ReleaseYear}}
rating: {{.Rating}}
{{if .IMDbID}}imdb: {{.IMDbID}}{{end}}
---

# {{.Title}} ({{.ReleaseYear}})

{{.Description}}

**Rating:** {{.Rating}}/10
```

**Use Cases:**
- Different frontmatter formats
- Custom sections
- Conditional fields
- Multi-language output

**Tasks:**
- [ ] Externalize current template
- [ ] Implement template loading
- [ ] Add template variables
- [ ] Create example templates
- [ ] Add template validation

---

### 5.2 Multiple Output Formats
**Priority:** Medium | **Complexity:** Medium | **Impact:** Medium

**Description:**
Support output formats beyond MDX: JSON, YAML, CSV, HTML.

**Formats:**

**JSON:**
```json
{
  "title": "The Matrix",
  "year": 1999,
  "rating": 8.7,
  "genres": ["Action", "Science Fiction"]
}
```

**CSV:**
```csv
title,year,rating,director,genres
The Matrix,1999,8.7,Wachowski Sisters,"Action, Sci-Fi"
```

**HTML:**
```html
<div class="movie">
  <h1>The Matrix (1999)</h1>
  <p>Rating: 8.7/10</p>
</div>
```

**Configuration:**
```yaml
output:
  formats:
    - mdx                  # Default
    - json                 # movies.json
    - csv                  # movies.csv
  json_output: "./data/movies.json"
  csv_output: "./data/movies.csv"
```

**Tasks:**
- [ ] Implement JSON writer
- [ ] Implement CSV writer
- [ ] Implement HTML writer
- [ ] Add format selection flag
- [ ] Support multiple simultaneous formats

---

### 5.3 Collection/Franchise Support
**Priority:** Medium | **Complexity:** Medium | **Impact:** Medium

**Description:**
Group movies by collection (e.g., "The Matrix Collection").

**TMDB API:**
```json
{
  "belongs_to_collection": {
    "id": 2344,
    "name": "The Matrix Collection",
    "poster_path": "/path.jpg",
    "backdrop_path": "/path.jpg"
  }
}
```

**MDX Output:**
```yaml
collection:
  id: 2344
  name: "The Matrix Collection"
  slug: "the-matrix-collection"
```

**Website Feature:**
- Collection detail pages
- "More in this series" section
- Collection browsing

**Tasks:**
- [ ] Fetch collection data from TMDB
- [ ] Add collection to Movie struct
- [ ] Generate collection MDX files
- [ ] Update website to show collections
- [ ] Add collection images

---

### 5.4 Rich Metadata Export
**Priority:** Low | **Complexity:** Low | **Impact:** Low

**Description:**
Export complete library metadata for backup/analysis.

**Formats:**
- SQLite database (queryable)
- Full JSON export
- Spreadsheet-friendly CSV

**CLI:**
```bash
# Export to SQLite
./scanner --export-db movies.db

# Export to JSON
./scanner --export-json movies.json

# Export to CSV with all fields
./scanner --export-csv movies.csv --all-fields
```

**Use Cases:**
- Library analysis
- Data backup
- Import to other tools
- Statistics generation

**Tasks:**
- [ ] Implement SQLite export
- [ ] Implement full JSON export
- [ ] Implement detailed CSV export
- [ ] Add compression option
- [ ] Add incremental export

---

### 5.5 Metadata Versioning
**Priority:** Low | **Complexity:** Medium | **Impact:** Low

**Description:**
Track metadata changes over time, allow rollback.

**Implementation:**
```yaml
# MDX file
metadata_version: 3
metadata_history:
  - version: 1
    date: "2025-01-01T00:00:00Z"
    source: "TMDB"
    rating: 8.5
  - version: 2
    date: "2025-06-01T00:00:00Z"
    source: "NFO"
    rating: 9.0
  - version: 3
    date: "2026-01-01T00:00:00Z"
    source: "NFO+TMDB"
    rating: 8.7
```

**CLI:**
```bash
# Show metadata history
./scanner --history the-matrix-1999.mdx

# Rollback to version 2
./scanner --rollback the-matrix-1999.mdx --version 2
```

**Tasks:**
- [ ] Add version tracking to MDX
- [ ] Store metadata snapshots
- [ ] Implement history viewing
- [ ] Add rollback feature
- [ ] Add diff visualization

---

## Database & Caching

### 6.1 SQLite Library Database
**Priority:** High | **Complexity:** Medium | **Impact:** High

**Description:**
Store all movie metadata in SQLite for fast queries and advanced features.

**Schema:**
```sql
CREATE TABLE movies (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL,
    slug TEXT UNIQUE,
    release_year INTEGER,
    rating REAL,
    tmdb_id INTEGER,
    imdb_id TEXT,
    file_path TEXT,
    scanned_at TIMESTAMP,
    metadata_source TEXT
);

CREATE TABLE genres (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE
);

CREATE TABLE movie_genres (
    movie_id INTEGER,
    genre_id INTEGER,
    FOREIGN KEY (movie_id) REFERENCES movies(id),
    FOREIGN KEY (genre_id) REFERENCES genres(id)
);

CREATE TABLE cast (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE movie_cast (
    movie_id INTEGER,
    cast_id INTEGER,
    character TEXT,
    "order" INTEGER,
    FOREIGN KEY (movie_id) REFERENCES movies(id),
    FOREIGN KEY (cast_id) REFERENCES cast(id)
);
```

**Benefits:**
- Fast full-text search
- Complex queries (find all 1999 action movies rated > 8)
- Statistics and analytics
- Deduplication
- Backup/restore

**Features:**
```bash
# Search database
./scanner query "SELECT * FROM movies WHERE rating > 8.5"

# Statistics
./scanner stats

# Find movies by actor
./scanner find-actor "Keanu Reeves"
```

**Tasks:**
- [ ] Design database schema
- [ ] Implement SQLite backend
- [ ] Add query interface
- [ ] Implement full-text search
- [ ] Add migration system
- [ ] Create backup/restore

---

### 6.2 Metadata Cache Database ✅ COMPLETED
**Priority:** Medium | **Complexity:** Medium | **Impact:** High | **Implemented:** US-026 + v1.3.0

**Description:**
Dedicated cache database for TMDB/IMDb responses.

**Schema:**
```sql
CREATE TABLE tmdb_cache (
    id INTEGER PRIMARY KEY,
    tmdb_id INTEGER UNIQUE,
    title TEXT,
    year INTEGER,
    response_json TEXT,
    cached_at TIMESTAMP,
    expires_at TIMESTAMP
);

CREATE INDEX idx_tmdb_title_year ON tmdb_cache(title, year);
```

**Features:**
- Persist cache across runs
- TTL-based expiration
- Queryable cache
- Cache statistics

**Benefits:**
- Survive restarts
- Share cache between machines
- Analyze cache efficiency
- Manual cache management

**Tasks:**
- [x] Create cache database schema
- [x] Implement cache storage
- [x] Add TTL management
- [x] Add cache pruning — lazy expiry on `Get()` + `--clear-cache` for full wipe
- [x] Add cache statistics

---

### 6.3 Full-Text Search
**Priority:** Medium | **Complexity:** Medium | **Impact:** Medium

**Description:**
Enable fast full-text search across all metadata.

**Implementation:**
```sql
-- SQLite FTS5
CREATE VIRTUAL TABLE movies_fts USING fts5(
    title,
    description,
    director,
    cast,
    genres,
    content=movies
);

-- Search query
SELECT * FROM movies_fts
WHERE movies_fts MATCH 'keanu reeves matrix'
ORDER BY rank;
```

**CLI:**
```bash
# Search all fields
./scanner search "keanu reeves"

# Search specific field
./scanner search --field title "matrix"

# Fuzzy search
./scanner search --fuzzy "matricx"
```

**Website Integration:**
- Search bar on website
- Auto-complete suggestions
- Faceted search (filter by genre, year, etc.)

**Tasks:**
- [ ] Implement FTS5 tables
- [ ] Add search indexing
- [ ] Create search CLI
- [ ] Add fuzzy matching
- [ ] Optimize search performance

---

### 6.4 Statistics & Analytics
**Priority:** Low | **Complexity:** Low | **Impact:** Low

**Description:**
Generate library statistics and insights.

**Statistics:**
```
Library Statistics
==================
Total Movies: 542
Total Runtime: 1,284 hours (53 days)
Average Rating: 7.8/10
Total File Size: 2.4 TB

By Decade:
  1980s: 23 movies
  1990s: 87 movies
  2000s: 156 movies
  2010s: 189 movies
  2020s: 87 movies

Top Genres:
  1. Action (145 movies)
  2. Drama (132 movies)
  3. Comedy (98 movies)

Top Directors:
  1. Christopher Nolan (12 movies)
  2. Steven Spielberg (11 movies)
  3. Quentin Tarantino (9 movies)

Metadata Sources:
  NFO: 423 movies (78%)
  TMDB: 119 movies (22%)
```

**Export Options:**
```bash
# Text report
./scanner stats

# JSON export
./scanner stats --json > stats.json

# HTML report
./scanner stats --html > report.html

# Chart generation
./scanner stats --charts ./charts/
```

**Tasks:**
- [ ] Implement statistics engine
- [ ] Add data aggregation
- [ ] Create text reporter
- [ ] Add chart generation
- [ ] Create HTML report template

---

### 6.5 Backup & Restore
**Priority:** Medium | **Complexity:** Low | **Impact:** Medium

**Description:**
Backup and restore all metadata, cache, and configuration.

**Backup Contents:**
- All MDX files
- SQLite databases
- Cache data
- Configuration
- Images (optional)

**Implementation:**
```bash
# Create backup
./scanner backup ./backups/backup-2026-01-27.tar.gz

# Restore from backup
./scanner restore ./backups/backup-2026-01-27.tar.gz

# List backups
./scanner backup --list

# Automated backups
./scanner backup --schedule daily
```

**Backup Format:**
```
backup-2026-01-27.tar.gz
├── metadata.json         # Library index
├── mdx/                  # All MDX files
├── db/                   # SQLite databases
├── cache/                # Cache data
├── config/               # Configuration
└── images/               # (optional) All images
```

**Tasks:**
- [ ] Implement backup creation
- [ ] Implement restore logic
- [ ] Add compression
- [ ] Add incremental backups
- [ ] Add scheduled backups

---

## Web Interface

### 7.1 Built-in Web UI
**Priority:** High | **Complexity:** High | **Impact:** High

**Description:**
Web interface for managing scanner, browsing library, editing metadata.

**Features:**

**Dashboard:**
- Library statistics
- Recent scans
- Failed scans
- System status

**Library Browser:**
- Grid/list view
- Filter by genre, year, rating
- Sort options
- Search

**Movie Editor:**
- Edit all metadata fields
- Upload custom images
- Edit NFO files
- Re-fetch from TMDB

**Scanner Control:**
- Trigger scans
- View progress
- Configure settings
- View logs

**Technology Stack:**
- Backend: Go HTTP server
- Frontend: React/Vue/Svelte
- Database: SQLite
- Real-time: WebSockets

**CLI:**
```bash
# Start web server
./scanner serve --port 8080

# Start with auth
./scanner serve --auth --password secret
```

**Tasks:**
- [ ] Design UI mockups
- [ ] Implement REST API
- [ ] Build frontend
- [ ] Add authentication
- [ ] Add WebSocket support
- [ ] Create Docker image

---

### 7.2 API Server
**Priority:** Medium | **Complexity:** Medium | **Impact:** Medium

**Description:**
RESTful API for external integrations.

**Endpoints:**
```
GET    /api/movies              # List all movies
GET    /api/movies/:slug        # Get movie details
POST   /api/movies              # Add movie
PUT    /api/movies/:slug        # Update movie
DELETE /api/movies/:slug        # Delete movie

GET    /api/scan                # Get scan status
POST   /api/scan                # Trigger scan
DELETE /api/scan                # Cancel scan

GET    /api/stats               # Library statistics
GET    /api/search?q=matrix     # Search movies
```

**Authentication:**
```yaml
api:
  enabled: true
  port: 8080
  auth:
    type: "token"              # token, oauth, none
    tokens:
      - "secret_token_123"
```

**Use Cases:**
- Mobile app integration
- Home automation (Home Assistant)
- Custom frontends
- Third-party tools

**Tasks:**
- [ ] Design API spec (OpenAPI)
- [ ] Implement REST endpoints
- [ ] Add authentication
- [ ] Add rate limiting
- [ ] Generate API docs
- [ ] Create client libraries

---

### 7.3 Real-time Updates
**Priority:** Low | **Complexity:** Medium | **Impact:** Low

**Description:**
WebSocket-based real-time scan progress and updates.

**Implementation:**
```javascript
// Frontend
const ws = new WebSocket('ws://localhost:8080/ws');

ws.on('message', (data) => {
  const event = JSON.parse(data);

  if (event.type === 'scan_progress') {
    updateProgress(event.current, event.total);
  }

  if (event.type === 'movie_added') {
    addMovieToUI(event.movie);
  }
});
```

**Events:**
- `scan_started`
- `scan_progress`
- `movie_added`
- `movie_updated`
- `scan_completed`
- `scan_error`

**Tasks:**
- [ ] Implement WebSocket server
- [ ] Add event broadcasting
- [ ] Create event types
- [ ] Add client reconnection
- [ ] Add event filtering

---

### 7.4 Mobile App
**Priority:** Low | **Complexity:** High | **Impact:** Low

**Description:**
Native or hybrid mobile app for library browsing.

**Features:**
- Browse library
- Search movies
- View details
- Trigger scans
- Edit metadata
- Watch trailers

**Technology Options:**
- React Native (cross-platform)
- Flutter (cross-platform)
- SwiftUI + Kotlin (native)

**Tasks:**
- [ ] Choose technology stack
- [ ] Design mobile UI
- [ ] Implement API client
- [ ] Add offline support
- [ ] Publish to app stores

---

### 7.5 Browser Extension
**Priority:** Low | **Complexity:** Low | **Impact:** Low

**Description:**
Browser extension to add movies from TMDB/IMDb pages.

**Features:**
- Detect TMDB/IMDb pages
- "Add to Library" button
- Quick add with metadata
- Custom notes

**Implementation:**
```javascript
// content script
if (window.location.hostname === 'www.themoviedb.org') {
  const tmdbId = extractTMDBId();

  addButton('Add to MovieVault', () => {
    fetch('http://localhost:8080/api/movies', {
      method: 'POST',
      body: JSON.stringify({ tmdb_id: tmdbId })
    });
  });
}
```

**Tasks:**
- [ ] Create manifest
- [ ] Implement content scripts
- [ ] Add UI injection
- [ ] Add API integration
- [ ] Publish to extension stores

---

## Library Management

### 8.1 Smart Playlists/Filters
**Priority:** Medium | **Complexity:** Medium | **Impact:** Medium

**Description:**
Dynamic collections based on rules (like iTunes smart playlists).

**Examples:**
```yaml
playlists:
  - name: "Hidden Gems"
    rules:
      - rating: ">= 8.0"
      - release_year: ">= 2010"
      - popularity: "< 100"  # Not well-known

  - name: "Rainy Day Movies"
    rules:
      - genres: ["Drama", "Romance"]
      - runtime: "> 120"

  - name: "90s Action"
    rules:
      - genres: "Action"
      - release_year: "1990-1999"
```

**UI:**
- Saved filters
- One-click access
- Auto-updating
- Share filters

**Tasks:**
- [ ] Design rule engine
- [ ] Implement filter parsing
- [ ] Add playlist storage
- [ ] Create playlist UI
- [ ] Add export/import

---

### 8.2 Watchlist Integration
**Priority:** Medium | **Complexity:** Medium | **Impact:** Medium

**Description:**
Track movies to watch, watched movies, favorites.

**Features:**
```yaml
# Movie metadata
watchlist:
  status: "to_watch"        # to_watch, watching, watched
  added: "2026-01-15"
  watched: "2026-01-20"
  favorite: true
  user_rating: 9.0
  notes: "Recommended by John"
```

**CLI:**
```bash
# Add to watchlist
./scanner watchlist add the-matrix-1999

# Mark as watched
./scanner watchlist watched the-matrix-1999

# List watchlist
./scanner watchlist list

# Statistics
./scanner watchlist stats
```

**Tasks:**
- [ ] Add watchlist fields to Movie
- [ ] Implement watchlist commands
- [ ] Add watchlist UI
- [ ] Export watchlist
- [ ] Sync with Trakt/IMDb

---

### 8.3 Custom Tags/Labels
**Priority:** Low | **Complexity:** Low | **Impact:** Medium

**Description:**
User-defined tags for organization.

**Examples:**
```yaml
tags:
  - "Kids Safe"
  - "Date Night"
  - "4K HDR"
  - "Director's Cut"
  - "Rewatchable"
  - "Criterion Collection"
```

**CLI:**
```bash
# Add tags
./scanner tag the-matrix-1999 "4K HDR" "Rewatchable"

# Search by tag
./scanner search --tag "Kids Safe"

# List all tags
./scanner tags list
```

**Use Cases:**
- Custom categorization
- Quality indicators
- Viewing context
- Collection membership

**Tasks:**
- [ ] Add tags to Movie struct
- [ ] Implement tag commands
- [ ] Add tag autocomplete
- [ ] Create tag cloud view
- [ ] Add tag filtering

---

### 8.4 Movie Recommendations
**Priority:** Low | **Complexity:** High | **Impact:** Medium

**Description:**
Suggest movies based on library and preferences.

**Algorithms:**

**1. Similarity-based:**
- Movies with similar genres
- Same director/actors
- Similar ratings

**2. Collaborative filtering:**
- What similar users liked
- Requires external data

**3. TMDB-based:**
- Use TMDB recommendations API
- "If you liked X, try Y"

**Implementation:**
```go
func GetRecommendations(movie *Movie, limit int) []*Movie {
    // Get TMDB recommendations
    tmdbRecs := tmdb.GetRecommendations(movie.TMDBID)

    // Filter by what's not in library
    filtered := filterNotInLibrary(tmdbRecs)

    // Sort by rating
    sort.Slice(filtered, func(i, j int) bool {
        return filtered[i].Rating > filtered[j].Rating
    })

    return filtered[:limit]
}
```

**UI:**
- "You might also like" section
- "Because you watched X" lists
- Discovery page

**Tasks:**
- [ ] Implement similarity algorithm
- [ ] Add TMDB recommendations
- [ ] Create recommendation UI
- [ ] Add user feedback
- [ ] Tune algorithms

---

### 8.5 Library Cleanup Tools
**Priority:** Medium | **Complexity:** Low | **Impact:** Medium

**Description:**
Tools to maintain library health.

**Features:**

**1. Orphan Detection:**
- MDX files without video files
- Images without MDX files

**2. Broken Links:**
- Missing cover images
- Invalid file paths

**3. Incomplete Metadata:**
- Missing required fields
- Low-quality data

**4. Quality Issues:**
- Duplicate movies
- Multiple versions (720p/1080p)

**CLI:**
```bash
# Find issues
./scanner cleanup --dry-run

# Fix automatically
./scanner cleanup --auto-fix

# Interactive mode
./scanner cleanup --interactive
```

**Report:**
```
Library Cleanup Report
======================

Orphaned MDX Files: 5
  - the-matrix-1999.mdx (video file deleted)
  - inception.mdx (video file moved)

Missing Images: 12
  - movie-1.jpg
  - movie-2.jpg

Incomplete Metadata: 8
  - missing-title.mdx (no title)
  - no-year.mdx (no release year)

Duplicates: 3 sets
  - The Matrix (2 copies)
  - Inception (2 copies)

Total Issues: 28
```

**Tasks:**
- [ ] Implement orphan detection
- [ ] Add broken link checking
- [ ] Implement auto-fix logic
- [ ] Create cleanup report
- [ ] Add interactive mode

---

## Media Analysis

### 9.1 Video File Analysis
**Priority:** Medium | **Complexity:** Medium | **Impact:** Medium

**Description:**
Extract technical details from video files using FFmpeg.

**Data to Extract:**
```yaml
video:
  codec: "H.264"
  resolution: "1920x1080"
  bitrate: "8000 kbps"
  framerate: "23.976 fps"
  duration: "136:23"
  hdr: true

audio:
  codec: "AAC"
  channels: "5.1"
  language: "en"

subtitles:
  - language: "en"
    forced: false
  - language: "es"
    forced: false
```

**Implementation:**
```go
import "github.com/vansante/go-ffprobe"

func AnalyzeVideo(path string) (*VideoInfo, error) {
    data, err := ffprobe.GetProbeData(path, 120*time.Second)
    if err != nil {
        return nil, err
    }

    return &VideoInfo{
        Codec:      data.Format.FormatName,
        Duration:   data.Format.Duration(),
        Resolution: fmt.Sprintf("%dx%d", data.FirstVideoStream().Width, data.FirstVideoStream().Height),
        // ...
    }, nil
}
```

**Benefits:**
- Show quality information
- Filter by resolution
- Identify HDR content
- Subtitle availability

**Tasks:**
- [ ] Add FFmpeg/FFprobe dependency
- [ ] Implement video analysis
- [ ] Add to Movie struct
- [ ] Display in MDX
- [ ] Add quality filters

---

### 9.2 Generate Video Thumbnails
**Priority:** Low | **Complexity:** Medium | **Impact:** Low

**Description:**
Generate thumbnail strips or preview images from video files.

**Thumbnail Types:**

**1. Single Frame:**
- Capture at 10% point
- Use for quick preview

**2. Thumbnail Strip:**
- 10 frames across movie
- Contact sheet style

**3. Animated Preview:**
- Short GIF of highlights
- For hover previews

**Implementation:**
```go
func GenerateThumbnail(videoPath string, timestamp time.Duration) error {
    cmd := exec.Command("ffmpeg",
        "-i", videoPath,
        "-ss", fmt.Sprintf("%f", timestamp.Seconds()),
        "-vframes", "1",
        "-vf", "scale=320:-1",
        outputPath,
    )
    return cmd.Run()
}
```

**Configuration:**
```yaml
thumbnails:
  enabled: true
  type: "strip"              # single, strip, animated
  count: 10                  # Frames for strip
  width: 320                 # Thumbnail width
  output: "./website/public/thumbnails"
```

**Tasks:**
- [ ] Implement thumbnail generation
- [ ] Add strip layout
- [ ] Add animated GIF support
- [ ] Optimize image sizes
- [ ] Add to website

---

### 9.3 Subtitle Management
**Priority:** Low | **Complexity:** Medium | **Impact:** Low

**Description:**
Detect, download, and manage subtitle files.

**Features:**

**1. Detection:**
- Find existing .srt files
- Parse subtitle languages
- Extract embedded subtitles

**2. Download:**
- OpenSubtitles API integration
- Auto-download missing subtitles
- Language preferences

**3. Management:**
- Rename to match video
- Convert formats (SRT, ASS, VTT)
- Fix encoding issues

**Implementation:**
```go
func FindSubtitles(videoPath string) ([]Subtitle, error) {
    dir := filepath.Dir(videoPath)
    base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))

    // Search for .srt files
    pattern := filepath.Join(dir, base+"*.srt")
    matches, _ := filepath.Glob(pattern)

    // Parse language from filename
    // movie.en.srt, movie.spa.srt, etc.
}
```

**Tasks:**
- [ ] Implement subtitle detection
- [ ] Add OpenSubtitles integration
- [ ] Implement auto-download
- [ ] Add format conversion
- [ ] Store subtitle info in MDX

---

### 9.4 Audio Track Detection
**Priority:** Low | **Complexity:** Low | **Impact:** Low

**Description:**
Detect and catalog audio tracks (languages, commentary, etc.).

**Data:**
```yaml
audio_tracks:
  - index: 0
    language: "en"
    channels: "5.1"
    codec: "DTS"
    title: "English DTS"
  - index: 1
    language: "en"
    channels: "2.0"
    codec: "AAC"
    title: "Director Commentary"
  - index: 2
    language: "es"
    channels: "5.1"
    codec: "AC3"
    title: "Spanish"
```

**Use Cases:**
- Show available languages
- Identify commentary tracks
- Find highest quality audio

**Tasks:**
- [ ] Extract audio track info
- [ ] Parse language codes
- [ ] Detect commentary tracks
- [ ] Add to Movie metadata
- [ ] Display in UI

---

### 9.5 Scene Detection & Chapters
**Priority:** Low | **Complexity:** High | **Impact:** Low

**Description:**
Detect scenes/chapters for better navigation.

**Methods:**

**1. Embedded Chapters:**
- Extract from MKV/MP4 metadata
- Use existing chapter markers

**2. Scene Detection:**
- FFmpeg scene detection
- ML-based scene segmentation
- Create chapter markers

**3. Chapter Generation:**
- Auto-create chapters every N minutes
- Use subtitle breaks
- Use audio analysis

**Implementation:**
```go
func ExtractChapters(videoPath string) ([]Chapter, error) {
    // Use ffprobe to get chapter data
    cmd := exec.Command("ffprobe",
        "-v", "quiet",
        "-print_format", "json",
        "-show_chapters",
        videoPath,
    )
    // Parse JSON output
}
```

**Tasks:**
- [ ] Extract embedded chapters
- [ ] Implement scene detection
- [ ] Generate chapter markers
- [ ] Store in metadata
- [ ] Add chapter navigation to player

---

## Integration & APIs

### 10.1 Plex Integration
**Priority:** Medium | **Complexity:** Medium | **Impact:** Medium

**Description:**
Sync with Plex Media Server for watch status and ratings.

**Features:**
- Import Plex library
- Sync watch history
- Import user ratings
- Export to Plex

**Implementation:**
```go
type PlexClient struct {
    baseURL string
    token   string
}

func (p *PlexClient) GetLibrary() ([]*PlexMovie, error) {
    resp, err := http.Get(p.baseURL + "/library/sections/1/all?X-Plex-Token=" + p.token)
    // Parse XML response
}
```

**Configuration:**
```yaml
plex:
  server: "http://localhost:32400"
  token: "your_plex_token"
  library_id: 1
  sync_watched: true
  sync_ratings: true
```

**Tasks:**
- [ ] Implement Plex API client
- [ ] Add library import
- [ ] Sync watch status
- [ ] Sync ratings
- [ ] Add bidirectional sync

---

### 10.2 Jellyfin Integration
**Priority:** High | **Complexity:** Medium | **Impact:** High

**Description:**
Deep integration with Jellyfin for two-way sync.

**Features:**
- Import Jellyfin library
- Sync .nfo updates
- Two-way watch status
- Trigger Jellyfin library refresh

**API:**
```go
type JellyfinClient struct {
    serverURL string
    apiKey    string
}

func (j *JellyfinClient) GetMovies() ([]*JellyfinMovie, error) {
    url := fmt.Sprintf("%s/Items?api_key=%s", j.serverURL, j.apiKey)
    // Make request
}

func (j *JellyfinClient) RefreshMetadata(itemId string) error {
    url := fmt.Sprintf("%s/Items/%s/Refresh?api_key=%s", j.serverURL, itemId, j.apiKey)
    // Trigger refresh
}
```

**Configuration:**
```yaml
jellyfin:
  server: "http://localhost:8096"
  api_key: "your_api_key"
  user_id: "user_id"
  sync_interval: 3600       # Seconds
```

**Tasks:**
- [ ] Implement Jellyfin API client
- [ ] Add library sync
- [ ] Implement watch status sync
- [ ] Add metadata refresh trigger
- [ ] Create sync daemon

---

### 10.3 Radarr/Sonarr Integration
**Priority:** Medium | **Complexity:** Medium | **Impact:** Medium

**Description:**
Integration with Radarr for automated downloads.

**Features:**
- Detect new movies added to Radarr
- Auto-scan when download completes
- Update Radarr with custom metadata
- Trigger missing movie downloads

**Webhooks:**
```go
func HandleRadarrWebhook(w http.ResponseWriter, r *http.Request) {
    var event RadarrEvent
    json.NewDecoder(r.Body).Decode(&event)

    if event.EventType == "Download" {
        // Scan the new file
        processFile(event.Movie.FolderPath)
    }
}
```

**Configuration:**
```yaml
radarr:
  url: "http://localhost:7878"
  api_key: "your_api_key"
  webhook_enabled: true
  auto_scan: true
```

**Tasks:**
- [ ] Implement Radarr API client
- [ ] Add webhook receiver
- [ ] Auto-scan on download
- [ ] Sync metadata
- [ ] Add Sonarr support (TV shows)

---

### 10.4 Home Assistant Integration
**Priority:** Low | **Complexity:** Medium | **Impact:** Low

**Description:**
Home Assistant integration for home automation.

**Features:**
- Sensor entities (library stats)
- Trigger scans via automation
- Notifications for new movies
- Media player integration

**MQTT Discovery:**
```yaml
# Home Assistant auto-discovery
homeassistant/sensor/filmscraper/config:
  name: "Film Library Count"
  state_topic: "filmscraper/library/count"
  unit_of_measurement: "movies"

homeassistant/sensor/filmscraper_rating/config:
  name: "Average Rating"
  state_topic: "filmscraper/library/avg_rating"
```

**Configuration:**
```yaml
homeassistant:
  mqtt_broker: "localhost:1883"
  discovery_prefix: "homeassistant"
  update_interval: 300
```

**Tasks:**
- [ ] Implement MQTT client
- [ ] Add Home Assistant discovery
- [ ] Publish sensor data
- [ ] Add automation triggers
- [ ] Create example automations

---

### 10.5 Discord/Slack Notifications
**Priority:** Low | **Complexity:** Low | **Impact:** Low

**Description:**
Send notifications to Discord/Slack for library updates.

**Features:**
- New movie added
- Scan completed
- Errors detected
- Daily/weekly summaries

**Discord Webhook:**
```go
func SendDiscordNotification(movie *Movie) error {
    webhook := "https://discord.com/api/webhooks/..."

    payload := DiscordWebhook{
        Embeds: []Embed{{
            Title:       fmt.Sprintf("New Movie: %s (%d)", movie.Title, movie.ReleaseYear),
            Description: movie.Description,
            Color:       3447003,
            Thumbnail:   Thumbnail{URL: movie.CoverImage},
            Fields: []Field{
                {Name: "Rating", Value: fmt.Sprintf("%.1f/10", movie.Rating)},
                {Name: "Genre", Value: strings.Join(movie.Genres, ", ")},
            },
        }},
    }

    // POST to webhook
}
```

**Configuration:**
```yaml
notifications:
  discord:
    enabled: true
    webhook: "https://discord.com/api/webhooks/..."
    events: ["movie_added", "scan_complete", "error"]

  slack:
    enabled: false
    webhook: "https://hooks.slack.com/..."
```

**Tasks:**
- [ ] Implement Discord webhooks
- [ ] Implement Slack webhooks
- [ ] Add event filtering
- [ ] Create rich embeds
- [ ] Add notification templates

---

## Quality of Life

### 11.1 Configuration Wizard
**Priority:** Medium | **Complexity:** Low | **Impact:** Medium

**Description:**
Interactive setup wizard for first-time users.

**CLI:**
```bash
./scanner init

Welcome to MovieVault!
========================

Let's set up your configuration.

[1/5] TMDB API Key
Enter your TMDB API key (get one from https://themoviedb.org):
> your_tmdb_api_key_here

[2/5] Movie Directories
Enter paths to scan (one per line, empty line to finish):
> /movies
> /downloads/movies
>

[3/5] Output Directory
Where should MDX files be saved?
> ./website/src/content/movies

[4/5] Features
Enable NFO parsing? (Y/n): Y
Download cover images? (Y/n): Y
Download backdrops? (Y/n): n

[5/5] Review & Save
---
tmdb:
  api_key: 0ce53...
scanner:
  directories:
    - /movies
    - /downloads/movies
output:
  mdx_dir: ./website/src/content/movies
---

Save configuration? (Y/n): Y

✓ Configuration saved to ./config/config.yaml
✓ Created output directories

Run './scanner' to start scanning!
```

**Tasks:**
- [ ] Implement interactive prompts
- [ ] Add validation
- [ ] Create config from wizard
- [ ] Add directory creation
- [ ] Test API key

---

### 11.2 Better Logging ⚡ IN PROGRESS
**Priority:** Medium | **Complexity:** Low | **Impact:** Medium | **Partially Implemented:** US-027 (structured verbose logging)

**Description:**
Structured logging with levels, rotation, and formatting.

**Logging Library:**
```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction()
defer logger.Sync()

logger.Info("Processing movie",
    zap.String("title", movie.Title),
    zap.Int("year", movie.ReleaseYear),
    zap.String("source", "NFO"),
)
```

**Configuration:**
```yaml
logging:
  level: "info"             # debug, info, warn, error
  format: "json"            # json, text
  output: "stdout"          # stdout, file
  file: "./logs/scanner.log"
  rotate:
    max_size: 100           # MB
    max_backups: 5
    max_age: 30             # Days
```

**Log Levels:**
- DEBUG: Detailed diagnostics
- INFO: General information
- WARN: Warning messages
- ERROR: Error messages

**Tasks:**
- [ ] Add zap logging library
- [ ] Implement log levels
- [ ] Add structured logging
- [ ] Implement log rotation
- [ ] Add JSON output option

---

### 11.3 Progress Bar & Better UI
**Priority:** Medium | **Complexity:** Low | **Impact:** High

**Description:**
Better CLI UI with progress bars and status updates.

**Using:** `github.com/schollz/progressbar/v3`

**Implementation:**
```go
bar := progressbar.NewOptions(len(files),
    progressbar.OptionSetDescription("Scanning movies"),
    progressbar.OptionSetWidth(50),
    progressbar.OptionThrottle(65*time.Millisecond),
    progressbar.OptionShowCount(),
    progressbar.OptionShowIts(),
    progressbar.OptionSetRenderBlankState(true),
)

for _, file := range files {
    processMovie(file)
    bar.Add(1)
}
```

**Output:**
```
Scanning movies [████████████████████----] 80% (40/50) 2.5 movies/sec

Currently processing: The Matrix (1999)
  ✓ Found NFO file
  ✓ Parsed metadata
  ⏳ Downloading cover image...
```

**Tasks:**
- [ ] Add progressbar library
- [ ] Implement progress tracking
- [ ] Add spinner for operations
- [ ] Improve status messages
- [ ] Add color output

---

### 11.4 Dry-Run Improvements
**Priority:** Low | **Complexity:** Low | **Impact:** Low

**Description:**
Enhanced dry-run mode with more details and simulation.

**Features:**
- Show exactly what would change
- Simulate API calls
- Preview generated MDX
- Estimate time/API usage

**Output:**
```bash
./scanner --dry-run --detailed

DRY RUN - No changes will be made
===================================

[1/50] The Matrix (1999).mkv
  Would scan: YES
  Metadata source: NFO
  Actions:
    ✓ Parse NFO file (0ms)
    ✓ Skip TMDB API call (saved 500ms)
    ✓ Download cover from TMDB (would take ~300ms)
    ✓ Write MDX file: the-matrix-1999.mdx

  Generated MDX preview:
  ---
  title: The Matrix
  rating: 8.7
  ...
  ---

Summary:
  Total files: 50
  Would process: 35 (15 already exist)
  NFO sources: 30
  TMDB sources: 5
  API calls saved: 30
  Estimated time: 2.5 minutes
  Estimated API usage: 5/1000 daily limit
```

**Tasks:**
- [ ] Add detailed dry-run output
- [ ] Add MDX preview
- [ ] Calculate estimates
- [ ] Show file operations
- [ ] Add confirmation prompt

---

### 11.5 Update Checker
**Priority:** Low | **Complexity:** Low | **Impact:** Low

**Description:**
Check for new versions and notify users.

**Implementation:**
```go
const VERSION = "1.1.0"

func CheckForUpdates() (*UpdateInfo, error) {
    resp, err := http.Get("https://api.github.com/repos/marco/filmScraper/releases/latest")
    var release GitHubRelease
    json.NewDecoder(resp.Body).Decode(&release)

    if release.TagName > VERSION {
        return &UpdateInfo{
            Current: VERSION,
            Latest:  release.TagName,
            URL:     release.HTMLURL,
        }, nil
    }

    return nil, nil
}
```

**Output:**
```
MovieVault v1.1.0

⚠ A new version is available: v1.2.0
Release notes: https://github.com/marco/filmScraper/releases/tag/v1.2.0

Run 'scanner update' to upgrade.
```

**Configuration:**
```yaml
updates:
  check_on_start: true
  auto_update: false
  channel: "stable"         # stable, beta, dev
```

**Tasks:**
- [ ] Implement version checking
- [ ] Add GitHub API integration
- [ ] Show release notes
- [ ] Add auto-update (optional)
- [ ] Add update channels

---

## Advanced Features

### 12.1 Machine Learning Enhancements
**Priority:** Low | **Complexity:** Very High | **Impact:** Medium

**Description:**
Use ML for better title extraction, genre prediction, and recommendations.

**Use Cases:**

**1. Title Extraction:**
- Train model on filename → title mappings
- Better than regex for edge cases

**2. Genre Prediction:**
- Predict genres from plot text
- Correct missing/wrong genres

**3. Content-Based Recommendations:**
- Analyze plot similarity
- Suggest similar unwatched movies

**4. Quality Assessment:**
- Predict movie quality from metadata
- Flag low-quality releases

**Tasks:**
- [ ] Research ML models
- [ ] Gather training data
- [ ] Train genre classifier
- [ ] Implement inference
- [ ] Add model updates

---

### 12.2 Multi-User Support
**Priority:** Low | **Complexity:** High | **Impact:** Medium

**Description:**
Support multiple users with separate watchlists, ratings, preferences.

**Schema:**
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    username TEXT UNIQUE,
    password_hash TEXT,
    created_at TIMESTAMP
);

CREATE TABLE user_movie_data (
    user_id INTEGER,
    movie_id INTEGER,
    watched BOOLEAN,
    watch_count INTEGER,
    rating REAL,
    favorite BOOLEAN,
    notes TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (movie_id) REFERENCES movies(id)
);
```

**Features:**
- User accounts
- Personal watchlists
- Individual ratings
- Watch history per user
- Personalized recommendations

**Tasks:**
- [ ] Implement user system
- [ ] Add authentication
- [ ] Create user profiles
- [ ] Add per-user data
- [ ] Build user switching UI

---

### 12.3 Cloud Sync
**Priority:** Low | **Complexity:** High | **Impact:** Medium

**Description:**
Sync library metadata across multiple devices.

**Backends:**
- Custom cloud service
- Google Drive / Dropbox
- Self-hosted sync (Syncthing)

**What to Sync:**
- Library database
- User data (watchlist, ratings)
- Configuration
- Cache (optional)

**Conflict Resolution:**
- Last-write-wins
- Merge strategies
- Manual resolution

**Tasks:**
- [ ] Design sync protocol
- [ ] Implement cloud backend
- [ ] Add conflict resolution
- [ ] Create sync UI
- [ ] Test multi-device

---

### 12.4 Plugin System
**Priority:** Low | **Complexity:** Very High | **Impact:** High

**Description:**
Extensible plugin architecture for custom functionality.

**Plugin Types:**

**1. Metadata Providers:**
- Custom sources
- Private APIs
- Local databases

**2. Output Formats:**
- Custom templates
- Different file formats
- External systems

**3. Processors:**
- Custom title extraction
- Metadata enrichment
- Post-processing

**4. UI Extensions:**
- Custom pages
- Additional widgets
- Themes

**Architecture:**
```go
type Plugin interface {
    Name() string
    Version() string
    Init(config map[string]interface{}) error
    Execute(ctx *Context) error
}

type PluginManager struct {
    plugins map[string]Plugin
}

func (pm *PluginManager) Load(path string) error {
    // Load Go plugin (.so file)
    // Or support scripting languages (Lua, Python)
}
```

**Tasks:**
- [ ] Design plugin API
- [ ] Implement plugin loader
- [ ] Create example plugins
- [ ] Add plugin marketplace
- [ ] Document plugin development

---

### 12.5 Offline Mode
**Priority:** Low | **Complexity:** Medium | **Impact:** Low

**Description:**
Full functionality without internet connection.

**Features:**
- Use cached metadata only
- Process NFO files
- Skip downloads
- Queue API calls for later

**Implementation:**
```go
type OfflineMode struct {
    cache       *MetadataCache
    queue       *RequestQueue
    isOnline    bool
}

func (o *OfflineMode) GetMovie(title string, year int) (*Movie, error) {
    // Try cache first
    if movie := o.cache.Get(title, year); movie != nil {
        return movie, nil
    }

    if !o.isOnline {
        // Queue for later
        o.queue.Add(title, year)
        return nil, ErrOffline
    }

    // Make API call
    return tmdb.Get(title, year)
}
```

**Configuration:**
```yaml
offline:
  enabled: false
  auto_detect: true         # Auto-detect connectivity
  queue_requests: true      # Queue failed requests
  retry_on_connect: true    # Process queue when online
```

**Tasks:**
- [ ] Implement offline detection
- [ ] Add request queueing
- [ ] Implement queue processing
- [ ] Add offline indicator
- [ ] Test offline scenarios

---

## Priority Matrix

### Completed

| Feature | User Story | Version |
|---------|-----------|---------|
| Better Title Extraction | US-013 – US-017 | 1.3.0 |
| NFO Image Download | US-018 – US-020 | 1.3.0 |
| Watch Folder / Auto-Scan | US-021 – US-023 | 1.3.0 |
| Duplicate Detection | US-024 – US-025 | 1.3.0 |
| Metadata Cache + Statistics | US-026 | 1.3.0 |
| Structured Verbose Logging | US-027 | 1.3.0 |
| Configuration Validation | US-028 | 1.3.0 |
| Smart Retry Logic | — | 1.3.0 |
| Local Metadata Cache (SQLite) | — | 1.3.0 |
| Direct TMDB Lookup via NFO ID | — | 1.3.0 |
| NFO `<art>` block parsing + local image paths | — | 1.3.2 |

### High Priority (Implement Soon)

| Feature | Complexity | Impact | Justification |
|---------|-----------|--------|---------------|
| Concurrent Scanning | Medium | High | 5x performance improvement |
| Incremental Scanning | Medium | High | Essential for large libraries |
| SQLite Library Database | Medium | High | Enables advanced features |
| Built-in Web UI | High | High | Major usability improvement |
| Jellyfin Integration | Medium | High | Target audience wants this |

### Medium Priority (Next Quarter)

| Feature | Complexity | Impact |
|---------|-----------|--------|
| IMDb Integration | Medium | High |
| Custom MDX Templates | Low | Medium |
| Collection/Franchise Support | Medium | Medium |
| API Server | Medium | Medium |
| Progress Bar & Better UI | Low | High |
| Configuration Wizard | Low | Medium |

### Low Priority (Future)

- All "Low" priority items from sections above
- Experimental features
- Nice-to-have improvements
- Community requests

---

## Contributing

Have ideas for new features? Please:

1. Check if it's already in this roadmap
2. Open an issue on GitHub
3. Describe your use case
4. Propose implementation approach

Pull requests welcome!

---

## Changelog

**2026-02-06:** Updated to v1.3.2. Full audit of roadmap vs codebase. Marked as completed: Direct TMDB Lookup via NFO ID (§1.2), Local Metadata Cache (§2.3, all tasks including `--clear-cache`), Metadata Cache Database pruning (§6.2). Added v1.3.2 NFO improvements (art block parsing, local image path support) to §1.1 tasks. Refreshed priority matrix.

**2026-02-03:** Marked 8 features complete (US-013–US-028): title extraction, NFO image downloads, watch mode, duplicate detection with quality comparison, SQLite cache with statistics, structured verbose logging, retry logic, and configuration validation. Updated priority matrix.

**2026-01-27:** Initial roadmap created after NFO support implementation

---

*This roadmap is a living document and will be updated as the project evolves.*
