// Package cache provides a caching layer for TMDB API responses.
package cache

import "time"

// Cache defines the interface for caching TMDB responses.
type Cache interface {
	// Get retrieves data from the cache by key.
	// Returns the data and true if found and not expired, otherwise nil and false.
	Get(key string) ([]byte, bool)

	// Set stores data in the cache with the given key and TTL.
	Set(key string, data []byte, ttl time.Duration) error

	// Clear removes all entries from the cache.
	Clear() error

	// Close closes the cache and releases resources.
	Close() error
}
