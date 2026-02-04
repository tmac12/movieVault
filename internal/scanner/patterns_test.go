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

func TestExtractDiscNumber(t *testing.T) {
	testCases := []struct {
		filename string
		expected int
	}{
		// Basic disc markers
		{"Movie.CD1.avi", 1},
		{"Movie.CD2.avi", 2},
		{"Movie.Disc1.avi", 1},
		{"Movie.Disc2.avi", 2},
		{"Movie.Disk1.avi", 1},
		{"Movie.Part2.avi", 2},
		{"Movie.Pt1.avi", 1},
		// Separator variants
		{"Movie-CD1.avi", 1},
		{"Movie_CD2.avi", 2},
		{"Movie CD1.avi", 1},
		{"Movie.CD.1.avi", 1},
		{"Movie-Disc-2.avi", 2},
		// Case insensitivity
		{"Movie.cd1.avi", 1},
		{"Movie.CD1.avi", 1},
		{"Movie.Cd2.avi", 2},
		{"Movie.DISC1.avi", 1},
		{"Movie.part2.avi", 2},
		// Real bug filenames from the library
		{"Rapunzel.L.Intreccio.della.Torre.2010.Italian.BDRip.Xvid-TRL.CD1.avi", 1},
		{"Rapunzel.L.Intreccio.della.Torre.2010.Italian.BDRip.Xvid-TRL.CD2.avi", 2},
		{"La.Bestia.nel.Cuore.2005.Italian.DVDRip.Xvid-GBM.CD1.avi", 1},
		{"La.Bestia.nel.Cuore.2005.Italian.DVDRip.Xvid-GBM.CD2.avi", 2},
		// Negative cases — no disc marker
		{"Movie.2020.avi", 0},
		{"Movie.avi", 0},
		// "ACDC" should NOT match — no separator before "CD"
		{"ACDC.Greatest.Hits.2020.avi", 0},
		{"The.ACDC.Story.2019.mkv", 0},
	}

	for _, tc := range testCases {
		got := ExtractDiscNumber(tc.filename)
		if got != tc.expected {
			t.Errorf("ExtractDiscNumber(%q) = %d, want %d", tc.filename, got, tc.expected)
		}
	}
}

func TestFilterMultiDiscDuplicates(t *testing.T) {
	testCases := []struct {
		name          string
		input         []FileInfo
		wantCount     int   // expected number of kept files
		wantSkipped   int   // expected number of skipped files
		wantFileNames []string // FileNames that should be in the output
	}{
		{
			name: "CD1+CD2 same dir — CD2 removed",
			input: []FileInfo{
				{Path: "/movies/Movie.CD1.avi", FileName: "Movie.CD1.avi", Title: "Movie", Year: 2020, DiscNumber: 1},
				{Path: "/movies/Movie.CD2.avi", FileName: "Movie.CD2.avi", Title: "Movie", Year: 2020, DiscNumber: 2},
			},
			wantCount:     1,
			wantSkipped:   1,
			wantFileNames: []string{"Movie.CD1.avi"},
		},
		{
			name: "CD2 only — kept (no CD1 to anchor to)",
			input: []FileInfo{
				{Path: "/movies/Movie.CD2.avi", FileName: "Movie.CD2.avi", Title: "Movie", Year: 2020, DiscNumber: 2},
			},
			wantCount:     1,
			wantSkipped:   0,
			wantFileNames: []string{"Movie.CD2.avi"},
		},
		{
			name: "single file no disc — untouched",
			input: []FileInfo{
				{Path: "/movies/Solo.Movie.avi", FileName: "Solo.Movie.avi", Title: "Solo Movie", Year: 2021, DiscNumber: 0},
			},
			wantCount:     1,
			wantSkipped:   0,
			wantFileNames: []string{"Solo.Movie.avi"},
		},
		{
			name: "two different movies each with CD1+CD2",
			input: []FileInfo{
				{Path: "/movies/Alpha.CD1.avi", FileName: "Alpha.CD1.avi", Title: "Alpha", Year: 2020, DiscNumber: 1},
				{Path: "/movies/Alpha.CD2.avi", FileName: "Alpha.CD2.avi", Title: "Alpha", Year: 2020, DiscNumber: 2},
				{Path: "/movies/Beta.CD1.avi", FileName: "Beta.CD1.avi", Title: "Beta", Year: 2021, DiscNumber: 1},
				{Path: "/movies/Beta.CD2.avi", FileName: "Beta.CD2.avi", Title: "Beta", Year: 2021, DiscNumber: 2},
			},
			wantCount:     2,
			wantSkipped:   2,
			wantFileNames: []string{"Alpha.CD1.avi", "Beta.CD1.avi"},
		},
		{
			name: "CD1+CD2 in different dirs — both kept",
			input: []FileInfo{
				{Path: "/movies/dirA/Movie.CD1.avi", FileName: "Movie.CD1.avi", Title: "Movie", Year: 2020, DiscNumber: 1},
				{Path: "/movies/dirB/Movie.CD2.avi", FileName: "Movie.CD2.avi", Title: "Movie", Year: 2020, DiscNumber: 2},
			},
			wantCount:     2,
			wantSkipped:   0,
			wantFileNames: []string{"Movie.CD1.avi", "Movie.CD2.avi"},
		},
		{
			name: "mixed disc and non-disc files",
			input: []FileInfo{
				{Path: "/movies/Single.avi", FileName: "Single.avi", Title: "Single", Year: 2019, DiscNumber: 0},
				{Path: "/movies/Multi.CD1.avi", FileName: "Multi.CD1.avi", Title: "Multi", Year: 2020, DiscNumber: 1},
				{Path: "/movies/Multi.CD2.avi", FileName: "Multi.CD2.avi", Title: "Multi", Year: 2020, DiscNumber: 2},
			},
			wantCount:     2,
			wantSkipped:   1,
			wantFileNames: []string{"Single.avi", "Multi.CD1.avi"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			kept, skipped := FilterMultiDiscDuplicates(tc.input)
			if len(kept) != tc.wantCount {
				t.Errorf("kept %d files, want %d", len(kept), tc.wantCount)
			}
			if len(skipped) != tc.wantSkipped {
				t.Errorf("skipped %d files, want %d", len(skipped), tc.wantSkipped)
			}
			// Verify exact set of kept filenames
			gotNames := make(map[string]bool)
			for _, f := range kept {
				gotNames[f.FileName] = true
			}
			for _, want := range tc.wantFileNames {
				if !gotNames[want] {
					t.Errorf("expected %q in kept files, but not found", want)
				}
			}
		})
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
