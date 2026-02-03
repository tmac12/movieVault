package nfo

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/marco/movieVault/internal/writer"
)

// Parser handles parsing of .nfo files
type Parser struct{}

// NewParser creates a new NFO parser instance
func NewParser() *Parser {
	return &Parser{}
}

// FindNFOFile locates the .nfo file for a given video file
// Priority order:
// 1. {filename}.nfo (e.g., "The Matrix (1999).nfo")
// 2. movie.nfo (Jellyfin standard)
func (p *Parser) FindNFOFile(videoPath string) (string, error) {
	dir := filepath.Dir(videoPath)
	baseNameWithExt := filepath.Base(videoPath)
	ext := filepath.Ext(baseNameWithExt)
	baseName := strings.TrimSuffix(baseNameWithExt, ext)

	// Try filename.nfo first
	fileNameNFO := filepath.Join(dir, baseName+".nfo")
	if _, err := os.Stat(fileNameNFO); err == nil {
		return fileNameNFO, nil
	}

	// Try movie.nfo
	movieNFO := filepath.Join(dir, "movie.nfo")
	if _, err := os.Stat(movieNFO); err == nil {
		return movieNFO, nil
	}

	return "", fmt.Errorf("no .nfo file found for %s", videoPath)
}

// ParseNFOFile reads and parses an .nfo XML file
func (p *Parser) ParseNFOFile(nfoPath string) (*NFOMovie, error) {
	data, err := os.ReadFile(nfoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read .nfo file: %w", err)
	}

	var nfo NFOMovie
	if err := xml.Unmarshal(data, &nfo); err != nil {
		return nil, fmt.Errorf("failed to parse .nfo XML: %w", err)
	}

	return &nfo, nil
}

// ConvertToMovie transforms NFO data to writer.Movie struct
func (p *Parser) ConvertToMovie(nfo *NFOMovie) *writer.Movie {
	movie := &writer.Movie{
		Title:       nfo.Title,
		Description: nfo.Plot,
		Rating:      nfo.Rating,
		ReleaseYear: nfo.Year,
		Runtime:     nfo.Runtime,
		Genres:      nfo.Genres,
		TMDBID:      nfo.TMDBID,
		IMDbID:      nfo.IMDbID,
		ScannedAt:   time.Now(),
	}

	// Parse year from premiered date if year is missing
	if movie.ReleaseYear == 0 && nfo.Premiered != "" {
		if t, err := time.Parse("2006-01-02", nfo.Premiered); err == nil {
			movie.ReleaseYear = t.Year()
		}
	}

	// Set release date
	if nfo.Premiered != "" {
		movie.ReleaseDate = nfo.Premiered
	}

	// Join multiple directors
	if len(nfo.Directors) > 0 {
		movie.Director = strings.Join(nfo.Directors, ", ")
	}

	// Extract top 5 cast members
	maxCast := 5
	if len(nfo.Actors) < maxCast {
		maxCast = len(nfo.Actors)
	}
	movie.Cast = make([]string, maxCast)
	for i := 0; i < maxCast; i++ {
		movie.Cast[i] = nfo.Actors[i].Name
	}

	// Extract poster URL from <thumb> elements (US-018)
	// Look for poster aspect or use first thumb as poster
	movie.PosterURL = extractPosterURL(nfo.Thumbs)

	// Extract backdrop URL from <fanart><thumb> elements (US-018)
	movie.BackdropURL = extractBackdropURL(nfo.Fanart)

	return movie
}

// extractPosterURL finds the best poster URL from NFO thumb elements
// Priority: "poster" aspect > first thumb with URL
func extractPosterURL(thumbs []NFOThumb) string {
	if len(thumbs) == 0 {
		return ""
	}

	// First look for explicit poster aspect
	for _, thumb := range thumbs {
		if strings.EqualFold(thumb.Aspect, "poster") && thumb.URL != "" {
			return strings.TrimSpace(thumb.URL)
		}
	}

	// Fall back to first thumb with a URL
	for _, thumb := range thumbs {
		if thumb.URL != "" {
			return strings.TrimSpace(thumb.URL)
		}
	}

	return ""
}

// extractBackdropURL finds the best backdrop URL from NFO fanart elements
// Returns first backdrop/fanart thumb URL found
func extractBackdropURL(fanart *NFOFanart) string {
	if fanart == nil || len(fanart.Thumbs) == 0 {
		return ""
	}

	// Return first fanart thumb with a URL
	for _, thumb := range fanart.Thumbs {
		if thumb.URL != "" {
			return strings.TrimSpace(thumb.URL)
		}
	}

	return ""
}

// GetMovieFromNFO is the main entry point: finds, parses, and converts NFO to Movie
func (p *Parser) GetMovieFromNFO(videoPath string) (*writer.Movie, error) {
	nfoPath, err := p.FindNFOFile(videoPath)
	if err != nil {
		return nil, err
	}

	nfo, err := p.ParseNFOFile(nfoPath)
	if err != nil {
		return nil, err
	}

	movie := p.ConvertToMovie(nfo)
	return movie, nil
}
