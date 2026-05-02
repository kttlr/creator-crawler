# Creator Crawler

> Disclaimer: Codex was used to help create this project, but it was not "vibe-coded".

A local Go app for finding YouTube creators who have covered specific games. It searches YouTube, pulls video/channel stats, filters obvious official/media/trailer results, and stores everything in SQLite.

## Setup

Create a YouTube Data API v3 key, then copy the env file and add it:

```bash
cp .env.example .env
```

```env
YOUTUBE_API_KEY=your_api_key_here
```

## Run the web UI

```bash
go run . serve
```

Open `http://localhost:8080`.

The app uses a local SQLite database at `data/creator-crawler.db` by default. A YouTube API key is only needed when running searches.

## CLI basics

```bash
# add a game
go run . game add "Elden Ring"

# list games
go run . game list

# search YouTube for creators/videos
go run . search --game "Elden Ring" --limit 100

# use a custom query for that game
go run . search --game "Elden Ring" --query "Elden Ring gameplay" --limit 100

# include results normally filtered out as official/media/trailers
go run . search --game "Elden Ring" --include-filtered

# use a custom database
go run . serve --db data/custom.db
```

Searches only work for games you have already added.

## What it stores

The SQLite database tracks:

- your games and tracked games
- tags and similar-game links
- YouTube search runs
- creators/channels
- videos and per-game video matches
- contacted/approved creator status for your own games

The My Games view can suggest similar tracked games by shared tags and suggest creators from approved/contacted videos on matching games.

## Notes

- Web UI binds to `localhost:8080` by default.
- Datastar is vendored in `static/vendor/`, so the UI does not need a CDN.
- YouTube search uses your query exactly, or the game name if `--query` is omitted.
- Shorts are inferred from videos 60 seconds or shorter.
- Video durations are stored in YouTube's ISO 8601 format.

## Development

The UI uses `templ` and Tailwind CSS v4. Generated `*_templ.go` files and compiled CSS are committed.

For live development:

```bash
just dev
```

Open `http://localhost:8090` for the live-reload proxy. The app still runs on `http://localhost:8080`.

Useful commands:

```bash
templ generate
pnpm build:css
pnpm watch:css
```
