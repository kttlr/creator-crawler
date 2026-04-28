package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"creator-crawler/internal/model"
)

const apiBase = "https://www.googleapis.com/youtube/v3"

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Search(ctx context.Context, query string, limit int, includeFiltered bool) ([]model.Result, error) {
	if limit <= 0 {
		limit = 100
	}

	videoIDs, err := c.searchVideoIDs(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	if len(videoIDs) == 0 {
		return nil, nil
	}

	videos, err := c.fetchVideos(ctx, videoIDs)
	if err != nil {
		return nil, err
	}

	channelIDs := uniqueChannelIDs(videos)
	channels, err := c.fetchChannels(ctx, channelIDs)
	if err != nil {
		return nil, err
	}

	results := make([]model.Result, 0, len(videos))
	for _, video := range videos {
		channel := channels[video.Snippet.ChannelID]
		reason := filteredReason(video, channel)
		if reason != "" && !includeFiltered {
			continue
		}

		publishedAt, _ := time.Parse(time.RFC3339, video.Snippet.PublishedAt)
		language := video.Snippet.DefaultAudioLanguage
		if language == "" {
			language = video.Snippet.DefaultLanguage
		}

		result := model.Result{
			CreatorName:     firstNonEmpty(channel.Snippet.Title, video.Snippet.ChannelTitle),
			ChannelID:       video.Snippet.ChannelID,
			ChannelURL:      "https://www.youtube.com/channel/" + video.Snippet.ChannelID,
			SubscriberCount: parseUint(channel.Statistics.SubscriberCount),
			VideoTitle:      video.Snippet.Title,
			VideoID:         video.ID,
			VideoURL:        "https://www.youtube.com/watch?v=" + video.ID,
			ViewCount:       parseUint(video.Statistics.ViewCount),
			LikeCount:       parseUint(video.Statistics.LikeCount),
			CommentCount:    parseUint(video.Statistics.CommentCount),
			PublishedAt:     publishedAt,
			Duration:        video.ContentDetails.Duration,
			Format:          classifyFormat(video),
			Language:        language,
			FilteredReason:  reason,
			Description:     video.Snippet.Description,
		}
		results = append(results, result)
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].ViewCount > results[j].ViewCount
	})

	return results, nil
}

func (c *Client) searchVideoIDs(ctx context.Context, query string, limit int) ([]string, error) {
	ids := make([]string, 0, limit)
	seen := map[string]bool{}
	pageToken := ""

	for len(ids) < limit {
		remaining := limit - len(ids)
		maxResults := 50
		if remaining < maxResults {
			maxResults = remaining
		}

		values := url.Values{}
		values.Set("part", "snippet")
		values.Set("q", query)
		values.Set("type", "video")
		values.Set("order", "viewCount")
		values.Set("maxResults", strconv.Itoa(maxResults))
		values.Set("key", c.apiKey)
		if pageToken != "" {
			values.Set("pageToken", pageToken)
		}

		var response searchResponse
		if err := c.get(ctx, "/search", values, &response); err != nil {
			return nil, err
		}

		for _, item := range response.Items {
			if item.ID.VideoID == "" || seen[item.ID.VideoID] {
				continue
			}
			seen[item.ID.VideoID] = true
			ids = append(ids, item.ID.VideoID)
		}

		if response.NextPageToken == "" {
			break
		}
		pageToken = response.NextPageToken
	}

	return ids, nil
}

func (c *Client) fetchVideos(ctx context.Context, ids []string) ([]videoItem, error) {
	videos := make([]videoItem, 0, len(ids))
	for _, batch := range chunks(ids, 50) {
		values := url.Values{}
		values.Set("part", "snippet,statistics,contentDetails,liveStreamingDetails")
		values.Set("id", strings.Join(batch, ","))
		values.Set("key", c.apiKey)

		var response videosResponse
		if err := c.get(ctx, "/videos", values, &response); err != nil {
			return nil, err
		}
		videos = append(videos, response.Items...)
	}
	return videos, nil
}

func (c *Client) fetchChannels(ctx context.Context, ids []string) (map[string]channelItem, error) {
	channels := map[string]channelItem{}
	for _, batch := range chunks(ids, 50) {
		values := url.Values{}
		values.Set("part", "snippet,statistics")
		values.Set("id", strings.Join(batch, ","))
		values.Set("key", c.apiKey)

		var response channelsResponse
		if err := c.get(ctx, "/channels", values, &response); err != nil {
			return nil, err
		}
		for _, item := range response.Items {
			channels[item.ID] = item
		}
	}
	return channels, nil
}

func (c *Client) get(ctx context.Context, path string, values url.Values, target any) error {
	endpoint := apiBase + path + "?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr apiErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error.Message != "" {
			return fmt.Errorf("youtube API error: %s", apiErr.Error.Message)
		}
		return fmt.Errorf("youtube API error: status %s", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func uniqueChannelIDs(videos []videoItem) []string {
	seen := map[string]bool{}
	var ids []string
	for _, video := range videos {
		id := video.Snippet.ChannelID
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

func chunks(values []string, size int) [][]string {
	var out [][]string
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		out = append(out, values[start:end])
	}
	return out
}

func parseUint(value string) uint64 {
	parsed, _ := strconv.ParseUint(value, 10, 64)
	return parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

type searchResponse struct {
	NextPageToken string       `json:"nextPageToken"`
	Items         []searchItem `json:"items"`
}

type searchItem struct {
	ID struct {
		VideoID string `json:"videoId"`
	} `json:"id"`
}

type videosResponse struct {
	Items []videoItem `json:"items"`
}

type videoItem struct {
	ID      string `json:"id"`
	Snippet struct {
		PublishedAt          string   `json:"publishedAt"`
		ChannelID            string   `json:"channelId"`
		Title                string   `json:"title"`
		Description          string   `json:"description"`
		ChannelTitle         string   `json:"channelTitle"`
		Tags                 []string `json:"tags"`
		DefaultLanguage      string   `json:"defaultLanguage"`
		DefaultAudioLanguage string   `json:"defaultAudioLanguage"`
		LiveBroadcastContent string   `json:"liveBroadcastContent"`
	} `json:"snippet"`
	Statistics struct {
		ViewCount    string `json:"viewCount"`
		LikeCount    string `json:"likeCount"`
		CommentCount string `json:"commentCount"`
	} `json:"statistics"`
	ContentDetails struct {
		Duration string `json:"duration"`
	} `json:"contentDetails"`
	LiveStreamingDetails struct {
		ActualStartTime    string `json:"actualStartTime"`
		ActualEndTime      string `json:"actualEndTime"`
		ScheduledStartTime string `json:"scheduledStartTime"`
	} `json:"liveStreamingDetails"`
}

type channelsResponse struct {
	Items []channelItem `json:"items"`
}

type channelItem struct {
	ID      string `json:"id"`
	Snippet struct {
		Title string `json:"title"`
	} `json:"snippet"`
	Statistics struct {
		SubscriberCount string `json:"subscriberCount"`
	} `json:"statistics"`
}

type apiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
