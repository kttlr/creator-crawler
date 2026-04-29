package views

import "creator-crawler/internal/storage"

type GamesPageData struct {
	Games []storage.GameSummary
	Error string
}

type GameDetailData struct {
	Game       storage.Game
	Videos     []storage.GameVideo
	Runs       []storage.SearchRun
	Filters    VideoFilters
	Notice     string
	Error      string
	SearchForm SearchForm
}

type SearchForm struct {
	Query           string
	Limit           int
	IncludeFiltered bool
}

type VideoFilters struct {
	Status string
	Query  string
	Sort   string
}
