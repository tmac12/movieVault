# MovieVault Post-Release Roadmap

**Last Updated:** January 29, 2026
**Current Status:** v1.0 Production Ready ✅

This document outlines the prioritized feature roadmap following the v1.0 release. These items were deferred during the production readiness push but should be implemented in upcoming releases.

---

## Release Status

### ✅ v1.0 - Production Ready (January 29, 2026)

**Completed Critical Items:**
1. ✅ Docker silent failure handling with health checks
2. ✅ 404 error page with search integration
3. ✅ SEO and social sharing metadata (Open Graph, Twitter Cards, schema.org)
4. ✅ Image error handling and loading states
5. ✅ Structured logging (log/slog)
6. ✅ Production deployment verification

**Key Features:**
- NFO file support with TMDB fallback
- Static Astro website generation
- TMDB metadata integration
- Docker deployment
- Responsive UI with search/filter

---

## 🎯 v1.1 - High Priority Improvements (1-2 weeks)

These items enhance UX and operational visibility without requiring architectural changes.

### 1.1.1 TMDB API Retry Logic
**Priority:** High | **Effort:** Medium | **Impact:** High

**Problem:** Single transient network failures currently fail entire movie processing.

**Implementation:**
```go
// internal/metadata/tmdb.go
func (c *Client) GetFullMovieData(title string, year int) (*Movie, error) {
    var lastErr error
    for attempt := 1; attempt <= 3; attempt++ {
        movie, err := c.fetchWithRetry(title, year)
        if err == nil {
            return movie, nil
        }
        lastErr = err
        slog.Warn("tmdb api retry", "attempt", attempt, "error", err)
        time.Sleep(time.Duration(attempt) * time.Second)
    }
    return nil, fmt.Errorf("failed after 3 attempts: %w", lastErr)
}
```

**Benefits:**
- Resilient to transient network issues
- Reduces failed scans
- Better user experience

---

### 1.1.2 Genre Filter URL Persistence
**Priority:** High | **Effort:** Low | **Impact:** Medium

**Problem:** Can't share filtered views, browser back button doesn't work with genre filters.

**Implementation:**
```astro
<!-- website/src/components/GenreFilter.astro -->
<script>
  // Update URL when genre selected
  filterButtons.forEach(btn => {
    btn.addEventListener('click', () => {
      const genre = btn.dataset.genre;
      const url = new URL(window.location);
      url.searchParams.set('genre', genre);
      window.history.pushState({}, '', url);
    });
  });

  // Read genre from URL on load
  const urlParams = new URLSearchParams(window.location.search);
  const genre = urlParams.get('genre');
  if (genre) {
    filterMoviesByGenre(genre);
  }
</script>
```

**Benefits:**
- Shareable filtered URLs
- Browser back/forward works
- Better bookmarking

---

### 1.1.3 Sorting Options
**Priority:** Medium | **Effort:** Low | **Impact:** Medium

**Problem:** Currently only sorted by rating; users may want alphabetical, year, recently added.

**Implementation:**
```astro
<!-- website/src/pages/index.astro -->
<select id="sort-select">
  <option value="rating">Highest Rated</option>
  <option value="title">Title (A-Z)</option>
  <option value="year-desc">Newest First</option>
  <option value="year-asc">Oldest First</option>
  <option value="added">Recently Added</option>
</select>

<script>
  const sortFunctions = {
    rating: (a, b) => b.data.rating - a.data.rating,
    title: (a, b) => a.data.title.localeCompare(b.data.title),
    'year-desc': (a, b) => b.data.releaseYear - a.data.releaseYear,
    'year-asc': (a, b) => a.data.releaseYear - b.data.releaseYear,
    added: (a, b) => new Date(b.data.scannedAt) - new Date(a.data.scannedAt)
  };
</script>
```

**Benefits:**
- User preference flexibility
- Better discovery
- Persist sort in URL (combine with 1.1.2)

---

### 1.1.4 Accessibility Improvements
**Priority:** Medium | **Effort:** Medium | **Impact:** High

**Problem:** Missing skip links, ARIA labels, focus states for keyboard/screen reader users.

**Implementation Tasks:**
- Add skip navigation link: `<a href="#main" class="skip-link">Skip to main content</a>`
- Add ARIA labels to search form, genre filters, movie cards
- Add keyboard focus styles (outline, focus-visible)
- Add proper heading hierarchy (h1 → h2 → h3)
- Add alt text validation for all images
- Test with screen reader (NVDA/VoiceOver)
- Add focus trap to modals (if any added later)

**Benefits:**
- WCAG 2.1 AA compliance
- Better keyboard navigation
- Screen reader compatible

---

### 1.1.5 Observability & Metrics
**Priority:** Medium | **Effort:** Medium-High | **Impact:** High

**Problem:** Can't monitor scan duration, success rate, TMDB API usage in production.

**Implementation:**
```go
// internal/metrics/metrics.go
type Metrics struct {
    ScanDuration      time.Duration
    FilesScanned      int
    SuccessCount      int
    ErrorCount        int
    TMDBAPICalls      int
    NFOFilesUsed      int
    ImagesDownloaded  int
}

func (m *Metrics) RecordToFile(path string) error {
    data, _ := json.MarshalIndent(m, "", "  ")
    return os.WriteFile(path, data, 0644)
}
```

**Docker Integration:**
```bash
# docker/entrypoint.sh
/usr/local/bin/scanner --config /config/config.yaml --metrics-output /data/metrics.json
```

