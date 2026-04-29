package views

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func GamePath(id int64) string {
	return "/games/" + strconv.FormatInt(id, 10)
}

func GameVideosPath(id int64) string {
	return GamePath(id) + "/videos"
}

func GameSearchesPath(id int64) string {
	return GamePath(id) + "/searches"
}

func VideoStatusPath(gameID int64, videoID string) string {
	return GameVideosPath(gameID) + "/" + videoID + "/status"
}

func VideoNotesPath(gameID int64, videoID string) string {
	return GameVideosPath(gameID) + "/" + videoID + "/notes"
}

func FormatCount(value int64) string {
	negative := value < 0
	if negative {
		value = -value
	}

	digits := strconv.FormatInt(value, 10)
	var parts []string
	for len(digits) > 3 {
		parts = append([]string{digits[len(digits)-3:]}, parts...)
		digits = digits[:len(digits)-3]
	}
	parts = append([]string{digits}, parts...)
	out := strings.Join(parts, ",")
	if negative {
		return "-" + out
	}
	return out
}

func FormatTime(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return parsed.Local().Format("Jan 2, 2006 3:04 PM")
}

func StatusLabel(status string) string {
	if status == "" {
		return "candidate"
	}
	return status
}

func SearchSummary(count int, dbPath string) string {
	return fmt.Sprintf("Saved %d videos to %s", count, dbPath)
}
