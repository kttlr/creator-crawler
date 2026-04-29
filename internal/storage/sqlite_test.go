package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"creator-crawler/internal/model"
)

func TestSaveSearchResultsPreservesGameVideoStatus(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	game, err := store.AddGame(ctx, "Elden Ring")
	if err != nil {
		t.Fatal(err)
	}

	result := model.Result{
		CreatorName:    "Creator",
		ChannelID:      "channel-1",
		ChannelURL:     "https://www.youtube.com/channel/channel-1",
		VideoTitle:     "Elden Ring Gameplay",
		VideoID:        "video-1",
		VideoURL:       "https://www.youtube.com/watch?v=video-1",
		ViewCount:      100,
		PublishedAt:    time.Now().UTC(),
		Duration:       "PT10M",
		Format:         "video",
		FilteredReason: "",
	}

	if _, err := store.SaveSearchResults(ctx, game, "Elden Ring", 100, false, []model.Result{result}); err != nil {
		t.Fatal(err)
	}

	if _, err := store.db.ExecContext(ctx, `UPDATE game_videos SET status = 'rejected', notes = 'bad fit' WHERE game_id = ? AND video_id = ?`, game.ID, result.VideoID); err != nil {
		t.Fatal(err)
	}

	result.ViewCount = 200
	if _, err := store.SaveSearchResults(ctx, game, "Elden Ring", 100, false, []model.Result{result}); err != nil {
		t.Fatal(err)
	}

	var status, notes string
	if err := store.db.QueryRowContext(ctx, `SELECT status, notes FROM game_videos WHERE game_id = ? AND video_id = ?`, game.ID, result.VideoID).Scan(&status, &notes); err != nil {
		t.Fatal(err)
	}
	if status != "rejected" {
		t.Fatalf("status = %q, want rejected", status)
	}
	if notes != "bad fit" {
		t.Fatalf("notes = %q, want bad fit", notes)
	}

	var viewCount int64
	if err := store.db.QueryRowContext(ctx, `SELECT view_count FROM videos WHERE video_id = ?`, result.VideoID).Scan(&viewCount); err != nil {
		t.Fatal(err)
	}
	if viewCount != 200 {
		t.Fatalf("view_count = %d, want 200", viewCount)
	}
}

