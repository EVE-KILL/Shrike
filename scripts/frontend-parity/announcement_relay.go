package main

// The public feed and its mutations use the real owned PostgreSQL API. Only
// the optional River/relay transport is replaced for deterministic comparisons.
import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/eve-kill/shrike/internal/api"
)

type fixtureAnnouncementRelay struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
}

func newFixtureAnnouncementRelay() *fixtureAnnouncementRelay {
	return &fixtureAnnouncementRelay{clients: make(map[*websocket.Conn]struct{})}
}

var fixtureAnnouncementMutation = regexp.MustCompile(`^/api/admin/announcements(?:/\d+(?:/archive)?)?$`)

func (f *fixtureAnnouncementRelay) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws/announcements" {
			f.socket(w, r)
			return
		}
		if r.URL.Path == "/__fixture/announcement-relay" {
			f.control(w, r)
			return
		}
		if !fixtureAnnouncementMutation.MatchString(r.URL.Path) || (r.Method != "POST" && r.Method != "PATCH" && r.Method != "DELETE") {
			next.ServeHTTP(w, r)
			return
		}
		recorder := httptest.NewRecorder()
		next.ServeHTTP(recorder, r)
		for name, values := range recorder.Header() {
			w.Header()[name] = values
		}
		w.WriteHeader(recorder.Code)
		_, _ = w.Write(recorder.Body.Bytes())
		if recorder.Code >= 300 {
			return
		}
		var data struct{ Announcement api.Announcement }
		if json.Unmarshal(recorder.Body.Bytes(), &data) != nil || data.Announcement.ID == 0 {
			return
		}
		event := "updated"
		if r.URL.Path == "/api/admin/announcements" {
			event = "new"
		}
		if strings.HasSuffix(r.URL.Path, "/archive") || r.Method == "DELETE" {
			event = "archived"
		}
		now := time.Now()
		if event != "archived" && (data.Announcement.StartsAt.After(now) || !data.Announcement.ExpiresAt.After(now)) {
			return
		}
		f.broadcast(false, event, data.Announcement)
	})
}
func (f *fixtureAnnouncementRelay) connections() []*websocket.Conn {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]*websocket.Conn, 0, len(f.clients))
	for c := range f.clients {
		result = append(result, c)
	}
	return result
}
func (f *fixtureAnnouncementRelay) broadcast(disconnect bool, event string, announcement api.Announcement) {
	payload, _ := json.Marshal(map[string]any{"channel": "announcements", "data": map[string]any{"event_type": event, "announcement": announcement}})
	for _, c := range f.connections() {
		if disconnect {
			_ = c.Close(websocket.StatusGoingAway, "Disposable reconnect test")
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = c.Write(ctx, websocket.MessageText, payload)
		cancel()
	}
}
func (f *fixtureAnnouncementRelay) control(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"connections": len(f.connections())})
		return
	}
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	var event struct {
		Disconnect   bool             `json:"disconnect"`
		Type         string           `json:"event_type"`
		Announcement api.Announcement `json:"announcement"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&event) != nil {
		w.WriteHeader(400)
		return
	}
	if !event.Disconnect && (event.Announcement.ID == 0 || (event.Type != "new" && event.Type != "updated" && event.Type != "expired" && event.Type != "archived")) {
		w.WriteHeader(400)
		return
	}
	f.broadcast(event.Disconnect, event.Type, event.Announcement)
	w.WriteHeader(204)
}
func (f *fixtureAnnouncementRelay) socket(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*.localhost:*", "localhost:*", "127.0.0.1:*"}})
	if err != nil {
		return
	}
	f.mu.Lock()
	f.clients[c] = struct{}{}
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		delete(f.clients, c)
		f.mu.Unlock()
		_ = c.Close(websocket.StatusNormalClosure, "")
	}()
	for {
		if _, _, err := c.Read(r.Context()); err != nil {
			return
		}
	}
}
