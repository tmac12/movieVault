package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Quality extraction patterns for US-025
var (
	// Resolution patterns for quality extraction
	resolutionExtractPattern = regexp.MustCompile(`(?i)\b(2160p|1080p|1080i|720p|720i|480p|4K)\b`)
	// Source quality patterns for quality extraction
	sourceExtractPattern = regexp.MustCompile(`(?i)\b(BluRay|BRRip|BDRip|WEB-DL|WEBRip|HDRip|DVDRip|HDTV|WEB|CAM|TS|TC|DVDSCR|R5|SCREENER)\b`)
)

// Resolution quality ranking (higher is better)
var resolutionRank = map[string]int{
	"2160p": 4,
	"4k":    4,
	"1080p": 3,
	"1080i": 2,
	"720p":  2,
	"720i":  1,
	"480p":  1,
	"":      0, // unknown
}

// Source quality ranking (higher is better)
var sourceRank = map[string]int{
	"bluray":  8,
	"bdrip":   7,
	"brrip":   7,
	"web-dl":  6,
	"webrip":  5,
	"hdrip":   4,
	"hdtv":    4,
	"dvdrip":  3,
	"dvdscr":  2,
	"screener": 2,
	"r5":      2,
	"ts":      1,
	"tc":      1,
	"cam":     0,
	"":        -1, // unknown
}

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
	// Quality fields (US-025)
	Resolution     string // e.g., "1080p", "2160p", "720p"
	Source         string // e.g., "BluRay", "WEB-DL", "HDRip"
	QualityScore   int    // Combined quality score for ranking
	IsRecommended  bool   // True if this is the recommended copy to keep
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
			// Mark recommended copy (US-025)
			markRecommended(movieList)
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
			// Mark recommended copy (US-025)
			markRecommended(movieList)
			duplicates = append(duplicates, DuplicateSet{
				Key:     key,
				KeyType: "title_year",
				Movies:  movieList,
			})
		}
	}

	return duplicates, nil
}

// markRecommended marks the highest quality copy as recommended (US-025)
func markRecommended(movies []DuplicateMovie) {
	if len(movies) == 0 {
		return
	}

	// Find the movie with highest quality score
	bestIdx := 0
	bestScore := movies[0].QualityScore

	for i := 1; i < len(movies); i++ {
		if movies[i].QualityScore > bestScore {
			bestScore = movies[i].QualityScore
			bestIdx = i
		}
	}

	// Mark the best one
	movies[bestIdx].IsRecommended = true
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

	// Extract quality info from filename (US-025)
	resolution, source := extractQualityInfo(fm.FileName)
	qualityScore := calculateQualityScore(resolution, source)

	return DuplicateMovie{
		Title:        fm.Title,
		ReleaseYear:  fm.ReleaseYear,
		TMDBID:       fm.TMDBID,
		FilePath:     fm.FilePath,
		FileName:     fm.FileName,
		Slug:         fm.Slug,
		Resolution:   resolution,
		Source:       source,
		QualityScore: qualityScore,
	}, nil
}

// extractQualityInfo extracts resolution and source quality from a filename (US-025)
func extractQualityInfo(filename string) (resolution string, source string) {
	// Extract resolution
	if match := resolutionExtractPattern.FindString(filename); match != "" {
		resolution = strings.ToLower(match)
		// Normalize 4K to 2160p for consistent comparison
		if resolution == "4k" {
			resolution = "2160p"
		}
	}

	// Extract source quality
	if match := sourceExtractPattern.FindString(filename); match != "" {
		source = match // Preserve original case for display
	}

	return resolution, source
}

// calculateQualityScore computes a combined quality score (US-025)
// Higher score = better quality
func calculateQualityScore(resolution, source string) int {
	// Get resolution rank (0-4)
	resRank := resolutionRank[strings.ToLower(resolution)]

	// Get source rank (0-8)
	srcRank := sourceRank[strings.ToLower(source)]
	if srcRank < 0 {
		srcRank = 0 // Treat unknown as 0 for scoring
	}

	// Combined score: resolution has higher weight (multiply by 10)
	// This ensures 2160p WEB-DL (46) > 1080p BluRay (38)
	// Adjust weights: resolution * 10 + source * 1
	return resRank*10 + srcRank
}

// PrintDuplicateReport outputs a formatted report of duplicates
// If detailed is true, shows full quality breakdown for each file (US-025)
func PrintDuplicateReport(duplicates []DuplicateSet, detailed bool) {
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
			// Show recommendation marker (US-025)
			recommendMarker := ""
			if movie.IsRecommended {
				recommendMarker = " ★ RECOMMENDED"
			}
			fmt.Printf("  [%d] %s (%d)%s\n", j+1, movie.Title, movie.ReleaseYear, recommendMarker)
			fmt.Printf("      File: %s\n", movie.FileName)

			// Show quality info (US-025)
			qualityStr := formatQualityString(movie.Resolution, movie.Source)
			if qualityStr != "" {
				fmt.Printf("      Quality: %s\n", qualityStr)
			}

			// Show detailed breakdown if requested (US-025)
			if detailed {
				fmt.Printf("      Path: %s\n", movie.FilePath)
				fmt.Printf("      Slug: %s\n", movie.Slug)
				fmt.Printf("      Resolution: %s (rank: %d)\n", displayResolution(movie.Resolution), resolutionRank[strings.ToLower(movie.Resolution)])
				fmt.Printf("      Source: %s (rank: %d)\n", displaySource(movie.Source), sourceRank[strings.ToLower(movie.Source)])
				fmt.Printf("      Quality Score: %d\n", movie.QualityScore)
			} else {
				fmt.Printf("      Path: %s\n", movie.FilePath)
				fmt.Printf("      Slug: %s\n", movie.Slug)
			}

			if movie.TMDBID > 0 {
				fmt.Printf("      TMDB: https://www.themoviedb.org/movie/%d\n", movie.TMDBID)
			}
			fmt.Println()
		}
	}
}

// formatQualityString creates a display string for resolution and source (US-025)
func formatQualityString(resolution, source string) string {
	parts := []string{}
	if resolution != "" {
		parts = append(parts, strings.ToUpper(resolution))
	}
	if source != "" {
		parts = append(parts, source)
	}
	if len(parts) == 0 {
		return "Unknown"
	}
	return strings.Join(parts, " ")
}

// displayResolution returns a display string for resolution (US-025)
func displayResolution(resolution string) string {
	if resolution == "" {
		return "Unknown"
	}
	return strings.ToUpper(resolution)
}

// displaySource returns a display string for source (US-025)
func displaySource(source string) string {
	if source == "" {
		return "Unknown"
	}
	return source
}
