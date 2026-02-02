package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/marco/movieVault/internal/config"
	"github.com/marco/movieVault/internal/metadata"
	"github.com/marco/movieVault/internal/metadata/cache"
	"github.com/marco/movieVault/internal/metadata/nfo"
	"github.com/marco/movieVault/internal/scanner"
	"github.com/marco/movieVault/internal/writer"
)

var (
	configPath   = flag.String("config", "./config/config.yaml", "Path to configuration file")
	forceRefresh = flag.Bool("force-refresh", false, "Re-fetch all metadata from TMDB even for existing MDX files")
	noBuild      = flag.Bool("no-build", false, "Skip Astro build step")
	dryRun       = flag.Bool("dry-run", false, "Show what would be done without actually doing it")
	verbose      = flag.Bool("verbose", false, "Show detailed logging")
	clearCache   = flag.Bool("clear-cache", false, "Clear the metadata cache and exit")
	testParser   = flag.Bool("test-parser", false, "Test title extraction without running full scan")
)

func main() {
	flag.Parse()

	// Handle --test-parser flag (US-017)
	if *testParser {
		exitCode := runTestParser()
		os.Exit(exitCode)
	}

	// Setup structured logger
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	startTime := time.Now()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "path", *configPath, "error", err)
		os.Exit(1)
	}

	slog.Info("configuration loaded",
		"path", *configPath,
		"directories", len(cfg.Scanner.Directories),
		"extensions", len(cfg.Scanner.Extensions),
		"nfo_enabled", cfg.Options.UseNFO,
		"nfo_fallback", cfg.Options.NFOFallbackTMDB,
	)

	if *verbose {
		slog.Debug("config details",
			"scan_dirs", cfg.Scanner.Directories,
			"mdx_dir", cfg.Output.MDXDir,
			"covers_dir", cfg.Output.CoversDir,
		)
	}

	// Handle --clear-cache flag
	if *clearCache {
		if !cfg.Cache.Enabled {
			fmt.Println("Cache is disabled in configuration.")
			os.Exit(0)
		}

		tmdbCache, err := cache.NewSQLiteCache(cfg.Cache.Path)
		if err != nil {
			slog.Error("failed to open cache", "path", cfg.Cache.Path, "error", err)
			os.Exit(1)
		}
		defer tmdbCache.Close()

		// Get count before clearing
		count, err := tmdbCache.Count()
		if err != nil {
			slog.Error("failed to count cache entries", "error", err)
			os.Exit(1)
		}

		// Clear the cache
		if err := tmdbCache.Clear(); err != nil {
			slog.Error("failed to clear cache", "error", err)
			os.Exit(1)
		}

		fmt.Printf("Cache cleared successfully. %d entries removed.\n", count)
		os.Exit(0)
	}

	// Create scanner with directory exclusions
	s := scanner.NewWithExclusions(cfg.Scanner.Extensions, cfg.Output.MDXDir, cfg.Scanner.ExcludeDirs)

	// Scan all directories
	slog.Info("scanning directories for video files", "count", len(cfg.Scanner.Directories))
	files, err := s.ScanAll(cfg.Scanner.Directories)
	if err != nil {
		slog.Error("failed to scan directories", "error", err)
		os.Exit(1)
	}

	slog.Info("scan complete", "files_found", len(files))

	// Filter files based on force-refresh flag
	var filesToProcess []scanner.FileInfo
	if *forceRefresh {
		filesToProcess = files
		slog.Info("force refresh enabled", "processing_all", true)
	} else {
		for _, file := range files {
			if file.ShouldScan {
				filesToProcess = append(filesToProcess, file)
			}
		}
		skippedCount := len(files) - len(filesToProcess)
		if skippedCount > 0 {
			slog.Info("skipping existing files", "count", skippedCount)
		}
	}

	if len(filesToProcess) == 0 {
		slog.Info("no new files to process")
		return
	}

	slog.Info("processing files", "count", len(filesToProcess))

	if *dryRun {
		fmt.Println("\nDRY RUN MODE - No actual changes will be made")
		for _, file := range filesToProcess {
			fmt.Printf("Would process: %s\n", file.FileName)
			fmt.Printf("  Title: %s\n", file.Title)
			if file.Year > 0 {
				fmt.Printf("  Year: %d\n", file.Year)
			}
			fmt.Printf("  Slug: %s\n", file.Slug)
			fmt.Println()
		}
		return
	}

	// Initialize cache if enabled
	var tmdbCache cache.Cache
	if cfg.Cache.Enabled {
		var err error
		tmdbCache, err = cache.NewSQLiteCache(cfg.Cache.Path)
		if err != nil {
			slog.Error("failed to initialize cache", "path", cfg.Cache.Path, "error", err)
			os.Exit(1)
		}
		defer tmdbCache.Close()
		slog.Info("cache initialized", "path", cfg.Cache.Path, "ttl_days", cfg.Cache.TTLDays)
	}

	// Create TMDB client with retry and cache configuration
	var retryLogFunc metadata.RetryLogFunc
	var cacheLogFunc metadata.CacheLogFunc
	if *verbose {
		retryLogFunc = func(attempt int, maxAttempts int, backoff time.Duration, err error) {
			slog.Debug("retrying tmdb request",
				"attempt", attempt,
				"max_attempts", maxAttempts,
				"backoff_ms", backoff.Milliseconds(),
				"error", err.Error(),
			)
		}
		cacheLogFunc = func(operation string, key string, hit bool) {
			switch operation {
			case "get":
				if hit {
					slog.Debug("cache hit", "key", key)
				} else {
					slog.Debug("cache miss", "key", key)
				}
			case "set":
				slog.Debug("cache store", "key", key)
			case "set_error":
				slog.Warn("cache store failed", "key", key)
			}
		}
	}
	tmdbClient := metadata.NewClientWithConfig(metadata.ClientConfig{
		APIKey:           cfg.TMDB.APIKey,
		Language:         cfg.TMDB.Language,
		RateLimitDelayMs: cfg.Options.RateLimitDelay,
		MaxAttempts:      cfg.Retry.MaxAttempts,
		InitialBackoffMs: cfg.Retry.InitialBackoffMs,
		RetryLogFunc:     retryLogFunc,
		Cache:            tmdbCache,
		CacheTTLDays:     cfg.Cache.TTLDays,
		CacheLogFunc:     cacheLogFunc,
		ForceRefresh:     *forceRefresh,
	})

	// Create MDX writer
	mdxWriter := writer.NewMDXWriter(cfg.Output.MDXDir, cfg.Output.CoversDir)

	// Process each file
	successCount := 0
	errorCount := 0
	nfoCount := 0
	tmdbCount := 0
	mixedCount := 0

	for i, file := range filesToProcess {
		slog.Info("processing file",
			"progress", fmt.Sprintf("%d/%d", i+1, len(filesToProcess)),
			"filename", file.FileName,
		)

		slog.Debug("file details",
			"title", file.Title,
			"year", file.Year,
			"path", file.Path,
		)

		// Fetch metadata from NFO or TMDB
		var movie *writer.Movie
		var err error
		var metadataSource string

		// Try NFO first if enabled
		if cfg.Options.UseNFO {
			nfoParser := nfo.NewParser()
			movie, err = nfoParser.GetMovieFromNFO(file.Path)

			if err != nil {
				// NFO not found or parse error - fall back to TMDB if enabled
				if cfg.Options.NFOFallbackTMDB {
					slog.Debug("nfo error, falling back to tmdb", "error", err)
					movie, err = tmdbClient.GetFullMovieData(file.Title, file.Year)
					metadataSource = "TMDB"
					slog.Debug("tmdb lookup method", "method", "search", "title", file.Title, "year", file.Year)
				}
			} else {
				metadataSource = "NFO"

				// Check if NFO has TMDB ID for direct lookup
				if movie.TMDBID > 0 && cfg.Options.NFOFallbackTMDB {
					slog.Debug("nfo contains tmdb id, using direct lookup", "tmdb_id", movie.TMDBID)
					tmdbMovie, tmdbErr := tmdbClient.GetMovieByID(movie.TMDBID)
					if tmdbErr != nil {
						// Check if movie not found (404) - fall back to search
						if errors.Is(tmdbErr, metadata.ErrMovieNotFound) {
							slog.Debug("tmdb id not found, falling back to search", "tmdb_id", movie.TMDBID)
							tmdbMovie, tmdbErr = tmdbClient.GetFullMovieData(file.Title, file.Year)
							slog.Debug("tmdb lookup method", "method", "search (fallback from direct)", "title", file.Title, "year", file.Year)
						}
					} else {
						slog.Debug("tmdb lookup method", "method", "direct ID", "tmdb_id", movie.TMDBID)
					}
					if tmdbErr == nil && tmdbMovie != nil {
						movie = mergeMovieData(movie, tmdbMovie)
						metadataSource = "NFO+TMDB"
					}
				} else if cfg.Options.NFOFallbackTMDB && (movie.Title == "" || movie.ReleaseYear == 0) {
					// Check for incomplete NFO data (no TMDB ID available)
					slog.Debug("nfo incomplete, enriching with tmdb search")
					tmdbMovie, tmdbErr := tmdbClient.GetFullMovieData(file.Title, file.Year)
					slog.Debug("tmdb lookup method", "method", "search", "title", file.Title, "year", file.Year)
					if tmdbErr == nil && tmdbMovie != nil {
						movie = mergeMovieData(movie, tmdbMovie)
						metadataSource = "NFO+TMDB"
					}
				}
			}
		} else {
			// NFO disabled, use TMDB only
			movie, err = tmdbClient.GetFullMovieData(file.Title, file.Year)
			metadataSource = "TMDB"
			slog.Debug("tmdb lookup method", "method", "search", "title", file.Title, "year", file.Year)
		}

		if err != nil {
			slog.Error("failed to fetch metadata",
				"filename", file.FileName,
				"title", file.Title,
				"error", err,
			)
			errorCount++
			continue
		}

		// Generate clean slug from metadata title (not from filename)
		movie.Slug = scanner.GenerateSlug(movie.Title, movie.ReleaseYear)

		// Add file information
		movie.FilePath = file.Path
		movie.FileName = file.FileName
		movie.FileSize = file.Size

		// Log successful metadata fetch
		slog.Info("metadata fetched",
			"movie", movie.Title,
			"year", movie.ReleaseYear,
			"source", metadataSource,
		)

		// Track metadata sources for summary
		switch metadataSource {
		case "NFO":
			nfoCount++
		case "TMDB":
			tmdbCount++
		case "NFO+TMDB":
			mixedCount++
		}

		slog.Debug("movie details",
			"tmdb_id", movie.TMDBID,
			"rating", movie.Rating,
			"genres", movie.Genres,
		)

		// Download cover image
		if cfg.Options.DownloadCovers {
			coverPath := mdxWriter.GetAbsoluteCoverPath(movie.Slug)
			movie.CoverImage = mdxWriter.GetCoverPath(movie.Slug)

			coverDownloaded := false
			coverSource := ""

			// Try NFO poster URL first if enabled (US-020)
			if cfg.Options.NFODownloadImages && movie.PosterURL != "" {
				if err := tmdbClient.DownloadImageFromURL(movie.PosterURL, coverPath); err != nil {
					slog.Debug("failed to download cover from nfo url, trying tmdb", "movie", movie.Title, "error", err)
				} else {
					coverDownloaded = true
					coverSource = "NFO"
				}
			}

			// Fall back to TMDB if NFO download failed or not enabled
			if !coverDownloaded {
				searchResult, _ := tmdbClient.SearchMovie(movie.Title, movie.ReleaseYear)
				if searchResult != nil && searchResult.PosterPath != "" {
					if err := tmdbClient.DownloadImage(searchResult.PosterPath, coverPath, "poster"); err != nil {
						slog.Warn("failed to download cover", "movie", movie.Title, "error", err)
					} else {
						coverDownloaded = true
						coverSource = "TMDB"
					}
				}
			}

			if coverDownloaded {
				slog.Debug("downloaded cover image", "movie", movie.Title, "source", coverSource)
			}
		}

		// Download backdrop image
		if cfg.Options.DownloadBackdrops {
			backdropPath := mdxWriter.GetAbsoluteBackdropPath(movie.Slug)
			movie.BackdropImage = mdxWriter.GetBackdropPath(movie.Slug)

			backdropDownloaded := false
			backdropSource := ""

			// Try NFO backdrop URL first if enabled (US-020)
			if cfg.Options.NFODownloadImages && movie.BackdropURL != "" {
				if err := tmdbClient.DownloadImageFromURL(movie.BackdropURL, backdropPath); err != nil {
					slog.Debug("failed to download backdrop from nfo url, trying tmdb", "movie", movie.Title, "error", err)
				} else {
					backdropDownloaded = true
					backdropSource = "NFO"
				}
			}

			// Fall back to TMDB if NFO download failed or not enabled
			if !backdropDownloaded {
				searchResult, _ := tmdbClient.SearchMovie(movie.Title, movie.ReleaseYear)
				if searchResult != nil && searchResult.BackdropPath != "" {
					if err := tmdbClient.DownloadImage(searchResult.BackdropPath, backdropPath, "backdrop"); err != nil {
						slog.Warn("failed to download backdrop", "movie", movie.Title, "error", err)
					} else {
						backdropDownloaded = true
						backdropSource = "TMDB"
					}
				}
			}

			if backdropDownloaded {
				slog.Debug("downloaded backdrop image", "movie", movie.Title, "source", backdropSource)
			}
		}

		// Write MDX file
		if err := mdxWriter.WriteMDXFile(movie); err != nil {
			slog.Error("failed to write mdx file",
				"movie", movie.Title,
				"slug", movie.Slug,
				"error", err,
			)
			errorCount++
			continue
		}

		slog.Info("mdx file created", "slug", movie.Slug)
		successCount++
	}

	// Print summary
	duration := time.Since(startTime)
	slog.Info("scan complete",
		"total_files", len(files),
		"processed", len(filesToProcess),
		"successful", successCount,
		"errors", errorCount,
		"duration_sec", duration.Seconds(),
	)

	// Show metadata source breakdown
	if successCount > 0 {
		slog.Info("metadata sources",
			"nfo_count", nfoCount,
			"nfo_percent", fmt.Sprintf("%.0f%%", float64(nfoCount)/float64(successCount)*100),
			"tmdb_count", tmdbCount,
			"tmdb_percent", fmt.Sprintf("%.0f%%", float64(tmdbCount)/float64(successCount)*100),
			"mixed_count", mixedCount,
			"mixed_percent", fmt.Sprintf("%.0f%%", float64(mixedCount)/float64(successCount)*100),
		)
	}

	// Build Astro site if enabled and not disabled via flag
	if cfg.Output.AutoBuild && !*noBuild && successCount > 0 {
		slog.Info("building astro website")
		websiteDir := cfg.Output.WebsiteDir
		if websiteDir == "" {
			websiteDir = "./website"
		}
		if err := buildAstroSite(websiteDir); err != nil {
			slog.Error("failed to build astro site", "error", err, "website_dir", websiteDir)
			slog.Info("manual build command", "command", fmt.Sprintf("cd %s && npm run build", websiteDir))
		} else {
			slog.Info("astro site built successfully")
		}
	}

	if errorCount > 0 {
		os.Exit(1)
	}
}

