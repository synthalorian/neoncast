package ingest

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"neoncast/internal/models"
	"neoncast/internal/store"
)

// Pipeline processes audio files into episodes.
type Pipeline struct {
	store       *store.Store
	contentPath string
	baseURL     string
}

// New creates an ingest pipeline.
func New(s *store.Store, contentPath, baseURL string) *Pipeline {
	return &Pipeline{
		store:       s,
		contentPath: contentPath,
		baseURL:     strings.TrimRight(baseURL, "/"),
	}
}

// Handle processes a single audio file.
func (p *Pipeline) Handle(path string) {
	log.Printf("ingesting: %s", path)

	if err := p.process(path); err != nil {
		log.Printf("ingest failed for %s: %v", path, err)
		return
	}

	log.Printf("ingest complete: %s", path)
}

func (p *Pipeline) process(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	filename := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(filename))

	// Determine MIME type
	mimeType := "audio/mpeg"
	switch ext {
	case ".m4a":
		mimeType = "audio/mp4"
	case ".ogg":
		mimeType = "audio/ogg"
	case ".wav":
		mimeType = "audio/wav"
	}

	// Copy to content directory
	if err := os.MkdirAll(p.contentPath, 0755); err != nil {
		return fmt.Errorf("create content dir: %w", err)
	}

	destPath := filepath.Join(p.contentPath, filename)

	// Skip if already ingested (check by GUID would need filename+timestamp)
	// Instead, check if file exists and has same size
	if existing, err := os.Stat(destPath); err == nil {
		if existing.Size() == info.Size() {
			log.Printf("skipping duplicate: %s", filename)
			return nil
		}
		// Different size, use a unique filename
		destPath = filepath.Join(p.contentPath, fmt.Sprintf("%d-%s", time.Now().Unix(), filename))
	}

	if err := copyFile(path, destPath); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	// Create episode
	guid := store.GenerateGUID(filename)
	ep := models.Episode{
		Title:      store.FileToTitle(filename),
		GUID:       guid,
		PubDate:    time.Now(),
		Duration:   0, // Phase 4 will extract real duration
		FileURL:    fmt.Sprintf("%s/episodes/%s", p.baseURL, filepath.Base(destPath)),
		FileLength: info.Size(),
		FileType:   mimeType,
		Explicit:   false,
	}

	// Try to extract description from a sidecar file
	sidecarPath := strings.TrimSuffix(path, ext) + ".txt"
	if data, err := os.ReadFile(sidecarPath); err == nil {
		ep.Description = strings.TrimSpace(string(data))
	}

	// Add to store
	if err := p.store.Add(ep); err != nil {
		return fmt.Errorf("add to store: %w", err)
	}

	if err := p.store.Save(); err != nil {
		return fmt.Errorf("save store: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	buf := make([]byte, 32*1024)
	for {
		n, err := in.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
		}
		if err != nil {
			break
		}
	}

	return out.Sync()
}
