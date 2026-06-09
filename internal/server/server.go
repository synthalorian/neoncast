package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"neoncast/internal/analytics"
	"neoncast/internal/config"
	"neoncast/internal/models"
	"neoncast/internal/rss"
	"neoncast/internal/store"
)

type publisher interface {
	Publish(ctx context.Context) error
}

// Server wraps the Echo instance and dependencies.
type Server struct {
	echo      *echo.Echo
	cfg       config.Config
	store     *store.Store
	analytics *analytics.Analytics
	publisher publisher
}

// New creates a new Server with routes registered.
func New(cfg config.Config, st *store.Store, an *analytics.Analytics, pub publisher) *Server {
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.CORS())

	s := &Server{
		echo:      e,
		cfg:       cfg,
		store:     st,
		analytics: an,
		publisher: pub,
	}

	s.registerRoutes()

	return s
}

// registerRoutes sets up all HTTP routes.
func (s *Server) registerRoutes() {
	s.echo.GET("/health", s.handleHealth)
	s.echo.GET("/feed", s.handleFeed)

	if s.analytics != nil {
		s.echo.Use(s.analytics.Track(s.store))
	}
	s.echo.Static("/episodes", s.cfg.ContentPath)

	s.echo.GET("/api/episodes", s.handleListEpisodes)
	s.echo.GET("/api/episodes/:guid", s.handleGetEpisode)
	s.echo.PUT("/api/episodes/:guid", s.handleUpdateEpisode)
	s.echo.DELETE("/api/episodes/:guid", s.handleDeleteEpisode)
	s.echo.POST("/api/upload", s.handleUpload)
	s.echo.GET("/api/podcast", s.handleGetPodcast)
	s.echo.PUT("/api/podcast", s.handleUpdatePodcast)

	s.echo.GET("/api/analytics/downloads", s.handleAnalyticsDownloads)
	s.echo.GET("/api/analytics/summary", s.handleAnalyticsSummary)
	s.echo.GET("/api/analytics/recent", s.handleAnalyticsRecent)

	// Admin dashboard
	s.echo.File("/admin", filepath.Join(s.cfg.StaticPath, "dashboard.html"))
	s.echo.GET("/admin/", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/admin")
	})

	// Static files (fallback)
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
		Title:       s.cfg.Podcast.Title,
		Description: s.cfg.Podcast.Description,
		Link:        s.cfg.BaseURL,
		Language:    s.cfg.Podcast.Language,
		Author:      s.cfg.Podcast.Author,
		Email:       s.cfg.Podcast.Email,
		Copyright:   s.cfg.Podcast.Copyright,
		ImageURL:    s.cfg.Podcast.ImageURL,
		Category:    s.cfg.Podcast.Category,
		Explicit:    s.cfg.Podcast.Explicit,
	}

	episodes := s.store.All()
	feedURL := s.cfg.BaseURL + "/feed"

	data, err := rss.Generate(podcast, episodes, s.cfg.HubURLs, feedURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.Blob(http.StatusOK, "application/rss+xml; charset=utf-8", data)
}

// handleListEpisodes returns all episodes.
func (s *Server) handleListEpisodes(c echo.Context) error {
	return c.JSON(http.StatusOK, s.store.All())
}

// handleGetEpisode returns a single episode by GUID.
func (s *Server) handleGetEpisode(c echo.Context) error {
	guid := c.Param("guid")
	ep, ok := s.store.GetByGUID(guid)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "episode not found",
		})
	}
	return c.JSON(http.StatusOK, ep)
}

// updateEpisodeRequest represents the fields that can be updated.
type updateEpisodeRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Duration    int    `json:"duration"`
	Explicit    bool   `json:"explicit"`
	Season      int    `json:"season"`
	Episode     int    `json:"episode"`
}

// handleUpdateEpisode updates episode metadata.
func (s *Server) handleUpdateEpisode(c echo.Context) error {
	guid := c.Param("guid")
	ep, ok := s.store.GetByGUID(guid)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "episode not found",
		})
	}

	var req updateEpisodeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	ep.Title = req.Title
	ep.Description = req.Description
	ep.Duration = req.Duration
	ep.Explicit = req.Explicit
	ep.Season = req.Season
	ep.Episode = req.Episode

	if err := s.store.Update(ep); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	if err := s.store.Save(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	s.notifyHubs()
	return c.JSON(http.StatusOK, ep)
}

