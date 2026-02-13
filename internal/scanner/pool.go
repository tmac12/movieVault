package scanner

import (
	"context"
	"sync"
	"sync/atomic"
)

// ProcessResult holds the outcome of processing a single file.
type ProcessResult struct {
	File           FileInfo
	MetadataSource string // "NFO", "TMDB", or "NFO+TMDB"
	Slug           string
	Err            error
}

// ProcessFunc processes a single FileInfo and returns the metadata source,
// generated slug, and any error encountered.
type ProcessFunc func(ctx context.Context, file FileInfo) (metadataSource string, slug string, err error)

// SlugGuard provides thread-safe slug deduplication. Multiple goroutines can
// safely call TryClaimSlug; only the first caller for a given slug succeeds.
type SlugGuard struct {
	mu    sync.Mutex
	slugs map[string]bool
}

// NewSlugGuard creates a new SlugGuard.
func NewSlugGuard() *SlugGuard {
	return &SlugGuard{slugs: make(map[string]bool)}
}

// TryClaimSlug attempts to claim a slug. Returns true if the slug was
// successfully claimed (first caller wins), false if already taken.
func (sg *SlugGuard) TryClaimSlug(slug string) bool {
	sg.mu.Lock()
	defer sg.mu.Unlock()
	if sg.slugs[slug] {
		return false
	}
	sg.slugs[slug] = true
	return true
}

// ProcessFilesConcurrently fans out file processing across N workers.
// The processedCount pointer is atomically incremented after each file
// completes (success or failure), enabling external progress reporting.
// Results are returned in no guaranteed order.
func ProcessFilesConcurrently(
	ctx context.Context,
	files []FileInfo,
	fn ProcessFunc,
	workers int,
	processedCount *int64,
) []ProcessResult {
	if workers <= 0 {
		workers = 1
	}

	jobs := make(chan FileInfo, len(files))
	results := make(chan ProcessResult, len(files))

	var wg sync.WaitGroup

	// Start worker goroutines
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range jobs {
				// Check for cancellation before processing
				if ctx.Err() != nil {
					results <- ProcessResult{File: file, Err: ctx.Err()}
					atomic.AddInt64(processedCount, 1)
					continue
				}

				source, slug, err := fn(ctx, file)
				results <- ProcessResult{
					File:           file,
					MetadataSource: source,
					Slug:           slug,
					Err:            err,
				}
				atomic.AddInt64(processedCount, 1)
			}
		}()
	}

	// Feed jobs
	for _, file := range files {
		jobs <- file
	}
	close(jobs)

	// Wait for all workers then close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var out []ProcessResult
	for r := range results {
		out = append(out, r)
	}
	return out
}
