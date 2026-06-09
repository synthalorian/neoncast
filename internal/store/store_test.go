package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"neoncast/internal/models"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)

	if s == nil {
		t.Fatal("expected store, got nil")
	}
	if s.path != filepath.Join(tmpDir, "episodes.json") {
		t.Errorf("expected path %s, got %s", filepath.Join(tmpDir, "episodes.json"), s.path)
	}
}

func TestLoadEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)

	if err := s.Load(); err != nil {
		t.Fatalf("expected no error for empty store, got %v", err)
	}

	if len(s.All()) != 0 {
		t.Errorf("expected 0 episodes, got %d", len(s.All()))
	}
}

func TestAddAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)

	ep := models.Episode{
		Title:   "Test Episode",
		GUID:    "test-001",
		PubDate: time.Now(),
	}

	if err := s.Add(ep); err != nil {
		t.Fatalf("add episode: %v", err)
	}

	if err := s.Save(); err != nil {
		t.Fatalf("save store: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		t.Fatal("expected store file to exist")
	}

	// Reload and verify
	s2 := New(tmpDir)
	if err := s2.Load(); err != nil {
		t.Fatalf("load store: %v", err)
	}

	eps := s2.All()
	if len(eps) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(eps))
	}
	if eps[0].GUID != "test-001" {
		t.Errorf("expected GUID test-001, got %s", eps[0].GUID)
	}
	if eps[0].Title != "Test Episode" {
		t.Errorf("expected title 'Test Episode', got %s", eps[0].Title)
	}
}

func TestAddDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)

	ep := models.Episode{GUID: "dup-001", Title: "First"}
	if err := s.Add(ep); err != nil {
		t.Fatalf("add first: %v", err)
	}

	if err := s.Add(ep); err == nil {
		t.Fatal("expected error for duplicate GUID, got nil")
	}
}

func TestUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)

	ep := models.Episode{GUID: "update-001", Title: "Original"}
	_ = s.Add(ep)

	updated := models.Episode{GUID: "update-001", Title: "Updated"}
	if err := s.Update(updated); err != nil {
		t.Fatalf("update: %v", err)
	}

	eps := s.All()
	if len(eps) != 1 || eps[0].Title != "Updated" {
		t.Errorf("expected title 'Updated', got %s", eps[0].Title)
	}
}

func TestUpdateNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)

	if err := s.Update(models.Episode{GUID: "missing"}); err == nil {
		t.Fatal("expected error for missing episode")
	}
}

func TestDelete(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)

	_ = s.Add(models.Episode{GUID: "del-001"})
	_ = s.Add(models.Episode{GUID: "del-002"})

	if err := s.Delete("del-001"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if len(s.All()) != 1 {
		t.Errorf("expected 1 episode, got %d", len(s.All()))
	}
	if s.Has("del-001") {
		t.Error("expected del-001 to be removed")
	}
}

func TestDeleteNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)

	if err := s.Delete("missing"); err == nil {
		t.Fatal("expected error for missing episode")
	}
}

func TestSortByDateDesc(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)

	now := time.Now()
	_ = s.Add(models.Episode{GUID: "oldest", PubDate: now.Add(-2 * time.Hour)})
	_ = s.Add(models.Episode{GUID: "newest", PubDate: now})
	_ = s.Add(models.Episode{GUID: "middle", PubDate: now.Add(-1 * time.Hour)})

	eps := s.All()
	if len(eps) != 3 {
		t.Fatalf("expected 3 episodes, got %d", len(eps))
	}

	if eps[0].GUID != "newest" {
		t.Errorf("expected newest first, got %s", eps[0].GUID)
	}
	if eps[1].GUID != "middle" {
		t.Errorf("expected middle second, got %s", eps[1].GUID)
	}
	if eps[2].GUID != "oldest" {
		t.Errorf("expected oldest third, got %s", eps[2].GUID)
	}
}

func TestFileToTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-episode.mp3", "My Episode"},
		{"hello_world.m4a", "Hello World"},
		{"UPPER-CASE.ogg", "Upper Case"},
		{"mixed-file_name.wav", "Mixed File Name"},
		{"noext", "Noext"},
	}

	for _, tt := range tests {
		got := FileToTitle(tt.input)
		if got != tt.expected {
			t.Errorf("FileToTitle(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestGenerateGUID(t *testing.T) {
	guid1 := GenerateGUID("test.mp3")
	if guid1 == "" {
		t.Error("expected non-empty GUID")
	}

	if !strings.Contains(guid1, "test-") {
		t.Errorf("expected GUID to contain 'test-', got %s", guid1)
	}
}
