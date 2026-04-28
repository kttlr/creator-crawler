package export

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"creator-crawler/internal/model"
)

func WriteCSV(results []model.Result, gameName, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%s-%s.csv", slugify(gameName), time.Now().Format("2006-01-02-150405"))
	path := filepath.Join(outputDir, filename)

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{
		"creator_name",
		"channel_id",
		"channel_url",
		"subscriber_count",
		"video_title",
		"video_id",
		"video_url",
		"view_count",
		"like_count",
		"comment_count",
		"published_at",
		"duration",
		"format",
		"language",
		"filtered_reason",
		"description",
	}
	if err := writer.Write(header); err != nil {
		return "", err
	}

	for _, result := range results {
		published := ""
		if !result.PublishedAt.IsZero() {
			published = result.PublishedAt.Format(time.RFC3339)
		}

		record := []string{
			result.CreatorName,
			result.ChannelID,
			result.ChannelURL,
			strconv.FormatUint(result.SubscriberCount, 10),
			result.VideoTitle,
			result.VideoID,
			result.VideoURL,
			strconv.FormatUint(result.ViewCount, 10),
			strconv.FormatUint(result.LikeCount, 10),
			strconv.FormatUint(result.CommentCount, 10),
			published,
			result.Duration,
			result.Format,
			result.Language,
			result.FilteredReason,
			result.Description,
		}
		if err := writer.Write(record); err != nil {
			return "", err
		}
	}

	if err := writer.Error(); err != nil {
		return "", err
	}

	return path, nil
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
		return "search"
	}
	return slug
}
