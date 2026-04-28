# Creator Crawler

Creator Crawler is a local Go CLI for finding YouTube creators who have made videos about a game. It searches YouTube, fetches video and channel statistics, lightly filters obvious official/media/trailer results, then writes a timestamped CSV sorted by view count.

## Setup

Create a YouTube Data API v3 key in Google Cloud Console, then add it to `.env`:

```bash
cp .env.example .env
```

```env
YOUTUBE_API_KEY=your_api_key_here
```

## Usage

```bash
go run . search "Elden Ring"
```

```bash
go run . search "Elden Ring" --limit 100
```

```bash
go run . search "Elden Ring" --output-dir outputs
```

By default, likely official/media/trailer rows are excluded. To include them with a `filtered_reason` column:

```bash
go run . search "Elden Ring" --include-filtered
```

CSV files are written as timestamped files:

```text
outputs/elden-ring-2026-04-28-153000.csv
```

## CSV Fields

The CSV contains one row per video:

```text
creator_name, channel_id, channel_url, subscriber_count, video_title, video_id, video_url, view_count, like_count, comment_count, published_at, duration, format, language, filtered_reason, description
```

## Notes

- Search uses the exact game name you provide.
- YouTube does not expose a reliable public “tagged game” search field like Twitch categories.
- Language is left blank when YouTube does not provide language metadata.
- Shorts are inferred from duration of 60 seconds or less.
- The `duration` column uses YouTube's ISO 8601 duration format.
