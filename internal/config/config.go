package config

import (
	"fmt"
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
}

// TMDBConfig holds TMDB API configuration
type TMDBConfig struct {
	APIKey   string `yaml:"api_key"`
	Language string `yaml:"language"`
}

// ScannerConfig holds scanner settings
type ScannerConfig struct {
	Directories []string `yaml:"directories"`
	Extensions  []string `yaml:"extensions"`
	ExcludeDirs []string `yaml:"exclude_dirs"`
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

	return &cfg, nil
}
