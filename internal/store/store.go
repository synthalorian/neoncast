package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"neoncast/internal/models"
)

// Store manages episode persistence with JSON-backed storage.
type Store struct {
	mu       sync.RWMutex
	path     string
	episodes []models.Episode
}

// New creates a Store and loads existing data.
func New(dataPath string) *Store {
	return &Store{
		path:     filepath.Join(dataPath, "episodes.json"),
		episodes: []models.Episode{},
	}
}

// Load reads episodes from disk.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read store: %w", err)
	}

	var episodes []models.Episode
	if err := json.Unmarshal(data, &episodes); err != nil {
		return fmt.Errorf("unmarshal store: %w", err)
	}

	s.episodes = episodes
	return nil
}

// Save writes episodes to disk.
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("mkdir store: %w", err)
	}

	data, err := json.MarshalIndent(s.episodes, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal store: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("write store: %w", err)
	}

	return nil
}

// All returns episodes sorted by pub date descending.
func (s *Store) All() []models.Episode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]models.Episode, len(s.episodes))
	copy(out, s.episodes)
	return out
}

// Add inserts a new episode if the GUID is not already present.
func (s *Store) Add(ep models.Episode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.episodes {
		if existing.GUID == ep.GUID {
			return fmt.Errorf("episode with GUID %q already exists", ep.GUID)
		}
	}

	s.episodes = append(s.episodes, ep)
	s.sortByDateDesc()
	return nil
}

// Update replaces an episode by GUID.
func (s *Store) Update(ep models.Episode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existing := range s.episodes {
		if existing.GUID == ep.GUID {
			s.episodes[i] = ep
			s.sortByDateDesc()
			return nil
		}
	}

	return fmt.Errorf("episode with GUID %q not found", ep.GUID)
}

// Delete removes an episode by GUID.
func (s *Store) Delete(guid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existing := range s.episodes {
		if existing.GUID == guid {
			s.episodes = append(s.episodes[:i], s.episodes[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("episode with GUID %q not found", guid)
}

// GetByGUID returns a single episode by GUID.
func (s *Store) GetByGUID(guid string) (models.Episode, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ep := range s.episodes {
		if ep.GUID == guid {
			return ep, true
		}
	}
	return models.Episode{}, false
}

// Has checks if an episode exists by GUID.
func (s *Store) Has(guid string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ep := range s.episodes {
		if ep.GUID == guid {
			return true
		}
	}
	return false
}

func (s *Store) sortByDateDesc() {
	for i := 0; i < len(s.episodes); i++ {
		for j := i + 1; j < len(s.episodes); j++ {
			if s.episodes[i].PubDate.Before(s.episodes[j].PubDate) {
				s.episodes[i], s.episodes[j] = s.episodes[j], s.episodes[i]
			}
		}
	}
}

// FileToTitle converts a filename like "my-episode.mp3" to "My Episode".
func FileToTitle(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.TrimSpace(base)

	words := strings.Fields(base)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}

	return strings.Join(words, " ")
}

// GenerateGUID creates a GUID from filename and timestamp.
func GenerateGUID(filename string) string {
	timestamp := time.Now().UTC().Format("20060102-150405")
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	return fmt.Sprintf("%s-%s", base, timestamp)
}
