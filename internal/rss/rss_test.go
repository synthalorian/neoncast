package rss

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"neoncast/internal/models"
)

func TestGenerateBasic(t *testing.T) {
	podcast := models.Podcast{
		Title:       "Test Podcast",
		Description: "A test podcast feed",
		Link:        "https://example.com",
		Language:    "en-us",
		Author:      "Test Author",
		Email:       "test@example.com",
		Copyright:   "2024 Test Author",
		ImageURL:    "https://example.com/cover.jpg",
		Category:    "Technology",
		Explicit:    false,
	}

	episodes := []models.Episode{
		{
			Title:       "Episode 1",
			Description: "The first episode",
			GUID:        "ep-001",
			PubDate:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Duration:    3661,
			FileURL:     "https://example.com/ep1.mp3",
			FileLength:  12345678,
			FileType:    "audio/mpeg",
			Explicit:    false,
			Season:      1,
			Episode:     1,
		},
	}

	data, err := Generate(podcast, episodes, nil, "")
	if err != nil {
		t.Fatalf("generate rss: %v", err)
	}

	output := string(data)

	// Check XML declaration
	if !strings.HasPrefix(output, xml.Header) {
		t.Error("expected XML declaration at start")
	}

	// Check RSS structure
	if !strings.Contains(output, `<rss xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd" version="2.0">`) {
		t.Error("expected rss root element with itunes namespace")
	}

	// Check channel elements
	if !strings.Contains(output, "<title>Test Podcast</title>") {
		t.Error("expected podcast title")
	}
	if !strings.Contains(output, "<description>A test podcast feed</description>") {
		t.Error("expected podcast description")
	}
	if !strings.Contains(output, "<link>https://example.com</link>") {
		t.Error("expected podcast link")
	}
	if !strings.Contains(output, "<language>en-us</language>") {
		t.Error("expected language")
	}
	if !strings.Contains(output, "<copyright>2024 Test Author</copyright>") {
		t.Error("expected copyright")
	}
	if !strings.Contains(output, "<ttl>60</ttl>") {
		t.Error("expected ttl=60")
	}

	// Check iTunes channel extensions
	if !strings.Contains(output, "<itunes:author>Test Author</itunes:author>") {
		t.Error("expected itunes:author")
	}
	if !strings.Contains(output, `<itunes:category text="Technology"></itunes:category>`) {
		t.Error("expected itunes:category")
	}
	if !strings.Contains(output, "<itunes:explicit>no</itunes:explicit>") {
		t.Error("expected itunes:explicit=no")
	}
	if !strings.Contains(output, `<itunes:image href="https://example.com/cover.jpg"></itunes:image>`) {
		t.Error("expected itunes:image")
	}
	if !strings.Contains(output, "<itunes:type>episodic</itunes:type>") {
		t.Error("expected itunes:type=episodic")
	}

	// Check iTunes owner
	if !strings.Contains(output, "<itunes:owner>") {
		t.Error("expected itunes:owner")
	}
	if !strings.Contains(output, "<itunes:name>Test Author</itunes:name>") {
		t.Error("expected itunes:name in owner")
	}
	if !strings.Contains(output, "<itunes:email>test@example.com</itunes:email>") {
		t.Error("expected itunes:email in owner")
	}

	// Check image element
	if !strings.Contains(output, "<image>") {
		t.Error("expected image element")
	}
	if !strings.Contains(output, "<url>https://example.com/cover.jpg</url>") {
		t.Error("expected image url")
	}

	// Check item elements
	if !strings.Contains(output, "<title>Episode 1</title>") {
		t.Error("expected episode title")
	}
	if !strings.Contains(output, "<description>The first episode</description>") {
		t.Error("expected episode description")
	}
	if !strings.Contains(output, `<guid isPermaLink="false">ep-001</guid>`) {
		t.Error("expected guid with isPermaLink=false")
	}
	if !strings.Contains(output, `<enclosure url="https://example.com/ep1.mp3" length="12345678" type="audio/mpeg"></enclosure>`) {
		t.Error("expected enclosure")
	}

	// Check iTunes item extensions
	if !strings.Contains(output, "<itunes:duration>1:01:01</itunes:duration>") {
		t.Error("expected itunes:duration in HH:MM:SS format")
	}
	if !strings.Contains(output, "<itunes:explicit>no</itunes:explicit>") {
		t.Error("expected item itunes:explicit")
	}
	if !strings.Contains(output, "<itunes:season>1</itunes:season>") {
		t.Error("expected itunes:season")
	}
	if !strings.Contains(output, "<itunes:episode>1</itunes:episode>") {
		t.Error("expected itunes:episode")
	}

	// Check pubDate format (RFC1123)
	if !strings.Contains(output, "Mon, 01 Jan 2024 00:00:00 UTC") {
		t.Error("expected RFC1123 formatted pubDate")
	}

	// Check duration format (HH:MM:SS for > 1 hour)
	if !strings.Contains(output, "<itunes:duration>1:01:01</itunes:duration>") {
		t.Error("expected duration in HH:MM:SS format")
	}
}

