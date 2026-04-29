package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"creator-crawler/internal/model"

	_ "modernc.org/sqlite"
)

const DefaultDBPath = "data/creator-crawler.db"

type Store struct {
	db *sql.DB
}

type Game struct {
	ID         int64
	Name       string
	Slug       string
	CreatedAt  string
	UpdatedAt  string
	VideoCount int64
}

type GameSummary struct {
	Game
	CandidateCount int64
	ApprovedCount  int64
	RejectedCount  int64
	ContactedCount int64
	IgnoredCount   int64
	LastSearchedAt string
}

type SearchRun struct {
	ID              int64
	GameID          int64
	Query           string
	LimitRequested  int64
	IncludeFiltered bool
	CreatedAt       string
	ResultCount     int64
}

type GameVideo struct {
	GameID          int64
	VideoID         string
	Status          string
	Notes           string
	CreatorName     string
	ChannelID       string
	ChannelURL      string
	SubscriberCount int64
	VideoTitle      string
	VideoURL        string
	ViewCount       int64
	LikeCount       int64
	CommentCount    int64
	PublishedAt     string
	Duration        string
	Format          string
	Language        string
	FilteredReason  string
	FirstSeenAt     string
	LastSeenAt      string
}

type CreatorSummary struct {
	ChannelID          string
	CreatorName        string
	ChannelURL         string
	SubscriberCount    int64
	ApprovedVideoCount int64
	GameCount          int64
	TotalViewCount     int64
	LastSeenAt         string
}

type CreatorVideo struct {
	GameID         int64
	GameName       string
	VideoID        string
	Status         string
	Notes          string
	VideoTitle     string
	VideoURL       string
	ViewCount      int64
	LikeCount      int64
	CommentCount   int64
	PublishedAt    string
	Duration       string
	Format         string
	Language       string
	FilteredReason string
	FirstSeenAt    string
	LastSeenAt     string
}

type ListApprovedCreatorsOptions struct {
	Query string
	Sort  string
}

type ListGameVideosOptions struct {
	GameID int64
	Status string
	Query  string
	Sort   string
}

func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultDBPath
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	store := &Store{db: db}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = store.Close()
		return nil, err
	}

	if err := store.Migrate(context.Background()); err != nil {
		_ = store.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *Store) AddGame(ctx context.Context, name string) (Game, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Game{}, errors.New("game name cannot be blank")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	slug := slugify(name)
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO games (name, slug, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, name, slug, now, now)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "constraint") {
			return Game{}, fmt.Errorf("game %q already exists", name)
		}
		return Game{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Game{}, err
	}

	return Game{ID: id, Name: name, Slug: slug, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *Store) FindGameByName(ctx context.Context, name string) (Game, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, created_at, updated_at
		FROM games
		WHERE name = ? COLLATE NOCASE
	`, strings.TrimSpace(name))

	var game Game
	if err := row.Scan(&game.ID, &game.Name, &game.Slug, &game.CreatedAt, &game.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Game{}, false, nil
		}
		return Game{}, false, err
	}

	return game, true, nil
}

func (s *Store) GetGame(ctx context.Context, id int64) (Game, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, created_at, updated_at
		FROM games
		WHERE id = ?
	`, id)

	var game Game
	if err := row.Scan(&game.ID, &game.Name, &game.Slug, &game.CreatedAt, &game.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Game{}, false, nil
		}
		return Game{}, false, err
	}

	return game, true, nil
}

func (s *Store) ListGames(ctx context.Context) ([]Game, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT g.id, g.name, g.slug, g.created_at, g.updated_at, COUNT(gv.video_id) AS video_count
		FROM games g
		LEFT JOIN game_videos gv ON gv.game_id = g.id
		GROUP BY g.id
		ORDER BY g.name COLLATE NOCASE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var game Game
		if err := rows.Scan(&game.ID, &game.Name, &game.Slug, &game.CreatedAt, &game.UpdatedAt, &game.VideoCount); err != nil {
			return nil, err
		}
		games = append(games, game)
	}

	return games, rows.Err()
}

