package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DuplicateSet represents a group of movies that are duplicates of each other
type DuplicateSet struct {
	Key     string         // The grouping key (TMDB ID or title+year)
	KeyType string         // "tmdb_id" or "title_year"
	Movies  []DuplicateMovie
}

// DuplicateMovie represents a single movie entry in a duplicate set
type DuplicateMovie struct {
	Title       string
	ReleaseYear int
	TMDBID      int
	FilePath    string
	FileName    string
	Slug        string
	MDXPath     string
}

// mdxFrontmatter represents the YAML frontmatter structure in MDX files
type mdxFrontmatter struct {
	Title       string `yaml:"title"`
	Slug        string `yaml:"slug"`
	ReleaseYear int    `yaml:"releaseYear"`
	TMDBID      int    `yaml:"tmdbId"`
	FilePath    string `yaml:"filePath"`
	FileName    string `yaml:"fileName"`
}

// DuplicateFinder handles finding duplicate movies in the library
type DuplicateFinder struct {
	mdxDir string
}

// NewDuplicateFinder creates a new DuplicateFinder instance
func NewDuplicateFinder(mdxDir string) *DuplicateFinder {
	return &DuplicateFinder{
		mdxDir: mdxDir,
	}
}

// FindDuplicates scans all MDX files and returns groups of duplicates
func (df *DuplicateFinder) FindDuplicates() ([]DuplicateSet, error) {
	// Read all MDX files
	movies, err := df.readAllMDXFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to read MDX files: %w", err)
	}

	// Group movies by TMDB ID
	tmdbGroups := make(map[int][]DuplicateMovie)
	// Group movies without TMDB ID by title+year
	titleYearGroups := make(map[string][]DuplicateMovie)

	for _, movie := range movies {
		if movie.TMDBID > 0 {
			tmdbGroups[movie.TMDBID] = append(tmdbGroups[movie.TMDBID], movie)
		} else {
			// Create key from lowercase title + year for matching
			key := fmt.Sprintf("%s|%d", strings.ToLower(movie.Title), movie.ReleaseYear)
			titleYearGroups[key] = append(titleYearGroups[key], movie)
		}
	}

	// Build duplicate sets (only groups with more than 1 movie)
	var duplicates []DuplicateSet

	// Process TMDB ID groups
	for tmdbID, movieList := range tmdbGroups {
		if len(movieList) > 1 {
			duplicates = append(duplicates, DuplicateSet{
				Key:     fmt.Sprintf("%d", tmdbID),
				KeyType: "tmdb_id",
				Movies:  movieList,
			})
		}
	}

	// Process title+year groups
	for key, movieList := range titleYearGroups {
		if len(movieList) > 1 {
			duplicates = append(duplicates, DuplicateSet{
				Key:     key,
				KeyType: "title_year",
				Movies:  movieList,
			})
		}
	}

	return duplicates, nil
}

// readAllMDXFiles reads all MDX files in the directory and extracts frontmatter
func (df *DuplicateFinder) readAllMDXFiles() ([]DuplicateMovie, error) {
	var movies []DuplicateMovie

	// Check if MDX directory exists
	if _, err := os.Stat(df.mdxDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("MDX directory does not exist: %s", df.mdxDir)
	}

	// Find all .mdx files
	pattern := filepath.Join(df.mdxDir, "*.mdx")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob MDX files: %w", err)
	}

	for _, mdxPath := range files {
		movie, err := df.parseMDXFile(mdxPath)
		if err != nil {
			// Log warning but continue processing other files
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse %s: %v\n", mdxPath, err)
			continue
		}
		movie.MDXPath = mdxPath
		movies = append(movies, movie)
	}

	return movies, nil
}

// parseMDXFile extracts frontmatter from a single MDX file
func (df *DuplicateFinder) parseMDXFile(mdxPath string) (DuplicateMovie, error) {
	content, err := os.ReadFile(mdxPath)
	if err != nil {
		return DuplicateMovie{}, fmt.Errorf("failed to read file: %w", err)
	}

	// Extract YAML frontmatter between --- markers
	contentStr := string(content)
	if !strings.HasPrefix(contentStr, "---") {
		return DuplicateMovie{}, fmt.Errorf("no frontmatter found")
	}

	// Find the closing --- delimiter
	endIndex := strings.Index(contentStr[3:], "---")
	if endIndex == -1 {
		return DuplicateMovie{}, fmt.Errorf("frontmatter not properly closed")
	}

	frontmatterYAML := contentStr[3 : endIndex+3]

	var fm mdxFrontmatter
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &fm); err != nil {
		return DuplicateMovie{}, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return DuplicateMovie{
		Title:       fm.Title,
		ReleaseYear: fm.ReleaseYear,
		TMDBID:      fm.TMDBID,
		FilePath:    fm.FilePath,
		FileName:    fm.FileName,
		Slug:        fm.Slug,
	}, nil
}

// PrintReport outputs a formatted report of duplicates
func PrintDuplicateReport(duplicates []DuplicateSet) {
	if len(duplicates) == 0 {
		fmt.Println("No duplicates found.")
		return
	}

	fmt.Printf("Found %d duplicate set(s):\n\n", len(duplicates))

	for i, set := range duplicates {
		fmt.Printf("━━━ Duplicate Set %d ━━━\n", i+1)

		// Print grouping info
		if set.KeyType == "tmdb_id" {
			fmt.Printf("TMDB ID: %s\n", set.Key)
		} else {
			parts := strings.Split(set.Key, "|")
			if len(parts) == 2 {
				fmt.Printf("Title: %s, Year: %s\n", parts[0], parts[1])
			}
		}

		fmt.Printf("Copies: %d\n\n", len(set.Movies))

		// Print each movie in the set
		for j, movie := range set.Movies {
			fmt.Printf("  [%d] %s (%d)\n", j+1, movie.Title, movie.ReleaseYear)
			fmt.Printf("      File: %s\n", movie.FileName)
			fmt.Printf("      Path: %s\n", movie.FilePath)
			fmt.Printf("      Slug: %s\n", movie.Slug)
			if movie.TMDBID > 0 {
				fmt.Printf("      TMDB: https://www.themoviedb.org/movie/%d\n", movie.TMDBID)
			}
			fmt.Println()
		}
	}
}
