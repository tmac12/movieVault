// Package cache provides a caching layer for TMDB API responses.
package cache

import "time"

// CacheStats holds statistics about cache operations.
type CacheStats struct {
	Hits       int64 // Number of cache hits
	Misses     int64 // Number of cache misses
	EntryCount int   // Current number of entries in cache
}

// HitRate returns the cache hit rate as a percentage (0-100).
// Returns 0 if no operations have been performed.
func (s CacheStats) HitRate() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total) * 100
}

// Cache defines the interface for caching TMDB responses.
type Cache interface {
	// Get retrieves data from the cache by key.
	// Returns the data and true if found and not expired, otherwise nil and false.
	Get(key string) ([]byte, bool)

	// Set stores data in the cache with the given key and TTL.
	Set(key string, data []byte, ttl time.Duration) error

	// Clear removes all entries from the cache.
	Clear() error

	// Count returns the number of entries in the cache.
	Count() (int, error)

	// Stats returns cache statistics including hits, misses, and entry count.
	Stats() (CacheStats, error)

	// ResetStats resets the hit and miss counters to zero.
	ResetStats()

	// Close closes the cache and releases resources.
	Close() error
}
