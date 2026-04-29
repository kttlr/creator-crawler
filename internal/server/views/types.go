package views

import "creator-crawler/internal/storage"

type GamesPageData struct {
	Games []storage.GameSummary
	Error string
}

type CreatorsPageData struct {
	Creators []storage.CreatorSummary
	Filters  CreatorFilters
}

type CreatorDetailData struct {
	Creator storage.CreatorSummary
	Games   []CreatorGameGroup
}

type CreatorGameGroup struct {
	GameID int64
	Name   string
	Videos []storage.CreatorVideo
}

type CreatorFilters struct {
	Query string
	Sort  string
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