**Benefits:**
- Track performance trends
- Identify slow scans
- Monitor API usage
- Alert on high error rates

---

## 🚀 v1.2 - Polish & Enhancements (1 month)

### 1.2.1 Related Movies Section
**Priority:** Low | **Effort:** Medium

Add "You might also like" section to movie detail pages based on shared genres/director.

---

### 1.2.2 Dark/Light Mode Toggle
**Priority:** Low | **Effort:** Low

User-selectable theme with localStorage persistence.

---

### 1.2.3 Director/Actor Landing Pages
**Priority:** Low | **Effort:** High

Create `/people/[name].astro` pages showing all movies by director/actor.

---

### 1.2.4 Recently Added Section
**Priority:** Low | **Effort:** Low

Homepage section showing last 10 scanned movies sorted by `scannedAt`.

---

### 1.2.5 Collection Statistics Dashboard
**Priority:** Low | **Effort:** Medium

Dashboard showing:
- Total movies, total runtime, total file size
- Genre breakdown (pie chart)
- Decade distribution (bar chart)
- Top rated movies
- Longest/shortest movies

---

### 1.2.6 Image Format Optimization
**Priority:** Low | **Effort:** Medium

Convert JPEG posters to WebP/AVIF for 40-60% size reduction:

```bash
# During image download
cwebp -q 85 input.jpg -o output.webp
```

---

## 📦 v2.0 - Major Features (3+ months)

### 2.1 SQLite Library Database
**Priority:** High | **Effort:** High | **Impact:** High

Replace MDX files with SQLite database for advanced queries and watch history.

**Benefits:**
- Faster search/filter
- Watch history tracking
- Custom collections
- Advanced queries (unwatched, by decade, etc.)

---

### 2.2 Advanced Search
**Priority:** Medium | **Effort:** High

Multi-field search with autocomplete:
- Title, director, cast, year range
- Genre combinations (Action + Comedy)
- Rating range (7.0-9.0)
- Runtime range

---

### 2.3 Watch History Tracking
**Priority:** Medium | **Effort:** High

Track watched status, watch date, rewatch count.

---

### 2.4 Multi-User Support
**Priority:** Low | **Effort:** Very High

User accounts with separate watch history and ratings.

---

### 2.5 Unit & Integration Tests
**Priority:** High | **Effort:** High

Comprehensive test suite:
- Unit tests for scanner, metadata, writer
- Integration tests for end-to-end workflow
- Docker test environment
- CI/CD pipeline

---

### 2.6 Operations Runbook
**Priority:** High | **Effort:** Medium

Production operations documentation:
- Deployment procedures
- Backup/restore procedures
- Troubleshooting guide
- Log analysis
- Performance tuning

---

## Priority Matrix

| Feature | Priority | Effort | Impact | Version |
|---------|----------|--------|--------|---------|
| TMDB Retry Logic | High | Medium | High | v1.1 |
| Genre URL Persistence | High | Low | Medium | v1.1 |
| Sorting Options | Medium | Low | Medium | v1.1 |
| Accessibility | Medium | Medium | High | v1.1 |
| Metrics/Observability | Medium | Medium-High | High | v1.1 |
| Related Movies | Low | Medium | Low | v1.2 |
| Dark Mode | Low | Low | Low | v1.2 |
| Director Pages | Low | High | Low | v1.2 |
| Recently Added | Low | Low | Low | v1.2 |
| Statistics Dashboard | Low | Medium | Low | v1.2 |
| Image Optimization | Low | Medium | Low | v1.2 |
| SQLite Database | High | High | High | v2.0 |
| Advanced Search | Medium | High | Medium | v2.0 |
| Watch History | Medium | High | Medium | v2.0 |
| Multi-User | Low | Very High | Low | v2.0 |
| Unit Tests | High | High | High | v2.0 |
| Ops Runbook | High | Medium | High | v2.0 |

---

## Implementation Order Recommendation

**Next Sprint (v1.1):**
1. TMDB retry logic (1.1.1) - Foundational reliability
2. Genre URL persistence (1.1.2) - Quick win
3. Sorting options (1.1.3) - Enhances 1.1.2
4. Accessibility basics (1.1.4) - Parallel track
5. Metrics foundation (1.1.5) - Enables monitoring

**Following Month (v1.2):**
- Focus on polish and user experience enhancements
- Low-hanging fruit: dark mode, recently added
- Skip high-effort items (director pages) unless requested

**Long Term (v2.0):**
- SQLite migration is prerequisite for watch history
- Unit tests should be written alongside features
- Ops runbook before production at scale

---

## Deferred/Not Planned

These features are out of scope for a personal movie library:

- ❌ Video streaming/playback (use Plex/Jellyfin)
- ❌ Subtitle management
- ❌ Torrent integration
- ❌ Mobile app (responsive web is sufficient)
- ❌ Social features (sharing, reviews, ratings)
- ❌ Content recommendations from external sources

---

## Contributing

If you implement any of these features, please:
1. Update this roadmap with completion date
2. Add entry to CHANGELOG.md
3. Update TECHNICAL_DOCUMENTATION.md if architecture changes
4. Add usage instructions to README.md
5. Create PR with clear description

---

## Feedback

Have ideas not listed here? Open an issue at:
https://github.com/your-username/movieVault/issues

---

**Last Review:** January 29, 2026
**Next Review:** March 1, 2026 (post v1.1 release)
