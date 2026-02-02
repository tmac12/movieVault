package metadata

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/marco/movieVault/internal/writer"
)

const (
	tmdbAPIBaseURL  = "https://api.themoviedb.org/3"
	tmdbImageBaseURL = "https://image.tmdb.org/t/p"
	posterSize      = "w500"
	backdropSize    = "w1280"
)

// Client represents a TMDB API client
type Client struct {
	apiKey     string
	language   string
	httpClient *http.Client
	rateDelay  time.Duration
}

// NewClient creates a new TMDB API client
func NewClient(apiKey string, language string, rateLimitDelayMs int) *Client {
	if language == "" {
		language = "en-US"
	}
	return &Client{
		apiKey:     apiKey,
		language:   language,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		rateDelay:  time.Duration(rateLimitDelayMs) * time.Millisecond,
	}
}

// SearchMovie searches for a movie by title and optional year
func (c *Client) SearchMovie(title string, year int) (*TMDBMovie, error) {
	// Build query parameters
	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("query", title)
	if year > 0 {
		params.Set("year", strconv.Itoa(year))
	}
	params.Set("language", c.language)
	params.Set("page", "1")

	// Make request
	searchURL := fmt.Sprintf("%s/search/movie?%s", tmdbAPIBaseURL, params.Encode())
	resp, err := c.httpClient.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to search movie: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TMDB API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var searchResp TMDBSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	// Return first result if available
	if len(searchResp.Results) == 0 {
		return nil, fmt.Errorf("no results found for '%s'", title)
	}

	// Rate limiting
	time.Sleep(c.rateDelay)

	return &searchResp.Results[0], nil
}

// GetMovieDetails fetches detailed information about a movie
func (c *Client) GetMovieDetails(tmdbID int) (*TMDBMovieDetails, error) {
	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("language", c.language)

	detailsURL := fmt.Sprintf("%s/movie/%d?%s", tmdbAPIBaseURL, tmdbID, params.Encode())
	resp, err := c.httpClient.Get(detailsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TMDB API error (status %d): %s", resp.StatusCode, string(body))
	}

	var details TMDBMovieDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("failed to decode movie details: %w", err)
	}

	// Rate limiting
	time.Sleep(c.rateDelay)

	return &details, nil
}

// GetMovieCredits fetches cast and crew information
func (c *Client) GetMovieCredits(tmdbID int) (*TMDBCreditsResponse, error) {
	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("language", c.language)

	creditsURL := fmt.Sprintf("%s/movie/%d/credits?%s", tmdbAPIBaseURL, tmdbID, params.Encode())
	resp, err := c.httpClient.Get(creditsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie credits: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TMDB API error (status %d): %s", resp.StatusCode, string(body))
	}

	var credits TMDBCreditsResponse
	if err := json.NewDecoder(resp.Body).Decode(&credits); err != nil {
		return nil, fmt.Errorf("failed to decode credits: %w", err)
	}

	// Rate limiting
	time.Sleep(c.rateDelay)

	return &credits, nil
}

// GetFullMovieData fetches all data needed for a Movie struct
func (c *Client) GetFullMovieData(title string, year int) (*writer.Movie, error) {
	// Search for the movie
	searchResult, err := c.SearchMovie(title, year)
	if err != nil {
		return nil, err
	}

	// Get detailed information
	details, err := c.GetMovieDetails(searchResult.ID)
	if err != nil {
		return nil, err
	}

	// Get credits
	credits, err := c.GetMovieCredits(searchResult.ID)
	if err != nil {
		return nil, err
	}

	// Extract genres
	var genres []string
	for _, genre := range details.Genres {
		genres = append(genres, genre.Name)
	}

	// Extract director(s)
	var directors []string
	for _, crew := range credits.Crew {
		if crew.Job == "Director" {
			directors = append(directors, crew.Name)
		}
	}
	director := strings.Join(directors, ", ")

	// Extract top cast (first 5)
	var cast []string
	maxCast := 5
	if len(credits.Cast) < maxCast {
		maxCast = len(credits.Cast)
	}
	for i := 0; i < maxCast; i++ {
		cast = append(cast, credits.Cast[i].Name)
	}

	// Extract release year
	releaseYear := 0
	if len(details.ReleaseDate) >= 4 {
		releaseYear, _ = strconv.Atoi(details.ReleaseDate[:4])
	}

	// Build Movie struct
	movie := &writer.Movie{
		Title:         details.Title,
		Description:   details.Overview,
		Rating:        details.VoteAverage,
		ReleaseYear:   releaseYear,
		ReleaseDate:   details.ReleaseDate,
		Runtime:       details.Runtime,
		Genres:        genres,
		Director:      director,
		Cast:          cast,
		TMDBID:        details.ID,
		IMDbID:        details.IMDbID,
		ScannedAt:     time.Now(),
	}

	return movie, nil
}

// ErrMovieNotFound is returned when a movie is not found by ID
var ErrMovieNotFound = fmt.Errorf("movie not found")

// GetMovieByID fetches a movie directly by its TMDB ID, bypassing search
func (c *Client) GetMovieByID(tmdbID int) (*writer.Movie, error) {
	// Get detailed information
	details, err := c.GetMovieDetails(tmdbID)
	if err != nil {
		// Check for 404 response
		if strings.Contains(err.Error(), "status 404") {
			return nil, ErrMovieNotFound
		}
		return nil, err
	}

	// Get credits
	credits, err := c.GetMovieCredits(tmdbID)
	if err != nil {
		return nil, err
	}

	// Extract genres
	var genres []string
	for _, genre := range details.Genres {
		genres = append(genres, genre.Name)
	}

	// Extract director(s)
	var directors []string
	for _, crew := range credits.Crew {
		if crew.Job == "Director" {
			directors = append(directors, crew.Name)
		}
	}
	director := strings.Join(directors, ", ")

	// Extract top cast (first 5)
	var cast []string
	maxCast := 5
	if len(credits.Cast) < maxCast {
		maxCast = len(credits.Cast)
	}
	for i := 0; i < maxCast; i++ {
		cast = append(cast, credits.Cast[i].Name)
	}

	// Extract release year
	releaseYear := 0
	if len(details.ReleaseDate) >= 4 {
		releaseYear, _ = strconv.Atoi(details.ReleaseDate[:4])
	}

	// Build Movie struct
	movie := &writer.Movie{
		Title:       details.Title,
		Description: details.Overview,
		Rating:      details.VoteAverage,
		ReleaseYear: releaseYear,
		ReleaseDate: details.ReleaseDate,
		Runtime:     details.Runtime,
		Genres:      genres,
		Director:    director,
		Cast:        cast,
		TMDBID:      details.ID,
		IMDbID:      details.IMDbID,
		ScannedAt:   time.Now(),
	}

	return movie, nil
}

// DownloadImage downloads an image from TMDB to a local path
func (c *Client) DownloadImage(imagePath string, outputPath string, imageType string) error {
	if imagePath == "" {
		return fmt.Errorf("image path is empty")
	}

	// Determine size based on type
	size := posterSize
	if imageType == "backdrop" {
		size = backdropSize
	}

	// Build image URL
	imageURL := fmt.Sprintf("%s/%s%s", tmdbImageBaseURL, size, imagePath)

	// Download image
	resp, err := c.httpClient.Get(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image (status %d)", resp.StatusCode)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Copy image data
	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return fmt.Errorf("failed to write image: %w", err)
	}

	// Rate limiting
	time.Sleep(c.rateDelay)

	return nil
}
