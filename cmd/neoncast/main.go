package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"neoncast/internal/server"
)

func main() {
	fmt.Println("neoncast -- self-hosted podcast hosting")

	cfg := server.DefaultConfig()
	srv := server.New(cfg)

	// Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nshutting down server...")
		if err := srv.Shutdown(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
		}
	}()

	addr := fmt.Sprintf("http://localhost:%s", cfg.Port)
	fmt.Printf("server listening on %s\n", addr)
	fmt.Printf("health check: %s/health\n", addr)

	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
