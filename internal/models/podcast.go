package models

import "time"

// Podcast represents a podcast feed channel.
type Podcast struct {
	Title       string
	Description string
	Link        string
	Language    string
	Author      string
	Email       string
	Copyright   string
	ImageURL    string
	Category    string
	Explicit    bool
}

// Episode represents a single podcast episode (RSS item).
type Episode struct {
	Title       string
	Description string
	GUID        string
	PubDate     time.Time
	Duration    int // seconds
	FileURL     string
	FileLength  int64
	FileType    string
	Explicit    bool
	Season      int
	Episode     int
}
