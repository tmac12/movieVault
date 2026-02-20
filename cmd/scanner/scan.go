package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/marco/movieVault/internal/config"
	"github.com/marco/movieVault/internal/metadata"
	"github.com/marco/movieVault/internal/metadata/nfo"
	"github.com/marco/movieVault/internal/scanner"
	"github.com/marco/movieVault/internal/writer"
)

// ScanResults holds the outcome of a scan operation
type ScanResults struct {
	TotalFiles     int
	ProcessedFiles int
	SuccessCount   int
	ErrorCount     int
	NFOCount       int
	TMDBCount      int
	MixedCount     int
	Duration       time.Duration
	Errors         []error
}

// runScan performs a full directory scan with concurrent processing
// This function is reusable by initial startup scans, scheduled scans, and future manual triggers
func runScan(
	ctx context.Context,
	cfg *config.Config,
	tmdbClient *metadata.Client,
	mdxWriter *writer.MDXWriter,
	forceRefresh bool,
	dryRun bool,
	verbose bool,
) *ScanResults {
	startTime := time.Now()
	results := &ScanResults{}

	// Create scanner with directory exclusions
	s := scanner.NewWithExclusions(cfg.Scanner.Extensions, cfg.Output.MDXDir, cfg.Scanner.ExcludeDirs)

	// Scan all directories
	slog.Info("scanning directories for video files", "count", len(cfg.Scanner.Directories))
	files, err := s.ScanAll(cfg.Scanner.Directories)
	if err != nil {
		slog.Error("failed to scan directories", "error", err)
		results.Errors = append(results.Errors, err)
		return results
	}

	slog.Info("scan complete", "files_found", len(files))
	results.TotalFiles = len(files)

	// Filter out secondary discs (CD2+) when CD1 exists in the same directory
	files, skippedDiscs := scanner.FilterMultiDiscDuplicates(files)
	for _, skip := range skippedDiscs {
		slog.Info("multi-disc: skipping secondary disc",
			"file", skip.FileName, "disc", skip.DiscNumber, "kept", skip.KeptFile)
	}

	// Filter files based on force-refresh flag
	var filesToProcess []scanner.FileInfo
	if forceRefresh {
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

	results.ProcessedFiles = len(filesToProcess)

	if len(filesToProcess) == 0 {
		slog.Info("no new files to process")
		results.Duration = time.Since(startTime)
		return results
	}

	slog.Info("processing files", "count", len(filesToProcess))

	if dryRun {
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
		results.Duration = time.Since(startTime)
		return results
	}

	// Progress reporter
	var processedCount int64
	totalFiles := int64(len(filesToProcess))
	progressDone := make(chan struct{})
	go func() {
		defer close(progressDone)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				current := atomic.LoadInt64(&processedCount)
				if current > 0 && current < totalFiles {
					slog.Info("progress", "processed", current, "total", totalFiles,
						"percent", fmt.Sprintf("%.0f%%", float64(current)/float64(totalFiles)*100))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Create SlugGuard for thread-safe slug deduplication
	slugGuard := scanner.NewSlugGuard()

	// Define per-file processing function
	processFn := func(ctx context.Context, file scanner.FileInfo) (string, string, error) {
		slog.Debug("file details",
			"title", file.Title,
			"year", file.Year,
			"path", file.Path,
		)

		// Fetch metadata from NFO or TMDB
		var movie *writer.Movie
		var err error
		var metadataSource string

		var tmdbLookupMethod string
		if cfg.Options.UseNFO {
			nfoParser := nfo.NewParser()
			movie, err = nfoParser.GetMovieFromNFO(file.Path)

			if err != nil {
				if cfg.Options.NFOFallbackTMDB {
					slog.Debug("metadata lookup",
						"file", file.FileName,
						"nfo_status", "not_found_or_error",
						"nfo_error", err.Error(),
						"action", "fallback_to_tmdb",
					)
					movie, err = tmdbClient.GetFullMovieData(file.Title, file.Year)
					metadataSource = "TMDB"
					tmdbLookupMethod = "search"
				}
			} else {
				metadataSource = "NFO"
				slog.Debug("metadata lookup",
					"file", file.FileName,
					"nfo_status", "found",
					"nfo_title", movie.Title,
					"nfo_tmdb_id", movie.TMDBID,
				)

				if movie.TMDBID > 0 && cfg.Options.NFOFallbackTMDB {
					slog.Debug("tmdb enrichment",
						"file", file.FileName,
						"method", "direct_id_lookup",
						"tmdb_id", movie.TMDBID,
					)
					tmdbMovie, tmdbErr := tmdbClient.GetMovieByID(movie.TMDBID)
					if tmdbErr != nil {
						if errors.Is(tmdbErr, metadata.ErrMovieNotFound) {
							slog.Debug("tmdb enrichment",
								"file", file.FileName,
								"method", "search_fallback",
								"reason", "direct_id_not_found",
								"tmdb_id", movie.TMDBID,
								"search_title", file.Title,
								"search_year", file.Year,
							)
							tmdbMovie, tmdbErr = tmdbClient.GetFullMovieData(file.Title, file.Year)
							tmdbLookupMethod = "search (fallback from direct)"
						}
					} else {
						tmdbLookupMethod = "direct ID"
					}
					if tmdbErr == nil && tmdbMovie != nil {
						movie = mergeMovieData(movie, tmdbMovie)
						metadataSource = "NFO+TMDB"
						slog.Debug("metadata merge",
							"file", file.FileName,
							"nfo_fields_kept", "title,year,rating,genres,director,cast",
							"tmdb_fields_filled", "missing_fields_only",
						)
					}
				} else if cfg.Options.NFOFallbackTMDB && (movie.Title == "" || movie.ReleaseYear == 0) {
					slog.Debug("tmdb enrichment",
						"file", file.FileName,
						"method", "search",
						"reason", "nfo_incomplete",
						"missing_title", movie.Title == "",
						"missing_year", movie.ReleaseYear == 0,
						"search_title", file.Title,
						"search_year", file.Year,
					)
					tmdbMovie, tmdbErr := tmdbClient.GetFullMovieData(file.Title, file.Year)
					tmdbLookupMethod = "search"
					if tmdbErr == nil && tmdbMovie != nil {
						movie = mergeMovieData(movie, tmdbMovie)
						metadataSource = "NFO+TMDB"
						slog.Debug("metadata merge",
							"file", file.FileName,
							"nfo_fields_kept", "available_nfo_data",
							"tmdb_fields_filled", "missing_fields",
						)
					}
				}
			}
		} else {
			slog.Debug("metadata lookup",
				"file", file.FileName,
				"nfo_status", "disabled",
				"action", "tmdb_search",
			)
			movie, err = tmdbClient.GetFullMovieData(file.Title, file.Year)
			metadataSource = "TMDB"
			tmdbLookupMethod = "search"
		}

		if tmdbLookupMethod != "" {
			slog.Debug("tmdb lookup completed",
				"file", file.FileName,
				"lookup_method", tmdbLookupMethod,
			)
		}

		if err != nil {
			return "", "", fmt.Errorf("failed to fetch metadata for %s: %w", file.FileName, err)
		}

		// Generate clean slug from metadata title (not from filename)
		movie.Slug = scanner.GenerateSlug(movie.Title, movie.ReleaseYear)

		// Thread-safe slug deduplication
		if !slugGuard.TryClaimSlug(movie.Slug) {
			slog.Info("skipping: slug already produced this run", "slug", movie.Slug, "file", file.FileName)
			return metadataSource, movie.Slug, nil
		}

		// Add file information
		movie.FilePath = file.Path
		movie.FileName = file.FileName
		movie.FileSize = file.Size
		movie.SourceDir = file.SourceDir

		slog.Info("metadata fetched",
			"movie", movie.Title,
			"year", movie.ReleaseYear,
			"source", metadataSource,
		)

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

			if cfg.Options.NFODownloadImages && movie.PosterURL != "" {
				slog.Debug("image download attempt",
					"file", file.FileName,
					"movie", movie.Title,
					"image_type", "cover",
					"source", "NFO",
					"url", movie.PosterURL,
				)
				if dlErr := tmdbClient.DownloadImageFromURL(movie.PosterURL, coverPath); dlErr != nil {
					slog.Debug("image download failed",
						"file", file.FileName,
						"movie", movie.Title,
						"image_type", "cover",
						"source", "NFO",
						"error", dlErr.Error(),
						"action", "fallback_to_tmdb",
					)
				} else {
					coverDownloaded = true
					coverSource = "NFO"
				}
			}

			if !coverDownloaded {
				slog.Debug("image download attempt",
					"file", file.FileName,
					"movie", movie.Title,
					"image_type", "cover",
					"source", "TMDB",
				)
				var tmdbPosterPath string
				if movie.TMDBID > 0 {
					if details, detErr := tmdbClient.GetMovieDetails(movie.TMDBID); detErr == nil && details.PosterPath != "" {
						tmdbPosterPath = details.PosterPath
					}
				}
				if tmdbPosterPath == "" {
					if searchResult, searchErr := tmdbClient.SearchMovie(movie.Title, movie.ReleaseYear); searchErr == nil && searchResult != nil {
						tmdbPosterPath = searchResult.PosterPath
					}
				}
				if tmdbPosterPath != "" {
					if dlErr := tmdbClient.DownloadImage(tmdbPosterPath, coverPath, "poster"); dlErr != nil {
						slog.Warn("image download failed",
							"file", file.FileName,
							"movie", movie.Title,
							"image_type", "cover",
							"source", "TMDB",
							"error", dlErr,
						)
					} else {
						coverDownloaded = true
						coverSource = "TMDB"
					}
				} else {
					slog.Debug("image not available",
						"file", file.FileName,
						"movie", movie.Title,
						"image_type", "cover",
						"reason", "no_poster_path_in_tmdb",
					)
				}
			}

			if coverDownloaded {
				slog.Debug("image download success",
					"file", file.FileName,
					"movie", movie.Title,
					"image_type", "cover",
					"source", coverSource,
					"path", coverPath,
				)
			}
		}

		// Download backdrop image
		if cfg.Options.DownloadBackdrops {
			backdropPath := mdxWriter.GetAbsoluteBackdropPath(movie.Slug)
			movie.BackdropImage = mdxWriter.GetBackdropPath(movie.Slug)

			backdropDownloaded := false
			backdropSource := ""

			if cfg.Options.NFODownloadImages && movie.BackdropURL != "" {
				slog.Debug("image download attempt",
					"file", file.FileName,
					"movie", movie.Title,
					"image_type", "backdrop",
					"source", "NFO",
					"url", movie.BackdropURL,
				)
				if dlErr := tmdbClient.DownloadImageFromURL(movie.BackdropURL, backdropPath); dlErr != nil {
					slog.Debug("image download failed",
						"file", file.FileName,
						"movie", movie.Title,
						"image_type", "backdrop",
						"source", "NFO",
						"error", dlErr.Error(),
						"action", "fallback_to_tmdb",
					)
				} else {
					backdropDownloaded = true
					backdropSource = "NFO"
				}
			}

			if !backdropDownloaded {
				slog.Debug("image download attempt",
					"file", file.FileName,
					"movie", movie.Title,
					"image_type", "backdrop",
					"source", "TMDB",
				)
				var tmdbBackdropPath string
				if movie.TMDBID > 0 {
					if details, detErr := tmdbClient.GetMovieDetails(movie.TMDBID); detErr == nil && details.BackdropPath != "" {
						tmdbBackdropPath = details.BackdropPath
					}
				}
				if tmdbBackdropPath == "" {
					if searchResult, searchErr := tmdbClient.SearchMovie(movie.Title, movie.ReleaseYear); searchErr == nil && searchResult != nil {
						tmdbBackdropPath = searchResult.BackdropPath
					}
				}
				if tmdbBackdropPath != "" {
					if dlErr := tmdbClient.DownloadImage(tmdbBackdropPath, backdropPath, "backdrop"); dlErr != nil {
						slog.Warn("image download failed",
							"file", file.FileName,
							"movie", movie.Title,
							"image_type", "backdrop",
							"source", "TMDB",
							"error", dlErr,
						)
					} else {
						backdropDownloaded = true
						backdropSource = "TMDB"
					}
				} else {
					slog.Debug("image not available",
						"file", file.FileName,
						"movie", movie.Title,
						"image_type", "backdrop",
						"reason", "no_backdrop_path_in_tmdb",
					)
				}
			}

			if backdropDownloaded {
				slog.Debug("image download success",
					"file", file.FileName,
					"movie", movie.Title,
					"image_type", "backdrop",
					"source", backdropSource,
					"path", backdropPath,
				)
			}
		}

		// Write MDX file
		if err := mdxWriter.WriteMDXFile(movie); err != nil {
			return metadataSource, movie.Slug, fmt.Errorf("failed to write mdx for %s: %w", movie.Title, err)
		}

		slog.Info("mdx file created", "slug", movie.Slug)
		return metadataSource, movie.Slug, nil
	}

	// Run concurrent processing
	processResults := scanner.ProcessFilesConcurrently(ctx, filesToProcess, processFn, cfg.Scanner.ConcurrentWorkers, &processedCount)

	// Stop progress reporter (use a separate context for graceful shutdown)
	close(progressDone)
	<-progressDone

	// Aggregate results
	for _, r := range processResults {
		if r.Err != nil {
			slog.Error("failed to process file",
				"filename", r.File.FileName,
				"error", r.Err,
			)
			results.ErrorCount++
			results.Errors = append(results.Errors, r.Err)
			continue
		}
		// Files that were slug-duplicates (TryClaimSlug returned false) get
		// a non-empty Slug but still succeed — they just don't produce output.
		// We count them as successful.
		results.SuccessCount++
		switch r.MetadataSource {
		case "NFO":
			results.NFOCount++
		case "TMDB":
			results.TMDBCount++
		case "NFO+TMDB":
			results.MixedCount++
		}
	}

	results.Duration = time.Since(startTime)

	// Print summary
	slog.Info("scan complete",
		"total_files", results.TotalFiles,
		"processed", results.ProcessedFiles,
		"successful", results.SuccessCount,
		"errors", results.ErrorCount,
		"duration_sec", results.Duration.Seconds(),
	)

	// Show metadata source breakdown
	if results.SuccessCount > 0 {
		slog.Info("metadata sources",
			"nfo_count", results.NFOCount,
			"nfo_percent", fmt.Sprintf("%.0f%%", float64(results.NFOCount)/float64(results.SuccessCount)*100),
			"tmdb_count", results.TMDBCount,
			"tmdb_percent", fmt.Sprintf("%.0f%%", float64(results.TMDBCount)/float64(results.SuccessCount)*100),
			"mixed_count", results.MixedCount,
			"mixed_percent", fmt.Sprintf("%.0f%%", float64(results.MixedCount)/float64(results.SuccessCount)*100),
		)
	}

	return results
}
