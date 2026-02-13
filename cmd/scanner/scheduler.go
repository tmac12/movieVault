package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/marco/movieVault/internal/config"
	"github.com/marco/movieVault/internal/metadata"
	"github.com/marco/movieVault/internal/writer"
)

// Global state for overlap prevention
var scanInProgress atomic.Bool

// startScheduler starts the scheduled scanning service
// Runs periodic scans at configured intervals, optionally running immediately on startup
func startScheduler(
	ctx context.Context,
	cfg *config.Config,
	tmdbClient *metadata.Client,
	mdxWriter *writer.MDXWriter,
	verbose bool,
) {
	interval := time.Duration(cfg.Scanner.ScheduleInterval) * time.Minute

	slog.Info("scheduled scanning started",
		"interval_minutes", cfg.Scanner.ScheduleInterval,
		"run_on_startup", *cfg.Scanner.ScheduleOnStartup,
	)

	// Run initial scan on startup if enabled
	if *cfg.Scanner.ScheduleOnStartup {
		slog.Info("running initial scheduled scan on startup")
		runScheduledScan(ctx, cfg, tmdbClient, mdxWriter, verbose)
	}

	// Create ticker for periodic scans
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Main scheduler loop
	for {
		select {
		case <-ticker.C:
			slog.Info("scheduled scan triggered",
				"interval_minutes", cfg.Scanner.ScheduleInterval,
			)
			runScheduledScan(ctx, cfg, tmdbClient, mdxWriter, verbose)

		case <-ctx.Done():
			slog.Info("scheduled scanning stopped")
			return
		}
	}
}

// runScheduledScan performs a single scheduled scan with overlap prevention
func runScheduledScan(
	ctx context.Context,
	cfg *config.Config,
	tmdbClient *metadata.Client,
	mdxWriter *writer.MDXWriter,
	verbose bool,
) {
	// Try to claim the scan lock atomically
	if !scanInProgress.CompareAndSwap(false, true) {
		slog.Warn("scheduled scan skipped: previous scan still running",
			"interval_minutes", cfg.Scanner.ScheduleInterval,
			"suggestion", "consider increasing schedule_interval")
		return
	}

	// Ensure flag is reset even on panic
	defer scanInProgress.Store(false)

	startTime := time.Now()
	slog.Info("scheduled scan started")

	// Run incremental scan (forceRefresh=false, dryRun=false)
	results := runScan(ctx, cfg, tmdbClient, mdxWriter, false, false, verbose)

	// Log completion with results
	slog.Info("scheduled scan completed",
		"duration_sec", results.Duration.Seconds(),
		"files_processed", results.ProcessedFiles,
		"successful", results.SuccessCount,
		"errors", results.ErrorCount,
	)

	// Trigger Astro build if enabled and files were successfully processed
	if cfg.Output.AutoBuild && results.SuccessCount > 0 {
		slog.Info("triggering astro build after scheduled scan")
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
	} else if results.ProcessedFiles == 0 {
		slog.Debug("scheduled scan: no new files to process")
	}

	slog.Info("scheduled scan cycle complete", "total_time_sec", time.Since(startTime).Seconds())
}
