package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"neoncast/internal/models"
	"neoncast/internal/rss"
)

// Config holds server configuration.
type Config struct {
	Port       string
	StaticPath string
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	staticPath := os.Getenv("STATIC_PATH")
	if staticPath == "" {
		staticPath = "static"
	}

	return Config{
		Port:       port,
		StaticPath: staticPath,
	}
}

// Server wraps the Echo instance.
type Server struct {
	echo *echo.Echo
	cfg  Config
}

// New creates a new Server with routes registered.
func New(cfg Config) *Server {
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	s := &Server{
		echo: e,
		cfg:  cfg,
	}

	s.registerRoutes()

	return s
}

// registerRoutes sets up all HTTP routes.
func (s *Server) registerRoutes() {
	s.echo.GET("/health", s.handleHealth)
	s.echo.GET("/feed", s.handleFeed)
	s.echo.Static("/", s.cfg.StaticPath)
}

// handleHealth returns service health status.
func (s *Server) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (s *Server) handleFeed(c echo.Context) error {
	podcast := models.Podcast{
		Title:       "neoncast Demo Feed",
		Description: "Auto-generated demo podcast feed",
		Link:        "http://localhost:" + s.cfg.Port,
		Language:    "en-us",
		Author:      "neoncast",
		Explicit:    false,
	}

	// Demo episodes for Phase 2; later phases will wire real data
	episodes := []models.Episode{
		{
			Title:       "Welcome to neoncast",
			Description: "This is a demo episode.",
			GUID:        "demo-001",
			PubDate:     time.Now().Add(-24 * time.Hour),
			Duration:    600,
			FileURL:     "/episodes/demo-001.mp3",
			FileLength:  1234567,
			FileType:    "audio/mpeg",
			Explicit:    false,
		},
	}

	data, err := rss.Generate(podcast, episodes)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.Blob(http.StatusOK, "application/rss+xml; charset=utf-8", data)
}

// Start begins listening for requests.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%s", s.cfg.Port)
	return s.echo.Start(addr)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

// Echo exposes the underlying Echo instance for testing.
func (s *Server) Echo() *echo.Echo {
	return s.echo
}
