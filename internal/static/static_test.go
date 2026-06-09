package static

import (
	"io/fs"
	"strings"
	"testing"
)

func TestFS(t *testing.T) {
	fsys := FS()
	if fsys == nil {
		t.Fatal("expected non-nil filesystem")
	}

	_, err := fsys.Open("dashboard.html")
	if err != nil {
		t.Fatalf("expected dashboard.html to exist: %v", err)
	}
}

func TestDashboardHTML(t *testing.T) {
	f, err := DashboardHTML()
	if err != nil {
		t.Fatalf("expected dashboard.html to open: %v", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		t.Fatalf("expected stat to succeed: %v", err)
	}

	if info.IsDir() {
		t.Error("expected dashboard.html to be a file, not a directory")
	}

	if info.Size() == 0 {
		t.Error("expected dashboard.html to have content")
	}
}

func TestReadDashboardHTML(t *testing.T) {
	data, err := ReadDashboardHTML()
	if err != nil {
		t.Fatalf("expected read to succeed: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty dashboard.html content")
	}

	content := string(data)
	if !strings.Contains(content, "neoncast Admin") {
		t.Error("expected dashboard to contain 'neoncast Admin'")
	}

	if !strings.Contains(content, "<!DOCTYPE html>") {
		t.Error("expected dashboard to contain HTML doctype")
	}
}

func TestFSEmbed(t *testing.T) {
	_, err := fs.ReadFile(FS(), "nonexistent.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
