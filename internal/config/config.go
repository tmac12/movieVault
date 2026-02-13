package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	TMDB    TMDBConfig    `yaml:"tmdb"`
	Scanner ScannerConfig `yaml:"scanner"`
	Output  OutputConfig  `yaml:"output"`
	Options OptionsConfig `yaml:"options"`
	Retry   RetryConfig   `yaml:"retry"`
	Cache   CacheConfig   `yaml:"cache"`
}

// TMDBConfig holds TMDB API configuration
type TMDBConfig struct {
	APIKey   string `yaml:"api_key"`
	Language string `yaml:"language"`
}

// ScannerConfig holds scanner settings
type ScannerConfig struct {
	Directories       []string `yaml:"directories"`
	Extensions        []string `yaml:"extensions"`
	ExcludeDirs       []string `yaml:"exclude_dirs"`
	ConcurrentWorkers int      `yaml:"concurrent_workers"` // Number of concurrent workers for parallel scanning (default: 5)
	WatchMode         bool     `yaml:"watch_mode"`         // Enable watch mode to monitor directories for changes (default: false)
	WatchDebounce     int      `yaml:"watch_debounce"`     // Seconds to wait after file change before processing (default: 30)
	WatchRecursive    *bool    `yaml:"watch_recursive"`    // Watch subdirectories recursively (default: true, use pointer to detect nil)
	ScheduleEnabled   bool     `yaml:"schedule_enabled"`   // Enable scheduled scans (default: false)
	ScheduleInterval  int      `yaml:"schedule_interval"`  // Minutes between scans (default: 60)
	ScheduleOnStartup *bool    `yaml:"schedule_on_startup"` // Run on startup (default: true, use pointer to detect nil)
}

// OutputConfig holds output directory settings
type OutputConfig struct {
	MDXDir         string `yaml:"mdx_dir"`
	CoversDir      string `yaml:"covers_dir"`
	WebsiteDir     string `yaml:"website_dir"`
	AutoBuild      bool   `yaml:"auto_build"`
	CleanupMissing bool   `yaml:"cleanup_missing"`
}

// OptionsConfig holds additional options
type OptionsConfig struct {
	RateLimitDelay    int  `yaml:"rate_limit_delay"`
	DownloadCovers    bool `yaml:"download_covers"`
	DownloadBackdrops bool `yaml:"download_backdrops"`
	UseNFO            bool `yaml:"use_nfo"`
	NFOFallbackTMDB   bool `yaml:"nfo_fallback_tmdb"`
	NFODownloadImages bool `yaml:"nfo_download_images"` // Download images from NFO URLs when available (default: false)
}

// RetryConfig holds retry behavior configuration
type RetryConfig struct {
	MaxAttempts      int `yaml:"max_attempts"`
	InitialBackoffMs int `yaml:"initial_backoff_ms"`
}

