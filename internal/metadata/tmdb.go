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

	"github.com/marco/movieVault/internal/metadata/cache"
	"github.com/marco/movieVault/internal/retry"
	"github.com/marco/movieVault/internal/writer"
)

const (
	tmdbAPIBaseURL  = "https://api.themoviedb.org/3"
	tmdbImageBaseURL = "https://image.tmdb.org/t/p"
	posterSize      = "w500"
	backdropSize    = "w1280"
)

// RetryLogFunc is a callback for logging retry attempts
type RetryLogFunc func(attempt int, maxAttempts int, backoff time.Duration, err error)

// CacheLogFunc is a callback for logging cache operations
type CacheLogFunc func(operation string, key string, hit bool)

// Client represents a TMDB API client
type Client struct {
	apiKey         string
	language       string
	httpClient     *http.Client
	rateDelay      time.Duration
	maxAttempts    int
	initialBackoff time.Duration
	retryLogFunc   RetryLogFunc
	cache          cache.Cache
	cacheTTL       time.Duration
	cacheLogFunc   CacheLogFunc
	forceRefresh   bool
}

// ClientConfig holds configuration for the TMDB client
type ClientConfig struct {
	APIKey           string
	Language         string
	RateLimitDelayMs int
	MaxAttempts      int
	InitialBackoffMs int
	RetryLogFunc     RetryLogFunc
	Cache            cache.Cache
	CacheTTLDays     int
	CacheLogFunc     CacheLogFunc
	ForceRefresh     bool
}

// NewClient creates a new TMDB API client
func NewClient(apiKey string, language string, rateLimitDelayMs int) *Client {
	return NewClientWithConfig(ClientConfig{
		APIKey:           apiKey,
		Language:         language,
		RateLimitDelayMs: rateLimitDelayMs,
		MaxAttempts:      3,
		InitialBackoffMs: 1000,
	})
}

// NewClientWithConfig creates a new TMDB API client with full configuration
func NewClientWithConfig(cfg ClientConfig) *Client {
	if cfg.Language == "" {
		cfg.Language = "en-US"
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.InitialBackoffMs <= 0 {
		cfg.InitialBackoffMs = 1000
	}
	if cfg.CacheTTLDays <= 0 {
		cfg.CacheTTLDays = 30
	}
	return &Client{
		apiKey:         cfg.APIKey,
		language:       cfg.Language,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		rateDelay:      time.Duration(cfg.RateLimitDelayMs) * time.Millisecond,
		maxAttempts:    cfg.MaxAttempts,
		initialBackoff: time.Duration(cfg.InitialBackoffMs) * time.Millisecond,
		retryLogFunc:   cfg.RetryLogFunc,
		cache:          cfg.Cache,
		cacheTTL:       time.Duration(cfg.CacheTTLDays) * 24 * time.Hour,
		cacheLogFunc:   cfg.CacheLogFunc,
		forceRefresh:   cfg.ForceRefresh,
	}
}

// doRequestWithRetry executes an HTTP GET request with retry logic
func (c *Client) doRequestWithRetry(requestURL string) (*http.Response, error) {
	var resp *http.Response
	var lastErr error
	attempt := 0

	err := retry.Retry(func() error {
		attempt++
		var reqErr error
		resp, reqErr = c.httpClient.Get(requestURL)
		if reqErr != nil {
			lastErr = reqErr
			// Log retry attempt if callback provided
			if c.retryLogFunc != nil && attempt < c.maxAttempts {
				backoff := c.initialBackoff * time.Duration(1<<(attempt-1))
				if retry.IsRateLimited(reqErr) {
					backoff *= 2
				}
				c.retryLogFunc(attempt, c.maxAttempts, backoff, reqErr)
			}
			return reqErr
		}

		// Check for retryable HTTP status codes
		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			statusErr := fmt.Errorf("TMDB API error (status %d): %s", resp.StatusCode, string(body))
			lastErr = statusErr
			// Log retry attempt if callback provided
			if c.retryLogFunc != nil && attempt < c.maxAttempts {
				backoff := c.initialBackoff * time.Duration(1<<(attempt-1))
				if resp.StatusCode == 429 {
					backoff *= 2
				}
				c.retryLogFunc(attempt, c.maxAttempts, backoff, statusErr)
			}
			return statusErr
		}

		return nil
	}, c.maxAttempts, c.initialBackoff)

	if err != nil {
		return nil, lastErr
	}
	return resp, nil
}

