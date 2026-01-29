package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileInfo represents a scanned video file with extracted information
type FileInfo struct {
	Path      string
	FileName  string
	Title     string
	Year      int
	Size      int64
	Slug      string
	ShouldScan bool // Whether to scan this file (false if MDX already exists)
}

// Scanner handles file system scanning for video files
type Scanner struct {
	extensions  []string
	mdxDir      string
	excludeDirs []string
}

// New creates a new Scanner instance
func New(extensions []string, mdxDir string) *Scanner {
	return &Scanner{
		extensions:  extensions,
		mdxDir:      mdxDir,
		excludeDirs: []string{},
	}
}

// NewWithExclusions creates a new Scanner instance with directory exclusions
func NewWithExclusions(extensions []string, mdxDir string, excludeDirs []string) *Scanner {
	return &Scanner{
		extensions:  extensions,
		mdxDir:      mdxDir,
		excludeDirs: excludeDirs,
	}
}

// IsExcludedDir checks if a directory should be excluded based on exclusion patterns
func (s *Scanner) IsExcludedDir(dirPath string) bool {
	dirName := strings.ToLower(filepath.Base(dirPath))

	for _, pattern := range s.excludeDirs {
		pattern = strings.ToLower(pattern)
		// Check for exact match or if pattern is contained in directory name
		if dirName == pattern || strings.Contains(dirName, pattern) {
			return true
		}
	}
	return false
}

// ScanDirectory recursively scans a directory for video files
func (s *Scanner) ScanDirectory(path string) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories we can't read
			if os.IsPermission(err) {
				return nil
			}
			return err
		}

		// Skip excluded directories
		if info.IsDir() {
			if s.IsExcludedDir(p) {
				fmt.Printf("Skipping excluded directory: %s\n", p)
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a media file
		if !s.IsMediaFile(info.Name()) {
			return nil
		}

		// Extract movie information from filename
		title, year := ExtractTitleAndYear(info.Name())
		slug := GenerateSlug(title, year)

		fileInfo := FileInfo{
			Path:       p,
			FileName:   info.Name(),
			Title:      title,
			Year:       year,
			Size:       info.Size(),
			Slug:       slug,
			ShouldScan: !s.MDXExists(slug),
		}

		files = append(files, fileInfo)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory %s: %w", path, err)
	}

	return files, nil
}

// IsMediaFile checks if a filename has a supported video extension
func (s *Scanner) IsMediaFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, validExt := range s.extensions {
		if ext == strings.ToLower(validExt) {
			return true
		}
	}
	return false
}

// MDXExists checks if an MDX file already exists for a given slug
func (s *Scanner) MDXExists(slug string) bool {
	mdxPath := filepath.Join(s.mdxDir, slug+".mdx")
	_, err := os.Stat(mdxPath)
	return err == nil
}

// ScanAll scans all directories and returns combined results
func (s *Scanner) ScanAll(directories []string) ([]FileInfo, error) {
	var allFiles []FileInfo

	for _, dir := range directories {
		// Check if directory exists
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Printf("Warning: Directory does not exist: %s\n", dir)
			continue
		}

		files, err := s.ScanDirectory(dir)
		if err != nil {
			return nil, err
		}

		allFiles = append(allFiles, files...)
	}

	return allFiles, nil
}
