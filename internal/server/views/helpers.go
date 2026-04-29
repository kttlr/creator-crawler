package views

import (
	"creator-crawler/internal/storage"

	"fmt"
	"net/url"
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

func GameTagsPath(id int64) string {
	return GamePath(id) + "/tags"
}

func MyGamesPath() string {
	return "/my-games"
}

func MyGamePath(id int64) string {
	return MyGamesPath() + "/" + strconv.FormatInt(id, 10)
}

func MyGameTagsPath(id int64) string {
	return MyGamePath(id) + "/tags"
}

func MyGameSimilarGamesPath(id int64) string {
	return MyGamePath(id) + "/games"
}

func MyGameSimilarGamePath(id int64, gameID int64) string {
	return MyGameSimilarGamesPath(id) + "/" + strconv.FormatInt(gameID, 10)
}

func MyGameCreatorsPath(id int64) string {
	return MyGamePath(id) + "/creators"
}

func MyGameCreatorPath(id int64, channelID string) string {
	return MyGameCreatorsPath(id) + "/" + url.PathEscape(channelID)
}

func MyGameCreatorContactedPath(id int64, channelID string) string {
	return MyGameCreatorPath(id, channelID) + "/contacted"
}

func CreatorsPath() string {
	return "/creators"
}

func CreatorPath(channelID string) string {
	return CreatorsPath() + "/" + url.PathEscape(channelID)
}

func CreatorEmailPath(channelID string) string {
	return CreatorPath(channelID) + "/email"
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

func TagsValue(tags []storage.Tag) string {
	values := make([]string, 0, len(tags))
	for _, tag := range tags {
		values = append(values, tag.Name)
	}
	return strings.Join(values, ", ")
}
