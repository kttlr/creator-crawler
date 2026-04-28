# Creator Crawler

Creator Crawler is a local Go CLI for finding YouTube creators who have made videos about a game. It searches YouTube, fetches video and channel statistics, lightly filters obvious official/media/trailer results, and stores results in SQLite.

## Setup

Create a YouTube Data API v3 key in Google Cloud Console, then add it to `.env`:

```bash
cp .env.example .env
```

```env
YOUTUBE_API_KEY=your_api_key_here
```

## Usage

Add a game before searching:

```bash
go run . game add "Elden Ring"
```

List tracked games:

```bash
go run . game list
```

Search YouTube for an existing game. The default query is the game name:

```bash
go run . search --game "Elden Ring" --limit 100
```

Use a different YouTube query while still tying results to the game:

```bash
go run . search --game "Elden Ring" --query "Elden Ring gameplay" --limit 100
```

By default, likely official/media/trailer rows are excluded. To include them with a `filtered_reason` value:

```bash
go run . search --game "Elden Ring" --include-filtered
```

The default SQLite database path is:

```text
data/creator-crawler.db
```

Use a custom database path:

```bash
go run . search --game "Elden Ring" --db data/custom.db
```

## Data Model

The SQLite database stores:

- `games`: games you explicitly choose to track.
- `search_runs`: each YouTube API search tied to a game.
- `creators`: YouTube channels.
- `videos`: YouTube videos, deduped globally by video ID.
- `search_results`: videos returned for a specific search run.
- `game_videos`: candidate videos associated with a game.

Searches do not create games automatically. If a game does not exist, the CLI fails with a message showing the `game add` command to run.

## Notes

- Search uses the exact query you provide, or the game name if `--query` is omitted.
- YouTube does not expose a reliable public “tagged game” search field like Twitch categories.
- Language is left blank when YouTube does not provide language metadata.
- Shorts are inferred from duration of 60 seconds or less.
- The `duration` field uses YouTube's ISO 8601 duration format.
