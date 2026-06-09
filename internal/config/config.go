package config

import (
	"os"
	"strings"
)

// PodcastMeta holds podcast feed metadata configurable via the admin API.
type PodcastMeta struct {
	Title       string
	Description string
	Author      string
	Email       string
	Copyright   string
	ImageURL    string
	Category    string
	Explicit    bool
	Language    string
}

// Config holds application configuration.
type Config struct {
	Port        string
	BaseURL     string
	StaticPath  string
	WatchPath   string
	ContentPath string
	DataPath    string
	HubURLs     []string
	Podcast     PodcastMeta
}

// Default returns sensible defaults.
func Default() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	baseURL := getEnv("BASE_URL", "http://localhost:"+port)
	staticPath := getEnv("STATIC_PATH", "static")
	watchPath := getEnv("WATCH_PATH", "watch")
	contentPath := getEnv("CONTENT_PATH", "content")
	dataPath := getEnv("DATA_PATH", "data")

	var hubURLs []string
	if v := os.Getenv("HUB_URLS"); v != "" {
		for _, u := range strings.Split(v, ",") {
			u = strings.TrimSpace(u)
			if u != "" {
				hubURLs = append(hubURLs, u)
			}
		}
	}

	return Config{
		Port:        port,
		BaseURL:     baseURL,
		StaticPath:  staticPath,
		WatchPath:   watchPath,
		ContentPath: contentPath,
		DataPath:    dataPath,
		HubURLs:     hubURLs,
		Podcast: PodcastMeta{
			Title:       "neoncast Feed",
			Description: "Auto-generated podcast feed",
			Author:      "neoncast",
			Language:    "en-us",
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
