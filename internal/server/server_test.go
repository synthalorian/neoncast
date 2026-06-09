package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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

	srv := New(cfg, st, nil, nil)

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

	srv := New(cfg, st, nil, nil)

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

	srv := New(cfg, st, nil, nil)

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

	srv := New(cfg, st, nil, nil)

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

	srv := New(cfg, st, nil, nil)

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

	srv := New(cfg, st, nil, nil)

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

	srv := New(cfg, st, nil, nil)

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

	srv := New(cfg, st, an, nil)

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

	srv := New(cfg, st, an, nil)

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

	srv := New(cfg, st, an, nil)

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

	srv := New(cfg, st, an, nil)

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

	srv := New(cfg, st, nil, nil)

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

	srv := New(cfg, st, an, nil)

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

type fakePublisher struct {
	calls atomic.Int32
}

func (f *fakePublisher) Publish(ctx context.Context) error {
	f.calls.Add(1)
	return nil
}

func TestFeedIncludesWebSubLinks(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{
		Port:    "8080",
		BaseURL: "https://podcast.example.com",
		HubURLs: []string{"https://hub.example.com/"},
		Podcast: config.PodcastMeta{
			Title:    "neoncast Feed",
			Language: "en-us",
		},
		StaticPath: tmpDir,
	}
	st := store.New(t.TempDir())

	srv := New(cfg, st, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/feed", nil)
	rec := httptest.NewRecorder()
	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `xmlns:atom="http://www.w3.org/2005/Atom"`) {
		t.Error("expected atom namespace in feed")
	}
	if !strings.Contains(body, `<atom:link rel="self" type="application/rss+xml" href="https://podcast.example.com/feed"></atom:link>`) {
		t.Error("expected atom self link")
	}
	if !strings.Contains(body, `<atom:link rel="hub" href="https://hub.example.com/"></atom:link>`) {
		t.Error("expected atom hub link")
	}
}

func TestUploadNotifiesPublisher(t *testing.T) {
	tmpDir := t.TempDir()
	contentDir := t.TempDir()
	cfg := config.Config{
		Port:        "8080",
		StaticPath:  tmpDir,
		ContentPath: contentDir,
		BaseURL:     "https://podcast.example.com",
		HubURLs:     []string{"https://hub.example.com/"},
	}
	st := store.New(t.TempDir())
	pub := &fakePublisher{}

	srv := New(cfg, st, nil, pub)

	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	part, err := mw.CreateFormFile("file", "upload-test.mp3")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.WriteString(part, "fake audio"); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/upload", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	time.Sleep(100 * time.Millisecond)
	if pub.calls.Load() != 1 {
		t.Errorf("expected publisher to be called once, got %d", pub.calls.Load())
	}
}

func TestListEpisodes(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(t.TempDir())

	_ = st.Add(models.Episode{Title: "Episode 1", GUID: "ep-1", PubDate: time.Now()})
	_ = st.Add(models.Episode{Title: "Episode 2", GUID: "ep-2", PubDate: time.Now().Add(-time.Hour)})
	_ = st.Save()

	srv := New(cfg, st, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/episodes", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"Title":"Episode 1"`) {
		t.Errorf("expected Episode 1 in response, got %s", body)
	}
	if !strings.Contains(body, `"Title":"Episode 2"`) {
		t.Errorf("expected Episode 2 in response, got %s", body)
	}
}

func TestGetEpisode(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(t.TempDir())

	_ = st.Add(models.Episode{Title: "Test Episode", GUID: "test-guid", PubDate: time.Now()})
	_ = st.Save()

	srv := New(cfg, st, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/episodes/test-guid", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"Title":"Test Episode"`) {
		t.Errorf("expected episode title, got %s", body)
	}
}

func TestGetEpisodeNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(t.TempDir())

	srv := New(cfg, st, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/episodes/nonexistent", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestUpdateEpisode(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(t.TempDir())

	_ = st.Add(models.Episode{Title: "Original Title", GUID: "update-guid", PubDate: time.Now()})
	_ = st.Save()

	srv := New(cfg, st, nil, nil)

	body := `{"title":"Updated Title","description":"New description","duration":3600,"explicit":true,"season":2,"episode":5}`
	req := httptest.NewRequest(http.MethodPut, "/api/episodes/update-guid", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	resp := rec.Body.String()
	if !strings.Contains(resp, `"Title":"Updated Title"`) {
		t.Errorf("expected updated title, got %s", resp)
	}

	ep, ok := st.GetByGUID("update-guid")
	if !ok {
		t.Fatal("episode not found after update")
	}
	if ep.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %s", ep.Title)
	}
	if ep.Description != "New description" {
		t.Errorf("expected description 'New description', got %s", ep.Description)
	}
	if ep.Duration != 3600 {
		t.Errorf("expected duration 3600, got %d", ep.Duration)
	}
	if !ep.Explicit {
		t.Error("expected explicit to be true")
	}
	if ep.Season != 2 {
		t.Errorf("expected season 2, got %d", ep.Season)
	}
	if ep.Episode != 5 {
		t.Errorf("expected episode 5, got %d", ep.Episode)
	}
}

func TestUpdateEpisodeNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(t.TempDir())

	srv := New(cfg, st, nil, nil)

	body := `{"title":"Updated Title"}`
	req := httptest.NewRequest(http.MethodPut, "/api/episodes/nonexistent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestDeleteEpisode(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(t.TempDir())

	_ = st.Add(models.Episode{Title: "To Delete", GUID: "delete-guid", PubDate: time.Now()})
	_ = st.Save()

	srv := New(cfg, st, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/episodes/delete-guid", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}

	if st.Has("delete-guid") {
		t.Error("expected episode to be deleted")
	}
}

func TestDeleteEpisodeNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(t.TempDir())

	srv := New(cfg, st, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/episodes/nonexistent", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestUploadEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	contentDir := t.TempDir()
	cfg := config.Config{
		Port:        "8080",
		StaticPath:  tmpDir,
		ContentPath: contentDir,
		BaseURL:     "https://podcast.example.com",
	}
	st := store.New(t.TempDir())

	srv := New(cfg, st, nil, nil)

	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	part, err := mw.CreateFormFile("file", "test-upload.mp3")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.WriteString(part, "fake mp3 audio data"); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/upload", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"Title":"Test Upload"`) {
		t.Errorf("expected title in response, got %s", body)
	}
	if !strings.Contains(body, `"FileType":"audio/mpeg"`) {
		t.Errorf("expected audio/mpeg file type, got %s", body)
	}

	expectedPath := filepath.Join(contentDir, "test-upload.mp3")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("expected uploaded file to exist")
	}

	if len(st.All()) != 1 {
		t.Errorf("expected 1 episode in store, got %d", len(st.All()))
	}
}

func TestUploadMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(t.TempDir())

	srv := New(cfg, st, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/upload", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestGetPodcast(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{
		Port:       "8080",
		StaticPath: tmpDir,
		Podcast: config.PodcastMeta{
			Title:    "Test Podcast",
			Author:   "Test Author",
			Language: "en",
		},
	}
	st := store.New(t.TempDir())

	srv := New(cfg, st, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/podcast", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"Title":"Test Podcast"`) {
		t.Errorf("expected podcast title, got %s", body)
	}
	if !strings.Contains(body, `"Author":"Test Author"`) {
		t.Errorf("expected podcast author, got %s", body)
	}
}

func TestUpdatePodcast(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{
		Port:       "8080",
		StaticPath: tmpDir,
		Podcast: config.PodcastMeta{
			Title:    "Original Title",
			Language: "en-us",
		},
	}
	st := store.New(t.TempDir())

	srv := New(cfg, st, nil, nil)

	body := `{"title":"New Title","description":"New description","author":"New Author","email":"new@example.com","copyright":"2024","image_url":"https://example.com/image.jpg","category":"Technology","explicit":true,"language":"en"}`
	req := httptest.NewRequest(http.MethodPut, "/api/podcast", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	resp := rec.Body.String()
	if !strings.Contains(resp, `"Title":"New Title"`) {
		t.Errorf("expected updated title, got %s", resp)
	}
	if !strings.Contains(resp, `"Author":"New Author"`) {
		t.Errorf("expected updated author, got %s", resp)
	}
	if !strings.Contains(resp, `"Language":"en"`) {
		t.Errorf("expected updated language, got %s", resp)
	}

	if srv.cfg.Podcast.Title != "New Title" {
		t.Errorf("expected title 'New Title', got %s", srv.cfg.Podcast.Title)
	}
	if srv.cfg.Podcast.Explicit != true {
		t.Error("expected explicit to be true")
	}
}

func TestAdminDashboardRoute(t *testing.T) {
	tmpDir := t.TempDir()
	dashboardFile := filepath.Join(tmpDir, "dashboard.html")
	if err := os.WriteFile(dashboardFile, []byte("<html><body>Admin Dashboard</body></html>"), 0644); err != nil {
		t.Fatalf("create dashboard file: %v", err)
	}

	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(t.TempDir())

	srv := New(cfg, st, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Admin Dashboard") {
		t.Errorf("expected dashboard content, got %s", body)
	}
}

func TestAdminDashboardRedirect(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(t.TempDir())

	srv := New(cfg, st, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected status 302, got %d", rec.Code)
	}

	location := rec.Header().Get("Location")
	if location != "/admin" {
		t.Errorf("expected redirect to /admin, got %s", location)
	}
}
