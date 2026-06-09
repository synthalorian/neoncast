package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"neoncast/internal/config"
	"neoncast/internal/models"
	"neoncast/internal/store"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{Port: "8080", StaticPath: tmpDir}
	st := store.New(tmpDir)

	srv := New(cfg, st)

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

	srv := New(cfg, st)

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

	srv := New(cfg, st)

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

	srv := New(cfg, st)

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

	srv := New(cfg, st)

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

	srv := New(cfg, st)

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

	srv := New(cfg, st)

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
