package nfo

import "encoding/xml"

// NFOMovie represents the structure of a Jellyfin .nfo XML file
type NFOMovie struct {
	XMLName   xml.Name    `xml:"movie"`
	Title     string      `xml:"title"`
	Plot      string      `xml:"plot"`
	Rating    float64     `xml:"rating"`
	Year      int         `xml:"year"`
	Premiered string      `xml:"premiered"`
	Runtime   int         `xml:"runtime"`
	Genres    []string    `xml:"genre"`
	Directors []string    `xml:"director"`
	Actors    []NFOActor  `xml:"actor"`
	TMDBID    int         `xml:"tmdbid"`
	IMDbID    string      `xml:"imdbid"`
	Thumbs    []NFOThumb  `xml:"thumb"`
	Fanart    *NFOFanart  `xml:"fanart"`
	Art       *NFOArt     `xml:"art"`
}

// NFOActor represents an actor in the .nfo file
type NFOActor struct {
	Name  string `xml:"name"`
	Role  string `xml:"role"`
	Thumb string `xml:"thumb"`
}

// NFOThumb represents a thumbnail/poster image
type NFOThumb struct {
	Aspect  string `xml:"aspect,attr"`
	URL     string `xml:",chardata"`
}

// NFOFanart represents fanart/backdrop images
type NFOFanart struct {
	Thumbs []NFOThumb `xml:"thumb"`
}

// NFOArt represents the <art> block used by Jellyfin/Kodi
type NFOArt struct {
	Poster string `xml:"poster"`
	Fanart string `xml:"fanart"`
}
