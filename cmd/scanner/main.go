package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Configuration loaded from: %s\n", *configPath)
		fmt.Printf("Scanning directories: %v\n", cfg.Scanner.Directories)
		fmt.Printf("Output MDX directory: %s\n", cfg.Output.MDXDir)
		fmt.Printf("Output covers directory: %s\n", cfg.Output.CoversDir)
	}

	// Create scanner
	s := scanner.New(cfg.Scanner.Extensions, cfg.Output.MDXDir)

	// Scan all directories
	fmt.Println("Scanning directories for video files...")
	files, err := s.ScanAll(cfg.Scanner.Directories)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directories: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d video files\n", len(files))

	// Filter files based on force-refresh flag
	var filesToProcess []scanner.FileInfo
	if *forceRefresh {
		filesToProcess = files
		fmt.Println("Force refresh enabled: processing all files")
	} else {
		for _, file := range files {
			if file.ShouldScan {
				filesToProcess = append(filesToProcess, file)
			}
		}
		skippedCount := len(files) - len(filesToProcess)
		if skippedCount > 0 {
			fmt.Printf("Skipping %d files (MDX already exists)\n", skippedCount)
		}
	}

	if len(filesToProcess) == 0 {
		fmt.Println("No new files to process")
		return
	}

	fmt.Printf("Processing %d files\n", len(filesToProcess))

	if *dryRun {
		fmt.Println("\nDRY RUN MODE - No actual changes will be made\n")
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
	tmdbClient := metadata.NewClient(cfg.TMDB.APIKey, cfg.Options.RateLimitDelay)

	// Create MDX writer
	mdxWriter := writer.NewMDXWriter(cfg.Output.MDXDir, cfg.Output.CoversDir)

	// Process each file
	successCount := 0
	errorCount := 0

	for i, file := range filesToProcess {
		fmt.Printf("\n[%d/%d] Processing: %s\n", i+1, len(filesToProcess), file.FileName)

		if *verbose {
			fmt.Printf("  Extracted title: %s\n", file.Title)
			if file.Year > 0 {
				fmt.Printf("  Extracted year: %d\n", file.Year)
			}
		}

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
					if *verbose {
						fmt.Printf("  NFO error: %v, falling back to TMDB\n", err)
					}
					movie, err = tmdbClient.GetFullMovieData(file.Title, file.Year)
					metadataSource = "TMDB"
				}
			} else {
				metadataSource = "NFO"

				// Check for incomplete NFO data
				if cfg.Options.NFOFallbackTMDB && (movie.Title == "" || movie.ReleaseYear == 0) {
					if *verbose {
						fmt.Printf("  NFO incomplete, enriching with TMDB\n")
					}
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
			fmt.Printf("  ❌ Error fetching metadata: %v\n", err)
			errorCount++
			continue
		}

		// Add file information
		movie.FilePath = file.Path
		movie.FileName = file.FileName
		movie.FileSize = file.Size
		movie.Slug = file.Slug

		if *verbose {
			fmt.Printf("  Metadata source: %s\n", metadataSource)
			fmt.Printf("  Found: %s (%d)\n", movie.Title, movie.ReleaseYear)
			if movie.TMDBID > 0 {
				fmt.Printf("  TMDB ID: %d\n", movie.TMDBID)
			}
			if movie.Rating > 0 {
				fmt.Printf("  Rating: %.1f/10\n", movie.Rating)
			}
		}

		// Download cover image
		if cfg.Options.DownloadCovers {
			coverPath := mdxWriter.GetAbsoluteCoverPath(movie.Slug)
			movie.CoverImage = mdxWriter.GetCoverPath(movie.Slug)

			// Get poster path from TMDB (we need to search again to get the poster path)
			searchResult, _ := tmdbClient.SearchMovie(movie.Title, movie.ReleaseYear)
			if searchResult != nil && searchResult.PosterPath != "" {
				if err := tmdbClient.DownloadImage(searchResult.PosterPath, coverPath, "poster"); err != nil {
					if *verbose {
						fmt.Printf("  Warning: Failed to download cover: %v\n", err)
					}
				} else if *verbose {
					fmt.Printf("  ✓ Downloaded cover image\n")
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
					if *verbose {
						fmt.Printf("  Warning: Failed to download backdrop: %v\n", err)
					}
				} else if *verbose {
					fmt.Printf("  ✓ Downloaded backdrop image\n")
				}
			}
		}

		// Write MDX file
		if err := mdxWriter.WriteMDXFile(movie); err != nil {
			fmt.Printf("  ❌ Error writing MDX file: %v\n", err)
			errorCount++
			continue
		}

		fmt.Printf("  ✓ Created: %s.mdx\n", movie.Slug)
		successCount++
	}

	// Print summary
	fmt.Printf("\n" + repeat("=", 50) + "\n")
	fmt.Printf("Summary:\n")
	fmt.Printf("  Total files scanned: %d\n", len(files))
	fmt.Printf("  Files processed: %d\n", len(filesToProcess))
	fmt.Printf("  Successful: %d\n", successCount)
	if errorCount > 0 {
		fmt.Printf("  Errors: %d\n", errorCount)
	}

	// Build Astro site if enabled and not disabled via flag
	if cfg.Output.AutoBuild && !*noBuild && successCount > 0 {
		fmt.Println("\nBuilding Astro website...")
		if err := buildAstroSite(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to build Astro site: %v\n", err)
			fmt.Fprintf(os.Stderr, "You can build manually with: cd website && npm run build\n")
		} else {
			fmt.Println("✓ Astro site built successfully")
		}
	}

	if errorCount > 0 {
		os.Exit(1)
	}
}

// buildAstroSite runs the Astro build command
func buildAstroSite() error {
	websiteDir := "./website"

	// Check if website directory exists
	if _, err := os.Stat(websiteDir); os.IsNotExist(err) {
		return fmt.Errorf("website directory does not exist")
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
