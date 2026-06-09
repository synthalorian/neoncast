package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"neoncast/internal/analytics"
	"neoncast/internal/config"
	"neoncast/internal/ingest"
	"neoncast/internal/server"
	"neoncast/internal/store"
	"neoncast/internal/watcher"
)

func main() {
	fmt.Println("neoncast -- self-hosted podcast hosting")

	cfg := config.Default()

	// Ensure directories exist
	for _, dir := range []string{cfg.WatchPath, cfg.ContentPath, cfg.DataPath, cfg.StaticPath} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("create directory %s: %v", dir, err)
		}
	}

	// Initialize store
	st := store.New(cfg.DataPath)
	if err := st.Load(); err != nil {
		log.Fatalf("load store: %v", err)
	}

	an := analytics.New(cfg.DataPath)
	if err := an.Load(); err != nil {
		log.Fatalf("load analytics: %v", err)
	}

	// Create server
	srv := server.New(cfg, st, an)

	// Create watcher + ingest pipeline
	baseURL := fmt.Sprintf("http://localhost:%s", cfg.Port)
	pipeline := ingest.New(st, cfg.ContentPath, baseURL)
	w := watcher.New(cfg.WatchPath, pipeline)

	if err := w.Start(); err != nil {
		log.Fatalf("start watcher: %v", err)
	}

	// Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nshutting down...")

		if err := w.Stop(); err != nil {
			log.Printf("watcher stop error: %v", err)
		}

		if err := srv.Shutdown(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
		}
	}()

	addr := fmt.Sprintf("http://localhost:%s", cfg.Port)
	fmt.Printf("server listening on %s\n", addr)
	fmt.Printf("health check: %s/health\n", addr)
	fmt.Printf("feed: %s/feed\n", addr)
	fmt.Printf("admin dashboard: %s/admin\n", addr)
	fmt.Printf("watch directory: %s\n", cfg.WatchPath)
	fmt.Printf("content directory: %s\n", cfg.ContentPath)

	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
