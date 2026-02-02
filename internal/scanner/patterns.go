package scanner

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	// Patterns to remove from filenames
	yearPattern = regexp.MustCompile(`[\[\(]?(\d{4})[\]\)]?`)
	// Resolution markers (US-010)
	resolutionPattern = regexp.MustCompile(`(?i)\b(2160p|1080p|1080i|720p|720i|480p|4K)\b`)
	// Source/quality markers (kept separate from resolution) (US-011)
	// Includes: BluRay, BRRip, WEB-DL, WEBRip, HDRip, DVDRip, HDTV, BDRip, WEB, AMZN, NF
	qualityPattern = regexp.MustCompile(`(?i)\b(BluRay|BRRip|WEB-DL|WEBRip|HDRip|DVDRip|HDTV|BDRip|WEB|AMZN|NF)\b`)
	// Codec markers (US-012)
	// Includes: x264, x265, HEVC, H.264, H.265, H264, H265, AVC, XviD, DivX, 10bit, HDR, HDR10, DV
	codecPattern = regexp.MustCompile(`(?i)\b(x264|x265|H\.?264|H\.?265|HEVC|XviD|DivX|AVC|10bit|HDR10|HDR|DV)\b`)
	// Audio codec markers (US-013)
	// Includes: AAC, AC3, DTS, DTS-HD, TrueHD, FLAC, MP3, DD5.1, DD2.0, Atmos, 5.1, 7.1, 2.0
	audioPattern = regexp.MustCompile(`(?i)\b(AAC|AC3|DTS-HD|DTS|TrueHD|FLAC|MP3|DD5\.1|DD2\.0|Atmos|7\.1|5\.1|2\.0|MA)\b`)
	languagePattern     = regexp.MustCompile(`(?i)\b(ita|eng|spa|fra|deu|jpn|kor|rus|chi|por|pol|nld|swe|nor|dan|fin|tur|ara|heb|tha|vie|ind|msa|hindi|tamil|multi|dual)\b`)
	subtitlePattern     = regexp.MustCompile(`(?i)\b(sub|subs|subtitle|subtitles|subbed)\b`)
	// Release group patterns (US-014)
	// Hyphenated suffixes at end: -SPARKS, -GECKOS, -FGT, -YIFY, etc.
	releaseGroupPattern = regexp.MustCompile(`(?i)[-\.]([A-Z0-9]+(\.[A-Z]+)*|MIRCrew|RARBG|YTS|YIFY|PublicHD|Tigole|QxR|UTR|ION10|EVO|CMRG|FGT|SPARKS|GECKOS|AMIABLE|DRONES|BLOW|GALACTICA|CODEX|SKIDROW|PLAZA|CPY|RELOADED|TERMiNAL|DEFLATE|CHD|RuDE|VETO|CiNEFiLE|PSYCHD)$`)
	// Bracketed groups: [YTS], [YIFY], [RARBG], [EVO], [FGT], etc.
	bracketedGroupPattern = regexp.MustCompile(`(?i)\[(YTS|YIFY|RARBG|EVO|FGT|MULTi|SPARKS|GECKOS|1080p|720p|2160p|4K|WEB|BRRip|BluRay|x264|x265|HEVC|HDR|DTS|AAC|FLAC|MP3|ENG|ITA|SPA|GER|FRA|RUS|JPN|KOR|CHI|MULTi|NF|AMZN|HULU|DSNP|MAX|PCOK|[A-Za-z0-9\.]+)\]`)
	// Generic bracket content (catches remaining)
	bracketPattern = regexp.MustCompile(`\[([^\]]+)\]`)
	// Edition markers (US-015)
	// Includes: Extended, Extended.Cut, Directors.Cut, Director's.Cut, Unrated, Theatrical, IMAX, Remastered
	// Also keeps: DC (Director's Cut abbreviation), UHD
	editionPattern = regexp.MustCompile(`(?i)\b(EXTENDED\.?CUT|EXTENDED|DIRECTOR\'?S\.?CUT|DIRECTORS\.?CUT|UNRATED|THEATRICAL|IMAX|REMASTERED|DC|UHD)\b`)
	// Legacy alias for backwards compatibility
	extraInfoPattern = editionPattern
)

// ExtractTitleAndYear extracts the movie title and year from a filename
func ExtractTitleAndYear(filename string) (title string, year int) {
	// Remove file extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Remove resolution markers FIRST (US-010)
	// This must happen before year extraction to prevent "1080p" from being
	// parsed as year "1080" with leftover "p"
	name = resolutionPattern.ReplaceAllString(name, " ")

	// Extract year if present
	yearMatches := yearPattern.FindStringSubmatch(name)
	if len(yearMatches) > 1 {
		year, _ = strconv.Atoi(yearMatches[1])
	}

	// Remove year from filename
	name = yearPattern.ReplaceAllString(name, "")

	// Remove quality markers
	name = qualityPattern.ReplaceAllString(name, " ")

	// Remove codec info
	name = codecPattern.ReplaceAllString(name, " ")

	// Remove audio info
	name = audioPattern.ReplaceAllString(name, " ")

	// Remove language codes
	name = languagePattern.ReplaceAllString(name, " ")

	// Remove subtitle markers
	name = subtitlePattern.ReplaceAllString(name, " ")

	// Remove edition markers (US-015)
	name = editionPattern.ReplaceAllString(name, " ")

	// Remove bracketed release groups first (US-014)
	// e.g., [YTS], [YIFY], [RARBG], [EVO], [FGT]
	name = bracketedGroupPattern.ReplaceAllString(name, " ")

	// Remove release group (usually after a dash at the end) (US-014)
	// e.g., -SPARKS, -GECKOS, -FGT, -YIFY
	name = releaseGroupPattern.ReplaceAllString(name, "")

	// Remove any remaining content in brackets
	name = bracketPattern.ReplaceAllString(name, " ")

	// Replace dots and underscores with spaces
	name = strings.ReplaceAll(name, ".", " ")
	name = strings.ReplaceAll(name, "_", " ")

	// Remove multiple spaces
	name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")

	// Trim whitespace
	title = strings.TrimSpace(name)

	return title, year
}

// GenerateSlug creates a URL-friendly slug from title and year
func GenerateSlug(title string, year int) string {
	// Convert to lowercase
	slug := strings.ToLower(title)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove special characters (keep only alphanumeric and hyphens)
	slug = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(slug, "")

	// Remove multiple consecutive hyphens
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	// Append year if available
	if year > 0 {
		slug = slug + "-" + strconv.Itoa(year)
	}

	return slug
}

// CleanTitle performs additional cleaning on the extracted title
func CleanTitle(title string) string {
	// Remove leading/trailing whitespace
	title = strings.TrimSpace(title)

	// Capitalize first letter of each word
	words := strings.Fields(title)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}