func (s *Store) ListGameSummaries(ctx context.Context) ([]GameSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			g.id,
			g.name,
			g.slug,
			g.created_at,
			g.updated_at,
			COALESCE(v.video_count, 0) AS video_count,
			COALESCE(v.candidate_count, 0) AS candidate_count,
			COALESCE(v.approved_count, 0) AS approved_count,
			COALESCE(v.rejected_count, 0) AS rejected_count,
			COALESCE(v.contacted_count, 0) AS contacted_count,
			COALESCE(v.ignored_count, 0) AS ignored_count,
			COALESCE(sr.last_searched_at, '') AS last_searched_at
		FROM games g
		LEFT JOIN (
			SELECT
				game_id,
				COUNT(video_id) AS video_count,
				SUM(CASE WHEN status = 'candidate' THEN 1 ELSE 0 END) AS candidate_count,
				SUM(CASE WHEN status = 'approved' THEN 1 ELSE 0 END) AS approved_count,
				SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END) AS rejected_count,
				SUM(CASE WHEN status = 'contacted' THEN 1 ELSE 0 END) AS contacted_count,
				SUM(CASE WHEN status = 'ignored' THEN 1 ELSE 0 END) AS ignored_count
			FROM game_videos
			GROUP BY game_id
		) v ON v.game_id = g.id
		LEFT JOIN (
			SELECT game_id, MAX(created_at) AS last_searched_at
			FROM search_runs
			GROUP BY game_id
		) sr ON sr.game_id = g.id
		ORDER BY g.name COLLATE NOCASE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []GameSummary
	for rows.Next() {
		var game GameSummary
		if err := rows.Scan(&game.ID, &game.Name, &game.Slug, &game.CreatedAt, &game.UpdatedAt, &game.VideoCount, &game.CandidateCount, &game.ApprovedCount, &game.RejectedCount, &game.ContactedCount, &game.IgnoredCount, &game.LastSearchedAt); err != nil {
			return nil, err
		}
		games = append(games, game)
	}

	return games, rows.Err()
}

