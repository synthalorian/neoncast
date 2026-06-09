package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"neoncast/internal/config"
	"neoncast/internal/models"
	"neoncast/internal/rss"
	"neoncast/internal/store"
)

// Server wraps the Echo instance and dependencies.
type Server struct {
	echo  *echo.Echo
	cfg   config.Config
	store *store.Store
}

// New creates a new Server with routes registered.
func New(cfg config.Config, st *store.Store) *Server {
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	s := &Server{
		echo:  e,
		cfg:   cfg,
		store: st,
	}

	s.registerRoutes()

	return s
}

// registerRoutes sets up all HTTP routes.
func (s *Server) registerRoutes() {
	s.echo.GET("/health", s.handleHealth)
	s.echo.GET("/feed", s.handleFeed)
	s.echo.Static("/episodes", s.cfg.ContentPath)
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
		Title:       "neoncast Feed",
		Description: "Auto-generated podcast feed",
		Link:        "http://localhost:" + s.cfg.Port,
		Language:    "en-us",
		Author:      "neoncast",
		Explicit:    false,
	}

	episodes := s.store.All()

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
