package main

// Browser fixtures replace only the external GIF provider and WS relay.
// Auth, Markdown, moderation, comment writes, and reports use the real local API.
import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

//go:embed media/fixture.mp4
var fixtureGIFVideo []byte

type commentMediaFixture struct {
	mu       sync.Mutex
	clients  map[*websocket.Conn]bool
	comments map[string]map[string]any
}

func newCommentMediaFixture() *commentMediaFixture {
	return &commentMediaFixture{clients: make(map[*websocket.Conn]bool), comments: make(map[string]map[string]any)}
}

// Let the current renderer exercise media against the retained local database.
// Provider reads are deterministic; writes/Markdown/auth still use the real API.
func (f *commentMediaFixture) facade(next http.Handler) http.Handler {
	wrapped := f.handler(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || (r.URL.Path != "/api/comments/klipy/search" && r.URL.Path != "/api/comments/klipy/trending") {
			wrapped.ServeHTTP(w, r)
			return
		}
		request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://api.klipy.com/fixture", nil)
		if err != nil {
			http.Error(w, "Invalid fixture provider request", http.StatusBadRequest)
			return
		}
		request.URL.RawQuery = r.URL.Query().Encode()
		response, err := f.RoundTrip(request)
		if err != nil {
			http.Error(w, "Fixture provider unavailable", http.StatusServiceUnavailable)
			return
		}
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode != 200 {
			http.Error(w, "Fixture provider failure", response.StatusCode)
			return
		}
		var data struct {
			Data struct {
				Items   []map[string]any `json:"data"`
				Page    int              `json:"current_page"`
				HasNext bool             `json:"has_next"`
			} `json:"data"`
		}
		if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid fixture provider response", http.StatusBadGateway)
			return
		}
		encoded, err := json.Marshal(data.Data.Items)
		if err != nil {
			http.Error(w, "Invalid fixture media", http.StatusBadGateway)
			return
		}
		encoded = bytes.ReplaceAll(encoded, []byte("http://frontend-parity.localhost:4450/"), []byte("http://"+r.Host+"/"))
		if err := json.Unmarshal(encoded, &data.Data.Items); err != nil {
			http.Error(w, "Invalid fixture media", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "private, no-store")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": data.Data.Items, "page": data.Data.Page, "per_page": 24, "has_next": data.Data.HasNext})
	})
}
func (f *commentMediaFixture) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Hostname() != "api.klipy.com" {
		return http.DefaultTransport.RoundTrip(r)
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	if r.URL.Query().Get("q") == "provider-error" {
		return &http.Response{StatusCode: 503, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":"fixture provider failure"}`)), Request: r}, nil
	}
	items := []any{}
	for i := 1; i <= 24; i++ {
		id := (page-1)*24 + i
		origin := "http://frontend-parity.localhost:4450/__fixture/media/"
		thumbnail := map[string]any{"url": origin + fmt.Sprintf("%d.svg", id), "width": 128, "height": 128, "size": 256}
		video := map[string]any{"url": origin + "fixture.mp4", "width": 128, "height": 128, "size": len(fixtureGIFVideo)}
		size := map[string]any{"gif": thumbnail, "webp": thumbnail, "mp4": video}
		items = append(items, map[string]any{"id": id, "title": fmt.Sprintf("Fixture GIF %d", id), "slug": fmt.Sprintf("fixture-%d", id), "file": map[string]any{"hd": size, "md": size, "sm": size, "xs": size}})
	}
	body, _ := json.Marshal(map[string]any{"result": true, "data": map[string]any{"data": items, "current_page": page, "per_page": 24, "has_next": page < 2}})
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(body)), Request: r}, nil
}
func (f *commentMediaFixture) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws/comments" {
			f.socket(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/__fixture/media/") {
			if strings.HasSuffix(r.URL.Path, ".mp4") {
				http.ServeContent(w, r, "fixture.mp4", time.Time{}, bytes.NewReader(fixtureGIFVideo))
				return
			}
			w.Header().Set("Content-Type", "image/svg+xml")
			id, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/__fixture/media/"), ".svg"))
			_, _ = fmt.Fprintf(w, `<svg xmlns="http://www.w3.org/2000/svg" width="128" height="128"><rect width="128" height="128" fill="hsl(%d 45%% 35%%)"/><rect x="24" y="24" width="80" height="80" rx="12" fill="#fbbf24"/><text x="64" y="72" text-anchor="middle" font-size="24" fill="#111">%d</text></svg>`, id*17, id)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/api/comments") || strings.Contains(r.URL.Path, "/preview") || strings.Contains(r.URL.Path, "/klipy/") || strings.HasSuffix(r.URL.Path, "/report") {
			next.ServeHTTP(w, r)
			return
		}
		result := httptest.NewRecorder()
		next.ServeHTTP(result, r)
		for key, values := range result.Header() {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(result.Code)
		_, _ = w.Write(result.Body.Bytes())
		if result.Code < 200 || result.Code >= 300 {
			return
		}
		var data struct {
			Comment map[string]any   `json:"comment"`
			Roots   []map[string]any `json:"roots"`
			Replies []map[string]any `json:"replies"`
		}
		_ = json.Unmarshal(result.Body.Bytes(), &data)
		f.mu.Lock()
		for _, comment := range append(data.Roots, data.Replies...) {
			f.comments[fmt.Sprint(comment["id"])] = comment
		}
		row := data.Comment
		if row != nil {
			f.comments[fmt.Sprint(row["id"])] = row
		}
		if r.Method == http.MethodDelete {
			row = f.comments[strings.TrimPrefix(r.URL.Path, "/api/comments/")]
		}
		f.mu.Unlock()
		if row == nil || r.Method == http.MethodGet {
			return
		}
		eventType := "new"
		if r.Method == http.MethodPatch {
			eventType = "edited"
		}
		if r.Method == http.MethodDelete {
			eventType = "deleted"
		}
		event := map[string]any{"event_type": eventType, "comment_id": row["id"], "target_type": row["target_type"], "target_id": row["target_id"], "target_slug": row["target_slug"], "domain_id": row["domain_id"], "parent_id": row["parent_id"], "parent_character_id": nil, "parent_character_name": nil, "ancestors": []any{}}
		// Match buildCommentEventPayload: Markdown/moderation fields are not
		// broadcast by the production relay. Editors fetch the current detail.
		if eventType != "deleted" {
			minimal := make(map[string]any)
			for _, key := range []string{"id", "target_type", "target_id", "target_slug", "domain_id", "parent_id", "root_id", "depth", "body_html", "character_id", "character_name", "corporation_id", "corporation_name", "alliance_id", "alliance_name", "created_at", "edited_at"} {
				minimal[key] = row[key]
			}
			event["comment"] = minimal
		}
		payload, _ := json.Marshal(map[string]any{"channel": "comments", "data": event})
		f.mu.Lock()
		clients := make([]*websocket.Conn, 0, len(f.clients))
		for client := range f.clients {
			clients = append(clients, client)
		}
		f.mu.Unlock()
		for _, client := range clients {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = client.Write(ctx, websocket.MessageText, payload)
			cancel()
		}
	})
}
func (f *commentMediaFixture) socket(w http.ResponseWriter, r *http.Request) {
	connection, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"frontend-parity.localhost:*", "127.0.0.1:*", "localhost:*"}})
	if err != nil {
		return
	}
	f.mu.Lock()
	f.clients[connection] = true
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		delete(f.clients, connection)
		f.mu.Unlock()
		_ = connection.Close(websocket.StatusNormalClosure, "")
	}()
	for {
		if _, _, err = connection.Read(r.Context()); err != nil {
			return
		}
	}
}
