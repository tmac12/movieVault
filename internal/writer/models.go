package writer

import (
	"time"
)

// Movie represents a movie with all its metadata
type Movie struct {
	Title         string    `yaml:"title"`
	Slug          string    `yaml:"slug"`
	Description   string    `yaml:"description"`
	CoverImage    string    `yaml:"coverImage"`
	BackdropImage string    `yaml:"backdropImage"`
	FilePath      string    `yaml:"filePath"`
	FileName      string    `yaml:"fileName"`
	Rating        float64   `yaml:"rating"`
	ReleaseYear   int       `yaml:"releaseYear"`
	ReleaseDate   string    `yaml:"releaseDate"`
	Runtime       int       `yaml:"runtime"`
	Genres        []string  `yaml:"genres"`
	Director      string    `yaml:"director"`
	Cast          []string  `yaml:"cast"`
	TMDBID        int       `yaml:"tmdbId"`
	IMDbID        string    `yaml:"imdbId,omitempty"`
	ScannedAt     time.Time `yaml:"scannedAt"`
	FileSize      int64     `yaml:"fileSize"`
}
