package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/starfederation/datastar-go/datastar"

	"creator-crawler/internal/config"
	"creator-crawler/internal/server/views"
	"creator-crawler/internal/storage"
	"creator-crawler/internal/youtube"
)

type Server struct {
	store  *storage.Store
	dbPath string
}

func New(store *storage.Store, dbPath string) *Server {
	return &Server{store: store, dbPath: dbPath}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("GET /", s.redirectHome)
	mux.HandleFunc("GET /my-games", s.myGames)
	mux.HandleFunc("POST /my-games", s.createMyGame)
	mux.HandleFunc("GET /my-games/{id}", s.myGameDetail)
	mux.HandleFunc("POST /my-games/{id}", s.updateMyGame)
	mux.HandleFunc("POST /my-games/{id}/tags", s.updateMyGameTags)
	mux.HandleFunc("POST /my-games/{id}/games", s.addMyGameSimilarGame)
	mux.HandleFunc("POST /my-games/{id}/games/{gameID}/remove", s.removeMyGameSimilarGame)
	mux.HandleFunc("POST /my-games/{id}/creators", s.addMyGameCreator)
	mux.HandleFunc("POST /my-games/{id}/creators/{channelID}/contacted", s.updateMyGameCreatorContacted)
	mux.HandleFunc("POST /my-games/{id}/creators/{channelID}/remove", s.removeMyGameCreator)
	mux.HandleFunc("GET /creators", s.creators)
	mux.HandleFunc("GET /creators/{channelID}", s.creatorDetail)
	mux.HandleFunc("POST /creators/{channelID}/email", s.updateCreatorEmail)
	mux.HandleFunc("GET /games", s.games)
	mux.HandleFunc("POST /games", s.createGame)
	mux.HandleFunc("GET /games/{id}", s.gameDetail)
	mux.HandleFunc("POST /games/{id}/tags", s.updateGameTags)
	mux.HandleFunc("POST /games/{id}/searches", s.createSearch)
	mux.HandleFunc("GET /games/{id}/videos", s.gameVideos)
	mux.HandleFunc("POST /games/{id}/videos/{videoID}/status", s.updateVideoStatus)
	mux.HandleFunc("POST /games/{id}/videos/{videoID}/notes", s.updateVideoNotes)
	return mux
}

func (s *Server) redirectHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/my-games", http.StatusFound)
}

func (s *Server) games(w http.ResponseWriter, r *http.Request) {
	games, err := s.store.ListGameSummaries(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, views.GamesPage(views.GamesPageData{Games: games}))
}

func (s *Server) createGame(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderGamesWithError(w, r, err.Error())
		return
	}
	game, err := s.store.AddGame(r.Context(), r.FormValue("name"))
	if err != nil {
		s.renderGamesWithError(w, r, err.Error())
		return
	}
	s.redirect(w, r, views.GamePath(game.ID))
}

func (s *Server) myGames(w http.ResponseWriter, r *http.Request) {
	myGames, err := s.store.ListMyGameSummaries(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, views.MyGamesPage(views.MyGamesPageData{MyGames: myGames}))
}

func (s *Server) createMyGame(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderMyGamesWithError(w, r, err.Error())
		return
	}
	game, err := s.store.AddMyGame(r.Context(), r.FormValue("name"), r.FormValue("description"), r.FormValue("notes"))
	if err != nil {
		s.renderMyGamesWithError(w, r, err.Error())
		return
	}
	if err := s.store.SetMyGameTags(r.Context(), game.ID, r.FormValue("tags")); err != nil {
		s.renderMyGamesWithError(w, r, err.Error())
		return
	}
	s.redirect(w, r, views.MyGamePath(game.ID))
}

func (s *Server) myGameDetail(w http.ResponseWriter, r *http.Request) {
	data, ok := s.loadMyGameDetail(w, r, "", "")
	if !ok {
		return
	}
	s.render(w, r, views.MyGameDetailPage(data))
}