// handleDeleteEpisode removes an episode.
func (s *Server) handleDeleteEpisode(c echo.Context) error {
	guid := c.Param("guid")

	if err := s.store.Delete(guid); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}

	if err := s.store.Save(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	s.notifyHubs()
	return c.NoContent(http.StatusNoContent)
}

// handleUpload accepts an audio file upload and creates an episode.
func (s *Server) handleUpload(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "missing file field",
		})
	}

	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "open uploaded file: " + err.Error(),
		})
	}
	defer src.Close()

	filename := filepath.Base(file.Filename)
	destPath := filepath.Join(s.cfg.ContentPath, filename)

	// Avoid overwriting existing files
	if _, err := os.Stat(destPath); err == nil {
		filename = fmt.Sprintf("%d-%s", time.Now().Unix(), filename)
		destPath = filepath.Join(s.cfg.ContentPath, filename)
	}

	if err := os.MkdirAll(s.cfg.ContentPath, 0755); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "create content dir: " + err.Error(),
		})
	}

	dst, err := os.Create(destPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "create destination file: " + err.Error(),
		})
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "copy file: " + err.Error(),
		})
	}

	ext := strings.ToLower(filepath.Ext(filename))
	mimeType := "audio/mpeg"
	switch ext {
	case ".m4a":
		mimeType = "audio/mp4"
	case ".ogg":
		mimeType = "audio/ogg"
	case ".wav":
		mimeType = "audio/wav"
	}

	info, err := os.Stat(destPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "stat file: " + err.Error(),
		})
	}

	guid := store.GenerateGUID(filename)
	ep := models.Episode{
		Title:      store.FileToTitle(filename),
		GUID:       guid,
		PubDate:    time.Now(),
		Duration:   0,
		FileURL:    fmt.Sprintf("%s/episodes/%s", s.cfg.BaseURL, filename),
		FileLength: info.Size(),
		FileType:   mimeType,
		Explicit:   false,
	}

	if err := s.store.Add(ep); err != nil {
		return c.JSON(http.StatusConflict, map[string]string{
			"error": err.Error(),
		})
	}

	if err := s.store.Save(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	s.notifyHubs()
	return c.JSON(http.StatusCreated, ep)
}

// handleGetPodcast returns the podcast configuration.
func (s *Server) handleGetPodcast(c echo.Context) error {
	return c.JSON(http.StatusOK, s.cfg.Podcast)
}

// updatePodcastRequest represents the podcast fields that can be updated.
type updatePodcastRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Email       string `json:"email"`
	Copyright   string `json:"copyright"`
	ImageURL    string `json:"image_url"`
	Category    string `json:"category"`
	Explicit    bool   `json:"explicit"`
	Language    string `json:"language"`
}

// handleUpdatePodcast updates podcast metadata.
func (s *Server) handleUpdatePodcast(c echo.Context) error {
	var req updatePodcastRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	s.cfg.Podcast.Title = req.Title
	s.cfg.Podcast.Description = req.Description
	s.cfg.Podcast.Author = req.Author
	s.cfg.Podcast.Email = req.Email
	s.cfg.Podcast.Copyright = req.Copyright
	s.cfg.Podcast.ImageURL = req.ImageURL
	s.cfg.Podcast.Category = req.Category
	s.cfg.Podcast.Explicit = req.Explicit
	s.cfg.Podcast.Language = req.Language

	s.notifyHubs()
	return c.JSON(http.StatusOK, s.cfg.Podcast)
}

func (s *Server) notifyHubs() {
	if s.publisher == nil {
		return
	}
	go s.publisher.Publish(context.Background())
}

func (s *Server) handleAnalyticsDownloads(c echo.Context) error {
	if s.analytics == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "analytics not available",
		})
	}

	summary := s.analytics.Summary()
	return c.JSON(http.StatusOK, summary.ByEpisode)
}

func (s *Server) handleAnalyticsSummary(c echo.Context) error {
	if s.analytics == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "analytics not available",
		})
	}

	return c.JSON(http.StatusOK, s.analytics.Summary())
}

func (s *Server) handleAnalyticsRecent(c echo.Context) error {
	if s.analytics == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "analytics not available",
		})
	}

	limit := 50
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	return c.JSON(http.StatusOK, s.analytics.Recent(limit))
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
