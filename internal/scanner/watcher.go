package scanner

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileHandler is called when a new file is detected and ready for processing
type FileHandler func(file FileInfo) error

// Watcher monitors directories for new video files
type Watcher struct {
	scanner       *Scanner
	directories   []string
	debounceDelay time.Duration
	recursive     bool
	handler       FileHandler
	watcher       *fsnotify.Watcher
	stopChan      chan struct{}
	doneChan      chan struct{}

	// Debouncing state
	mu            sync.Mutex
	pendingFiles  map[string]time.Time // file path -> last event time
	pendingTimers map[string]*time.Timer
}

// WatcherConfig holds configuration for the file watcher
type WatcherConfig struct {
	Directories   []string
	Extensions    []string
	MDXDir        string
	ExcludeDirs   []string
	DebounceDelay time.Duration // How long to wait after last event before processing
	Recursive     bool          // Watch subdirectories
}

// NewWatcher creates a new directory watcher
func NewWatcher(cfg WatcherConfig, handler FileHandler) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	s := NewWithExclusions(cfg.Extensions, cfg.MDXDir, cfg.ExcludeDirs)

	return &Watcher{
		scanner:       s,
		directories:   cfg.Directories,
		debounceDelay: cfg.DebounceDelay,
		recursive:     cfg.Recursive,
		handler:       handler,
		watcher:       fsWatcher,
		stopChan:      make(chan struct{}),
		doneChan:      make(chan struct{}),
		pendingFiles:  make(map[string]time.Time),
		pendingTimers: make(map[string]*time.Timer),
	}, nil
}

// Start begins watching directories for changes
func (w *Watcher) Start() error {
	// Add all configured directories to watch
	for _, dir := range w.directories {
		if err := w.addDirectory(dir); err != nil {
			slog.Warn("failed to watch directory", "path", dir, "error", err)
		}
	}

	// Start event processing goroutine
	go w.processEvents()

	slog.Info("file watcher started",
		"directories", len(w.directories),
		"debounce_seconds", w.debounceDelay.Seconds(),
		"recursive", w.recursive,
	)

	return nil
}

// Stop stops watching directories
func (w *Watcher) Stop() error {
	close(w.stopChan)
	<-w.doneChan // Wait for event loop to finish

	// Cancel any pending timers
	w.mu.Lock()
	for _, timer := range w.pendingTimers {
		timer.Stop()
	}
	w.mu.Unlock()

	return w.watcher.Close()
}

// Wait blocks until the watcher is stopped
func (w *Watcher) Wait() {
	<-w.doneChan
}

// addDirectory adds a directory (and optionally subdirectories) to watch
func (w *Watcher) addDirectory(path string) error {
	// Check if directory exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("directory does not exist: %s", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}

	if w.recursive {
		// Walk directory tree and add all subdirectories
		return filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip directories we can't access
			}
			if info.IsDir() {
				// Skip excluded directories
				if w.scanner.IsExcludedDir(p) {
					slog.Debug("skipping excluded directory", "path", p)
					return filepath.SkipDir
				}
				if err := w.watcher.Add(p); err != nil {
					slog.Warn("failed to add directory to watch", "path", p, "error", err)
				} else {
					slog.Debug("watching directory", "path", p)
				}
			}
			return nil
		})
	}

	// Non-recursive: just watch the top-level directory
	if err := w.watcher.Add(path); err != nil {
		return fmt.Errorf("failed to add directory to watch: %w", err)
	}
	slog.Debug("watching directory", "path", path)
	return nil
}

// processEvents handles fsnotify events
func (w *Watcher) processEvents() {
	defer close(w.doneChan)

	for {
		select {
		case <-w.stopChan:
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("watcher error", "error", err)
		}
	}
}

// handleEvent processes a single fsnotify event
func (w *Watcher) handleEvent(event fsnotify.Event) {
	path := event.Name

	// Handle directory creation (for recursive watching)
	if event.Has(fsnotify.Create) {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			if w.recursive && !w.scanner.IsExcludedDir(path) {
				if err := w.addDirectory(path); err != nil {
					slog.Warn("failed to add new directory to watch", "path", path, "error", err)
				} else {
					slog.Info("new directory detected, now watching", "path", path)
				}
			}
			return
		}
	}

	// Only process files with matching extensions
	filename := filepath.Base(path)
	if !w.scanner.IsMediaFile(filename) {
		return
	}

	// Handle file creation and write events (new files or file modifications)
	if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
		slog.Debug("file event detected",
			"event", event.Op.String(),
			"file", filename,
		)
		w.scheduleProcessing(path)
	}
}

// scheduleProcessing schedules a file for processing after debounce delay
func (w *Watcher) scheduleProcessing(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Update last event time
	w.pendingFiles[path] = time.Now()

	// Cancel existing timer if any
	if timer, exists := w.pendingTimers[path]; exists {
		timer.Stop()
	}

	// Create new timer for debounce
	w.pendingTimers[path] = time.AfterFunc(w.debounceDelay, func() {
		w.processFile(path)
	})

	slog.Debug("file scheduled for processing",
		"file", filepath.Base(path),
		"debounce_seconds", w.debounceDelay.Seconds(),
	)
}

// processFile processes a single file after debounce period
func (w *Watcher) processFile(path string) {
	w.mu.Lock()
	delete(w.pendingFiles, path)
	delete(w.pendingTimers, path)
	w.mu.Unlock()

	// Verify file still exists (might have been moved/deleted)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Debug("file no longer exists, skipping", "path", path)
			return
		}
		slog.Error("failed to stat file", "path", path, "error", err)
		return
	}

	// Skip directories
	if info.IsDir() {
		return
	}

	// Extract movie information from filename
	filename := filepath.Base(path)
	title, year := ExtractTitleAndYear(filename)
	slug := GenerateSlug(title, year)

	fileInfo := FileInfo{
		Path:       path,
		FileName:   filename,
		Title:      title,
		Year:       year,
		Size:       info.Size(),
		Slug:       slug,
		ShouldScan: !w.scanner.MDXExists(slug),
	}

	// Skip if MDX already exists
	if !fileInfo.ShouldScan {
		slog.Debug("mdx already exists, skipping", "file", filename, "slug", slug)
		return
	}

	slog.Info("processing new file", "file", filename, "title", title, "year", year)

	// Call the handler
	if err := w.handler(fileInfo); err != nil {
		slog.Error("failed to process file", "file", filename, "error", err)
	}
}

// IsValidMediaFile checks if a path is a valid media file for the configured extensions
func (w *Watcher) IsValidMediaFile(path string) bool {
	return w.scanner.IsMediaFile(filepath.Base(path))
}

// normalizeEventPath normalizes a path from fsnotify events
func normalizeEventPath(path string) string {
	// Clean the path
	path = filepath.Clean(path)
	// Convert to absolute if relative
	if !filepath.IsAbs(path) {
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}
	return path
}

// isHiddenFile checks if a file or any parent directory is hidden
func isHiddenFile(path string) bool {
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if len(part) > 0 && part[0] == '.' && part != "." && part != ".." {
			return true
		}
	}
	return false
}