func (s *Server) updateMyGame(w http.ResponseWriter, r *http.Request) {
	myGameID, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderMyGameDetailWithMessage(w, r, "", err.Error())
		return
	}
	if err := s.store.UpdateMyGame(r.Context(), myGameID, r.FormValue("name"), r.FormValue("description"), r.FormValue("notes")); err != nil {
		s.renderMyGameDetailWithMessage(w, r, "", err.Error())
		return
	}
	s.redirect(w, r, views.MyGamePath(myGameID))
}

func (s *Server) updateMyGameTags(w http.ResponseWriter, r *http.Request) {
	myGameID, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderMyGameDetailWithMessage(w, r, "", err.Error())
		return
	}
	if err := s.store.SetMyGameTags(r.Context(), myGameID, r.FormValue("tags")); err != nil {
		s.renderMyGameDetailWithMessage(w, r, "", err.Error())
		return
	}
	s.redirect(w, r, views.MyGamePath(myGameID))
}

func (s *Server) addMyGameSimilarGame(w http.ResponseWriter, r *http.Request) {
	myGameID, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderMyGameDetailWithMessage(w, r, "", err.Error())
		return
	}
	gameID, err := strconv.ParseInt(r.FormValue("game_id"), 10, 64)
	if err != nil || gameID <= 0 {
		s.renderMyGameDetailWithMessage(w, r, "", "choose a tracked game")
		return
	}
	if err := s.store.AddSimilarGameToMyGame(r.Context(), myGameID, gameID, r.FormValue("notes")); err != nil {
		s.renderMyGameDetailWithMessage(w, r, "", err.Error())
		return
	}
	s.redirect(w, r, views.MyGamePath(myGameID))
}

func (s *Server) removeMyGameSimilarGame(w http.ResponseWriter, r *http.Request) {
	myGameID, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	gameID, ok := s.pathInt(w, r, "gameID")
	if !ok {
		return
	}
	if err := s.store.RemoveSimilarGameFromMyGame(r.Context(), myGameID, gameID); err != nil {
		s.renderMyGameDetailWithMessage(w, r, "", err.Error())
		return
	}
	s.redirect(w, r, views.MyGamePath(myGameID))
}

func (s *Server) addMyGameCreator(w http.ResponseWriter, r *http.Request) {
	myGameID, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderMyGameDetailWithMessage(w, r, "", err.Error())
		return
	}
	channelID := strings.TrimSpace(r.FormValue("channel_id"))
	if channelID == "" {
		s.renderMyGameDetailWithMessage(w, r, "", "choose a creator")
		return
	}
	if err := s.store.AddCreatorToMyGame(r.Context(), myGameID, channelID, r.FormValue("notes")); err != nil {
		s.renderMyGameDetailWithMessage(w, r, "", err.Error())
		return
	}
	s.redirect(w, r, views.MyGamePath(myGameID))
}

func (s *Server) removeMyGameCreator(w http.ResponseWriter, r *http.Request) {
	myGameID, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	channelID := strings.TrimSpace(r.PathValue("channelID"))
	if channelID == "" {
		http.NotFound(w, r)
		return
	}
	if err := s.store.RemoveCreatorFromMyGame(r.Context(), myGameID, channelID); err != nil {
		s.renderMyGameDetailWithMessage(w, r, "", err.Error())
		return
	}
	s.redirect(w, r, views.MyGamePath(myGameID))
}

func (s *Server) updateMyGameCreatorContacted(w http.ResponseWriter, r *http.Request) {
	myGameID, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	channelID := strings.TrimSpace(r.PathValue("channelID"))
	if channelID == "" {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderMyGameDetailWithMessage(w, r, "", err.Error())
		return
	}
	contacted := r.FormValue("contacted") == "true"
	if err := s.store.UpdateMyGameCreatorContacted(r.Context(), myGameID, channelID, contacted); err != nil {
		s.renderMyGameDetailWithMessage(w, r, "", err.Error())
		return
	}
	s.redirect(w, r, views.MyGamePath(myGameID))
}

func (s *Server) updateCreatorEmail(w http.ResponseWriter, r *http.Request) {
	channelID := strings.TrimSpace(r.PathValue("channelID"))
	if channelID == "" {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, err)
		return
	}
	if err := s.store.UpdateCreatorEmail(r.Context(), channelID, r.FormValue("email")); err != nil {
		s.renderError(w, r, err)
		return
	}
	s.redirect(w, r, views.CreatorPath(channelID))
}

