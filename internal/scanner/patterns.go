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
	// Source/quality markers (kept separate from resolution)
	qualityPattern = regexp.MustCompile(`(?i)\b(BluRay|BDRip|WEB-DL|WEBRip|HDRip|DVDRip|HDTV)\b`)
	codecPattern   = regexp.MustCompile(`(?i)\b(x264|x265|H\.?264|H\.?265|HEVC|XviD|DivX|AVC)\b`)
	audioPattern        = regexp.MustCompile(`(?i)\b(AAC|AC3|DTS|DD5\.1|TrueHD|Atmos|DTS-HD|MA|FLAC)\b`)
	languagePattern     = regexp.MustCompile(`(?i)\b(ita|eng|spa|fra|deu|jpn|kor|rus|chi|por|pol|nld|swe|nor|dan|fin|tur|ara|heb|tha|vie|ind|msa|hindi|tamil|multi|dual)\b`)
	subtitlePattern     = regexp.MustCompile(`(?i)\b(sub|subs|subtitle|subtitles|subbed)\b`)
	releaseGroupPattern = regexp.MustCompile(`(?i)[-\.]([A-Z0-9]+(\.[A-Z]+)*|MIRCrew|RARBG|YTS|YIFY|PublicHD|Tigole|QxR|UTR|ION10|EVO|CMRG|FGT)$`)
	bracketPattern      = regexp.MustCompile(`\[([^\]]+)\]`)
	extraInfoPattern    = regexp.MustCompile(`(?i)\b(EXTENDED|UNRATED|DIRECTOR.?S.?CUT|REMASTERED|THEATRICAL|IMAX|DC|UHD|HDR|HDR10)\b`)
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

	// Remove extra info
	name = extraInfoPattern.ReplaceAllString(name, " ")

	// Remove release group (usually after a dash at the end)
	name = releaseGroupPattern.ReplaceAllString(name, "")

	// Remove content in brackets
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
