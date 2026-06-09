package config

import "os"

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
	StaticPath  string
	WatchPath   string
	ContentPath string
	DataPath    string
	Podcast     PodcastMeta
}

// Default returns sensible defaults.
func Default() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	staticPath := getEnv("STATIC_PATH", "static")
	watchPath := getEnv("WATCH_PATH", "watch")
	contentPath := getEnv("CONTENT_PATH", "content")
	dataPath := getEnv("DATA_PATH", "data")

	return Config{
		Port:        port,
		StaticPath:  staticPath,
		WatchPath:   watchPath,
		ContentPath: contentPath,
		DataPath:    dataPath,
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