func TestListApprovedCreatorsAndCreatorVideos(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	eldenRing, err := store.AddGame(ctx, "Elden Ring")
	if err != nil {
		t.Fatal(err)
	}
	balatro, err := store.AddGame(ctx, "Balatro")
	if err != nil {
		t.Fatal(err)
	}

	results := []model.Result{
		{
			CreatorName:     "Creator",
			ChannelID:       "channel-1",
			ChannelURL:      "https://www.youtube.com/channel/channel-1",
			SubscriberCount: 1000,
			VideoTitle:      "Elden Ring Gameplay",
			VideoID:         "video-1",
			VideoURL:        "https://www.youtube.com/watch?v=video-1",
			ViewCount:       100,
			PublishedAt:     time.Now().UTC(),
			Duration:        "PT10M",
			Format:          "video",
		},
		{
			CreatorName:     "Other Creator",
			ChannelID:       "channel-2",
			ChannelURL:      "https://www.youtube.com/channel/channel-2",
			SubscriberCount: 2000,
			VideoTitle:      "Elden Ring Trailer",
			VideoID:         "video-2",
			VideoURL:        "https://www.youtube.com/watch?v=video-2",
			ViewCount:       500,
			PublishedAt:     time.Now().UTC(),
			Duration:        "PT2M",
			Format:          "video",
		},
	}

	if _, err := store.SaveSearchResults(ctx, eldenRing, "Elden Ring", 100, false, results); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveSearchResults(ctx, balatro, "Balatro", 100, false, []model.Result{{
		CreatorName:     "Creator",
		ChannelID:       "channel-1",
		ChannelURL:      "https://www.youtube.com/channel/channel-1",
		SubscriberCount: 1000,
		VideoTitle:      "Balatro Gameplay",
		VideoID:         "video-3",
		VideoURL:        "https://www.youtube.com/watch?v=video-3",
		ViewCount:       250,
		PublishedAt:     time.Now().UTC(),
		Duration:        "PT12M",
		Format:          "video",
	}}); err != nil {
		t.Fatal(err)
	}

	if _, err := store.db.ExecContext(ctx, `UPDATE game_videos SET status = 'approved' WHERE video_id IN ('video-1', 'video-3')`); err != nil {
		t.Fatal(err)
	}

	creators, err := store.ListApprovedCreators(ctx, ListApprovedCreatorsOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(creators) != 1 {
		t.Fatalf("len(creators) = %d, want 1", len(creators))
	}
	if creators[0].ChannelID != "channel-1" {
		t.Fatalf("ChannelID = %q, want channel-1", creators[0].ChannelID)
	}
	if creators[0].ApprovedVideoCount != 2 {
		t.Fatalf("ApprovedVideoCount = %d, want 2", creators[0].ApprovedVideoCount)
	}
	if creators[0].GameCount != 2 {
		t.Fatalf("GameCount = %d, want 2", creators[0].GameCount)
	}

	videos, err := store.ListCreatorVideos(ctx, "channel-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(videos) != 2 {
		t.Fatalf("len(videos) = %d, want 2", len(videos))
	}
	if videos[0].GameName != "Balatro" || videos[1].GameName != "Elden Ring" {
		t.Fatalf("games = %q, %q; want Balatro, Elden Ring", videos[0].GameName, videos[1].GameName)
	}
}

func TestMyGameTagsAssociationsAndSuggestions(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	myGame, err := store.AddMyGame(ctx, "My Deckbuilder", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetMyGameTags(ctx, myGame.ID, "Deckbuilder, Roguelike, deckbuilder"); err != nil {
		t.Fatal(err)
	}

	balatro, err := store.AddGame(ctx, "Balatro")
	if err != nil {
		t.Fatal(err)
	}
	eldenRing, err := store.AddGame(ctx, "Elden Ring")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetGameTags(ctx, balatro.ID, "Deckbuilder, Roguelike"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetGameTags(ctx, eldenRing.ID, "Souls-like"); err != nil {
		t.Fatal(err)
	}

	tags, err := store.ListTagsForMyGame(ctx, myGame.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 {
		t.Fatalf("len(tags) = %d, want 2", len(tags))
	}

	suggestedGames, err := store.ListSuggestedGamesForMyGame(ctx, myGame.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestedGames) != 1 {
		t.Fatalf("len(suggestedGames) = %d, want 1", len(suggestedGames))
	}
	if suggestedGames[0].ID != balatro.ID || suggestedGames[0].SharedTagCount != 2 {
		t.Fatalf("suggested game = (%d, %d), want (%d, 2)", suggestedGames[0].ID, suggestedGames[0].SharedTagCount, balatro.ID)
	}

	if err := store.AddSimilarGameToMyGame(ctx, myGame.ID, balatro.ID, "good fit"); err != nil {
		t.Fatal(err)
	}
	suggestedGames, err = store.ListSuggestedGamesForMyGame(ctx, myGame.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestedGames) != 0 {
		t.Fatalf("len(suggestedGames) = %d, want 0 after explicit association", len(suggestedGames))
	}

	similarGames, err := store.ListMyGameSimilarGames(ctx, myGame.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(similarGames) != 1 || similarGames[0].SharedTagCount != 2 {
		t.Fatalf("similarGames = %#v, want one game with two shared tags", similarGames)
	}

	result := model.Result{
		CreatorName:     "Creator",
		ChannelID:       "channel-1",
		ChannelURL:      "https://www.youtube.com/channel/channel-1",
		SubscriberCount: 1000,
		VideoTitle:      "Balatro Gameplay",
		VideoID:         "video-1",
		VideoURL:        "https://www.youtube.com/watch?v=video-1",
		ViewCount:       100,
		PublishedAt:     time.Now().UTC(),
		Duration:        "PT10M",
		Format:          "video",
	}
	if _, err := store.SaveSearchResults(ctx, balatro, "Balatro", 100, false, []model.Result{result}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE game_videos SET status = 'approved' WHERE game_id = ? AND video_id = ?`, balatro.ID, result.VideoID); err != nil {
		t.Fatal(err)
	}

	suggestedCreators, err := store.ListSuggestedCreatorsForMyGame(ctx, myGame.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestedCreators) != 1 || suggestedCreators[0].ChannelID != "channel-1" {
		t.Fatalf("suggestedCreators = %#v, want channel-1", suggestedCreators)
	}

	if err := store.AddCreatorToMyGame(ctx, myGame.ID, "channel-1", "wishlist"); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateCreatorEmail(ctx, "channel-1", "creator@example.com"); err != nil {
		t.Fatal(err)
	}
	creators, err := store.ListCreatorsForMyGame(ctx, myGame.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(creators) != 1 || creators[0].Notes != "wishlist" || creators[0].Email != "creator@example.com" || creators[0].ContactedAt != "" {
		t.Fatalf("creators = %#v, want one creator with notes, email, and no contacted timestamp", creators)
	}

	if err := store.UpdateMyGameCreatorContacted(ctx, myGame.ID, "channel-1", true); err != nil {
		t.Fatal(err)
	}
	creators, err = store.ListCreatorsForMyGame(ctx, myGame.ID)
	if err != nil {
		t.Fatal(err)
	}
	if creators[0].ContactedAt == "" {
		t.Fatalf("ContactedAt is blank, want timestamp")
	}

	otherMyGame, err := store.AddMyGame(ctx, "Other Game", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddCreatorToMyGame(ctx, otherMyGame.ID, "channel-1", ""); err != nil {
		t.Fatal(err)
	}
	otherCreators, err := store.ListCreatorsForMyGame(ctx, otherMyGame.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(otherCreators) != 1 || otherCreators[0].ContactedAt != "" {
		t.Fatalf("otherCreators = %#v, want uncontacted association", otherCreators)
	}

	if err := store.UpdateMyGameCreatorContacted(ctx, myGame.ID, "channel-1", false); err != nil {
		t.Fatal(err)
	}
	creators, err = store.ListCreatorsForMyGame(ctx, myGame.ID)
	if err != nil {
		t.Fatal(err)
	}
	if creators[0].ContactedAt != "" {
		t.Fatalf("ContactedAt = %q, want blank", creators[0].ContactedAt)
	}
}
