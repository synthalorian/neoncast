package analytics

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"neoncast/internal/models"
	"neoncast/internal/store"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	a := New(tmpDir)

	if a == nil {
		t.Fatal("expected analytics, got nil")
	}
	if a.path != filepath.Join(tmpDir, "analytics.json") {
		t.Errorf("expected path %s, got %s", filepath.Join(tmpDir, "analytics.json"), a.path)
	}
}

func TestLoadEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	a := New(tmpDir)

	if err := a.Load(); err != nil {
		t.Fatalf("expected no error for empty analytics, got %v", err)
	}

	if len(a.Recent(10)) != 0 {
		t.Errorf("expected 0 events, got %d", len(a.Recent(10)))
	}
}

func TestRecordAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	a := New(tmpDir)

	event := DownloadEvent{
		Timestamp:   time.Now(),
		EpisodeGUID: "ep-001",
		IP:          "127.0.0.1",
		UserAgent:   "TestAgent/1.0",
		StatusCode:  200,
		Bytes:       1024,
	}

	a.Record(event)
	if err := a.Save(); err != nil {
		t.Fatalf("save analytics: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(a.path); os.IsNotExist(err) {
		t.Fatal("expected analytics file to exist")
	}

	// Reload and verify
	a2 := New(tmpDir)
	if err := a2.Load(); err != nil {
		t.Fatalf("load analytics: %v", err)
	}

	recent := a2.Recent(10)
	if len(recent) != 1 {
		t.Fatalf("expected 1 event, got %d", len(recent))
	}
	if recent[0].EpisodeGUID != "ep-001" {
		t.Errorf("expected GUID ep-001, got %s", recent[0].EpisodeGUID)
	}
	if recent[0].IP != "127.0.0.1" {
		t.Errorf("expected IP 127.0.0.1, got %s", recent[0].IP)
	}
}

func TestRecentLimit(t *testing.T) {
	tmpDir := t.TempDir()
	a := New(tmpDir)

	for i := 0; i < 5; i++ {
		a.Record(DownloadEvent{
			Timestamp:   time.Now(),
			EpisodeGUID: fmt.Sprintf("ep-%d", i),
		})
	}

	recent := a.Recent(3)
	if len(recent) != 3 {
		t.Fatalf("expected 3 events, got %d", len(recent))
	}
	// Should return the most recent (last appended)
	if recent[0].EpisodeGUID != "ep-2" {
		t.Errorf("expected first event ep-2, got %s", recent[0].EpisodeGUID)
	}
	if recent[2].EpisodeGUID != "ep-4" {
		t.Errorf("expected last event ep-4, got %s", recent[2].EpisodeGUID)
	}
}

func TestSummary(t *testing.T) {
	tmpDir := t.TempDir()
	a := New(tmpDir)

	a.Record(DownloadEvent{EpisodeGUID: "ep-001", IP: "1.1.1.1", UserAgent: "Mozilla/5.0"})
	a.Record(DownloadEvent{EpisodeGUID: "ep-001", IP: "1.1.1.1", UserAgent: "Mozilla/5.0"})
	a.Record(DownloadEvent{EpisodeGUID: "ep-002", IP: "2.2.2.2", UserAgent: "curl/7.0"})

	summary := a.Summary()

	if summary.TotalDownloads != 3 {
		t.Errorf("expected 3 total downloads, got %d", summary.TotalDownloads)
	}
	if summary.UniqueIPs != 2 {
		t.Errorf("expected 2 unique IPs, got %d", summary.UniqueIPs)
	}
	if summary.ByEpisode["ep-001"] != 2 {
		t.Errorf("expected ep-001 count 2, got %d", summary.ByEpisode["ep-001"])
	}
	if summary.ByEpisode["ep-002"] != 1 {
		t.Errorf("expected ep-002 count 1, got %d", summary.ByEpisode["ep-002"])
	}
	if summary.ByClient["Browser"] != 2 {
		t.Errorf("expected Browser count 2, got %d", summary.ByClient["Browser"])
	}
	if summary.ByClient["curl"] != 1 {
		t.Errorf("expected curl count 1, got %d", summary.ByClient["curl"])
	}
}

func TestTrackMiddleware(t *testing.T) {
	tmpDir := t.TempDir()
	contentDir := t.TempDir()
	st := store.New(tmpDir)

	// Add a test episode
	_ = st.Add(models.Episode{
		Title:   "Test Episode",
		GUID:    "test-guid-001",
		FileURL: "http://localhost:8080/episodes/test-episode.mp3",
	})

	analytics := New(tmpDir)

	// Create test audio file in content directory
	testFile := filepath.Join(contentDir, "test-episode.mp3")
	_ = os.WriteFile(testFile, []byte("fake mp3 content"), 0644)

	// Set up Echo server
	e := echo.New()
	e.Use(analytics.Track(st))
	e.Static("/episodes", contentDir)

	// Make a request
	req := httptest.NewRequest(http.MethodGet, "/episodes/test-episode.mp3", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// Give the async save time to complete
	time.Sleep(100 * time.Millisecond)

	// Verify event was recorded
	recent := analytics.Recent(10)
	if len(recent) != 1 {
		t.Fatalf("expected 1 tracked event, got %d", len(recent))
	}

	ev := recent[0]
	if ev.EpisodeGUID != "test-guid-001" {
		t.Errorf("expected GUID test-guid-001, got %s", ev.EpisodeGUID)
	}
	if ev.EpisodeTitle != "Test Episode" {
		t.Errorf("expected title 'Test Episode', got %s", ev.EpisodeTitle)
	}
	if ev.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", ev.StatusCode)
	}
	if ev.Bytes == 0 {
		t.Error("expected non-zero bytes")
	}
}

func TestTrackMiddlewareSkipsNonEpisodes(t *testing.T) {
	tmpDir := t.TempDir()
	st := store.New(tmpDir)
	analytics := New(tmpDir)

	e := echo.New()
	e.Use(analytics.Track(st))
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if len(analytics.Recent(10)) != 0 {
		t.Error("expected no tracking for non-episode paths")
	}
}

func TestTrackMiddlewareSkipsNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	contentDir := t.TempDir()
	st := store.New(tmpDir)
	analytics := New(tmpDir)

	e := echo.New()
	e.Use(analytics.Track(st))
	e.Static("/episodes", contentDir)

	req := httptest.NewRequest(http.MethodGet, "/episodes/nonexistent.mp3", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if len(analytics.Recent(10)) != 0 {
		t.Error("expected no tracking for 404 responses")
	}
}

func TestNormalizeClient(t *testing.T) {
	tests := []struct {
		ua       string
		expected string
	}{
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36", "Browser"},
		{"AppleCoreMedia/1.0.0.19A404", "Apple Podcasts"},
		{"iTunes/12.11.0", "Apple Podcasts"},
		{"Spotify/1.0", "Spotify"},
		{"Overcast/3.0", "Overcast"},
		{"Pocket Casts/7.0", "Pocket Casts"},
		{"Stitcher/4.0", "Stitcher"},
		{"curl/7.64.1", "curl"},
		{"Wget/1.20.3", "wget"},
		{"SomeUnknownClient/1.0", "Unknown"},
	}

	for _, tt := range tests {
		got := normalizeClient(tt.ua)
		if got != tt.expected {
			t.Errorf("normalizeClient(%q) = %q, want %q", tt.ua, got, tt.expected)
		}
	}
}