// buildAstroSite runs the Astro build command
func buildAstroSite(websiteDir string) error {
	// Check if website directory exists
	if _, err := os.Stat(websiteDir); os.IsNotExist(err) {
		return fmt.Errorf("website directory does not exist at: %s", websiteDir)
	}

	// Check if package.json exists (confirm it's a Node.js project)
	packageJSON := filepath.Join(websiteDir, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		return fmt.Errorf("package.json not found in %s (not a Node.js project?)", websiteDir)
	}

	// Check if node_modules exists
	nodeModules := filepath.Join(websiteDir, "node_modules")
	if _, err := os.Stat(nodeModules); os.IsNotExist(err) {
		fmt.Println("Installing npm dependencies...")
		installCmd := exec.Command("npm", "install")
		installCmd.Dir = websiteDir
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("npm install failed: %w", err)
		}
	}

	// Run build command
	buildCmd := exec.Command("npm", "run", "build")
	buildCmd.Dir = websiteDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	return buildCmd.Run()
}

// mergeMovieData merges NFO data (priority) with TMDB data (fallback)
func mergeMovieData(nfoMovie, tmdbMovie *writer.Movie) *writer.Movie {
	merged := nfoMovie

	// Fill missing fields from TMDB
	if merged.Title == "" {
		merged.Title = tmdbMovie.Title
	}
	if merged.Description == "" {
		merged.Description = tmdbMovie.Description
	}
	if merged.Rating == 0 {
		merged.Rating = tmdbMovie.Rating
	}
	if merged.ReleaseYear == 0 {
		merged.ReleaseYear = tmdbMovie.ReleaseYear
	}
	if merged.ReleaseDate == "" {
		merged.ReleaseDate = tmdbMovie.ReleaseDate
	}
	if merged.Runtime == 0 {
		merged.Runtime = tmdbMovie.Runtime
	}
	if len(merged.Genres) == 0 {
		merged.Genres = tmdbMovie.Genres
	}
	if merged.Director == "" {
		merged.Director = tmdbMovie.Director
	}
	if len(merged.Cast) == 0 {
		merged.Cast = tmdbMovie.Cast
	}
	if merged.TMDBID == 0 {
		merged.TMDBID = tmdbMovie.TMDBID
	}
	if merged.IMDbID == "" {
		merged.IMDbID = tmdbMovie.IMDbID
	}

	return merged
}

