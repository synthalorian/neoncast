package config

import "os"

// Config holds application configuration.
type Config struct {
	Port        string
	StaticPath  string
	WatchPath   string
	ContentPath string
	DataPath    string
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
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
