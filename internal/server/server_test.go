package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	// Save and restore env vars
	origPort := os.Getenv("PORT")
	origStatic := os.Getenv("STATIC_PATH")
	defer func() {
		os.Setenv("PORT", origPort)
		os.Setenv("STATIC_PATH", origStatic)
	}()

	t.Run("defaults", func(t *testing.T) {
		os.Unsetenv("PORT")
		os.Unsetenv("STATIC_PATH")

		cfg := DefaultConfig()
		if cfg.Port != "8080" {
			t.Errorf("expected default port 8080, got %s", cfg.Port)
		}
		if cfg.StaticPath != "static" {
			t.Errorf("expected default static path 'static', got %s", cfg.StaticPath)
		}
	})

	t.Run("from env", func(t *testing.T) {
		os.Setenv("PORT", "3000")
		os.Setenv("STATIC_PATH", "/tmp/static")

		cfg := DefaultConfig()
		if cfg.Port != "3000" {
			t.Errorf("expected port 3000, got %s", cfg.Port)
		}
		if cfg.StaticPath != "/tmp/static" {
			t.Errorf("expected static path '/tmp/static', got %s", cfg.StaticPath)
		}
	})
}

func TestNew(t *testing.T) {
	cfg := Config{Port: "8080", StaticPath: "static"}
	srv := New(cfg)

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
	// Create a temp static dir for the test
	tmpDir := t.TempDir()
	cfg := Config{Port: "8080", StaticPath: tmpDir}
	srv := New(cfg)

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
	// Create a temp static dir with a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "hello from static"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg := Config{Port: "8080", StaticPath: tmpDir}
	srv := New(cfg)

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
	cfg := Config{Port: "8080", StaticPath: tmpDir}
	srv := New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent.txt", nil)
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestMiddleware(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{Port: "8080", StaticPath: tmpDir}
	srv := New(cfg)

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
	cfg := Config{Port: "8080", StaticPath: tmpDir}
	srv := New(cfg)

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
	if !strings.Contains(body, "<title>neoncast Demo Feed</title>") {
		t.Error("expected demo podcast title")
	}
	if !strings.Contains(body, "<title>Welcome to neoncast</title>") {
		t.Error("expected demo episode title")
	}
	if !strings.Contains(body, `<enclosure url="/episodes/demo-001.mp3"`) {
		t.Error("expected enclosure element")
	}
}
