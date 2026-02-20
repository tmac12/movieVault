package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// MDXWriter handles writing movie data to MDX files
type MDXWriter struct {
	mdxDir     string
	coversDir  string
}

// NewMDXWriter creates a new MDX writer
func NewMDXWriter(mdxDir, coversDir string) *MDXWriter {
	return &MDXWriter{
		mdxDir:    mdxDir,
		coversDir: coversDir,
	}
}

// WriteMDXFile writes a movie to an MDX file
func (w *MDXWriter) WriteMDXFile(movie *Movie) error {
	// Generate MDX content
	content, err := w.GenerateMDX(movie)
	if err != nil {
		return fmt.Errorf("failed to generate MDX: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(w.mdxDir, 0755); err != nil {
		return fmt.Errorf("failed to create MDX directory: %w", err)
	}

	// Write to file
	filePath := filepath.Join(w.mdxDir, movie.Slug+".mdx")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write MDX file: %w", err)
	}

	return nil
}

// GenerateMDX creates MDX content with YAML frontmatter
func (w *MDXWriter) GenerateMDX(movie *Movie) (string, error) {
	var sb strings.Builder

	// Write frontmatter delimiter
	sb.WriteString("---\n")

	// Marshal movie data to YAML, forcing double-quote style on path fields
	// to prevent yaml.v3 from emitting unquoted scalars for paths that contain
	// ": " (e.g. "All Is Lost: Tutto Perduto"), which YAML parsers read as mappings.
	var docNode yaml.Node
	if err := docNode.Encode(movie); err != nil {
		return "", fmt.Errorf("failed to marshal movie to YAML: %w", err)
	}
	forceQuotedFields(&docNode, "filePath", "fileName")
	yamlData, err := yaml.Marshal(&docNode)
	if err != nil {
		return "", fmt.Errorf("failed to marshal movie to YAML: %w", err)
	}

	sb.Write(yamlData)
	sb.WriteString("---\n\n")

	// Write markdown content
	sb.WriteString(fmt.Sprintf("# %s", movie.Title))
	if movie.ReleaseYear > 0 {
		sb.WriteString(fmt.Sprintf(" (%d)", movie.ReleaseYear))
	}
	sb.WriteString("\n\n")

	// Synopsis section
	if movie.Description != "" {
		sb.WriteString("## Synopsis\n\n")
		sb.WriteString(movie.Description)
		sb.WriteString("\n\n")
	}

	// Details section
	sb.WriteString("## Details\n\n")

	if movie.Rating > 0 {
		sb.WriteString(fmt.Sprintf("- **Rating**: %.1f/10\n", movie.Rating))
	}

	if movie.Runtime > 0 {
		sb.WriteString(fmt.Sprintf("- **Runtime**: %d minutes\n", movie.Runtime))
	}

	if movie.Director != "" {
		sb.WriteString(fmt.Sprintf("- **Director**: %s\n", movie.Director))
	}

	if len(movie.Genres) > 0 {
		sb.WriteString(fmt.Sprintf("- **Genres**: %s\n", strings.Join(movie.Genres, ", ")))
	}

	if len(movie.Cast) > 0 {
		sb.WriteString(fmt.Sprintf("- **Cast**: %s\n", strings.Join(movie.Cast, ", ")))
	}

	sb.WriteString("\n")

	// File information section
	sb.WriteString("## File Information\n\n")
	sb.WriteString(fmt.Sprintf("- **Location**: `%s`\n", movie.FilePath))
	sb.WriteString(fmt.Sprintf("- **Filename**: `%s`\n", movie.FileName))

	if movie.FileSize > 0 {
		sb.WriteString(fmt.Sprintf("- **Size**: %s\n", formatFileSize(movie.FileSize)))
	}

	sb.WriteString(fmt.Sprintf("- **Last Scanned**: %s\n", movie.ScannedAt.Format("January 2, 2006")))

	// Links section
	if movie.TMDBID > 0 || movie.IMDbID != "" {
		sb.WriteString("\n## Links\n\n")

		if movie.TMDBID > 0 {
			sb.WriteString(fmt.Sprintf("- [View on TMDB](https://www.themoviedb.org/movie/%d)\n", movie.TMDBID))
		}

		if movie.IMDbID != "" {
			sb.WriteString(fmt.Sprintf("- [View on IMDb](https://www.imdb.com/title/%s)\n", movie.IMDbID))
		}
	}

	return sb.String(), nil
}

// GetCoverPath returns the relative path for a cover image
func (w *MDXWriter) GetCoverPath(slug string) string {
	return fmt.Sprintf("/covers/%s.jpg", slug)
}

// GetBackdropPath returns the relative path for a backdrop image
func (w *MDXWriter) GetBackdropPath(slug string) string {
	return fmt.Sprintf("/covers/%s-backdrop.jpg", slug)
}

// GetAbsoluteCoverPath returns the absolute file system path for a cover image
func (w *MDXWriter) GetAbsoluteCoverPath(slug string) string {
	return filepath.Join(w.coversDir, slug+".jpg")
}

// GetAbsoluteBackdropPath returns the absolute file system path for a backdrop image
func (w *MDXWriter) GetAbsoluteBackdropPath(slug string) string {
	return filepath.Join(w.coversDir, slug+"-backdrop.jpg")
}

// forceQuotedFields sets DoubleQuotedStyle on the named scalar fields inside a
// yaml.DocumentNode → MappingNode tree. This prevents yaml.v3 from emitting
// bare scalars for file paths that contain ": ", which YAML parsers would
// otherwise interpret as a nested mapping.
func forceQuotedFields(doc *yaml.Node, keys ...string) {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return
	}
	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return
	}
	keySet := make(map[string]bool, len(keys))
	for _, k := range keys {
		keySet[k] = true
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if keySet[mapping.Content[i].Value] {
			mapping.Content[i+1].Style = yaml.DoubleQuotedStyle
		}
	}
}

// formatFileSize formats a file size in bytes to a human-readable string
func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}