// CacheConfig holds cache behavior configuration
type CacheConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
	TTLDays int    `yaml:"ttl_days"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	// Expand ~ to home directory if present
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Read the config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables in the YAML content
	expandedData := os.ExpandEnv(string(data))

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal([]byte(expandedData), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate required fields
	if cfg.TMDB.APIKey == "" || cfg.TMDB.APIKey == "your_api_key_here" {
		return nil, fmt.Errorf("TMDB API key is required. Get one from https://www.themoviedb.org/settings/api")
	}

	// Set default language if not specified
	if cfg.TMDB.Language == "" {
		cfg.TMDB.Language = "en-US"
	}

	// Set default retry settings
	if cfg.Retry.MaxAttempts == 0 {
		cfg.Retry.MaxAttempts = 3
	}
	if cfg.Retry.InitialBackoffMs == 0 {
		cfg.Retry.InitialBackoffMs = 1000
	}

	// Set default cache settings
	// Default Path is always set; if user provides no cache section, we also default Enabled to true.
	// If user explicitly sets enabled: false with a custom path, we respect that.
	if cfg.Cache.Path == "" {
		cfg.Cache.Path = "./data/cache.db"
		// Only default Enabled to true if the entire cache section was omitted (Path was empty)
		cfg.Cache.Enabled = true
	}
	if cfg.Cache.TTLDays == 0 {
		cfg.Cache.TTLDays = 30
	}

	// Set default concurrent workers
	if cfg.Scanner.ConcurrentWorkers == 0 {
		cfg.Scanner.ConcurrentWorkers = 5
	}

	// Set default watch settings
	// WatchMode defaults to false (Go zero value) - no explicit set needed
	if cfg.Scanner.WatchDebounce == 0 {
		cfg.Scanner.WatchDebounce = 30
	}
	// WatchRecursive defaults to true. We use *bool to distinguish "not set" from "explicitly false".
	if cfg.Scanner.WatchRecursive == nil {
		defaultTrue := true
		cfg.Scanner.WatchRecursive = &defaultTrue
	}

	// Set default schedule settings
	// ScheduleEnabled defaults to false (Go zero value) - no explicit set needed
	if cfg.Scanner.ScheduleInterval == 0 {
		cfg.Scanner.ScheduleInterval = 60
	}
	// ScheduleOnStartup defaults to true. We use *bool to distinguish "not set" from "explicitly false".
	if cfg.Scanner.ScheduleOnStartup == nil {
		defaultTrue := true
		cfg.Scanner.ScheduleOnStartup = &defaultTrue
	}

	if len(cfg.Scanner.Directories) == 0 {
		return nil, fmt.Errorf("at least one scan directory is required")
	}

	if cfg.Output.MDXDir == "" {
		return nil, fmt.Errorf("mdx_dir is required")
	}

	if cfg.Output.CoversDir == "" {
		return nil, fmt.Errorf("covers_dir is required")
	}

	// Ensure output directories exist
	if err := os.MkdirAll(cfg.Output.MDXDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create MDX directory: %w", err)
	}

	if err := os.MkdirAll(cfg.Output.CoversDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create covers directory: %w", err)
	}

	// US-028: Validate configuration options
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validate performs validation on configuration options (US-028)
func (cfg *Config) validate() error {
	// Validate concurrent_workers is positive
	if cfg.Scanner.ConcurrentWorkers < 1 {
		return fmt.Errorf("scanner.concurrent_workers must be at least 1 (got %d)", cfg.Scanner.ConcurrentWorkers)
	}
	if cfg.Scanner.ConcurrentWorkers > 20 {
		slog.Warn("high concurrent_workers value may cause TMDB rate limit issues", "workers", cfg.Scanner.ConcurrentWorkers)
	}

	// Validate retry.max_attempts is positive
	if cfg.Retry.MaxAttempts <= 0 {
		return fmt.Errorf("retry.max_attempts must be positive (got %d)", cfg.Retry.MaxAttempts)
	}

	// Validate retry.initial_backoff_ms is positive
	if cfg.Retry.InitialBackoffMs <= 0 {
		return fmt.Errorf("retry.initial_backoff_ms must be positive (got %d)", cfg.Retry.InitialBackoffMs)
	}

	// Validate cache path parent directory exists and is writable when cache is enabled
	if cfg.Cache.Enabled {
		cacheParentDir := filepath.Dir(cfg.Cache.Path)
		if cacheParentDir != "" && cacheParentDir != "." {
			// Try to create parent directory if it doesn't exist
			if err := os.MkdirAll(cacheParentDir, 0755); err != nil {
				return fmt.Errorf("cache path parent directory is not writable: %s (%w)", cacheParentDir, err)
			}
			// Check if the directory is writable by attempting to create a temp file
			testFile := filepath.Join(cacheParentDir, ".write-test")
			f, err := os.Create(testFile)
			if err != nil {
				return fmt.Errorf("cache path parent directory is not writable: %s (%w)", cacheParentDir, err)
			}
			f.Close()
			os.Remove(testFile)
		}
	}

	// Warn if nfo_download_images: true but use_nfo: false
	if cfg.Options.NFODownloadImages && !cfg.Options.UseNFO {
		slog.Warn("nfo_download_images is enabled but use_nfo is disabled; NFO image URLs will not be available")
	}

	// Warn if watch_mode: true but no directories configured
	if cfg.Scanner.WatchMode && len(cfg.Scanner.Directories) == 0 {
		slog.Warn("watch_mode is enabled but no directories are configured; nothing to watch")
	}

	// Validate cache TTL is positive when cache is enabled
	if cfg.Cache.Enabled && cfg.Cache.TTLDays <= 0 {
		return fmt.Errorf("cache.ttl_days must be positive when cache is enabled (got %d)", cfg.Cache.TTLDays)
	}

	// Validate schedule settings
	if cfg.Scanner.ScheduleEnabled {
		if cfg.Scanner.ScheduleInterval <= 0 {
			return fmt.Errorf("scanner.schedule_interval must be positive when schedule_enabled is true (got %d)", cfg.Scanner.ScheduleInterval)
		}
		if cfg.Scanner.ScheduleInterval < 5 {
			slog.Warn("very frequent scheduled scans may cause high CPU/API usage",
				"interval_minutes", cfg.Scanner.ScheduleInterval,
				"suggestion", "consider using watch mode for real-time scanning or increasing interval")
		}
	}

	// Info log if both watch and schedule are enabled (not an error, just informational)
	if cfg.Scanner.WatchMode && cfg.Scanner.ScheduleEnabled {
		slog.Info("both watch mode and scheduled scanning enabled",
			"watch_mode", "immediate processing on file changes",
			"schedule_mode", fmt.Sprintf("periodic scans every %d minutes", cfg.Scanner.ScheduleInterval))
	}

	return nil
}
