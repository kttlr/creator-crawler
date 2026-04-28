package youtube

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

func classifyFormat(video videoItem) string {
	if strings.Contains(strings.ToLower(video.Snippet.Title), "premiere") {
		return "premiere"
	}

	if video.Snippet.LiveBroadcastContent == "live" || video.Snippet.LiveBroadcastContent == "upcoming" {
		return "livestream"
	}

	if video.LiveStreamingDetails.ActualStartTime != "" || video.LiveStreamingDetails.ScheduledStartTime != "" {
		if video.LiveStreamingDetails.ActualEndTime == "" {
			return "livestream"
		}
		return "video"
	}

	duration, ok := parseISO8601Duration(video.ContentDetails.Duration)
	if !ok {
		return "unknown"
	}
	if duration <= time.Minute {
		return "short"
	}
	return "video"
}

func filteredReason(video videoItem, channel channelItem) string {
	text := strings.ToLower(strings.Join([]string{
		video.Snippet.Title,
		video.Snippet.ChannelTitle,
		channel.Snippet.Title,
	}, " "))

	filters := []string{
		"official",
		"launch trailer",
		"gameplay trailer",
		"announcement trailer",
		"trailer",
		"developer",
		"publisher",
		"nintendo",
		"playstation",
		"xbox",
		"ign",
		"gamespot",
		"pc gamer",
		"game informer",
	}

	for _, filter := range filters {
		if strings.Contains(text, filter) {
			return filter
		}
	}
	return ""
}

var durationPattern = regexp.MustCompile(`^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?$`)

func parseISO8601Duration(value string) (time.Duration, bool) {
	match := durationPattern.FindStringSubmatch(value)
	if match == nil {
		return 0, false
	}

	days := parseInt(match[1])
	hours := parseInt(match[2])
	minutes := parseInt(match[3])
	seconds := parseInt(match[4])

	duration := time.Duration(days)*24*time.Hour + time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second
	return duration, true
}

func parseInt(value string) int {
	if value == "" {
		return 0
	}
	parsed, _ := strconv.Atoi(value)
	return parsed
}