func (s *Store) ListSearchRuns(ctx context.Context, gameID int64, limit int) ([]SearchRun, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT sr.id, sr.game_id, sr.query, sr.limit_requested, sr.include_filtered, sr.created_at, COUNT(sre.video_id) AS result_count
		FROM search_runs sr
		LEFT JOIN search_results sre ON sre.search_run_id = sr.id
		WHERE sr.game_id = ?
		GROUP BY sr.id
		ORDER BY sr.created_at DESC
		LIMIT ?
	`, gameID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []SearchRun
	for rows.Next() {
		var run SearchRun
		var includeFiltered int
		if err := rows.Scan(&run.ID, &run.GameID, &run.Query, &run.LimitRequested, &includeFiltered, &run.CreatedAt, &run.ResultCount); err != nil {
			return nil, err
		}
		run.IncludeFiltered = includeFiltered == 1
		runs = append(runs, run)
	}

	return runs, rows.Err()
}

func (s *Store) ListGameVideos(ctx context.Context, opts ListGameVideosOptions) ([]GameVideo, error) {
	where := []string{"gv.game_id = ?"}
	args := []any{opts.GameID}

	if strings.TrimSpace(opts.Status) != "" && opts.Status != "all" {
		where = append(where, "gv.status = ?")
		args = append(args, opts.Status)
	}
	if strings.TrimSpace(opts.Query) != "" {
		where = append(where, "(v.video_title LIKE ? OR c.creator_name LIKE ?)")
		like := "%" + strings.TrimSpace(opts.Query) + "%"
		args = append(args, like, like)
	}

	orderBy := "v.view_count DESC"
	switch opts.Sort {
	case "published":
		orderBy = "v.published_at DESC"
	case "subscribers":
		orderBy = "c.subscriber_count DESC"
	case "last_seen":
		orderBy = "gv.last_seen_at DESC"
	}

	query := fmt.Sprintf(`
		SELECT
			gv.game_id,
			gv.video_id,
			gv.status,
			gv.notes,
			c.creator_name,
			c.channel_id,
			c.channel_url,
			c.subscriber_count,
			v.video_title,
			v.video_url,
			v.view_count,
			v.like_count,
			v.comment_count,
			COALESCE(v.published_at, ''),
			v.duration,
			v.format,
			v.language,
			v.filtered_reason,
			gv.first_seen_at,
			gv.last_seen_at
		FROM game_videos gv
		JOIN videos v ON v.video_id = gv.video_id
		JOIN creators c ON c.channel_id = v.channel_id
		WHERE %s
		ORDER BY %s
		LIMIT 500
	`, strings.Join(where, " AND "), orderBy)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []GameVideo
	for rows.Next() {
		var video GameVideo
		if err := rows.Scan(&video.GameID, &video.VideoID, &video.Status, &video.Notes, &video.CreatorName, &video.ChannelID, &video.ChannelURL, &video.SubscriberCount, &video.VideoTitle, &video.VideoURL, &video.ViewCount, &video.LikeCount, &video.CommentCount, &video.PublishedAt, &video.Duration, &video.Format, &video.Language, &video.FilteredReason, &video.FirstSeenAt, &video.LastSeenAt); err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}

	return videos, rows.Err()
}

func (s *Store) ListApprovedCreators(ctx context.Context, opts ListApprovedCreatorsOptions) ([]CreatorSummary, error) {
	where := []string{"gv.status = 'approved'"}
	var args []any

	if strings.TrimSpace(opts.Query) != "" {
		where = append(where, "c.creator_name LIKE ?")
		args = append(args, "%"+strings.TrimSpace(opts.Query)+"%")
	}

	orderBy := "approved_video_count DESC, c.subscriber_count DESC"
	switch opts.Sort {
	case "subscribers":
		orderBy = "c.subscriber_count DESC, approved_video_count DESC"
	case "games":
		orderBy = "game_count DESC, approved_video_count DESC"
	case "last_seen":
		orderBy = "last_seen_at DESC"
	case "views":
		orderBy = "total_view_count DESC"
	}

	query := fmt.Sprintf(`
		SELECT
			c.channel_id,
			c.creator_name,
			c.channel_url,
			c.subscriber_count,
			COUNT(DISTINCT gv.video_id) AS approved_video_count,
			COUNT(DISTINCT gv.game_id) AS game_count,
			COALESCE(SUM(v.view_count), 0) AS total_view_count,
			MAX(gv.last_seen_at) AS last_seen_at
		FROM creators c
		JOIN videos v ON v.channel_id = c.channel_id
		JOIN game_videos gv ON gv.video_id = v.video_id
		WHERE %s
		GROUP BY c.channel_id
		ORDER BY %s
		LIMIT 500
	`, strings.Join(where, " AND "), orderBy)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creators []CreatorSummary
	for rows.Next() {
		var creator CreatorSummary
		if err := rows.Scan(&creator.ChannelID, &creator.CreatorName, &creator.ChannelURL, &creator.SubscriberCount, &creator.ApprovedVideoCount, &creator.GameCount, &creator.TotalViewCount, &creator.LastSeenAt); err != nil {
			return nil, err
		}
		creators = append(creators, creator)
	}

	return creators, rows.Err()
}

func (s *Store) GetCreator(ctx context.Context, channelID string) (CreatorSummary, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			c.channel_id,
			c.creator_name,
			c.channel_url,
			c.subscriber_count,
			COUNT(DISTINCT CASE WHEN gv.status = 'approved' THEN gv.video_id END) AS approved_video_count,
			COUNT(DISTINCT gv.game_id) AS game_count,
			COALESCE(SUM(CASE WHEN gv.status = 'approved' THEN v.view_count ELSE 0 END), 0) AS total_view_count,
			COALESCE(MAX(gv.last_seen_at), '') AS last_seen_at
		FROM creators c
		LEFT JOIN videos v ON v.channel_id = c.channel_id
		LEFT JOIN game_videos gv ON gv.video_id = v.video_id
		WHERE c.channel_id = ?
		GROUP BY c.channel_id
	`, strings.TrimSpace(channelID))

	var creator CreatorSummary
	if err := row.Scan(&creator.ChannelID, &creator.CreatorName, &creator.ChannelURL, &creator.SubscriberCount, &creator.ApprovedVideoCount, &creator.GameCount, &creator.TotalViewCount, &creator.LastSeenAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CreatorSummary{}, false, nil
		}
		return CreatorSummary{}, false, err
	}

	return creator, true, nil
}

