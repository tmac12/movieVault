package cache

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteCache implements the Cache interface using SQLite for persistence.
type SQLiteCache struct {
	db     *sql.DB
	hits   int64 // atomic counter for cache hits
	misses int64 // atomic counter for cache misses
}

// NewSQLiteCache creates a new SQLite-backed cache.
// The database file and table are auto-created if they don't exist.
func NewSQLiteCache(dbPath string) (*SQLiteCache, error) {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache database: %w", err)
	}

	// Create table if not exists
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS cache (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cache_key TEXT UNIQUE NOT NULL,
			response_json BLOB NOT NULL,
			cached_at DATETIME NOT NULL,
			expires_at DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_cache_key ON cache(cache_key);
		CREATE INDEX IF NOT EXISTS idx_expires_at ON cache(expires_at);
	`
	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create cache table: %w", err)
	}

	return &SQLiteCache{db: db}, nil
}

// Get retrieves data from the cache by key.
// Returns the data and true if found and not expired, otherwise nil and false.
func (c *SQLiteCache) Get(key string) ([]byte, bool) {
	var data []byte
	var expiresAt time.Time

	err := c.db.QueryRow(
		"SELECT response_json, expires_at FROM cache WHERE cache_key = ?",
		key,
	).Scan(&data, &expiresAt)

	if err != nil {
		// Not found or other error
		atomic.AddInt64(&c.misses, 1)
		return nil, false
	}

	// Check if expired
	if time.Now().After(expiresAt) {
		// Entry is expired, delete it
		c.db.Exec("DELETE FROM cache WHERE cache_key = ?", key)
		atomic.AddInt64(&c.misses, 1)
		return nil, false
	}

	atomic.AddInt64(&c.hits, 1)
	return data, true
}

// Set stores data in the cache with the given key and TTL.
func (c *SQLiteCache) Set(key string, data []byte, ttl time.Duration) error {
	now := time.Now()
	expiresAt := now.Add(ttl)

	// Use INSERT OR REPLACE to handle both new entries and updates
	_, err := c.db.Exec(
		`INSERT OR REPLACE INTO cache (cache_key, response_json, cached_at, expires_at)
		 VALUES (?, ?, ?, ?)`,
		key, data, now, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("failed to set cache entry: %w", err)
	}

	return nil
}

// Clear removes all entries from the cache.
func (c *SQLiteCache) Clear() error {
	_, err := c.db.Exec("DELETE FROM cache")
	if err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}
	return nil
}

// Count returns the number of entries in the cache.
func (c *SQLiteCache) Count() (int, error) {
	var count int
	err := c.db.QueryRow("SELECT COUNT(*) FROM cache").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count cache entries: %w", err)
	}
	return count, nil
}

// Stats returns cache statistics including hits, misses, and entry count.
func (c *SQLiteCache) Stats() (CacheStats, error) {
	count, err := c.Count()
	if err != nil {
		return CacheStats{}, err
	}
	return CacheStats{
		Hits:       atomic.LoadInt64(&c.hits),
		Misses:     atomic.LoadInt64(&c.misses),
		EntryCount: count,
	}, nil
}

// ResetStats resets the hit and miss counters to zero.
func (c *SQLiteCache) ResetStats() {
	atomic.StoreInt64(&c.hits, 0)
	atomic.StoreInt64(&c.misses, 0)
}

// Close closes the database connection.
func (c *SQLiteCache) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}
