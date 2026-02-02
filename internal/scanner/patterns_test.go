package scanner

import (
	"testing"
)

// TestYearStartingTitles tests US-016 acceptance criteria
// Movies that start with years (like "2001: A Space Odyssey" and "1917")
// should be parsed correctly
func TestYearStartingTitles(t *testing.T) {
	testCases := []struct {
		filename      string
		expectedTitle string
		expectedYear  int
	}{
		// US-016 acceptance criteria test cases
		{"2001.A.Space.Odyssey.1968.mkv", "2001 A Space Odyssey", 1968},
		{"1917.2019.1080p.mkv", "1917", 2019},
		// Additional year-starting title cases
		{"2001.A.Space.Odyssey.1968.BluRay.mkv", "2001 A Space Odyssey", 1968},
		{"1917.2019.BluRay.x264.mkv", "1917", 2019},
		{"1984.1984.1080p.mkv", "1984", 1984},
		{"2012.2009.WEB-DL.mkv", "2012", 2009},
		{"300.2006.1080p.BluRay.mkv", "300", 2006}, // 3-digit title (not a year)
		// Year in parentheses should always be treated as release year
		{"2001.A.Space.Odyssey.(1968).mkv", "2001 A Space Odyssey", 1968},
		{"1917.(2019).mkv", "1917", 2019},
		// Year in brackets should always be treated as release year
		{"2001.A.Space.Odyssey.[1968].mkv", "2001 A Space Odyssey", 1968},
		// Normal cases (should still work)
		{"The.Matrix.1999.1080p.mkv", "The Matrix", 1999},
		{"Inception.2010.BluRay.mkv", "Inception", 2010},
		{"Movie.2020.mkv", "Movie", 2020},
	}

	for _, tc := range testCases {
		title, year := ExtractTitleAndYear(tc.filename)
		if title != tc.expectedTitle || year != tc.expectedYear {
			t.Errorf("ExtractTitleAndYear(%q) = (%q, %d), want (%q, %d)",
				tc.filename, title, year, tc.expectedTitle, tc.expectedYear)
		}
	}
}

func TestEditionMarkers(t *testing.T) {
	testCases := []struct {
		filename     string
		expectedTitle string
		expectedYear  int
	}{
		// US-015 acceptance criteria test cases
		{"Movie.2020.Extended.Cut.mkv", "Movie", 2020},
		{"Movie.2020.Directors.Cut.mkv", "Movie", 2020},
		{"Movie.2020.Director's.Cut.mkv", "Movie", 2020},
		{"Movie.2020.Unrated.mkv", "Movie", 2020},
		{"Movie.2020.Theatrical.mkv", "Movie", 2020},
		{"Movie.2020.IMAX.mkv", "Movie", 2020},
		{"Movie.2020.Remastered.mkv", "Movie", 2020},
		// Additional cases
		{"The.Matrix.1999.Extended.Cut.1080p.BluRay.mkv", "The Matrix", 1999},
		{"Blade.Runner.1982.Directors.Cut.2160p.mkv", "Blade Runner", 1982},
		{"Alien.1979.IMAX.Remastered.mkv", "Alien", 1979},
		// Case insensitivity
		{"Movie.2020.extended.cut.mkv", "Movie", 2020},
		{"Movie.2020.DIRECTORS.CUT.mkv", "Movie", 2020},
	}

	for _, tc := range testCases {
		title, year := ExtractTitleAndYear(tc.filename)
		if title != tc.expectedTitle || year != tc.expectedYear {
			t.Errorf("ExtractTitleAndYear(%q) = (%q, %d), want (%q, %d)",
				tc.filename, title, year, tc.expectedTitle, tc.expectedYear)
		}
	}
}