func (s *Store) ListCreatorVideos(ctx context.Context, channelID string) ([]CreatorVideo, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			g.id,
			g.name,
			v.video_id,
			gv.status,
			gv.notes,
			v.video_title,
			v.video_url,
			v.view_count,
			v.like_count,
			v.comment_count,
			COALESCE(v.published_at, ''),
			v.duration,
			v.format,
			v.language,
			v.filtered_reason,
			gv.first_seen_at,
			gv.last_seen_at
		FROM videos v
		JOIN game_videos gv ON gv.video_id = v.video_id
		JOIN games g ON g.id = gv.game_id
		WHERE v.channel_id = ?
		ORDER BY g.name COLLATE NOCASE, gv.status = 'approved' DESC, v.view_count DESC
		LIMIT 1000
	`, strings.TrimSpace(channelID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []CreatorVideo
	for rows.Next() {
		var video CreatorVideo
		if err := rows.Scan(&video.GameID, &video.GameName, &video.VideoID, &video.Status, &video.Notes, &video.VideoTitle, &video.VideoURL, &video.ViewCount, &video.LikeCount, &video.CommentCount, &video.PublishedAt, &video.Duration, &video.Format, &video.Language, &video.FilteredReason, &video.FirstSeenAt, &video.LastSeenAt); err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}

	return videos, rows.Err()
}

func (s *Store) UpdateGameVideoStatus(ctx context.Context, gameID int64, videoID, status string) error {
	if !validStatus(status) {
		return fmt.Errorf("invalid status %q", status)
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE game_videos
		SET status = ?
		WHERE game_id = ? AND video_id = ?
	`, status, gameID, videoID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) UpdateGameVideoNotes(ctx context.Context, gameID int64, videoID, notes string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE game_videos
		SET notes = ?
		WHERE game_id = ? AND video_id = ?
	`, notes, gameID, videoID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func validStatus(status string) bool {
	switch status {
	case "candidate", "approved", "rejected", "contacted", "ignored":
		return true
	default:
		return false
	}
}

func (s *Store) SaveSearchResults(ctx context.Context, game Game, query string, limit int, includeFiltered bool, results []model.Result) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	searchRun, err := tx.ExecContext(ctx, `
		INSERT INTO search_runs (game_id, query, limit_requested, include_filtered, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, game.ID, query, limit, boolInt(includeFiltered), now)
	if err != nil {
		return 0, err
	}

	searchRunID, err := searchRun.LastInsertId()
	if err != nil {
		return 0, err
	}

	for position, result := range results {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO creators (channel_id, creator_name, channel_url, subscriber_count, updated_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(channel_id) DO UPDATE SET
				creator_name = excluded.creator_name,
				channel_url = excluded.channel_url,
				subscriber_count = excluded.subscriber_count,
				updated_at = excluded.updated_at
		`, result.ChannelID, result.CreatorName, result.ChannelURL, safeInt64(result.SubscriberCount), now); err != nil {
			return 0, err
		}

		publishedAt := ""
		if !result.PublishedAt.IsZero() {
			publishedAt = result.PublishedAt.UTC().Format(time.RFC3339)
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO videos (
				video_id, channel_id, video_title, video_url, view_count, like_count, comment_count,
				published_at, duration, format, language, filtered_reason, description, updated_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(video_id) DO UPDATE SET
				channel_id = excluded.channel_id,
				video_title = excluded.video_title,
				video_url = excluded.video_url,
				view_count = excluded.view_count,
				like_count = excluded.like_count,
				comment_count = excluded.comment_count,
				published_at = excluded.published_at,
				duration = excluded.duration,
				format = excluded.format,
				language = excluded.language,
				filtered_reason = excluded.filtered_reason,
				description = excluded.description,
				updated_at = excluded.updated_at
		`, result.VideoID, result.ChannelID, result.VideoTitle, result.VideoURL, safeInt64(result.ViewCount), safeInt64(result.LikeCount), safeInt64(result.CommentCount), publishedAt, result.Duration, result.Format, result.Language, result.FilteredReason, result.Description, now); err != nil {
			return 0, err
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO search_results (search_run_id, video_id, position)
			VALUES (?, ?, ?)
		`, searchRunID, result.VideoID, position+1); err != nil {
			return 0, err
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO game_videos (game_id, video_id, status, source_search_run_id, first_seen_at, last_seen_at, notes)
			VALUES (?, ?, 'candidate', ?, ?, ?, '')
			ON CONFLICT(game_id, video_id) DO UPDATE SET
				source_search_run_id = excluded.source_search_run_id,
				last_seen_at = excluded.last_seen_at
		`, game.ID, result.VideoID, searchRunID, now, now); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return searchRunID, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func safeInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value)
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false

	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}

		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "game"
	}
	return slug
}
