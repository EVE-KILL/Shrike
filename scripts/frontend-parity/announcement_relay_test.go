package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/eve-kill/shrike/internal/api"
)

func TestAnnouncementFixtureRelaysSuccessfulRealAPIResponseWithCanonicalMetadata(t *testing.T) {
	f := newFixtureAnnouncementRelay()
	now := time.Now().UTC()
	a := api.Announcement{ID: 12, Tier: 1, Title: "Fixture", StartsAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	server := httptest.NewServer(f.wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		_ = json.NewEncoder(w).Encode(map[string]any{"announcement": a})
	})))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws/announcements", &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://announcement-parity.localhost:4451"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	for len(f.connections()) != 1 {
		select {
		case <-ctx.Done():
			t.Fatal("fixture connection did not register")
		case <-time.After(time.Millisecond):
		}
	}
	for _, check := range []struct{ Method, Path, Event string }{{"POST", "/api/admin/announcements", "new"}, {"PATCH", "/api/admin/announcements/12", "updated"}, {"POST", "/api/admin/announcements/12/archive", "archived"}} {
		req, _ := http.NewRequestWithContext(ctx, check.Method, server.URL+check.Path, nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 200 || res.Header.Get("Cache-Control") != "private, no-store" {
			t.Fatal("fixture changed real API result")
		}
		_, raw, err := c.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var event struct {
			Channel string
			Data    struct {
				Type         string `json:"event_type"`
				Announcement api.Announcement
			}
		}
		if json.Unmarshal(raw, &event) != nil || event.Channel != "announcements" || event.Data.Type != check.Event || !event.Data.Announcement.CreatedAt.Equal(now) {
			t.Fatal("fixture lifecycle envelope diverged")
		}
	}
	w := httptest.NewRecorder()
	f.control(w, httptest.NewRequest("POST", "/__fixture/announcement-relay", strings.NewReader(`{"event_type":"invalid","announcement":{"id":1}}`)))
	if w.Code != 400 {
		t.Fatal("fixture accepted invalid lifecycle event")
	}
}
