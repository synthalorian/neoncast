package watcher

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type testHandler struct {
	mu     sync.Mutex
	files  []string
	called int
}

func (h *testHandler) Handle(path string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.files = append(h.files, path)
	h.called++
}

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	h := &testHandler{}

	w := New(tmpDir, h)
	if w == nil {
		t.Fatal("expected watcher, got nil")
	}
	if w.path != tmpDir {
		t.Errorf("expected path %s, got %s", tmpDir, w.path)
	}
}

func TestStartAndStop(t *testing.T) {
	tmpDir := t.TempDir()
	h := &testHandler{}
	w := New(tmpDir, h)

	if err := w.Start(); err != nil {
		t.Fatalf("start watcher: %v", err)
	}

	if err := w.Stop(); err != nil {
		t.Fatalf("stop watcher: %v", err)
	}
}

func TestScanExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files before starting watcher
	_ = os.WriteFile(filepath.Join(tmpDir, "existing.mp3"), []byte("test"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("skip"), 0644)

	h := &testHandler{}
	w := New(tmpDir, h)

	if err := w.Start(); err != nil {
		t.Fatalf("start watcher: %v", err)
	}
	defer w.Stop()

	// Give scanner time to process
	time.Sleep(100 * time.Millisecond)

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.called != 1 {
		t.Errorf("expected 1 call, got %d", h.called)
	}
	if len(h.files) != 1 || filepath.Base(h.files[0]) != "existing.mp3" {
		t.Errorf("expected existing.mp3, got %v", h.files)
	}
}

func TestWatchNewFile(t *testing.T) {
	tmpDir := t.TempDir()

	h := &testHandler{}
	w := New(tmpDir, h)

	if err := w.Start(); err != nil {
		t.Fatalf("start watcher: %v", err)
	}
	defer w.Stop()

	// Create a new file
	testFile := filepath.Join(tmpDir, "new-episode.mp3")
	if err := os.WriteFile(testFile, []byte("audio data"), 0644); err != nil {
		t.Fatalf("create test file: %v", err)
	}

	// Wait for watcher to process
	time.Sleep(200 * time.Millisecond)

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.called < 1 {
		t.Errorf("expected at least 1 call, got %d", h.called)
	}

	found := false
	for _, f := range h.files {
		if filepath.Base(f) == "new-episode.mp3" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected new-episode.mp3 in handled files, got %v", h.files)
	}
}

func TestWatchSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()

	h := &testHandler{}
	w := New(tmpDir, h)

	if err := w.Start(); err != nil {
		t.Fatalf("start watcher: %v", err)
	}
	defer w.Stop()

	// Create subdirectory with file
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("create subdir: %v", err)
	}

	testFile := filepath.Join(subDir, "sub-episode.mp3")
	if err := os.WriteFile(testFile, []byte("audio"), 0644); err != nil {
		t.Fatalf("create test file: %v", err)
	}

	// Wait for watcher
	time.Sleep(200 * time.Millisecond)

	h.mu.Lock()
	defer h.mu.Unlock()

	found := false
	for _, f := range h.files {
		if filepath.Base(f) == "sub-episode.mp3" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected sub-episode.mp3 in handled files, got %v", h.files)
	}
}

func TestIsAudioFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"episode.mp3", true},
		{"episode.m4a", true},
		{"episode.ogg", true},
		{"episode.wav", true},
		{"episode.txt", false},
		{"episode", false},
		{"episode.MP3", true},
	}

	for _, tt := range tests {
		got := isAudioFile(tt.path)
		if got != tt.expected {
			t.Errorf("isAudioFile(%q) = %v, want %v", tt.path, got, tt.expected)
		}
	}
}

func TestHandlerFunc(t *testing.T) {
	var called bool
	h := HandlerFunc(func(path string) {
		called = true
	})

	h.Handle("test.mp3")
	if !called {
		t.Error("expected HandlerFunc to be called")
	}
}
