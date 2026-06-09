package watcher

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Handler is called when a new file is detected.
type Handler interface {
	Handle(path string)
}

// HandlerFunc adapts a function to the Handler interface.
type HandlerFunc func(path string)

func (f HandlerFunc) Handle(path string) { f(path) }

// Watcher monitors a directory for new MP3 files.
type Watcher struct {
	path      string
	handler   Handler
	watcher   *fsnotify.Watcher
	stop      chan struct{}
	wg        sync.WaitGroup
	seenFiles sync.Map // deduplication for rapid events
}

// New creates a Watcher for the given directory.
func New(watchPath string, handler Handler) *Watcher {
	return &Watcher{
		path:    watchPath,
		handler: handler,
		stop:    make(chan struct{}),
	}
}

// Start begins watching the directory. It also rescans existing files.
func (w *Watcher) Start() error {
	if err := os.MkdirAll(w.path, 0755); err != nil {
		return fmt.Errorf("create watch dir: %w", err)
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create fsnotify watcher: %w", err)
	}
	w.watcher = fsw

	if err := fsw.Add(w.path); err != nil {
		return fmt.Errorf("watch path: %w", err)
	}

	// Process existing files first
	if err := w.scanExisting(); err != nil {
		log.Printf("scan existing files: %v", err)
	}

	w.wg.Add(1)
	go w.loop()

	log.Printf("watching directory: %s", w.path)
	return nil
}

// Stop shuts down the watcher gracefully.
func (w *Watcher) Stop() error {
	close(w.stop)
	if w.watcher != nil {
		w.watcher.Close()
	}
	w.wg.Wait()
	return nil
}

func (w *Watcher) loop() {
	defer w.wg.Done()

	for {
		select {
		case <-w.stop:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				w.handlePath(event.Name)
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				w.handlePath(event.Name)
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

func (w *Watcher) handlePath(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}

	if info.IsDir() {
		// Watch new subdirectories
		_ = w.watcher.Add(path)
		_ = w.scanDir(path)
		return
	}

	if !isAudioFile(path) {
		return
	}

	// Deduplicate rapid events for the same file
	if _, loaded := w.seenFiles.LoadOrStore(path, true); loaded {
		return
	}

	w.handler.Handle(path)
}

func (w *Watcher) scanExisting() error {
	return w.scanDir(w.path)
}

func (w *Watcher) scanDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			_ = w.watcher.Add(path)
			_ = w.scanDir(path)
			continue
		}
		if isAudioFile(path) {
			w.handler.Handle(path)
		}
	}

	return nil
}

func isAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".mp3" || ext == ".m4a" || ext == ".ogg" || ext == ".wav"
}