// getFromCache retrieves data from cache if available and not force-refreshing
func (c *Client) getFromCache(key string) ([]byte, bool) {
	if c.cache == nil || c.forceRefresh {
		return nil, false
	}
	data, found := c.cache.Get(key)
	if c.cacheLogFunc != nil {
		c.cacheLogFunc("get", key, found)
	}
	return data, found
}

// setToCache stores data in cache if caching is enabled
func (c *Client) setToCache(key string, data []byte) {
	if c.cache == nil {
		return
	}
	if err := c.cache.Set(key, data, c.cacheTTL); err != nil {
		// Log error but don't fail the operation
		if c.cacheLogFunc != nil {
			c.cacheLogFunc("set_error", key, false)
		}
	} else if c.cacheLogFunc != nil {
		c.cacheLogFunc("set", key, true)
	}
}

// SearchMovie searches for a movie by title and optional year
func (c *Client) SearchMovie(title string, year int) (*TMDBMovie, error) {
	// Build cache key
	cacheKey := fmt.Sprintf("tmdb:search:%s:%d", title, year)

	// Check cache first
	if cachedData, found := c.getFromCache(cacheKey); found {
		var cachedResult TMDBMovie
		if err := json.Unmarshal(cachedData, &cachedResult); err == nil {
			return &cachedResult, nil
		}
	}

	// Build query parameters
	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("query", title)
	if year > 0 {
		params.Set("year", strconv.Itoa(year))
	}
	params.Set("language", c.language)
	params.Set("page", "1")

	// Make request with retry
	searchURL := fmt.Sprintf("%s/search/movie?%s", tmdbAPIBaseURL, params.Encode())
	resp, err := c.doRequestWithRetry(searchURL)
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

	// Cache the result
	if resultData, err := json.Marshal(searchResp.Results[0]); err == nil {
		c.setToCache(cacheKey, resultData)
	}

	// Rate limiting
	time.Sleep(c.rateDelay)

	return &searchResp.Results[0], nil
}

// GetMovieDetails fetches detailed information about a movie
func (c *Client) GetMovieDetails(tmdbID int) (*TMDBMovieDetails, error) {
	// Build cache key
	cacheKey := fmt.Sprintf("tmdb:movie:%d", tmdbID)

	// Check cache first
	if cachedData, found := c.getFromCache(cacheKey); found {
		var cachedResult TMDBMovieDetails
		if err := json.Unmarshal(cachedData, &cachedResult); err == nil {
			return &cachedResult, nil
		}
	}

	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("language", c.language)

	detailsURL := fmt.Sprintf("%s/movie/%d?%s", tmdbAPIBaseURL, tmdbID, params.Encode())
	resp, err := c.doRequestWithRetry(detailsURL)
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

	// Cache the result
	if resultData, err := json.Marshal(details); err == nil {
		c.setToCache(cacheKey, resultData)
	}

	// Rate limiting
	time.Sleep(c.rateDelay)

	return &details, nil
}

// GetMovieCredits fetches cast and crew information
func (c *Client) GetMovieCredits(tmdbID int) (*TMDBCreditsResponse, error) {
	// Build cache key
	cacheKey := fmt.Sprintf("tmdb:credits:%d", tmdbID)

	// Check cache first
	if cachedData, found := c.getFromCache(cacheKey); found {
		var cachedResult TMDBCreditsResponse
		if err := json.Unmarshal(cachedData, &cachedResult); err == nil {
			return &cachedResult, nil
		}
	}

	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("language", c.language)

	creditsURL := fmt.Sprintf("%s/movie/%d/credits?%s", tmdbAPIBaseURL, tmdbID, params.Encode())
	resp, err := c.doRequestWithRetry(creditsURL)
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

	// Cache the result
	if resultData, err := json.Marshal(credits); err == nil {
		c.setToCache(cacheKey, resultData)
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

	// Download image with retry
	resp, err := c.doRequestWithRetry(imageURL)
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

// DownloadImageFromURL downloads an image from an arbitrary URL or copies from a local path (US-020)
// Used for downloading images from NFO-provided URLs or local filesystem paths
func (c *Client) DownloadImageFromURL(imageURL string, outputPath string) error {
	if imageURL == "" {
		return fmt.Errorf("image URL is empty")
	}

	// Local filesystem path — copy directly
	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		return copyLocalImage(imageURL, outputPath)
	}

	// Download image with retry
	resp, err := c.doRequestWithRetry(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download image from URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image from URL (status %d)", resp.StatusCode)
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

// copyLocalImage copies an image from a local filesystem path to the output path
func copyLocalImage(srcPath string, outputPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open local image %s: %w", srcPath, err)
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	dst, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy local image: %w", err)
	}

	return nil
}
