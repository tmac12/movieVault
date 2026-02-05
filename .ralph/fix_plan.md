# Ralph Fix Plan

## High Priority
- [ ] Add `GetMovieByID(tmdbID int)` method to TMDB client
- [ ] When NFO contains `<tmdbid>`, use direct lookup instead of search
- [ ] Fall back to title/year search only if TMDB ID missing or invalid
- [ ] Log which method was used (direct vs search) in verbose mode
- [ ] Handle invalid/deleted TMDB IDs gracefully (fall back to search)
- [ ] Typecheck passes
- [ ] Remove resolution markers: 1080p, 720p, 2160p, 4K, 480p
- [ ] Remove source markers: BluRay, BRRip, WEB-DL, WEBRip, HDRip, DVDRip, HDTV
- [ ] Remove codec markers: x264, x265, HEVC, H.264, H.265, AVC, XviD, DivX
- [ ] Remove audio markers: AAC, AC3, DTS, DTS-HD, TrueHD, FLAC, MP3
- [ ] Patterns are case-insensitive
- [ ] "The.Matrix.1999.1080p.BluRay.x264.mkv" extracts as "The Matrix" (1999)
- [ ] Typecheck passes
- [ ] Remove bracketed groups: [YTS], [YIFY], [RARBG], [EVO], etc.
- [ ] Remove hyphenated suffixes: -SPARKS, -GECKOS, -FGT, etc.
- [ ] Handle groups at start or end of filename
- [ ] Preserve legitimate brackets in titles (e.g., "[REC]" the movie)
- [ ] "Inception.2010.1080p.BluRay-SPARKS.mkv" extracts as "Inception" (2010)
- [ ] Typecheck passes
- [ ] Handle year-starting titles: "2001.A.Space.Odyssey.1968" → "2001: A Space Odyssey" (1968)
- [ ] Handle multi-part titles: "The.Lord.of.the.Rings.The.Fellowship.2001" → correct title
- [ ] Handle extended editions: "Movie.2020.Extended.Cut" → "Movie" (2020), not "Movie Extended Cut"
- [ ] Handle director's cuts: "Movie.2020.Directors.Cut" → "Movie" (2020)
- [ ] Preserve intentional dots in titles when possible
- [ ] Typecheck passes
- [ ] Add `--test-parser` flag that accepts filenames as arguments
- [ ] Output extracted title, year, and matched patterns for each filename
- [ ] Support reading filenames from stdin for batch testing
- [ ] Exit with non-zero code if any extraction fails
- [ ] Example: `./scanner --test-parser "The.Matrix.1999.1080p.mkv"` outputs title/year


## Medium Priority


## Low Priority


## Completed
- [x] Project enabled for Ralph

## Notes
- Focus on MVP functionality first
- Ensure each feature is properly tested
- Update this file after each major milestone
