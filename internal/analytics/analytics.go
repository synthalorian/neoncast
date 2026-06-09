package analytics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	"neoncast/internal/store"
)

// DownloadEvent represents a single episode download.
type DownloadEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	EpisodeGUID string    `json:"episode_guid"`
	EpisodeTitle string   `json:"episode_title"`
	IP          string    `json:"ip"`
	UserAgent   string    `json:"user_agent"`
	Referrer    string    `json:"referrer"`
	StatusCode  int       `json:"status_code"`
	Bytes       int64     `json:"bytes"`
}

// Analytics manages download tracking with JSON-backed storage.
type Analytics struct {
	mu     sync.RWMutex
	path   string
	events []DownloadEvent
}

// New creates an Analytics instance.
func New(dataPath string) *Analytics {
	return &Analytics{
		path:   filepath.Join(dataPath, "analytics.json"),
		events: []DownloadEvent{},
	}
}

// Load reads analytics data from disk.
func (a *Analytics) Load() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	data, err := os.ReadFile(a.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read analytics: %w", err)
	}

	var events []DownloadEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return fmt.Errorf("unmarshal analytics: %w", err)
	}

	a.events = events
	return nil
}

// Save writes analytics data to disk.
func (a *Analytics) Save() error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(a.path), 0755); err != nil {
		return fmt.Errorf("mkdir analytics: %w", err)
	}

	data, err := json.MarshalIndent(a.events, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal analytics: %w", err)
	}

	if err := os.WriteFile(a.path, data, 0644); err != nil {
		return fmt.Errorf("write analytics: %w", err)
	}

	return nil
}

// Record adds a download event.
func (a *Analytics) Record(event DownloadEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.events = append(a.events, event)
}

// Summary returns aggregated download statistics.
func (a *Analytics) Summary() Summary {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := Summary{
		TotalDownloads: int64(len(a.events)),
		ByEpisode:      make(map[string]int64),
		ByClient:       make(map[string]int64),
	}

	uniqueIPs := make(map[string]struct{})
	for _, ev := range a.events {
		uniqueIPs[ev.IP] = struct{}{}
		if ev.EpisodeGUID != "" {
			result.ByEpisode[ev.EpisodeGUID]++
		}
		client := normalizeClient(ev.UserAgent)
		result.ByClient[client]++
	}

	result.UniqueIPs = len(uniqueIPs)
	return result
}

// Recent returns the most recent download events up to limit.
func (a *Analytics) Recent(limit int) []DownloadEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.events) == 0 {
		return []DownloadEvent{}
	}

	start := len(a.events) - limit
	if start < 0 {
		start = 0
	}

	out := make([]DownloadEvent, len(a.events[start:]))
	copy(out, a.events[start:])
	return out
}

// Summary holds aggregated analytics data.
type Summary struct {
	TotalDownloads int64            `json:"total_downloads"`
	UniqueIPs      int              `json:"unique_ips"`
	ByEpisode      map[string]int64 `json:"by_episode"`
	ByClient       map[string]int64 `json:"by_client"`
}

// Track returns Echo middleware that records downloads for /episodes/* paths.
func (a *Analytics) Track(episodeStore *store.Store) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			if !strings.HasPrefix(path, "/episodes/") {
				return next(c)
			}

			// Execute the handler first
			err := next(c)

			status := c.Response().Status
			if err == nil && status == http.StatusOK {
				filename := filepath.Base(path)

				// Look up episode by filename match on FileURL
				var guid, title string
				episodes := episodeStore.All()
				for _, ep := range episodes {
					if strings.HasSuffix(ep.FileURL, "/"+filename) {
						guid = ep.GUID
						title = ep.Title
						break
					}
				}

				event := DownloadEvent{
					Timestamp:    time.Now(),
					EpisodeGUID:  guid,
					EpisodeTitle: title,
					IP:           c.RealIP(),
					UserAgent:    c.Request().UserAgent(),
					Referrer:     c.Request().Referer(),
					StatusCode:   status,
					Bytes:        c.Response().Size,
				}

				a.Record(event)
				// Save asynchronously to avoid blocking the response
				go func() {
					if err := a.Save(); err != nil {
						// Best effort; log but don't fail the request
					}
				}()
			}

			return err
		}
	}
}

// normalizeClient extracts a simplified client name from a User-Agent string.
func normalizeClient(ua string) string {
	ua = strings.ToLower(ua)
	switch {
		case strings.Contains(ua, "applecoremedia") || strings.Contains(ua, "itunes") || strings.Contains(ua, "podcasts"):
			return "Apple Podcasts"
		case strings.Contains(ua, "spotify"):
			return "Spotify"
		case strings.Contains(ua, "overcast"):
			return "Overcast"
		case strings.Contains(ua, "pocket casts"):
			return "Pocket Casts"
		case strings.Contains(ua, "stitcher"):
			return "Stitcher"
		case strings.Contains(ua, "curl"):
			return "curl"
		case strings.Contains(ua, "wget"):
			return "wget"
		case strings.Contains(ua, "mozilla") || strings.Contains(ua, "chrome") || strings.Contains(ua, "safari"):
			return "Browser"
		default:
			return "Unknown"
	}
}
