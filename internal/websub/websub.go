package websub

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Publisher sends WebSub publish notifications to configured hubs.
type Publisher struct {
	client  *http.Client
	hubs    []string
	feedURL string
}

// New creates a Publisher for the given hubs and feed URL.
// If hubs is empty, Publish becomes a no-op.
func New(hubs []string, feedURL string) *Publisher {
	return NewWithClient(hubs, feedURL, &http.Client{Timeout: 30 * time.Second})
}

// NewWithClient creates a Publisher with a custom HTTP client.
func NewWithClient(hubs []string, feedURL string, client *http.Client) *Publisher {
	filtered := make([]string, 0, len(hubs))
	for _, h := range hubs {
		if strings.TrimSpace(h) != "" {
			filtered = append(filtered, h)
		}
	}
	return &Publisher{
		client:  client,
		hubs:    filtered,
		feedURL: feedURL,
	}
}

// Publish notifies all configured hubs that the feed has updated.
// Errors from individual hubs are aggregated and returned.
func (p *Publisher) Publish(ctx context.Context) error {
	if len(p.hubs) == 0 || p.feedURL == "" {
		return nil
	}

	body := url.Values{}
	body.Set("hub.mode", "publish")
	body.Set("hub.url", p.feedURL)
	encoded := body.Encode()

	var wg sync.WaitGroup
	errCh := make(chan error, len(p.hubs))

	for _, hub := range p.hubs {
		wg.Add(1)
		go func(hub string) {
			defer wg.Done()
			if err := p.publishToHub(ctx, hub, encoded); err != nil {
				errCh <- err
			}
		}(hub)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (p *Publisher) publishToHub(ctx context.Context, hub, body string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hub, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("websub: create request for %s: %w", hub, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("websub: post to %s: %w", hub, err)
	}
	defer resp.Body.Close()

	// Read and discard body to keep connection alive.
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("websub: hub %s returned status %d", hub, resp.StatusCode)
	}
	return nil
}
