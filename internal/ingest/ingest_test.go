package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"neoncast/internal/store"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	st := store.New(tmpDir)

	p := New(st, tmpDir, "http://localhost:8080", nil)
	if p == nil {
		t.Fatal("expected pipeline, got nil")
	}
}

func TestProcessFile(t *testing.T) {
	tmpDir := t.TempDir()
	contentDir := t.TempDir()
	st := store.New(tmpDir)

	p := New(st, contentDir, "http://localhost:8080", nil)

	// Create a fake mp3 file
	srcFile := filepath.Join(tmpDir, "my-test-episode.mp3")
	content := []byte("fake mp3 content for testing")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("create test file: %v", err)
	}

	p.Handle(srcFile)

	// Check store has the episode
	eps := st.All()
	if len(eps) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(eps))
	}

	ep := eps[0]
	if ep.Title != "My Test Episode" {
		t.Errorf("expected title 'My Test Episode', got %s", ep.Title)
	}
	if ep.FileType != "audio/mpeg" {
		t.Errorf("expected audio/mpeg, got %s", ep.FileType)
	}
	if ep.FileLength != int64(len(content)) {
		t.Errorf("expected length %d, got %d", len(content), ep.FileLength)
	}
	if !strings.HasSuffix(ep.FileURL, "/my-test-episode.mp3") {
		t.Errorf("expected URL ending with /my-test-episode.mp3, got %s", ep.FileURL)
	}

	// Check file was copied
	copiedFile := filepath.Join(contentDir, "my-test-episode.mp3")
	if _, err := os.Stat(copiedFile); os.IsNotExist(err) {
		t.Error("expected file to be copied to content directory")
	}
}

func TestProcessWithSidecar(t *testing.T) {
	tmpDir := t.TempDir()
	contentDir := t.TempDir()
	st := store.New(tmpDir)

	p := New(st, contentDir, "http://localhost:8080", nil)

	// Create mp3 + sidecar description
	srcFile := filepath.Join(tmpDir, "described.mp3")
	_ = os.WriteFile(srcFile, []byte("audio"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "described.txt"), []byte("This is the episode description."), 0644)

	p.Handle(srcFile)

	eps := st.All()
	if len(eps) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(eps))
	}

	if eps[0].Description != "This is the episode description." {
		t.Errorf("expected description from sidecar, got %q", eps[0].Description)
	}
}

func TestProcessDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	contentDir := t.TempDir()
	st := store.New(tmpDir)

	p := New(st, contentDir, "http://localhost:8080", nil)

	srcFile := filepath.Join(tmpDir, "dup.mp3")
	_ = os.WriteFile(srcFile, []byte("audio"), 0644)

	// Process twice
	p.Handle(srcFile)
	p.Handle(srcFile)

	eps := st.All()
	if len(eps) != 1 {
		t.Errorf("expected 1 episode (duplicate skipped), got %d", len(eps))
	}
}

func TestProcessM4A(t *testing.T) {
	tmpDir := t.TempDir()
	contentDir := t.TempDir()
	st := store.New(tmpDir)

	p := New(st, contentDir, "http://localhost:8080", nil)

	srcFile := filepath.Join(tmpDir, "episode.m4a")
	_ = os.WriteFile(srcFile, []byte("audio"), 0644)

	p.Handle(srcFile)

	eps := st.All()
	if len(eps) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(eps))
	}

	if eps[0].FileType != "audio/mp4" {
		t.Errorf("expected audio/mp4 for m4a, got %s", eps[0].FileType)
	}
}

func TestProcessOGG(t *testing.T) {
	tmpDir := t.TempDir()
	contentDir := t.TempDir()
	st := store.New(tmpDir)

	p := New(st, contentDir, "http://localhost:8080", nil)

	srcFile := filepath.Join(tmpDir, "episode.ogg")
	_ = os.WriteFile(srcFile, []byte("audio"), 0644)

	p.Handle(srcFile)

	eps := st.All()
	if len(eps) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(eps))
	}

	if eps[0].FileType != "audio/ogg" {
		t.Errorf("expected audio/ogg for ogg, got %s", eps[0].FileType)
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")

	content := []byte("hello world test content")
	_ = os.WriteFile(src, content, 0644)

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copy file: %v", err)
	}

	copied, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read copied file: %v", err)
	}

	if string(copied) != string(content) {
		t.Errorf("copied content mismatch: got %q, want %q", copied, content)
	}
}
