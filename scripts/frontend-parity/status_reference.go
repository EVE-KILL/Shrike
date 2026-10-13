package main

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/coder/websocket"
)

func statusReferenceRenderer(reference, renderer http.Handler, scripts map[string][]byte) http.Handler {
	return componentReferenceRenderer(reference, renderer, scripts, "status", regexp.MustCompile(`src="/assets/[^/"<>]+/status\.js"`), func(path string) bool { return path == "/status" }, []string{"status"}, "status.js")
}

// This relay is loopback-only and in-memory. It replays captured Go status DTOs
// to both versions, without closing or changing any upstream connection.
type fixtureStatusRelay struct {
	mu       sync.Mutex
	clients  map[*websocket.Conn]struct{}
	snapshot json.RawMessage
	blocked  bool
	accepted int
}

func newFixtureStatusRelay() *fixtureStatusRelay {
	return &fixtureStatusRelay{clients: make(map[*websocket.Conn]struct{})}
}

func (f *fixtureStatusRelay) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ws/status":
			f.socket(w, r)
		case "/__fixture/status-relay":
			f.control(w, r)
		default:
			next.ServeHTTP(w, r)
		}
	})
}

func (f *fixtureStatusRelay) state() ([]*websocket.Conn, json.RawMessage, bool, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	clients := make([]*websocket.Conn, 0, len(f.clients))
	for client := range f.clients {
		clients = append(clients, client)
	}
	return clients, append(json.RawMessage(nil), f.snapshot...), f.blocked, f.accepted
}

func writeStatusSnapshot(c *websocket.Conn, snapshot json.RawMessage) {
	if len(snapshot) == 0 {
		return
	}
	payload, _ := json.Marshal(struct {
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}{"status", snapshot})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = c.Write(ctx, websocket.MessageText, payload)
}

func (f *fixtureStatusRelay) control(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if r.Method == http.MethodGet {
		clients, snapshot, blocked, accepted := f.state()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"connections": len(clients), "accepted": accepted, "blocked": blocked, "hasSnapshot": len(snapshot) > 0})
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var control struct {
		Snapshot   json.RawMessage `json:"snapshot"`
		Disconnect bool            `json:"disconnect"`
		Blocked    *bool           `json:"blocked"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&control) != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(control.Snapshot) > 0 {
		var snapshot map[string]json.RawMessage
		if json.Unmarshal(control.Snapshot, &snapshot) != nil || snapshot == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	f.mu.Lock()
	if control.Blocked != nil {
		f.blocked = *control.Blocked
	}
	if len(control.Snapshot) > 0 {
		f.snapshot = append(json.RawMessage(nil), control.Snapshot...)
	}
	f.mu.Unlock()
	clients, snapshot, _, _ := f.state()
	for _, client := range clients {
		if control.Disconnect {
			_ = client.Close(websocket.StatusGoingAway, "Disposable reconnect check")
		} else if len(control.Snapshot) > 0 {
			writeStatusSnapshot(client, snapshot)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (f *fixtureStatusRelay) socket(w http.ResponseWriter, r *http.Request) {
	_, _, blocked, _ := f.state()
	if blocked {
		http.Error(w, "Disposable relay unavailable", http.StatusServiceUnavailable)
		return
	}
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*.localhost:*", "localhost:*", "127.0.0.1:*"}})
	if err != nil {
		return
	}
	f.mu.Lock()
	f.clients[c] = struct{}{}
	f.accepted++
	snapshot := append(json.RawMessage(nil), f.snapshot...)
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		delete(f.clients, c)
		f.mu.Unlock()
		_ = c.Close(websocket.StatusNormalClosure, "")
	}()
	writeStatusSnapshot(c, snapshot)
	for {
		if _, _, err := c.Read(r.Context()); err != nil {
			return
		}
	}
}