func (s *Server) creators(w http.ResponseWriter, r *http.Request) {
	filters := views.CreatorFilters{
		Query: r.URL.Query().Get("q"),
		Sort:  firstNonEmpty(r.URL.Query().Get("sort"), "approved"),
	}
	creators, err := s.store.ListApprovedCreators(r.Context(), storage.ListApprovedCreatorsOptions{Query: filters.Query, Sort: filters.Sort})
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, views.CreatorsPage(views.CreatorsPageData{Creators: creators, Filters: filters}))
}

func (s *Server) creatorDetail(w http.ResponseWriter, r *http.Request) {
	channelID := strings.TrimSpace(r.PathValue("channelID"))
	if channelID == "" {
		http.NotFound(w, r)
		return
	}

	creator, found, err := s.store.GetCreator(r.Context(), channelID)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	videos, err := s.store.ListCreatorVideos(r.Context(), channelID)
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	s.render(w, r, views.CreatorDetailPage(views.CreatorDetailData{Creator: creator, Games: groupCreatorVideos(videos)}))
}

func (s *Server) gameDetail(w http.ResponseWriter, r *http.Request) {
	data, ok := s.loadGameDetail(w, r, "", "")
	if !ok {
		return
	}
	s.render(w, r, views.GameDetailPage(data))
}

func (s *Server) createSearch(w http.ResponseWriter, r *http.Request) {
	gameID, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	game, found, err := s.store.GetGame(r.Context(), gameID)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		data, ok := s.loadGameDetail(w, r, "", err.Error())
		if ok {
			s.patch(w, r, views.GameDetailContent(data), "game-content")
		}
		return
	}

	query := strings.TrimSpace(r.FormValue("query"))
	if query == "" {
		query = game.Name
	}
	limit := parsePositiveInt(r.FormValue("limit"), 100)
	includeFiltered := r.FormValue("include_filtered") == "on"

	cfg, err := config.Load()
	if err != nil {
		data, ok := s.loadGameDetail(w, r, "", err.Error())
		if ok {
			s.patch(w, r, views.GameDetailContent(data), "game-content")
		}
		return
	}

	client := youtube.NewClient(cfg.YouTubeAPIKey)
	results, err := client.Search(r.Context(), query, limit, includeFiltered)
	if err != nil {
		data, ok := s.loadGameDetail(w, r, "", err.Error())
		if ok {
			s.patch(w, r, views.GameDetailContent(data), "game-content")
		}
		return
	}

	searchRunID, err := s.store.SaveSearchResults(r.Context(), game, query, limit, includeFiltered, results)
	if err != nil {
		data, ok := s.loadGameDetail(w, r, "", err.Error())
		if ok {
			s.patch(w, r, views.GameDetailContent(data), "game-content")
		}
		return
	}

	notice := fmt.Sprintf("Stored search run %d. %s.", searchRunID, views.SearchSummary(len(results), s.dbPath))
	data, ok := s.loadGameDetail(w, r, notice, "")
	if !ok {
		return
	}
	data.SearchForm = views.SearchForm{Query: query, Limit: limit, IncludeFiltered: includeFiltered}
	s.patch(w, r, views.GameDetailContent(data), "game-content")
}

func (s *Server) updateGameTags(w http.ResponseWriter, r *http.Request) {
	gameID, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, err)
		return
	}
	if err := s.store.SetGameTags(r.Context(), gameID, r.FormValue("tags")); err != nil {
		s.renderError(w, r, err)
		return
	}
	s.redirect(w, r, views.GamePath(gameID))
}

func (s *Server) gameVideos(w http.ResponseWriter, r *http.Request) {
	data, ok := s.loadGameDetail(w, r, "", "")
	if !ok {
		return
	}
	s.patch(w, r, views.VideosSection(data.Game, data.Videos, data.Filters), "videos-section")
}

