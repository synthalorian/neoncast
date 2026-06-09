package websub

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPublishNoHubs(t *testing.T) {
	pub := New(nil, "http://example.com/feed")
	if err := pub.Publish(context.Background()); err != nil {
		t.Fatalf("expected no error with no hubs, got %v", err)
	}
}

func TestPublishNoFeedURL(t *testing.T) {
	pub := New([]string{"https://hub.example.com/"}, "")
	if err := pub.Publish(context.Background()); err != nil {
		t.Fatalf("expected no error with empty feed URL, got %v", err)
	}
}

func TestPublishNotifiesHub(t *testing.T) {
	var called atomic.Bool
	var received url.Values

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Store(true)
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/x-www-form-urlencoded" {
			t.Errorf("expected form content-type, got %s", ct)
		}
		body, _ := io.ReadAll(r.Body)
		var err error
		received, err = url.ParseQuery(string(body))
		if err != nil {
			t.Errorf("parse body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	pub := New([]string{ts.URL}, "http://example.com/feed")
	if err := pub.Publish(context.Background()); err != nil {
		t.Fatalf("publish: %v", err)
	}

	if !called.Load() {
		t.Fatal("expected hub to be called")
	}
	if got := received.Get("hub.mode"); got != "publish" {
		t.Errorf("expected hub.mode=publish, got %q", got)
	}
	if got := received.Get("hub.url"); got != "http://example.com/feed" {
		t.Errorf("expected hub.url=http://example.com/feed, got %q", got)
	}
}

func TestPublishMultipleHubs(t *testing.T) {
	var count atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	pub := New([]string{ts.URL, ts.URL}, "http://example.com/feed")
	if err := pub.Publish(context.Background()); err != nil {
		t.Fatalf("publish: %v", err)
	}

	if got := count.Load(); got != 2 {
		t.Errorf("expected 2 hub calls, got %d", got)
	}
}

func TestPublishHubError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer ts.Close()

	pub := New([]string{ts.URL}, "http://example.com/feed")
	err := pub.Publish(context.Background())
	if err == nil {
		t.Fatal("expected error from failing hub")
	}
	if !strings.Contains(err.Error(), "returned status 400") {
		t.Errorf("expected status 400 in error, got %v", err)
	}
}

func TestPublishEmptyHubsFiltered(t *testing.T) {
	pub := New([]string{"", "   "}, "http://example.com/feed")
	if err := pub.Publish(context.Background()); err != nil {
		t.Fatalf("expected no error when all hubs are empty, got %v", err)
	}
}
