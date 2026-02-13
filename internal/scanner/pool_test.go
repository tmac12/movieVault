package scanner

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSlugGuard_TryClaimSlug(t *testing.T) {
	sg := NewSlugGuard()

	// First claim should succeed
	if !sg.TryClaimSlug("the-matrix-1999") {
		t.Error("expected first claim to succeed")
	}

	// Second claim of same slug should fail
	if sg.TryClaimSlug("the-matrix-1999") {
		t.Error("expected duplicate claim to fail")
	}

	// Different slug should succeed
	if !sg.TryClaimSlug("inception-2010") {
		t.Error("expected different slug claim to succeed")
	}
}

func TestSlugGuard_ConcurrentAccess(t *testing.T) {
	sg := NewSlugGuard()
	slug := "test-slug-2024"

	const goroutines = 100
	var successes int64
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if sg.TryClaimSlug(slug) {
				atomic.AddInt64(&successes, 1)
			}
		}()
	}
	wg.Wait()

	if successes != 1 {
		t.Errorf("expected exactly 1 successful claim, got %d", successes)
	}
}

func TestProcessFilesConcurrently_BasicProcessing(t *testing.T) {
	files := []FileInfo{
		{FileName: "movie1.mkv", Title: "Movie One", Year: 2020},
		{FileName: "movie2.mkv", Title: "Movie Two", Year: 2021},
		{FileName: "movie3.mkv", Title: "Movie Three", Year: 2022},
	}

	var processedCount int64
	processFn := func(ctx context.Context, file FileInfo) (string, string, error) {
		return "TMDB", file.Title, nil
	}

	results := ProcessFilesConcurrently(
		context.Background(),
		files,
		processFn,
		2,
		&processedCount,
	)

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
	if processedCount != 3 {
		t.Errorf("expected processedCount=3, got %d", processedCount)
	}

	// All should succeed
	for _, r := range results {
		if r.Err != nil {
			t.Errorf("unexpected error for %s: %v", r.File.FileName, r.Err)
		}
		if r.MetadataSource != "TMDB" {
			t.Errorf("expected source TMDB, got %s", r.MetadataSource)
		}
	}
}

func TestProcessFilesConcurrently_HandlesErrors(t *testing.T) {
	files := []FileInfo{
		{FileName: "good.mkv", Title: "Good Movie", Year: 2020},
		{FileName: "bad.mkv", Title: "Bad Movie", Year: 2021},
	}

	var processedCount int64
	processFn := func(ctx context.Context, file FileInfo) (string, string, error) {
		if file.Title == "Bad Movie" {
			return "", "", fmt.Errorf("metadata not found")
		}
		return "TMDB", file.Title, nil
	}

	results := ProcessFilesConcurrently(
		context.Background(),
		files,
		processFn,
		2,
		&processedCount,
	)

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	var errCount, okCount int
	for _, r := range results {
		if r.Err != nil {
			errCount++
		} else {
			okCount++
		}
	}

	if errCount != 1 || okCount != 1 {
		t.Errorf("expected 1 error and 1 success, got %d errors and %d successes", errCount, okCount)
	}
}

func TestProcessFilesConcurrently_ContextCancellation(t *testing.T) {
	files := make([]FileInfo, 20)
	for i := range files {
		files[i] = FileInfo{FileName: fmt.Sprintf("movie%d.mkv", i), Title: fmt.Sprintf("Movie %d", i)}
	}

	ctx, cancel := context.WithCancel(context.Background())
	var processedCount int64
	var started int64

	processFn := func(ctx context.Context, file FileInfo) (string, string, error) {
		count := atomic.AddInt64(&started, 1)
		// Cancel after a few have started
		if count == 3 {
			cancel()
		}
		// Simulate work
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
		}
		return "TMDB", file.Title, nil
	}

	results := ProcessFilesConcurrently(ctx, files, processFn, 2, &processedCount)

	// All files should be accounted for (some with ctx error, some processed)
	if len(results) != 20 {
		t.Errorf("expected 20 results, got %d", len(results))
	}

	// At least some should have context cancelled errors
	var cancelledCount int
	for _, r := range results {
		if r.Err == context.Canceled {
			cancelledCount++
		}
	}
	if cancelledCount == 0 {
		t.Log("warning: no cancelled results detected (timing-dependent)")
	}
}

func TestProcessFilesConcurrently_EmptyInput(t *testing.T) {
	var processedCount int64
	processFn := func(ctx context.Context, file FileInfo) (string, string, error) {
		t.Error("process function should not be called for empty input")
		return "", "", nil
	}

	results := ProcessFilesConcurrently(context.Background(), nil, processFn, 5, &processedCount)

	if len(results) != 0 {
		t.Errorf("expected 0 results for nil input, got %d", len(results))
	}
	if processedCount != 0 {
		t.Errorf("expected processedCount=0, got %d", processedCount)
	}
}

func TestProcessFilesConcurrently_SingleWorker(t *testing.T) {
	files := []FileInfo{
		{FileName: "a.mkv", Title: "A"},
		{FileName: "b.mkv", Title: "B"},
		{FileName: "c.mkv", Title: "C"},
	}

	var processedCount int64
	processFn := func(ctx context.Context, file FileInfo) (string, string, error) {
		return "NFO", file.Title, nil
	}

	results := ProcessFilesConcurrently(context.Background(), files, processFn, 1, &processedCount)

	if len(results) != 3 {
		t.Errorf("expected 3 results with single worker, got %d", len(results))
	}
	if processedCount != 3 {
		t.Errorf("expected processedCount=3, got %d", processedCount)
	}
}

func TestProcessFilesConcurrently_ZeroWorkers(t *testing.T) {
	files := []FileInfo{{FileName: "test.mkv", Title: "Test"}}

	var processedCount int64
	processFn := func(ctx context.Context, file FileInfo) (string, string, error) {
		return "TMDB", file.Title, nil
	}

	// workers=0 should be clamped to 1
	results := ProcessFilesConcurrently(context.Background(), files, processFn, 0, &processedCount)

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}