func (s *Server) updateVideoStatus(w http.ResponseWriter, r *http.Request) {
	gameID, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	videoID := r.PathValue("videoID")
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, err)
		return
	}
	if err := s.store.UpdateGameVideoStatus(r.Context(), gameID, videoID, r.FormValue("status")); err != nil {
		s.renderError(w, r, err)
		return
	}
	s.renderUpdatedVideoRow(w, r, gameID, videoID)
}

func (s *Server) updateVideoNotes(w http.ResponseWriter, r *http.Request) {
	gameID, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	videoID := r.PathValue("videoID")
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, err)
		return
	}
	if err := s.store.UpdateGameVideoNotes(r.Context(), gameID, videoID, r.FormValue("notes")); err != nil {
		s.renderError(w, r, err)
		return
	}
	s.renderUpdatedVideoRow(w, r, gameID, videoID)
}

func (s *Server) renderUpdatedVideoRow(w http.ResponseWriter, r *http.Request, gameID int64, videoID string) {
	videos, err := s.store.ListGameVideos(r.Context(), storage.ListGameVideosOptions{GameID: gameID, Status: "all"})
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	for _, video := range videos {
		if video.VideoID == videoID {
			s.patch(w, r, views.VideoRow(video), "video-row-"+video.VideoID)
			return
		}
	}
	http.NotFound(w, r)
}

func (s *Server) loadGameDetail(w http.ResponseWriter, r *http.Request, notice, message string) (views.GameDetailData, bool) {
	gameID, ok := s.pathInt(w, r, "id")
	if !ok {
		return views.GameDetailData{}, false
	}
	game, found, err := s.store.GetGame(r.Context(), gameID)
	if err != nil {
		s.renderError(w, r, err)
		return views.GameDetailData{}, false
	}
	if !found {
		http.NotFound(w, r)
		return views.GameDetailData{}, false
	}

	filters := views.VideoFilters{
		Status: firstNonEmpty(r.URL.Query().Get("status"), "candidate"),
		Query:  r.URL.Query().Get("q"),
		Sort:   firstNonEmpty(r.URL.Query().Get("sort"), "views"),
	}
	videos, err := s.store.ListGameVideos(r.Context(), storage.ListGameVideosOptions{GameID: game.ID, Status: filters.Status, Query: filters.Query, Sort: filters.Sort})
	if err != nil {
		s.renderError(w, r, err)
		return views.GameDetailData{}, false
	}
	runs, err := s.store.ListSearchRuns(r.Context(), game.ID, 10)
	if err != nil {
		s.renderError(w, r, err)
		return views.GameDetailData{}, false
	}
	tags, err := s.store.ListTagsForGame(r.Context(), game.ID)
	if err != nil {
		s.renderError(w, r, err)
		return views.GameDetailData{}, false
	}

	return views.GameDetailData{
		Game:    game,
		Tags:    tags,
		Videos:  videos,
		Runs:    runs,
		Filters: filters,
		Notice:  notice,
		Error:   message,
		SearchForm: views.SearchForm{
			Query: game.Name,
			Limit: 100,
		},
	}, true
}

