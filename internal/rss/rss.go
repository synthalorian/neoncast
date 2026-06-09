package rss

import (
	"encoding/xml"
	"fmt"
	"time"

	"neoncast/internal/models"
)

const (
	itunesNS = "http://www.itunes.com/dtds/podcast-1.0.dtd"
	atomNS   = "http://www.w3.org/2005/Atom"
)

// RSS is the root rss element.
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	NS      string   `xml:"xmlns:itunes,attr"`
	AtomNS  string   `xml:"xmlns:atom,attr,omitempty"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

// AtomLink represents an Atom link element for self/hub discovery.
type AtomLink struct {
	XMLName xml.Name `xml:"atom:link"`
	Rel     string   `xml:"rel,attr"`
	Type    string   `xml:"type,attr,omitempty"`
	Href    string   `xml:"href,attr"`
}

// Channel represents the podcast channel metadata.
type Channel struct {
	Title          string          `xml:"title"`
	Description    string          `xml:"description"`
	Link           string          `xml:"link"`
	AtomLinks      []AtomLink      `xml:"atom:link"`
	Language       string          `xml:"language,omitempty"`
	Copyright      string          `xml:"copyright,omitempty"`
	LastBuildDate  string          `xml:"lastBuildDate"`
	PubDate        string          `xml:"pubDate,omitempty"`
	TTL            int             `xml:"ttl,omitempty"`
	Image          *Image          `xml:"image,omitempty"`
	ITunesAuthor   string          `xml:"itunes:author,omitempty"`
	ITunesCategory *ITunesCategory `xml:"itunes:category,omitempty"`
	ITunesExplicit string          `xml:"itunes:explicit"`
	ITunesImage    *ITunesImage    `xml:"itunes:image,omitempty"`
	ITunesOwner    *ITunesOwner    `xml:"itunes:owner,omitempty"`
	ITunesType     string          `xml:"itunes:type,omitempty"`
	Items          []Item          `xml:"item"`
}

// Image is the standard RSS image element.
type Image struct {
	URL   string `xml:"url"`
	Title string `xml:"title"`
	Link  string `xml:"link"`
}

// ITunesOwner represents the iTunes owner element.
type ITunesOwner struct {
	Name  string `xml:"itunes:name"`
	Email string `xml:"itunes:email"`
}

// ITunesImage represents the iTunes image element.
type ITunesImage struct {
	Href string `xml:"href,attr"`
}

// ITunesCategory represents the iTunes category element.
type ITunesCategory struct {
	Text string `xml:"text,attr"`
}

// Item represents a single episode in the RSS feed.
type Item struct {
	Title          string       `xml:"title"`
	Description    string       `xml:"description"`
	Link           string       `xml:"link,omitempty"`
	GUID           *GUID        `xml:"guid"`
	PubDate        string       `xml:"pubDate"`
	Enclosure      *Enclosure   `xml:"enclosure"`
	ITunesDuration string       `xml:"itunes:duration,omitempty"`
	ITunesExplicit string       `xml:"itunes:explicit,omitempty"`
	ITunesImage    *ITunesImage `xml:"itunes:image,omitempty"`
	ITunesSeason   int          `xml:"itunes:season,omitempty"`
	ITunesEpisode  int          `xml:"itunes:episode,omitempty"`
}

// GUID represents the unique identifier for an item.
type GUID struct {
	Value       string `xml:",chardata"`
	IsPermaLink bool   `xml:"isPermaLink,attr"`
}

// Enclosure represents the media file attachment.
type Enclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

func boolToExplicit(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func formatDate(t time.Time) string {
	return t.UTC().Format(time.RFC1123)
}

// Generate creates an RSS 2.0 feed with iTunes extensions from podcast and episode data.
// hubURLs are advertised as WebSub hubs via atom:link elements; feedURL is advertised as the self link.
func Generate(podcast models.Podcast, episodes []models.Episode, hubURLs []string, feedURL string) ([]byte, error) {
	now := time.Now()

	ch := Channel{
		Title:          podcast.Title,
		Description:    podcast.Description,
		Link:           podcast.Link,
		Language:       podcast.Language,
		Copyright:      podcast.Copyright,
		LastBuildDate:  formatDate(now),
		TTL:            60,
		ITunesAuthor:   podcast.Author,
		ITunesExplicit: boolToExplicit(podcast.Explicit),
		ITunesType:     "episodic",
	}

	if feedURL != "" {
		ch.AtomLinks = append(ch.AtomLinks, AtomLink{
			Rel:  "self",
			Type: "application/rss+xml",
			Href: feedURL,
		})
	}
	for _, hub := range hubURLs {
		if hub != "" {
			ch.AtomLinks = append(ch.AtomLinks, AtomLink{Rel: "hub", Href: hub})
		}
	}

	if podcast.ImageURL != "" {
		ch.Image = &Image{
			URL:   podcast.ImageURL,
			Title: podcast.Title,
			Link:  podcast.Link,
		}
		ch.ITunesImage = &ITunesImage{Href: podcast.ImageURL}
	}

	if podcast.Category != "" {
		ch.ITunesCategory = &ITunesCategory{Text: podcast.Category}
	}

	if podcast.Author != "" || podcast.Email != "" {
		ch.ITunesOwner = &ITunesOwner{
			Name:  podcast.Author,
			Email: podcast.Email,
		}
	}

	if len(episodes) > 0 {
		// Use the most recent episode's pubDate as channel pubDate
		ch.PubDate = formatDate(episodes[0].PubDate)
	}

	for _, ep := range episodes {
		item := Item{
			Title:       ep.Title,
			Description: ep.Description,
			GUID: &GUID{
				Value:       ep.GUID,
				IsPermaLink: false,
			},
			PubDate:        formatDate(ep.PubDate),
			ITunesDuration: formatDuration(ep.Duration),
			ITunesExplicit: boolToExplicit(ep.Explicit),
		}

		if ep.FileURL != "" {
			item.Enclosure = &Enclosure{
				URL:    ep.FileURL,
				Length: ep.FileLength,
				Type:   ep.FileType,
			}
		}

		if ep.Season > 0 {
			item.ITunesSeason = ep.Season
		}
		if ep.Episode > 0 {
			item.ITunesEpisode = ep.Episode
		}

		ch.Items = append(ch.Items, item)
	}

	rss := RSS{
		NS:      itunesNS,
		Version: "2.0",
		Channel: ch,
	}
	if len(ch.AtomLinks) > 0 {
		rss.AtomNS = atomNS
	}

	// Marshal with indentation for readability
	output, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal rss: %w", err)
	}

	// Prepend XML declaration
	header := []byte(xml.Header)
	return append(header, output...), nil
}
