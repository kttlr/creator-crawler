package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	YouTubeAPIKey string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{YouTubeAPIKey: strings.TrimSpace(os.Getenv("YOUTUBE_API_KEY"))}
	if cfg.YouTubeAPIKey == "" {
		return Config{}, fmt.Errorf("missing YOUTUBE_API_KEY\n\nCreate a .env file:\nYOUTUBE_API_KEY=your_api_key_here\n\nEnable YouTube Data API v3 in Google Cloud Console and create an API key")
	}

	return cfg, nil
}
