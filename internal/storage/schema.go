package storage

const schema = `
CREATE TABLE IF NOT EXISTS games (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL COLLATE NOCASE UNIQUE,
  slug TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS my_games (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL COLLATE NOCASE UNIQUE,
  slug TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tags (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL COLLATE NOCASE UNIQUE,
  slug TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS my_game_tags (
  my_game_id INTEGER NOT NULL,
  tag_id INTEGER NOT NULL,
  PRIMARY KEY (my_game_id, tag_id),
  FOREIGN KEY (my_game_id) REFERENCES my_games(id),
  FOREIGN KEY (tag_id) REFERENCES tags(id)
);

CREATE TABLE IF NOT EXISTS game_tags (
  game_id INTEGER NOT NULL,
  tag_id INTEGER NOT NULL,
  PRIMARY KEY (game_id, tag_id),
  FOREIGN KEY (game_id) REFERENCES games(id),
  FOREIGN KEY (tag_id) REFERENCES tags(id)
);

CREATE TABLE IF NOT EXISTS my_game_similar_games (
  my_game_id INTEGER NOT NULL,
  game_id INTEGER NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  PRIMARY KEY (my_game_id, game_id),
  FOREIGN KEY (my_game_id) REFERENCES my_games(id),
  FOREIGN KEY (game_id) REFERENCES games(id)
);

CREATE TABLE IF NOT EXISTS my_game_creators (
  my_game_id INTEGER NOT NULL,
  channel_id TEXT NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  contacted_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  PRIMARY KEY (my_game_id, channel_id),
  FOREIGN KEY (my_game_id) REFERENCES my_games(id),
  FOREIGN KEY (channel_id) REFERENCES creators(channel_id)
);

CREATE TABLE IF NOT EXISTS search_runs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  game_id INTEGER NOT NULL,
  query TEXT NOT NULL,
  limit_requested INTEGER NOT NULL,
  include_filtered INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (game_id) REFERENCES games(id)
);

CREATE TABLE IF NOT EXISTS creators (
  channel_id TEXT PRIMARY KEY,
  creator_name TEXT NOT NULL,
  channel_url TEXT NOT NULL,
  email TEXT NOT NULL DEFAULT '',
  subscriber_count INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS videos (
  video_id TEXT PRIMARY KEY,
  channel_id TEXT NOT NULL,
  video_title TEXT NOT NULL,
  video_url TEXT NOT NULL,
  view_count INTEGER NOT NULL DEFAULT 0,
  like_count INTEGER NOT NULL DEFAULT 0,
  comment_count INTEGER NOT NULL DEFAULT 0,
  published_at TEXT,
  duration TEXT NOT NULL DEFAULT '',
  format TEXT NOT NULL DEFAULT '',
  language TEXT NOT NULL DEFAULT '',
  filtered_reason TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL,
  FOREIGN KEY (channel_id) REFERENCES creators(channel_id)
);

CREATE TABLE IF NOT EXISTS search_results (
  search_run_id INTEGER NOT NULL,
  video_id TEXT NOT NULL,
  position INTEGER NOT NULL,
  PRIMARY KEY (search_run_id, video_id),
  FOREIGN KEY (search_run_id) REFERENCES search_runs(id),
  FOREIGN KEY (video_id) REFERENCES videos(video_id)
);

CREATE TABLE IF NOT EXISTS game_videos (
  game_id INTEGER NOT NULL,
  video_id TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'candidate',
  source_search_run_id INTEGER,
  first_seen_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (game_id, video_id),
  FOREIGN KEY (game_id) REFERENCES games(id),
  FOREIGN KEY (video_id) REFERENCES videos(video_id),
  FOREIGN KEY (source_search_run_id) REFERENCES search_runs(id)
);

CREATE INDEX IF NOT EXISTS idx_search_runs_game_created_at
ON search_runs(game_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_videos_view_count
ON videos(view_count DESC);

CREATE INDEX IF NOT EXISTS idx_videos_channel_id
ON videos(channel_id);

CREATE INDEX IF NOT EXISTS idx_search_results_search_run_position
ON search_results(search_run_id, position);

CREATE INDEX IF NOT EXISTS idx_game_videos_game_status
ON game_videos(game_id, status);

CREATE INDEX IF NOT EXISTS idx_my_game_tags_tag
ON my_game_tags(tag_id);

CREATE INDEX IF NOT EXISTS idx_game_tags_tag
ON game_tags(tag_id);

CREATE INDEX IF NOT EXISTS idx_my_game_similar_games_game
ON my_game_similar_games(game_id);

CREATE INDEX IF NOT EXISTS idx_my_game_creators_channel
ON my_game_creators(channel_id);
`
