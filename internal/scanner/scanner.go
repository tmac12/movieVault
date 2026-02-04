package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileInfo represents a scanned video file with extracted information
type FileInfo struct {
	Path       string
	FileName   string
	Title      string
	Year       int
	Size       int64
	Slug       string
	DiscNumber int  // Disc/part number extracted from filename (0 = not a multi-disc file)
	ShouldScan bool // Whether to scan this file (false if MDX already exists)
}

// SkippedDisc records a secondary disc that was filtered out by FilterMultiDiscDuplicates.
type SkippedDisc struct {
	FileName   string
	DiscNumber int
	KeptFile   string // FileName of the primary (disc 1) file that was kept
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
		discNumber := ExtractDiscNumber(info.Name())

		fileInfo := FileInfo{
			Path:       p,
			FileName:   info.Name(),
			Title:      title,
			Year:       year,
			Size:       info.Size(),
			Slug:       slug,
			DiscNumber: discNumber,
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

// discGroupKey is the grouping key for multi-disc files: same directory + same movie.
type discGroupKey struct {
	Dir   string
	Title string // normalized (lowercase, disc markers stripped)
	Year  int
}

// FilterMultiDiscDuplicates removes secondary discs (CD2+) when a CD1 sibling exists
// in the same directory for the same movie. Non-disc files and lone secondary discs
// (no CD1 present) are left untouched. Original order is preserved.
func FilterMultiDiscDuplicates(files []FileInfo) ([]FileInfo, []SkippedDisc) {
	// Separate disc files from everything else
	var nonDisc []FileInfo
	var discFiles []FileInfo
	for _, f := range files {
		if f.DiscNumber == 0 {
			nonDisc = append(nonDisc, f)
		} else {
			discFiles = append(discFiles, f)
		}
	}

	if len(discFiles) == 0 {
		return files, nil
	}

	// Group disc files by {Dir, NormalizedTitle, Year}
	groups := make(map[discGroupKey][]FileInfo)
	for _, f := range discFiles {
		key := discGroupKey{
			Dir:   filepath.Dir(f.Path),
			Title: normalizeTitle(f.Title),
			Year:  f.Year,
		}
		groups[key] = append(groups[key], f)
	}

	// Per group: keep only disc 1 if it exists; otherwise keep all
	keptDiscs := make(map[string]bool) // keyed by Path
	var skipped []SkippedDisc
	for _, group := range groups {
		// Find disc 1
		var disc1 *FileInfo
		for i := range group {
			if group[i].DiscNumber == 1 {
				disc1 = &group[i]
				break
			}
		}
		if disc1 == nil {
			// No disc 1 in group — keep everything (standalone discs)
			for _, f := range group {
				keptDiscs[f.Path] = true
			}
		} else {
			keptDiscs[disc1.Path] = true
			for _, f := range group {
				if f.DiscNumber != 1 {
					skipped = append(skipped, SkippedDisc{
						FileName:   f.FileName,
						DiscNumber: f.DiscNumber,
						KeptFile:   disc1.FileName,
					})
				}
			}
		}
	}

	// Rebuild output preserving original order: all nonDisc + kept disc files
	var result []FileInfo
	result = append(result, nonDisc...)
	for _, f := range discFiles {
		if keptDiscs[f.Path] {
			result = append(result, f)
		}
	}

	return result, skipped
}

// PrimarySiblingExists checks whether a disc-1 sibling for the given file exists
// in the same directory. Used by watch mode, which processes files one at a time.
func PrimarySiblingExists(file FileInfo, extensions []string) bool {
	entries, err := os.ReadDir(filepath.Dir(file.Path))
	if err != nil {
		return false
	}

	targetTitle := normalizeTitle(file.Title)
	targetYear := file.Year

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		supported := false
		for _, e := range extensions {
			if ext == strings.ToLower(e) {
				supported = true
				break
			}
		}
		if !supported {
			continue
		}
		// Skip the file itself
		if name == file.FileName {
			continue
		}
		t, y := ExtractTitleAndYear(name)
		if normalizeTitle(t) == targetTitle && y == targetYear && ExtractDiscNumber(name) == 1 {
			return true
		}
	}
	return false
}
