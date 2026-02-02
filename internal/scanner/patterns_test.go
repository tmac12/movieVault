package scanner

import (
	"testing"
)

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
