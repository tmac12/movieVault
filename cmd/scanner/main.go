package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/marco/movieVault/internal/config"
	"github.com/marco/movieVault/internal/metadata"
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
)

func main() {
	flag.Parse()

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

	// Create TMDB client
	tmdbClient := metadata.NewClient(cfg.TMDB.APIKey, cfg.TMDB.Language, cfg.Options.RateLimitDelay)

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
				}
			} else {
				metadataSource = "NFO"

				// Check for incomplete NFO data
				if cfg.Options.NFOFallbackTMDB && (movie.Title == "" || movie.ReleaseYear == 0) {
					slog.Debug("nfo incomplete, enriching with tmdb")
					tmdbMovie, tmdbErr := tmdbClient.GetFullMovieData(file.Title, file.Year)
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

			// Get poster path from TMDB (we need to search again to get the poster path)
			searchResult, _ := tmdbClient.SearchMovie(movie.Title, movie.ReleaseYear)
			if searchResult != nil && searchResult.PosterPath != "" {
				if err := tmdbClient.DownloadImage(searchResult.PosterPath, coverPath, "poster"); err != nil {
					slog.Warn("failed to download cover", "movie", movie.Title, "error", err)
				} else {
					slog.Debug("downloaded cover image", "movie", movie.Title)
				}
			}
		}

		// Download backdrop image
		if cfg.Options.DownloadBackdrops {
			backdropPath := mdxWriter.GetAbsoluteBackdropPath(movie.Slug)
			movie.BackdropImage = mdxWriter.GetBackdropPath(movie.Slug)

			searchResult, _ := tmdbClient.SearchMovie(movie.Title, movie.ReleaseYear)
			if searchResult != nil && searchResult.BackdropPath != "" {
				if err := tmdbClient.DownloadImage(searchResult.BackdropPath, backdropPath, "backdrop"); err != nil {
					slog.Warn("failed to download backdrop", "movie", movie.Title, "error", err)
				} else {
					slog.Debug("downloaded backdrop image", "movie", movie.Title)
				}
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

// Helper function to repeat a string (not available in older Go versions)
func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
