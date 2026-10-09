package main

// Opt-in transport fixture; notification persistence and read state still use
// the real disposable PostgreSQL/API. Never proxies mutations to production.
import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type fixtureTrackerRelay struct {
	mu      sync.Mutex
	clients map[*websocket.Conn][]string
}

func newFixtureTrackerRelay() *fixtureTrackerRelay {
	return &fixtureTrackerRelay{clients: make(map[*websocket.Conn][]string)}
}

func (f *fixtureTrackerRelay) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws/killlist" {
			f.socket(w, r)
			return
		}
		if r.URL.Path != "/__fixture/tracker-relay" {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "private, no-store")
		f.mu.Lock()
		clients := make([]*websocket.Conn, 0, len(f.clients))
		topics := [][]string{}
		for connection, subscriptions := range f.clients {
			clients = append(clients, connection)
			topics = append(topics, subscriptions)
		}
		f.mu.Unlock()
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"connections": len(clients), "topics": topics})
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		var event struct {
			Disconnect  bool            `json:"disconnect"`
			Killmail    json.RawMessage `json:"killmail"`
			RoutingKeys []string        `json:"routing_keys"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384)).Decode(&event) != nil {
			w.WriteHeader(400)
			return
		}
		for _, connection := range clients {
			if event.Disconnect {
				_ = connection.Close(websocket.StatusGoingAway, "Disposable reconnect test")
				continue
			}
			if len(event.Killmail) > 0 {
				keys := event.RoutingKeys
				if len(keys) == 0 {
					keys = []string{"victim.4242"}
				}
				f.mu.Lock()
				subscriptions := append([]string(nil), f.clients[connection]...)
				f.mu.Unlock()
				matches := false
				for _, topic := range subscriptions {
					if topic == "all" {
						matches = true
					}
					for _, key := range keys {
						if key == topic {
							matches = true
						}
					}
				}
				if !matches {
					continue
				}
				payload, _ := json.Marshal(map[string]any{"channel": "killlist", "data": map[string]any{"killmail": event.Killmail}, "routing_keys": keys})
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				_ = connection.Write(ctx, websocket.MessageText, payload)
				cancel()
			}
		}
		w.WriteHeader(204)
	})
}

func (f *fixtureTrackerRelay) socket(w http.ResponseWriter, r *http.Request) {
	connection, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*.localhost:*", "127.0.0.1:*", "localhost:*"}})
	if err != nil {
		return
	}
	f.mu.Lock()
	f.clients[connection] = nil
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		delete(f.clients, connection)
		f.mu.Unlock()
		_ = connection.Close(websocket.StatusNormalClosure, "")
	}()
	for {
		_, payload, err := connection.Read(r.Context())
		if err != nil {
			return
		}
		var message struct {
			Action string   `json:"action"`
			Topics []string `json:"topics"`
		}
		if json.Unmarshal(payload, &message) == nil && message.Action == "subscribe" {
			f.mu.Lock()
			f.clients[connection] = append([]string(nil), message.Topics...)
			f.mu.Unlock()
		}
	}
}