func TestGenerateExplicit(t *testing.T) {
	podcast := models.Podcast{
		Title:    "Explicit Podcast",
		Link:     "https://example.com",
		Explicit: true,
	}

	data, err := Generate(podcast, nil, nil, "")
	if err != nil {
		t.Fatalf("generate rss: %v", err)
	}

	output := string(data)
	if !strings.Contains(output, "<itunes:explicit>yes</itunes:explicit>") {
		t.Error("expected explicit=yes")
	}
}

func TestGenerateNoEpisodes(t *testing.T) {
	podcast := models.Podcast{
		Title:       "Empty Podcast",
		Description: "No episodes yet",
		Link:        "https://example.com",
	}

	data, err := Generate(podcast, nil, nil, "")
	if err != nil {
		t.Fatalf("generate rss: %v", err)
	}

	output := string(data)
	if !strings.Contains(output, "<title>Empty Podcast</title>") {
		t.Error("expected title")
	}
	if strings.Contains(output, "<item>") {
		t.Error("expected no items")
	}
}

func TestGenerateMinimalEpisode(t *testing.T) {
	podcast := models.Podcast{
		Title: "Minimal",
		Link:  "https://example.com",
	}

	episodes := []models.Episode{
		{
			Title:   "Minimal Episode",
			GUID:    "min-001",
			PubDate: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			Duration: 600,
		},
	}

	data, err := Generate(podcast, episodes, nil, "")
	if err != nil {
		t.Fatalf("generate rss: %v", err)
	}

	output := string(data)
	if !strings.Contains(output, "<title>Minimal Episode</title>") {
		t.Error("expected episode title")
	}
	// Should not have enclosure when FileURL is empty
	if strings.Contains(output, "<enclosure") {
		t.Error("expected no enclosure when FileURL is empty")
	}
	// Should not have season/episode when zero
	if strings.Contains(output, "<itunes:season>") {
		t.Error("expected no itunes:season when zero")
	}
	if strings.Contains(output, "<itunes:episode>") {
		t.Error("expected no itunes:episode when zero")
	}
	// Duration should be MM:SS format for < 1 hour
	if !strings.Contains(output, "<itunes:duration>10:00</itunes:duration>") {
		t.Error("expected duration in MM:SS format")
	}
}

func TestGenerateMultipleEpisodes(t *testing.T) {
	podcast := models.Podcast{
		Title: "Multi",
		Link:  "https://example.com",
	}

	episodes := []models.Episode{
		{
			Title:   "Episode 2",
			GUID:    "ep-002",
			PubDate: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Title:   "Episode 1",
			GUID:    "ep-001",
			PubDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	data, err := Generate(podcast, episodes, nil, "")
	if err != nil {
		t.Fatalf("generate rss: %v", err)
	}

	// Verify both episodes exist
	output := string(data)
	if !strings.Contains(output, "<title>Episode 2</title>") {
		t.Error("expected episode 2")
	}
	if !strings.Contains(output, "<title>Episode 1</title>") {
		t.Error("expected episode 1")
	}

	// Channel pubDate should be from the first episode
	if !strings.Contains(output, "<pubDate>Thu, 01 Feb 2024 00:00:00 UTC</pubDate>") {
		t.Error("expected channel pubDate from most recent episode")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0:00"},
		{30, "0:30"},
		{60, "1:00"},
		{90, "1:30"},
		{3661, "1:01:01"},
		{7200, "2:00:00"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.input)
		if got != tt.expected {
			t.Errorf("formatDuration(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestGenerateWebSubLinks(t *testing.T) {
	podcast := models.Podcast{
		Title: "WebSub Podcast",
		Link:  "https://example.com",
	}

	hubs := []string{"https://hub1.example.com/", "https://hub2.example.com/"}
	feedURL := "https://example.com/feed"

	data, err := Generate(podcast, nil, hubs, feedURL)
	if err != nil {
		t.Fatalf("generate rss: %v", err)
	}

	output := string(data)
	if !strings.Contains(output, `xmlns:atom="http://www.w3.org/2005/Atom"`) {
		t.Error("expected atom namespace")
	}
	if !strings.Contains(output, `<atom:link rel="self" type="application/rss+xml" href="https://example.com/feed"></atom:link>`) {
		t.Error("expected atom self link")
	}
	if !strings.Contains(output, `<atom:link rel="hub" href="https://hub1.example.com/"></atom:link>`) {
		t.Error("expected first hub link")
	}
	if !strings.Contains(output, `<atom:link rel="hub" href="https://hub2.example.com/"></atom:link>`) {
		t.Error("expected second hub link")
	}
}

func TestGenerateNoWebSubLinks(t *testing.T) {
	podcast := models.Podcast{
		Title: "Plain Podcast",
		Link:  "https://example.com",
	}

	data, err := Generate(podcast, nil, nil, "")
	if err != nil {
		t.Fatalf("generate rss: %v", err)
	}

	output := string(data)
	if strings.Contains(output, "atom:link") {
		t.Error("expected no atom links when hubs and feedURL are empty")
	}
	if strings.Contains(output, "xmlns:atom") {
		t.Error("expected no atom namespace when no atom links are present")
	}
}

func TestFormatDate(t *testing.T) {
	ts := time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC)
	got := formatDate(ts)
	want := "Sat, 15 Jun 2024 12:30:00 UTC"
	if got != want {
		t.Errorf("formatDate() = %q, want %q", got, want)
	}
}