// runTestParser tests title extraction on filenames without running a full scan (US-017)
// Returns exit code: 0 if all extractions produced valid titles, 1 if any produced empty title
func runTestParser() int {
	filenames := flag.Args()

	// If no arguments provided, read from stdin
	if len(filenames) == 0 {
		stdinReader := bufio.NewScanner(os.Stdin)
		for stdinReader.Scan() {
			line := stdinReader.Text()
			if line != "" {
				filenames = append(filenames, line)
			}
		}
		if err := stdinReader.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			return 1
		}
	}

	if len(filenames) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: scanner --test-parser <filename> [filename2] ...")
		fmt.Fprintln(os.Stderr, "       echo 'filename.mkv' | scanner --test-parser")
		return 1
	}

	hasEmptyTitle := false

	for _, filename := range filenames {
		title, year := scanner.ExtractTitleAndYear(filename)
		slug := scanner.GenerateSlug(title, year)

		// Detect which patterns matched
		patternsMatched := detectPatternsMatched(filename)

		fmt.Printf("Filename: %s\n", filename)
		fmt.Printf("  Title: %s\n", title)
		if year > 0 {
			fmt.Printf("  Year: %d\n", year)
		} else {
			fmt.Printf("  Year: (not found)\n")
		}
		fmt.Printf("  Slug: %s\n", slug)
		if len(patternsMatched) > 0 {
			fmt.Printf("  Patterns matched: %s\n", patternsMatched)
		} else {
			fmt.Printf("  Patterns matched: (none)\n")
		}
		fmt.Println()

		if title == "" {
			hasEmptyTitle = true
		}
	}

	if hasEmptyTitle {
		return 1
	}
	return 0
}

