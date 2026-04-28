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
