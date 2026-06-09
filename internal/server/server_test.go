package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"neoncast/internal/analytics"
	"neoncast/internal/config"
	"neoncast/internal/models"
	"neoncast/internal/store"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(tmpDir)

	srv := New(cfg, st, nil)

	if srv == nil {
		t.Fatal("expected server, got nil")
	}
	if srv.cfg.Port != "8080" {
		t.Errorf("expected port 8080, got %s", srv.cfg.Port)
	}
	if srv.echo == nil {
		t.Error("expected echo instance, got nil")
	}
}

func TestHealthEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(tmpDir)

	srv := New(cfg, st, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Errorf("expected status ok in body, got %s", body)
	}
}

func TestStaticFileServing(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "hello from static"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(t.TempDir())

	srv := New(cfg, st, nil)

	req := httptest.NewRequest(http.MethodGet, "/test.txt", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if body != content {
		t.Errorf("expected body %q, got %q", content, body)
	}
}

func TestStaticFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(t.TempDir())

	srv := New(cfg, st, nil)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent.txt", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestMiddleware(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(t.TempDir())

	srv := New(cfg, st, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	// Check that request ID middleware added a header
	if rec.Header().Get("X-Request-Id") == "" {
		t.Error("expected X-Request-Id header to be set")
	}
}

func TestFeedEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{
		Port:       "8080",
		StaticPath: tmpDir,
		Podcast: config.PodcastMeta{
			Title:    "neoncast Feed",
			Language: "en-us",
		},
	}
	st := store.New(t.TempDir())

	srv := New(cfg, st, nil)

	req := httptest.NewRequest(http.MethodGet, "/feed", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/rss+xml") {
		t.Errorf("expected Content-Type application/rss+xml, got %s", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("expected XML declaration in response")
	}
	if !strings.Contains(body, `<rss xmlns:itunes=`) {
		t.Error("expected rss root with itunes namespace")
	}
	if !strings.Contains(body, "<title>neoncast Feed</title>") {
		t.Error("expected podcast title")
	}
}

func TestFeedWithEpisodes(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{
		Port:        "8080",
		StaticPath:  tmpDir,
		ContentPath: tmpDir,
		Podcast: config.PodcastMeta{
			Title:    "neoncast Feed",
			Language: "en-us",
		},
	}
	st := store.New(t.TempDir())

	// Add a test episode
	ep := store.GenerateGUID("test-episode.mp3")
	_ = st.Add(models.Episode{
		Title:    "Test Episode",
		GUID:     ep,
		FileURL:  "/episodes/test-episode.mp3",
	})
	_ = st.Save()

	srv := New(cfg, st, nil)

	req := httptest.NewRequest(http.MethodGet, "/feed", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "<title>Test Episode</title>") {
		t.Errorf("expected episode title in feed, got:\n%s", body)
	}
	if !strings.Contains(body, `<enclosure url="/episodes/test-episode.mp3"`) {
		t.Errorf("expected enclosure in feed, got:\n%s", body)
	}
}

func TestAnalyticsDownloadsEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(tmpDir)
	an := analytics.New(tmpDir)

	an.Record(analytics.DownloadEvent{EpisodeGUID: "ep-001", IP: "1.1.1.1"})
	an.Record(analytics.DownloadEvent{EpisodeGUID: "ep-001", IP: "2.2.2.2"})
	an.Record(analytics.DownloadEvent{EpisodeGUID: "ep-002", IP: "3.3.3.3"})

	srv := New(cfg, st, an)

	req := httptest.NewRequest(http.MethodGet, "/api/analytics/downloads", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"ep-001":2`) {
		t.Errorf("expected ep-001 count 2, got %s", body)
	}
	if !strings.Contains(body, `"ep-002":1`) {
		t.Errorf("expected ep-002 count 1, got %s", body)
	}
}

func TestAnalyticsSummaryEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(tmpDir)
	an := analytics.New(tmpDir)

	an.Record(analytics.DownloadEvent{EpisodeGUID: "ep-001", IP: "1.1.1.1", UserAgent: "Mozilla/5.0"})
	an.Record(analytics.DownloadEvent{EpisodeGUID: "ep-002", IP: "2.2.2.2", UserAgent: "curl/7.0"})

	srv := New(cfg, st, an)

	req := httptest.NewRequest(http.MethodGet, "/api/analytics/summary", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"total_downloads":2`) {
		t.Errorf("expected total_downloads 2, got %s", body)
	}
	if !strings.Contains(body, `"unique_ips":2`) {
		t.Errorf("expected unique_ips 2, got %s", body)
	}
}

func TestAnalyticsRecentEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(tmpDir)
	an := analytics.New(tmpDir)

	an.Record(analytics.DownloadEvent{EpisodeGUID: "ep-001", IP: "1.1.1.1"})

	srv := New(cfg, st, an)

	req := httptest.NewRequest(http.MethodGet, "/api/analytics/recent", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"episode_guid":"ep-001"`) {
		t.Errorf("expected ep-001 in recent events, got %s", body)
	}
}

func TestAnalyticsRecentLimit(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(tmpDir)
	an := analytics.New(tmpDir)

	for i := 0; i < 5; i++ {
		an.Record(analytics.DownloadEvent{EpisodeGUID: fmt.Sprintf("ep-%d", i)})
	}

	srv := New(cfg, st, an)

	req := httptest.NewRequest(http.MethodGet, "/api/analytics/recent?limit=2", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	count := strings.Count(body, `"episode_guid"`)
	if count != 2 {
		t.Errorf("expected 2 events with limit=2, got %d", count)
	}
}

func TestAnalyticsEndpointsUnavailable(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(tmpDir)

	srv := New(cfg, st, nil)

	endpoints := []string{
		"/api/analytics/downloads",
		"/api/analytics/summary",
		"/api/analytics/recent",
	}

	for _, endpoint := range endpoints {
		req := httptest.NewRequest(http.MethodGet, endpoint, nil)
		rec := httptest.NewRecorder()

		srv.echo.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503 for %s, got %d", endpoint, rec.Code)
		}
	}
}

func TestAnalyticsTrackDownload(t *testing.T) {
	tmpDir := t.TempDir()
	contentDir := t.TempDir()
	cfg := config.Config{
		Port:        "8080",
		StaticPath:  tmpDir,
		ContentPath: contentDir,
	}
	st := store.New(tmpDir)
	an := analytics.New(tmpDir)

	_ = st.Add(models.Episode{
		Title:   "Analytics Test",
		GUID:    "analytics-001",
		FileURL: "http://localhost:8080/episodes/analytics-test.mp3",
	})
	_ = os.WriteFile(filepath.Join(contentDir, "analytics-test.mp3"), []byte("fake audio"), 0644)

	srv := New(cfg, st, an)

	req := httptest.NewRequest(http.MethodGet, "/episodes/analytics-test.mp3", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	time.Sleep(100 * time.Millisecond)

	recent := an.Recent(10)
	if len(recent) != 1 {
		t.Fatalf("expected 1 tracked download, got %d", len(recent))
	}

	if recent[0].EpisodeGUID != "analytics-001" {
		t.Errorf("expected GUID analytics-001, got %s", recent[0].EpisodeGUID)
	}
	if recent[0].EpisodeTitle != "Analytics Test" {
		t.Errorf("expected title 'Analytics Test', got %s", recent[0].EpisodeTitle)
	}
}