func (s *Server) loadMyGameDetail(w http.ResponseWriter, r *http.Request, notice, message string) (views.MyGameDetailData, bool) {
	myGameID, ok := s.pathInt(w, r, "id")
	if !ok {
		return views.MyGameDetailData{}, false
	}
	myGame, found, err := s.store.GetMyGame(r.Context(), myGameID)
	if err != nil {
		s.renderError(w, r, err)
		return views.MyGameDetailData{}, false
	}
	if !found {
		http.NotFound(w, r)
		return views.MyGameDetailData{}, false
	}

	tags, err := s.store.ListTagsForMyGame(r.Context(), myGame.ID)
	if err != nil {
		s.renderError(w, r, err)
		return views.MyGameDetailData{}, false
	}
	similarGames, err := s.store.ListMyGameSimilarGames(r.Context(), myGame.ID)
	if err != nil {
		s.renderError(w, r, err)
		return views.MyGameDetailData{}, false
	}
	suggestedGames, err := s.store.ListSuggestedGamesForMyGame(r.Context(), myGame.ID)
	if err != nil {
		s.renderError(w, r, err)
		return views.MyGameDetailData{}, false
	}
	creators, err := s.store.ListCreatorsForMyGame(r.Context(), myGame.ID)
	if err != nil {
		s.renderError(w, r, err)
		return views.MyGameDetailData{}, false
	}
	suggestedCreators, err := s.store.ListSuggestedCreatorsForMyGame(r.Context(), myGame.ID)
	if err != nil {
		s.renderError(w, r, err)
		return views.MyGameDetailData{}, false
	}
	allGames, err := s.store.ListGames(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return views.MyGameDetailData{}, false
	}
	approvedCreators, err := s.store.ListApprovedCreators(r.Context(), storage.ListApprovedCreatorsOptions{})
	if err != nil {
		s.renderError(w, r, err)
		return views.MyGameDetailData{}, false
	}

	return views.MyGameDetailData{
		MyGame:            myGame,
		Tags:              tags,
		SimilarGames:      similarGames,
		SuggestedGames:    suggestedGames,
		Creators:          creators,
		SuggestedCreators: suggestedCreators,
		AllGames:          allGames,
		ApprovedCreators:  approvedCreators,
		Notice:            notice,
		Error:             message,
	}, true
}

func (s *Server) renderGamesWithError(w http.ResponseWriter, r *http.Request, message string) {
	games, err := s.store.ListGameSummaries(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.patch(w, r, views.GamesPage(views.GamesPageData{Games: games, Error: message}), "")
}

func (s *Server) renderMyGamesWithError(w http.ResponseWriter, r *http.Request, message string) {
	myGames, err := s.store.ListMyGameSummaries(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.patch(w, r, views.MyGamesPage(views.MyGamesPageData{MyGames: myGames, Error: message}), "")
}

func (s *Server) renderMyGameDetailWithMessage(w http.ResponseWriter, r *http.Request, notice, message string) {
	data, ok := s.loadMyGameDetail(w, r, notice, message)
	if !ok {
		return
	}
	s.render(w, r, views.MyGameDetailPage(data))
}

func groupCreatorVideos(videos []storage.CreatorVideo) []views.CreatorGameGroup {
	groups := make([]views.CreatorGameGroup, 0)
	indexByGameID := make(map[int64]int)
	for _, video := range videos {
		index, ok := indexByGameID[video.GameID]
		if !ok {
			index = len(groups)
			indexByGameID[video.GameID] = index
			groups = append(groups, views.CreatorGameGroup{GameID: video.GameID, Name: video.GameName})
		}
		groups[index].Videos = append(groups[index].Videos, video)
	}
	return groups
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) patch(w http.ResponseWriter, r *http.Request, component templ.Component, selectorID string) {
	if !isDatastar(r) {
		s.render(w, r, component)
		return
	}

	sse := datastar.NewSSE(w, r)
	options := []datastar.PatchElementOption{datastar.WithModeOuter()}
	if selectorID != "" {
		options = append(options, datastar.WithSelectorID(selectorID))
	}
	if err := sse.PatchElementTempl(component, options...); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request, path string) {
	if !isDatastar(r) {
		http.Redirect(w, r, path, http.StatusSeeOther)
		return
	}

	if err := datastar.NewSSE(w, r).Redirect(path); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func isDatastar(r *http.Request) bool {
	return r.Header.Get("Datastar-Request") == "true"
}

func (s *Server) renderError(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func (s *Server) pathInt(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	value, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || value <= 0 {
		http.NotFound(w, r)
		return 0, false
	}
	return value, true
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func Serve(ctx context.Context, addr, dbPath string) error {
	if strings.TrimSpace(addr) == "" {
		addr = "localhost:8080"
	}
	store, err := storage.Open(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	server := &http.Server{Addr: addr, Handler: New(store, dbPath).Handler()}
	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()

	if !strings.HasPrefix(addr, "localhost:") && !strings.HasPrefix(addr, "127.0.0.1:") {
		fmt.Printf("Warning: serving on %s may expose this local app on your network.\n", addr)
	}
	fmt.Printf("Serving Creator Crawler at http://%s\n", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
