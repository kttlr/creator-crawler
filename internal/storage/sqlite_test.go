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
