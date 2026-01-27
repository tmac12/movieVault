package metadata

// TMDBSearchResponse represents the response from TMDB search API
type TMDBSearchResponse struct {
	Page         int            `json:"page"`
	Results      []TMDBMovie    `json:"results"`
	TotalPages   int            `json:"total_pages"`
	TotalResults int            `json:"total_results"`
}

// TMDBMovie represents a movie from TMDB API
type TMDBMovie struct {
	ID               int      `json:"id"`
	Title            string   `json:"title"`
	OriginalTitle    string   `json:"original_title"`
	Overview         string   `json:"overview"`
	PosterPath       string   `json:"poster_path"`
	BackdropPath     string   `json:"backdrop_path"`
	ReleaseDate      string   `json:"release_date"`
	VoteAverage      float64  `json:"vote_average"`
	VoteCount        int      `json:"vote_count"`
	Popularity       float64  `json:"popularity"`
	GenreIDs         []int    `json:"genre_ids"`
	Adult            bool     `json:"adult"`
	Video            bool     `json:"video"`
	OriginalLanguage string   `json:"original_language"`
}

// TMDBMovieDetails represents detailed movie information from TMDB
type TMDBMovieDetails struct {
	ID               int                  `json:"id"`
	Title            string               `json:"title"`
	OriginalTitle    string               `json:"original_title"`
	Overview         string               `json:"overview"`
	Tagline          string               `json:"tagline"`
	PosterPath       string               `json:"poster_path"`
	BackdropPath     string               `json:"backdrop_path"`
	ReleaseDate      string               `json:"release_date"`
	Runtime          int                  `json:"runtime"`
	VoteAverage      float64              `json:"vote_average"`
	VoteCount        int                  `json:"vote_count"`
	Popularity       float64              `json:"popularity"`
	Budget           int64                `json:"budget"`
	Revenue          int64                `json:"revenue"`
	Genres           []TMDBGenre          `json:"genres"`
	ProductionCompanies []TMDBCompany     `json:"production_companies"`
	SpokenLanguages  []TMDBLanguage       `json:"spoken_languages"`
	Status           string               `json:"status"`
	IMDbID           string               `json:"imdb_id"`
	Homepage         string               `json:"homepage"`
	Adult            bool                 `json:"adult"`
	Video            bool                 `json:"video"`
	OriginalLanguage string               `json:"original_language"`
}

// TMDBGenre represents a movie genre
type TMDBGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// TMDBCompany represents a production company
type TMDBCompany struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	LogoPath      string `json:"logo_path"`
	OriginCountry string `json:"origin_country"`
}

// TMDBLanguage represents a spoken language
type TMDBLanguage struct {
	ISO6391     string `json:"iso_639_1"`
	EnglishName string `json:"english_name"`
	Name        string `json:"name"`
}

// TMDBCreditsResponse represents the credits (cast and crew) response
type TMDBCreditsResponse struct {
	ID   int              `json:"id"`
	Cast []TMDBCastMember `json:"cast"`
	Crew []TMDBCrewMember `json:"crew"`
}

// TMDBCastMember represents a cast member
type TMDBCastMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Character   string `json:"character"`
	Order       int    `json:"order"`
	ProfilePath string `json:"profile_path"`
}

// TMDBCrewMember represents a crew member
type TMDBCrewMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Job         string `json:"job"`
	Department  string `json:"department"`
	ProfilePath string `json:"profile_path"`
}
