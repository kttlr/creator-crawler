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
