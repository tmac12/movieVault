# Technical Documentation - MovieVault

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Go Scanner Components](#go-scanner-components)
3. [Astro Website Components](#astro-website-components)
4. [Data Flow](#data-flow)
5. [File Formats](#file-formats)
6. [API Integration](#api-integration)
7. [Docker Architecture](#docker-architecture)
8. [Performance Optimizations](#performance-optimizations)
9. [Extension Guide](#extension-guide)

---

## Architecture Overview

MovieVault uses a two-part architecture:

1. **Scanner (Go)**: Discovers files, fetches metadata, generates MDX
2. **Website (Astro)**: Consumes MDX, builds static HTML

```
┌─────────────────┐
│  Movie Files    │
│  (Video Files)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐      ┌──────────────┐
│   Go Scanner    │─────▶│  NFO Files   │ (priority)
│   - File Walk   │      └──────────────┘
│   - Parse Names │      ┌──────────────┐
│   - NFO Parse   │─────▶│  TMDB API    │ (fallback)
│   - Fetch Meta  │      └──────────────┘
│   - Watch Mode  │      ┌──────────────┐
└────────┬────────┘      │ SQLite Cache │
         │               └──────────────┘
         │
         ▼
┌─────────────────┐
│   MDX Files     │
│   (Content)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Astro Builder  │
│  - Read MDX     │
│  - Generate HTML│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Static Website │
│  (HTML/CSS/JS)  │
└─────────────────┘
```

### Design Principles

1. **Separation of Concerns**: Scanner and website are independent
2. **Static Generation**: No runtime database or server-side processing
3. **File-based Storage**: MDX files as single source of truth
4. **Idempotent Operations**: Running scanner multiple times produces same result
5. **Progressive Enhancement**: Website works without JavaScript

---

## Go Scanner Components

### 1. Configuration System (`internal/config/`)

#### Purpose
Load and validate configuration from YAML files with environment variable support.

#### Key File: `config.go`

```go
type Config struct {
    TMDB    TMDBConfig    // API credentials
    Scanner ScannerConfig // Scan settings
    Output  OutputConfig  // Output paths
    Options OptionsConfig // Misc options
}
```

#### Features

**Environment Variable Expansion**
```yaml
tmdb:
  api_key: "${TMDB_API_KEY}"  # Reads from environment
```

The `os.ExpandEnv()` function replaces `${VAR}` placeholders before parsing.

**Validation**
- Checks API key is not default placeholder
- Ensures at least one scan directory exists
- Validates output directories are set
- Auto-creates output directories if missing

**Path Handling**
- Expands `~` to home directory
- Converts relative paths to absolute
- Creates parent directories as needed

#### Usage Example

```go
cfg, err := config.Load("./config/config.yaml")
if err != nil {
    log.Fatal(err)
}
// cfg.TMDB.APIKey is now available
```

---

### 2. File Scanner (`internal/scanner/`)

#### Purpose
Recursively scan directories for video files and extract movie information from filenames.

#### Key File: `scanner.go`

**Core Structure**
```go
type Scanner struct {
    extensions []string  // Allowed file extensions
    mdxDir     string   // Where MDX files are stored
}

type FileInfo struct {
    Path       string  // Full path to video file
    FileName   string  // Original filename
    Title      string  // Extracted title
    Year       int     // Extracted year
    Size       int64   // File size in bytes
    Slug       string  // URL-friendly identifier
    ShouldScan bool    // Whether to process this file
}
```

**Smart Scanning Algorithm**

```go
func (s *Scanner) ScanDirectory(path string) ([]FileInfo, error) {
    var files []FileInfo

    // Walk directory tree
    filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
        // Skip directories
        if info.IsDir() {
            return nil
        }

        // Check file extension
        if !s.IsMediaFile(info.Name()) {
            return nil
        }

        // Extract title and year from filename
        title, year := ExtractTitleAndYear(info.Name())

        // Generate slug for MDX filename
        slug := GenerateSlug(title, year)

        // Check if MDX already exists (SMART SCANNING)
        shouldScan := !s.MDXExists(slug)

        files = append(files, FileInfo{
            Path:       p,
            FileName:   info.Name(),
            Title:      title,
            Year:       year,
            Size:       info.Size(),
            Slug:       slug,
            ShouldScan: shouldScan,
        })

        return nil
    })

    return files, nil
}
```

**Key Optimization: ShouldScan Flag**

The `ShouldScan` boolean is the key to performance:
- Checks if `{mdxDir}/{slug}.mdx` exists
- If exists → skip (no TMDB API call needed)
- If missing → process (fetch from TMDB)

This makes subsequent scans 100x faster.

---

### 3. Filename Parser (`internal/scanner/patterns.go`)

#### Purpose
Extract clean movie titles and years from messy filenames.

#### Patterns Removed

```go
var (
    // Year: (2023), [2023], 2023
    yearPattern = regexp.MustCompile(`[\[\(]?(\d{4})[\]\)]?`)

    // Quality: 1080p, 720p, 4K, BluRay, WEB-DL
    qualityPattern = regexp.MustCompile(
        `(?i)(1080p|720p|480p|2160p|4K|BluRay|BDRip|WEB-DL|WEBRip|HDRip|DVDRip|HDTV)`
    )

    // Codec: x264, x265, H.264, HEVC
    codecPattern = regexp.MustCompile(
        `(?i)(x264|x265|H\.?264|H\.?265|HEVC|XviD|DivX)`
    )

    // Audio: AAC, AC3, DTS
    audioPattern = regexp.MustCompile(
        `(?i)(AAC|AC3|DTS|DD5\.1|TrueHD|Atmos)`
    )

    // Release group: -GROUP at end
    releaseGroupPattern = regexp.MustCompile(`-[A-Z0-9]+$`)

    // Brackets: [anything]
    bracketPattern = regexp.MustCompile(`\[([^\]]+)\]`)

    // Extra info: EXTENDED, DIRECTOR'S CUT, etc.
    extraInfoPattern = regexp.MustCompile(
        `(?i)(EXTENDED|UNRATED|DIRECTOR.?S.?CUT|REMASTERED|THEATRICAL)`
    )
)
```

#### Algorithm

```go
func ExtractTitleAndYear(filename string) (title string, year int) {
    // 1. Remove file extension
    name := strings.TrimSuffix(filename, filepath.Ext(filename))

    // 2. Extract year (before removing it)
    yearMatches := yearPattern.FindStringSubmatch(name)
    if len(yearMatches) > 1 {
        year, _ = strconv.Atoi(yearMatches[1])
    }

    // 3. Remove all patterns (year, quality, codec, etc.)
    name = yearPattern.ReplaceAllString(name, "")
    name = qualityPattern.ReplaceAllString(name, "")
    name = codecPattern.ReplaceAllString(name, "")
    name = audioPattern.ReplaceAllString(name, "")
    name = extraInfoPattern.ReplaceAllString(name, "")
    name = bracketPattern.ReplaceAllString(name, "")
    name = releaseGroupPattern.ReplaceAllString(name, "")

    // 4. Replace dots/underscores with spaces
    name = strings.ReplaceAll(name, ".", " ")
    name = strings.ReplaceAll(name, "_", " ")

    // 5. Remove multiple spaces
    name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")

    // 6. Trim whitespace
    title = strings.TrimSpace(name)

    return title, year
}
```

#### Examples

| Input Filename | Extracted Title | Year |
|----------------|-----------------|------|
| `The.Matrix.1999.1080p.BluRay.x264-GROUP.mkv` | The Matrix | 1999 |
| `Inception (2010) [1080p].mp4` | Inception | 2010 |
| `Interstellar.2014.2160p.4K.HEVC.mkv` | Interstellar | 2014 |
| `The_Godfather_1972_REMASTERED.avi` | The Godfather | 1972 |

#### Slug Generation

```go
func GenerateSlug(title string, year int) string {
    slug := strings.ToLower(title)                    // lowercase
    slug = strings.ReplaceAll(slug, " ", "-")         // spaces to hyphens
    slug = regexp.MustCompile(`[^a-z0-9-]+`).        // remove non-alphanumeric
           ReplaceAllString(slug, "")
    slug = regexp.MustCompile(`-+`).                 // collapse multiple hyphens
           ReplaceAllString(slug, "-")
    slug = strings.Trim(slug, "-")                   // trim edges

    if year > 0 {
        slug = slug + "-" + strconv.Itoa(year)       // append year
    }

    return slug
}
```

**Example**: "The Matrix" (1999) → `the-matrix-1999`

---

### 4. TMDB API Client (`internal/metadata/`)

#### Purpose
Fetch movie metadata from The Movie Database API.

#### Key File: `tmdb.go`

**Client Structure**
```go
type Client struct {
    apiKey     string
    httpClient *http.Client
    rateDelay  time.Duration  // Delay between requests
}
```

**Three-Step Data Fetching**

```go
func (c *Client) GetFullMovieData(title string, year int) (*writer.Movie, error) {
    // Step 1: Search for movie
    searchResult, err := c.SearchMovie(title, year)

    // Step 2: Get detailed information
    details, err := c.GetMovieDetails(searchResult.ID)

    // Step 3: Get cast and crew
    credits, err := c.GetMovieCredits(searchResult.ID)

    // Combine into Movie struct
    movie := buildMovieStruct(details, credits)

    return movie, nil
}
```

#### API Endpoints

**Search Movies**
```
GET https://api.themoviedb.org/3/search/movie
Parameters:
  - api_key: Your API key
  - query: Movie title
  - year: Release year (optional)
  - language: en-US
```

**Get Movie Details**
```
GET https://api.themoviedb.org/3/movie/{movie_id}
Parameters:
  - api_key: Your API key
  - language: en-US
```

**Get Movie Credits**
```
GET https://api.themoviedb.org/3/movie/{movie_id}/credits
Parameters:
  - api_key: Your API key
  - language: en-US
```

#### Rate Limiting

```go
func (c *Client) SearchMovie(...) (*TMDBMovie, error) {
    // Make API request
    resp, err := c.httpClient.Get(searchURL)

    // ... process response ...

    // Rate limiting - wait before next request
    time.Sleep(c.rateDelay)  // Default: 250ms

    return result, nil
}
```

**Why 250ms?**
- TMDB allows 40 requests per 10 seconds
- 250ms = 4 requests per second = well under limit
- Provides safety margin

#### Image Downloads

```go
func (c *Client) DownloadImage(imagePath string, outputPath string, imageType string) error {
    // Build URL
    size := posterSize    // "w500"
    if imageType == "backdrop" {
        size = backdropSize  // "w1280"
    }

    imageURL := fmt.Sprintf("%s/%s%s", tmdbImageBaseURL, size, imagePath)

    // Download
    resp, err := c.httpClient.Get(imageURL)

    // Save to file
    outFile, err := os.Create(outputPath)
    io.Copy(outFile, resp.Body)

    // Rate limit
    time.Sleep(c.rateDelay)

    return nil
}
```

**Image Sizes**
- **Posters**: w500 (500px wide) - good balance of quality and file size
- **Backdrops**: w1280 (1280px wide) - high quality for hero images

#### Error Handling

```go
if resp.StatusCode != http.StatusOK {
    body, _ := io.ReadAll(resp.Body)
    return nil, fmt.Errorf("TMDB API error (status %d): %s",
                          resp.StatusCode, string(body))
}
```

Common errors:
- `401`: Invalid API key
- `404`: Movie not found
- `429`: Rate limit exceeded (shouldn't happen with delays)

---

### 5. MDX Writer (`internal/writer/`)

#### Purpose
Generate MDX files with YAML frontmatter from movie data.

#### Key File: `mdx.go`

**Structure**
```go
type MDXWriter struct {
    mdxDir     string  // Where to write MDX files
    coversDir  string  // Where covers are stored
}
```

**MDX Generation Process**

```go
func (w *MDXWriter) GenerateMDX(movie *Movie) (string, error) {
    var sb strings.Builder

    // 1. Write frontmatter delimiter
    sb.WriteString("---\n")

    // 2. Marshal Movie struct to YAML
    yamlData, err := yaml.Marshal(movie)
    sb.Write(yamlData)

    sb.WriteString("---\n\n")

    // 3. Write markdown content
    sb.WriteString(fmt.Sprintf("# %s (%d)\n\n", movie.Title, movie.ReleaseYear))

    // Synopsis section
    sb.WriteString("## Synopsis\n\n")
    sb.WriteString(movie.Description + "\n\n")

    // Details section
    sb.WriteString("## Details\n\n")
    sb.WriteString(fmt.Sprintf("- **Rating**: %.1f/10\n", movie.Rating))
    sb.WriteString(fmt.Sprintf("- **Runtime**: %d minutes\n", movie.Runtime))
    sb.WriteString(fmt.Sprintf("- **Director**: %s\n", movie.Director))
    sb.WriteString(fmt.Sprintf("- **Genres**: %s\n",
                              strings.Join(movie.Genres, ", ")))

    // File information section
    sb.WriteString("## File Information\n\n")
    sb.WriteString(fmt.Sprintf("- **Location**: `%s`\n", movie.FilePath))
    sb.WriteString(fmt.Sprintf("- **Size**: %s\n", formatFileSize(movie.FileSize)))

    return sb.String(), nil
}
```

#### Output Example

```mdx
---
title: "The Matrix"
slug: "the-matrix-1999"
description: "Set in the 22nd century..."
coverImage: "/covers/the-matrix-1999.jpg"
backdropImage: "/covers/the-matrix-1999-backdrop.jpg"
filePath: "/home/marco/Movies/The.Matrix.1999.1080p.mkv"
fileName: "The.Matrix.1999.1080p.mkv"
rating: 8.7
releaseYear: 1999
releaseDate: "1999-03-31"
runtime: 136
genres:
  - Action
  - Science Fiction
director: "Lana Wachowski, Lilly Wachowski"
cast:
  - Keanu Reeves
  - Laurence Fishburne
  - Carrie-Anne Moss
tmdbId: 603
imdbId: "tt0133093"
scannedAt: 2026-01-27T10:30:00Z
fileSize: 2147483648
---

# The Matrix (1999)

## Synopsis

Set in the 22nd century, The Matrix tells the story of a computer hacker...

## Details

- **Rating**: 8.7/10
- **Runtime**: 136 minutes
- **Director**: Lana Wachowski, Lilly Wachowski
- **Genres**: Action, Science Fiction
- **Cast**: Keanu Reeves, Laurence Fishburne, Carrie-Anne Moss

## File Information

- **Location**: `/home/marco/Movies/The.Matrix.1999.1080p.mkv`
- **Size**: 2.00 GB
- **Last Scanned**: January 27, 2026

## Links

- [View on TMDB](https://www.themoviedb.org/movie/603)
- [View on IMDb](https://www.imdb.com/title/tt0133093)
```

#### File Size Formatting

```go
func formatFileSize(bytes int64) string {
    const (
        KB = 1024
        MB = KB * 1024
        GB = MB * 1024
        TB = GB * 1024
    )

    switch {
    case bytes >= TB:
        return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
    case bytes >= GB:
        return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
    case bytes >= MB:
        return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
    default:
        return fmt.Sprintf("%d bytes", bytes)
    }
}
```

---

### 6. CLI Application (`cmd/scanner/main.go`)

#### Purpose
Command-line interface for running the scanner.

#### Command-Line Flags

```go
var (
    configPath     = flag.String("config", "./config/config.yaml",
                                "Path to configuration file")
    forceRefresh   = flag.Bool("force-refresh", false,
                              "Re-fetch all metadata from TMDB")
    noBuild        = flag.Bool("no-build", false,
                              "Skip Astro build step")
    dryRun         = flag.Bool("dry-run", false,
                              "Show what would be done")
    verbose        = flag.Bool("verbose", false,
                              "Show detailed logging")
    watch          = flag.Bool("watch", false,
                              "Watch directories for new files continuously")
    testParser     = flag.Bool("test-parser", false,
                              "Test title extraction on given filenames")
    findDuplicates = flag.Bool("find-duplicates", false,
                              "Scan MDX files for duplicate movies")
    detailed       = flag.Bool("detailed", false,
                              "Show quality breakdown (used with --find-duplicates)")
    cacheStats     = flag.Bool("cache-stats", false,
                              "Display cache hit/miss statistics and exit")
)
```

#### Main Workflow

```go
func main() {
    // 1. Parse flags
    flag.Parse()

    // 2. Load configuration
    cfg, err := config.Load(*configPath)

    // 3. Create scanner
    scanner := scanner.New(cfg.Scanner.Extensions, cfg.Output.MDXDir)

    // 4. Scan all directories
    files, err := scanner.ScanAll(cfg.Scanner.Directories)

    // 5. Filter files (skip existing unless --force-refresh)
    filesToProcess := filterFiles(files, *forceRefresh)

    // 6. Process each file
    for _, file := range filesToProcess {
        // Fetch metadata from TMDB
        movie, err := tmdbClient.GetFullMovieData(file.Title, file.Year)

        // Download images
        downloadImages(movie, cfg)

        // Write MDX file
        mdxWriter.WriteMDXFile(movie)
    }

    // 7. Build Astro site (if enabled)
    if cfg.Output.AutoBuild && !*noBuild {
        buildAstroSite()
    }

    // 8. Print summary
    printSummary(files, successCount, errorCount)
}
```

#### Progress Reporting

```go
for i, file := range filesToProcess {
    fmt.Printf("\n[%d/%d] Processing: %s\n", i+1, len(filesToProcess), file.FileName)

    if *verbose {
        fmt.Printf("  Extracted title: %s\n", file.Title)
        if file.Year > 0 {
            fmt.Printf("  Extracted year: %d\n", file.Year)
        }
    }

    // ... process file ...

    fmt.Printf("  ✓ Created: %s.mdx\n", movie.Slug)
}
```

#### Astro Build Integration

```go
func buildAstroSite() error {
    websiteDir := "./website"

    // Check if node_modules exists, install if not
    nodeModules := filepath.Join(websiteDir, "node_modules")
    if _, err := os.Stat(nodeModules); os.IsNotExist(err) {
        fmt.Println("Installing npm dependencies...")
        installCmd := exec.Command("npm", "install")
        installCmd.Dir = websiteDir
        installCmd.Run()
    }

    // Run build
    buildCmd := exec.Command("npm", "run", "build")
    buildCmd.Dir = websiteDir
    buildCmd.Stdout = os.Stdout
    buildCmd.Stderr = os.Stderr

    return buildCmd.Run()
}
```

---

### 7. NFO Parser (`internal/metadata/nfo/`)

#### Purpose
Parse Jellyfin `.nfo` XML files as a priority metadata source before falling back to TMDB.

#### File Discovery

NFO files are located using a two-step priority search:

```go
func (p *Parser) FindNFOFile(videoPath string) (string, error) {
    // 1. {filename}.nfo — matches the video filename exactly
    fileNameNFO := filepath.Join(dir, baseName+".nfo")

    // 2. movie.nfo — Jellyfin/Kodi convention in the same directory
    movieNFO := filepath.Join(dir, "movie.nfo")
}
```

#### Parsing & Conversion

`ParseNFOFile()` unmarshals the XML into `NFOMovie` structs defined in `types.go`. `ConvertToMovie()` maps NFO fields to the canonical `writer.Movie`:

- Joins multiple `<director>` elements with `", "`
- Extracts top 5 cast members from `<actor>` elements
- Falls back to parsing year from `<premiered>` if `<year>` is empty
- Extracts poster URL from `<thumb>` elements (prefers `aspect="poster"`)
- Extracts backdrop URL from `<fanart><thumb>` elements

#### Image URL Extraction

When `nfo_download_images` is enabled, poster and backdrop URLs are extracted from the NFO and attempted first. TMDB images are used as fallback if the NFO URLs fail or are absent.

```go
// Priority: explicit poster aspect > first thumb with URL
func extractPosterURL(thumbs []NFOThumb) string { ... }

// Returns first fanart thumb URL found
func extractBackdropURL(fanart *NFOFanart) string { ... }
```

`PosterURL` and `BackdropURL` are transient fields on the `Movie` struct (tagged `yaml:"-"`) — used during image download but not persisted to MDX.

---

### 8. Watch Mode (`internal/scanner/watcher.go`)

#### Purpose
Continuously monitor configured directories for new video files and automatically process them through the metadata pipeline.

#### Architecture

Watch mode uses the `fsnotify` library (v1.9.0) for cross-platform filesystem event monitoring:

```go
type Watcher struct {
    watcher *fsnotify.Watcher
    config  WatchConfig          // debounce delay, recursive flag
    handler FileHandler          // callback to process new files
    pending map[string]*time.Timer  // debounce timers per file
}
```

#### Debounce Mechanism

Large files (movie downloads) take time to copy. A configurable debounce delay prevents processing a file before it is fully written:

1. File creation event received
2. Timer started (default: 30 seconds)
3. If another write event arrives for the same file, timer resets
4. When timer fires, file is handed to the processing pipeline

#### Event Handling

| Event  | Behavior |
|--------|----------|
| Create | Start debounce timer → process when stable |
| Write  | Reset debounce timer for that file |
| Rename | Cancel pending timer, log the move |
| Remove | Cancel pending timer, log warning (MDX not auto-deleted) |

#### Lifecycle

Watch mode runs until interrupted by `SIGINT` or `SIGTERM`, which trigger graceful shutdown of all pending timers and the fsnotify watcher.

---

### 9. Duplicate Detection (`internal/scanner/duplicates.go`)

#### Purpose
Identify movies that appear multiple times in the library, and recommend which copy to keep based on quality.

#### Detection Logic

Duplicates are grouped by two methods:

1. **TMDB ID match** — movies sharing the same `tmdbId` in their MDX frontmatter
2. **Title + Year match** — for movies without a TMDB ID, normalized title and release year are compared

```go
type DuplicateSet struct {
    Movies []DuplicateMovie
}

type DuplicateMovie struct {
    FilePath      string
    Resolution    string  // e.g. "1080p"
    Source        string  // e.g. "BluRay"
    QualityScore  int     // combined ranking score
    IsRecommended bool    // highest quality in the set
}
```

#### Quality Scoring

Each copy receives a composite score: `resolution_rank × 10 + source_rank`.

**Resolution ranks:**

| Resolution   | Rank |
|--------------|------|
| 2160p / 4K   | 4    |
| 1080p        | 3    |
| 720p / 1080i | 2    |
| 480p / 720i  | 1    |

**Source ranks:**

| Source         | Rank |
|----------------|------|
| BluRay         | 8    |
| BRRip          | 7    |
| WEB-DL         | 6    |
| WEBRip         | 5    |
| HDRip / HDTV   | 4    |
| DVDRip         | 3    |
| DVDScr         | 2    |
| TS / TC        | 1    |
| CAM            | 0    |

The copy with the highest combined score is marked `IsRecommended`. The `--detailed` flag shows the full score breakdown per copy.

---

### 10. Cache System (`internal/metadata/cache/`)

#### Purpose
Persist TMDB API responses locally in a SQLite database to avoid redundant API calls across scanner runs.

#### Architecture

```go
type SQLiteCache struct {
    db    *sql.DB
    ttl   time.Duration      // configurable via cache.ttl_days
    stats CacheStats         // atomic hit/miss counters
}

type CacheStats struct {
    Hits       int64
    Misses     int64
    EntryCount int64
}
```

#### Lifecycle

1. On lookup, check cache first — increment `Hits` or `Misses` accordingly
2. On miss, fetch from TMDB, store result with a `cached_at` timestamp
3. Entries older than `ttl_days` are treated as expired
4. `--cache-stats` flag prints a summary without triggering a scan:

```
Cache Statistics: hits=450, misses=50, hit_rate=90.0%, entry_count=500
```

---

## Astro Website Components

### 1. Content Collections (`src/content/config.ts`)

#### Purpose
Define type-safe schema for movie MDX files.

```typescript
import { defineCollection, z } from 'astro:content';

const moviesCollection = defineCollection({
  type: 'content',  // MDX files
  schema: z.object({
    // Schema matches Go Movie struct exactly
    title: z.string(),
    slug: z.string(),
    description: z.string(),
    coverImage: z.string(),
    backdropImage: z.string().optional(),
    filePath: z.string(),
    fileName: z.string(),
    rating: z.number(),
    releaseYear: z.number(),
    releaseDate: z.string(),
    runtime: z.number(),
    genres: z.array(z.string()),
    director: z.string(),
    cast: z.array(z.string()),
    tmdbId: z.number(),
    imdbId: z.string().optional(),
    scannedAt: z.date(),
    fileSize: z.number(),
  }),
});

export const collections = {
  movies: moviesCollection,
};
```

#### Benefits

1. **Type Safety**: TypeScript knows the shape of movie data
2. **Validation**: Astro validates MDX frontmatter at build time
3. **Autocomplete**: IDEs provide autocomplete for movie properties
4. **Error Detection**: Catch schema mismatches early

#### Usage in Pages

```astro
---
import { getCollection } from 'astro:content';

// TypeScript knows the type!
const movies = await getCollection('movies');

// Autocomplete works:
movies[0].data.title
movies[0].data.rating
---
```

---

### 2. Base Layout (`src/layouts/BaseLayout.astro`)

#### Purpose
Common HTML structure for all pages.

```astro
---
interface Props {
  title?: string;
  description?: string;
}

const {
  title = "MovieVault - My Movie Collection",
  description = "Browse my personal movie collection"
} = Astro.props;
---

<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{title}</title>
  <meta name="description" content={description}>

  <!-- Open Graph for social media -->
  <meta property="og:title" content={title}>
  <meta property="og:description" content={description}>
</head>
<body>
  <header>
    <nav class="container">
      <a href="/" class="logo">🎬 MovieVault</a>
      <div class="nav-links">
        <a href="/">Movies</a>
        <a href="/search">Search</a>
      </div>
    </nav>
  </header>

  <main class="container">
    <slot />  <!-- Page content goes here -->
  </main>

  <footer>
    <p>Powered by Astro & TMDB</p>
  </footer>
</body>
</html>
```

#### Key Features

- **Props Interface**: Type-safe props with defaults
- **SEO Meta Tags**: Title, description, Open Graph
- **Responsive Container**: Max-width with padding
- **Slot**: Placeholder for page content
- **Global Styles**: Imported via `<style is:global>`

---

### 3. Homepage (`src/pages/index.astro`)

#### Purpose
Display all movies in a grid with statistics.

```astro
---
import { getCollection } from 'astro:content';
import BaseLayout from '../layouts/BaseLayout.astro';
import MovieGrid from '../components/MovieGrid.astro';
import SearchBar from '../components/SearchBar.astro';

// Get all movies, sorted by rating
const movies = (await getCollection('movies'))
  .sort((a, b) => b.data.rating - a.data.rating);

// Calculate statistics
const totalMovies = movies.length;
const totalRuntime = movies.reduce((sum, m) => sum + m.data.runtime, 0);
const totalSize = movies.reduce((sum, m) => sum + m.data.fileSize, 0);

// Format functions
const formatSize = (bytes: number) => {
  const gb = bytes / (1024 * 1024 * 1024);
  return gb >= 1000
    ? `${(gb / 1024).toFixed(2)} TB`
    : `${gb.toFixed(2)} GB`;
};

const formatRuntime = (minutes: number) => {
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  return days > 0
    ? `${days} days, ${hours % 24} hours`
    : `${hours} hours, ${minutes % 60} minutes`;
};
---

<BaseLayout title="MovieVault - My Movie Collection">
  <div class="page-header">
    <h1>My Movie Collection</h1>

    <div class="stats">
      <div class="stat">
        <span class="stat-value">{totalMovies}</span>
        <span class="stat-label">Movies</span>
      </div>
      <div class="stat">
        <span class="stat-value">{formatSize(totalSize)}</span>
        <span class="stat-label">Total Size</span>
      </div>
      <div class="stat">
        <span class="stat-value">{formatRuntime(totalRuntime)}</span>
        <span class="stat-label">Total Runtime</span>
      </div>
    </div>
  </div>

  <SearchBar />
  <MovieGrid movies={movies} />
</BaseLayout>
```

#### Statistics Calculation

**Total Movies**: Simple count
```typescript
const totalMovies = movies.length;
```

**Total Runtime**: Sum of all movie runtimes
```typescript
const totalRuntime = movies.reduce((sum, movie) => {
  return sum + movie.data.runtime;
}, 0);
```

**Total Size**: Sum of all file sizes
```typescript
const totalSize = movies.reduce((sum, movie) => {
  return sum + movie.data.fileSize;
}, 0);
```

---

### 4. Movie Detail Page (`src/pages/movies/[...slug].astro`)

#### Purpose
Display full information for a single movie.

#### Dynamic Routes

Astro generates a static HTML page for each movie:

```astro
---
export async function getStaticPaths() {
  const movies = await getCollection('movies');

  return movies.map((movie) => ({
    params: { slug: movie.data.slug },  // URL parameter
    props: { movie },                    // Passed to page
  }));
}
---
```

**Generated Routes**:
- `/movies/the-matrix-1999/` → `the-matrix-1999.mdx`
- `/movies/inception-2010/` → `inception-2010.mdx`
- `/movies/interstellar-2014/` → `interstellar-2014.mdx`

#### Rendering MDX Content

```astro
---
interface Props {
  movie: CollectionEntry<'movies'>;
}

const { movie } = Astro.props;

// Render MDX to HTML
const { Content } = await movie.render();
---

<article>
  <!-- Backdrop image -->
  {movie.data.backdropImage && (
    <div class="backdrop">
      <img src={movie.data.backdropImage} alt={movie.data.title} />
    </div>
  )}

  <!-- Movie info from frontmatter -->
  <h1>{movie.data.title}</h1>
  <p>{movie.data.description}</p>

  <!-- Rendered MDX content -->
  <div class="movie-content">
    <Content />
  </div>
</article>
```

---

### 5. Search Page (`src/pages/search.astro`)

#### Purpose
Search movies by title, cast, director, or genre.

```astro
---
const allMovies = await getCollection('movies');

// Get search query from URL
const url = new URL(Astro.request.url);
const query = url.searchParams.get('q')?.toLowerCase() || '';

// Filter movies
const filteredMovies = query
  ? allMovies.filter((movie) => {
      // Build searchable text
      const searchableText = [
        movie.data.title,
        movie.data.description,
        movie.data.director,
        ...movie.data.cast,
        ...movie.data.genres,
      ].join(' ').toLowerCase();

      // Check if query matches
      return searchableText.includes(query);
    })
  : [];
---

<BaseLayout title={`Search: ${query}`}>
  <h1>Search Movies</h1>
  <SearchBar initialQuery={query} />

  {query && (
    <p>Found {filteredMovies.length} movies matching "{query}"</p>
  )}

  <MovieGrid movies={filteredMovies} />
</BaseLayout>
```

#### Search Algorithm

1. **Build searchable text**: Combine all searchable fields
2. **Convert to lowercase**: Case-insensitive search
3. **Simple includes()**: Fast substring matching
4. **No external dependencies**: Pure JavaScript

For fuzzy search, could integrate Fuse.js:
```typescript
import Fuse from 'fuse.js';

const fuse = new Fuse(allMovies, {
  keys: ['data.title', 'data.cast', 'data.director'],
  threshold: 0.3,
});

const results = fuse.search(query);
```

---

### 6. Components

#### MovieCard Component

```astro
---
interface Props {
  title: string;
  slug: string;
  coverImage: string;
  rating: number;
  releaseYear: number;
  genres: string[];
}

const { title, slug, coverImage, rating, releaseYear, genres } = Astro.props;
---

<a href={`/movies/${slug}`} class="movie-card">
  <div class="poster-container">
    <img src={coverImage} alt={`${title} poster`} loading="lazy" />

    <!-- Hover overlay -->
    <div class="overlay">
      <h3>{title}</h3>
      <p>{releaseYear}</p>
    </div>
  </div>

  <div class="card-footer">
    <div class="rating">
      <span>⭐</span>
      <span>{rating.toFixed(1)}</span>
    </div>
    <div class="genres">
      {genres.slice(0, 2).join(', ')}
    </div>
  </div>
</a>

<style>
  .movie-card {
    transition: transform 0.2s;
  }

  .movie-card:hover {
    transform: translateY(-4px);
  }

  .overlay {
    opacity: 0;
    transition: opacity 0.3s;
  }

  .movie-card:hover .overlay {
    opacity: 1;
  }
</style>
```

**Key Features**:
- Lazy loading images (`loading="lazy"`)
- Hover effects (transform, overlay)
- Scoped styles (only affect this component)
- Semantic HTML (`<a>` for clickable card)

#### GenreFilter Component

```astro
---
const { movies } = Astro.props;

// Extract unique genres
const allGenres = new Set<string>();
movies.forEach((movie) => {
  movie.data.genres.forEach((genre) => allGenres.add(genre));
});

const genres = Array.from(allGenres).sort();
---

<div class="genre-filter">
  <h3>Filter by Genre</h3>
  <div class="genre-chips" data-genre-filter>
    <button class="genre-chip active" data-genre="all">All</button>
    {genres.map((genre) => (
      <button class="genre-chip" data-genre={genre.toLowerCase()}>
        {genre}
      </button>
    ))}
  </div>
</div>

<script>
  // Client-side filtering
  document.addEventListener('DOMContentLoaded', () => {
    const chips = document.querySelectorAll('.genre-chip');
    const cards = document.querySelectorAll('.movie-card');

    chips.forEach((chip) => {
      chip.addEventListener('click', () => {
        const genre = chip.getAttribute('data-genre');

        // Update active chip
        chips.forEach((c) => c.classList.remove('active'));
        chip.classList.add('active');

        // Filter cards
        cards.forEach((card) => {
          const cardGenres = card.querySelector('.genres')?.textContent || '';

          if (genre === 'all' || cardGenres.toLowerCase().includes(genre)) {
            card.style.display = '';
          } else {
            card.style.display = 'none';
          }
        });
      });
    });
  });
</script>
```

**Client-Side Filtering**:
- No page reload required
- Instant filtering
- Works with existing movie cards
- Progressive enhancement (works without JS)

---

## Data Flow

### Full System Data Flow

```
┌─────────────────────────────────────────────────────────────┐
│ 1. FILE DISCOVERY                                           │
│ Scanner walks directories → Finds video files               │
│ /Movies/The.Matrix.1999.1080p.mkv                           │
└─────────┬───────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. FILENAME PARSING                                         │
│ Extract: "The Matrix" + 1999                                │
│ Generate slug: "the-matrix-1999"                            │
│ Check: MDX exists? NO → Continue                            │
└─────────┬───────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. METADATA LOOKUP                                          │
│ Priority 1: Parse NFO file (if use_nfo enabled)             │
│ Priority 2: Search TMDB API (fallback)                      │
│ Priority 3: Merge NFO + TMDB if NFO data is incomplete      │
│ Result: TMDB ID 603                                         │
└─────────┬───────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. FETCH DETAILS (TMDB, if needed)                          │
│ GET /movie/603 → Details                                    │
│ GET /movie/603/credits → Cast & Crew                        │
│ Rate limit: Wait 250ms between requests                     │
│ Results cached in SQLite for future runs                    │
└─────────┬───────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────┐
│ 5. DOWNLOAD IMAGES                                          │
│ Poster: /covers/the-matrix-1999.jpg                         │
│ Backdrop: /covers/the-matrix-1999-backdrop.jpg             │
└─────────┬───────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────┐
│ 6. GENERATE MDX                                             │
│ Create: website/src/content/movies/the-matrix-1999.mdx     │
│ YAML frontmatter + Markdown content                         │
└─────────┬───────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────┐
│ 7. ASTRO BUILD                                              │
│ Read MDX files → Validate schema → Generate HTML            │
│ Output: website/dist/                                       │
└─────────┬───────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────┐
│ 8. STATIC WEBSITE                                           │
│ /index.html → Homepage with all movies                      │
│ /movies/the-matrix-1999/index.html → Movie detail page     │
│ /search/index.html → Search functionality                   │
└─────────────────────────────────────────────────────────────┘
```

### Scanner → MDX Flow

```go
// 1. Discover file
fileInfo := scanner.ScanDirectory("/Movies")
// FileInfo{
//   Path: "/Movies/The.Matrix.1999.1080p.mkv",
//   Title: "The Matrix",
//   Year: 1999,
//   Slug: "the-matrix-1999"
// }

// 2. Fetch from TMDB
movie := tmdbClient.GetFullMovieData("The Matrix", 1999)
// Movie{
//   Title: "The Matrix",
//   Rating: 8.7,
//   Genres: ["Action", "Science Fiction"],
//   ...
// }

// 3. Add file info
movie.FilePath = fileInfo.Path
movie.FileName = fileInfo.FileName
movie.FileSize = fileInfo.Size
movie.Slug = fileInfo.Slug

// 4. Download images
tmdbClient.DownloadImage(posterPath, "/covers/the-matrix-1999.jpg", "poster")
movie.CoverImage = "/covers/the-matrix-1999.jpg"

// 5. Generate MDX
mdxWriter.WriteMDXFile(movie)
// Creates: website/src/content/movies/the-matrix-1999.mdx
```

### MDX → HTML Flow

```typescript
// 1. Astro reads MDX
const movies = await getCollection('movies');

// 2. For each movie, generate route
export async function getStaticPaths() {
  return movies.map(movie => ({
    params: { slug: movie.data.slug },
    props: { movie }
  }));
}

// 3. Render page
const { movie } = Astro.props;
const { Content } = await movie.render();

// 4. Output HTML
// website/dist/movies/the-matrix-1999/index.html
```

---

## Performance Optimizations

### 1. Smart Scanning (Skip Existing)

**Problem**: First scan takes 100 seconds for 100 movies

**Solution**: Check if MDX exists before fetching from TMDB

```go
func (s *Scanner) MDXExists(slug string) bool {
    mdxPath := filepath.Join(s.mdxDir, slug+".mdx")
    _, err := os.Stat(mdxPath)
    return err == nil  // true if file exists
}

// In scanning loop
if fileInfo.ShouldScan {
    // Only fetch if MDX doesn't exist
    movie := tmdbClient.GetFullMovieData(title, year)
} else {
    // Skip - MDX already exists
}
```

**Result**: Subsequent scans take 1 second instead of 100 seconds

### 2. Rate Limiting

**Problem**: TMDB may rate limit if too many requests

**Solution**: Add delay between requests

```go
time.Sleep(250 * time.Millisecond)  // 4 requests/second
```

**Trade-off**: Slower first scan, but respectful to API

### 3. Static Site Generation

**Problem**: Database queries slow down page loads

**Solution**: Pre-generate all HTML at build time

```typescript
// All movies loaded at BUILD time, not runtime
const movies = await getCollection('movies');
```

**Result**: Instant page loads (just serving static HTML)

### 4. Lazy Image Loading

**Problem**: Loading 100 poster images slows page load

**Solution**: Browser-native lazy loading

```html
<img src="/covers/movie.jpg" loading="lazy" />
```

**Result**: Only loads images when scrolled into view

### 5. Image Optimization

**Problem**: Full-size TMDB images are huge

**Solution**: Use optimal sizes from TMDB

```go
posterSize := "w500"       // 500px wide (not original)
backdropSize := "w1280"    // 1280px wide (not original)
```

**Result**: 5MB per movie instead of 20MB

---

## Extension Guide

### Adding New Metadata Fields

**1. Update Go struct** (`internal/writer/models.go`):
```go
type Movie struct {
    // ... existing fields ...
    Country string `yaml:"country"`  // NEW
}
```

**2. Fetch from TMDB** (`internal/metadata/tmdb.go`):
```go
movie.Country = details.ProductionCountries[0].Name
```

**3. Update Astro schema** (`website/src/content/config.ts`):
```typescript
schema: z.object({
  // ... existing fields ...
  country: z.string(),  // NEW
})
```

**4. Display on page** (`website/src/pages/movies/[...slug].astro`):
```astro
<div class="detail-item">
  <strong>Country:</strong>
  <span>{movie.data.country}</span>
</div>
```

### Adding TV Show Support

**1. Create new content collection**:
```typescript
// website/src/content/config.ts
const showsCollection = defineCollection({
  type: 'content',
  schema: z.object({
    title: z.string(),
    seasons: z.number(),
    episodes: z.number(),
    // ... other TV-specific fields
  }),
});

export const collections = {
  movies: moviesCollection,
  shows: showsCollection,  // NEW
};
```

**2. Create TV scanner** (`internal/scanner/tv.go`):
```go
func (s *Scanner) ScanForTVShows(path string) ([]ShowInfo, error) {
    // Detect TV show directory structure
    // Parse S01E01 format
    // Return show info
}
```

**3. Add TV pages**:
- `src/pages/shows/index.astro` - All shows
- `src/pages/shows/[...slug].astro` - Show details

### Adding User Ratings

**Option 1: Local Storage (Client-Side)**

```typescript
// components/RatingWidget.astro
<script>
  function saveRating(movieId: string, rating: number) {
    localStorage.setItem(`rating-${movieId}`, rating.toString());
  }

  function getRating(movieId: string): number {
    return parseInt(localStorage.getItem(`rating-${movieId}`) || '0');
  }
</script>
```

**Option 2: JSON File (Server-Side)**

```go
// internal/ratings/ratings.go
type UserRatings struct {
    Ratings map[string]float64 `json:"ratings"`
}

func SaveRating(movieId string, rating float64) error {
    // Read existing ratings.json
    // Update rating
    // Write back to file
}
```

Then rebuild site after rating changes.

### Adding Video Streaming

**Warning**: Complex! Requires video transcoding.

**Basic approach**:

```astro
---
// src/pages/watch/[slug].astro
const { movie } = Astro.props;
---

<video controls>
  <source src={`/api/stream/${movie.data.slug}`} type="video/mp4">
</video>
```

**Backend needed**:
- Node.js/Go API server
- FFmpeg for transcoding
- HLS/DASH for adaptive streaming
- Authentication for security

---

## Docker Architecture

### Multi-Stage Build Process

```dockerfile
# Stage 1: Build Go binary
FROM golang:1.21-alpine AS go-builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o scanner cmd/scanner/main.go

# Stage 2: Prepare Node.js environment
FROM node:20-alpine AS web-builder
WORKDIR /build
COPY website/package*.json ./
RUN npm ci --only=production
COPY website/ ./

# Stage 3: Final runtime
FROM nginx:alpine
RUN apk add --no-cache nodejs npm
COPY --from=go-builder /build/scanner /usr/local/bin/scanner
COPY --from=web-builder /build /app/website
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf
COPY docker/entrypoint.sh /entrypoint.sh
EXPOSE 80
ENTRYPOINT ["/entrypoint.sh"]
```

### Why Multi-Stage?

1. **Smaller final image**: Go builder (~500MB) not in final image
2. **Faster builds**: Layers cached independently
3. **Security**: No build tools in production image

### Volume Strategy

```yaml
volumes:
  # Read-only: Movie files (never modify originals)
  - /path/to/movies:/movies:ro

  # Read-write: Generated content (MDX, covers)
  - ./data/movies:/data/movies
  - ./data/covers:/data/covers

  # Read-only: Configuration (sensitive API keys)
  - ./config/config.yaml:/config/config.yaml:ro
```

### Entrypoint Script Flow

```bash
#!/bin/sh
# 1. Run scanner if AUTO_SCAN=true
if [ "$AUTO_SCAN" = "true" ]; then
  scanner --config /config/config.yaml
fi

# 2. Build Astro if not built
if [ ! -d "/app/website/dist" ]; then
  cd /app/website && npm run build
fi

# 3. Copy to nginx directory
cp -r /app/website/dist/* /usr/share/nginx/html/

# 4. Start nginx
exec nginx -g 'daemon off;'
```

---

## Conclusion

This documentation covers the complete technical architecture of MovieVault. Each component is designed to be:

- **Modular**: Components can be modified independently
- **Testable**: Clear interfaces and separation of concerns
- **Performant**: Optimized for large collections
- **Maintainable**: Clean code with good documentation
- **Extensible**: Easy to add new features

For implementation details, see the source code with inline comments.
