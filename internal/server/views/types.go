package views

import "creator-crawler/internal/storage"

type GamesPageData struct {
	Games []storage.GameSummary
	Error string
}

type MyGamesPageData struct {
	MyGames []storage.MyGameSummary
	Error   string
}

type MyGameDetailData struct {
	MyGame            storage.MyGame
	Tags              []storage.Tag
	SimilarGames      []storage.MyGameSimilarGame
	SuggestedGames    []storage.MyGameSimilarGame
	Creators          []storage.MyGameCreator
	SuggestedCreators []storage.SuggestedCreator
	AllGames          []storage.Game
	ApprovedCreators  []storage.CreatorSummary
	Notice            string
	Error             string
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
	Tags       []storage.Tag
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