// detectPatternsMatched returns a comma-separated list of pattern categories that matched
func detectPatternsMatched(filename string) string {
	var patterns []string

	// Remove extension for pattern matching (same as ExtractTitleAndYear)
	name := filename
	if idx := len(name) - len(filepath.Ext(name)); idx > 0 {
		name = name[:idx]
	}

	// Check each pattern category
	if matched, _ := regexp.MatchString(`(?i)\b(2160p|1080p|1080i|720p|720i|480p|4K)\b`, name); matched {
		patterns = append(patterns, "resolution")
	}
	if matched, _ := regexp.MatchString(`(?i)[\[\(](\d{4})[\]\)]`, name); matched {
		patterns = append(patterns, "year-bracketed")
	} else if matched, _ := regexp.MatchString(`\d{4}`, name); matched {
		patterns = append(patterns, "year")
	}
	if matched, _ := regexp.MatchString(`(?i)\b(BluRay|BRRip|WEB-DL|WEBRip|HDRip|DVDRip|HDTV|BDRip|WEB|AMZN|NF)\b`, name); matched {
		patterns = append(patterns, "quality")
	}
	if matched, _ := regexp.MatchString(`(?i)\b(x264|x265|H\.?264|H\.?265|HEVC|XviD|DivX|AVC|10bit|HDR10|HDR|DV)\b`, name); matched {
		patterns = append(patterns, "codec")
	}
	if matched, _ := regexp.MatchString(`(?i)\b(AAC|AC3|DTS-HD|DTS|TrueHD|FLAC|MP3|DD5\.1|DD2\.0|Atmos|7\.1|5\.1|2\.0|MA)\b`, name); matched {
		patterns = append(patterns, "audio")
	}
	if matched, _ := regexp.MatchString(`(?i)\b(ita|eng|spa|fra|deu|jpn|kor|rus|chi|por|pol|nld|swe|nor|dan|fin|tur|ara|heb|tha|vie|ind|msa|hindi|tamil|multi|dual)\b`, name); matched {
		patterns = append(patterns, "language")
	}
	if matched, _ := regexp.MatchString(`(?i)\b(EXTENDED\.?CUT|EXTENDED|DIRECTOR\'?S\.?CUT|DIRECTORS\.?CUT|UNRATED|THEATRICAL|IMAX|REMASTERED|DC|UHD)\b`, name); matched {
		patterns = append(patterns, "edition")
	}
	if matched, _ := regexp.MatchString(`(?i)[-\.]([A-Z0-9]+|MIRCrew|RARBG|YTS|YIFY|SPARKS|GECKOS|AMIABLE|CODEX|SKIDROW|PLAZA|CPY|RELOADED)$`, name); matched {
		patterns = append(patterns, "release-group")
	}
	if matched, _ := regexp.MatchString(`(?i)\[(YTS|YIFY|RARBG|EVO|FGT|SPARKS|GECKOS|[A-Za-z0-9\.]+)\]`, name); matched {
		patterns = append(patterns, "bracketed-group")
	}

	return strings.Join(patterns, ", ")
}

// Helper function to repeat a string (not available in older Go versions)
func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
