package main

// Pure transport fixture shared by both renderers. Submission POSTs are handled
// here, before either proxy, and never reach a database or a job dispatcher.
import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var fixtureSubmissionLink = regexp.MustCompile(`(?i)https?://esi\.evetech\.net/(?:latest/|v[0-9]+/)?killmails/([0-9]+)/[a-f0-9]{20,64}/?`)

type fixtureSubmissionConfig struct {
	Existing []int64         `json:"existing"`
	Ready    []int64         `json:"ready"`
	Reject   bool            `json:"reject"`
	Detail   json.RawMessage `json:"detail,omitempty"`
}

type fixtureSubmissions struct {
	mu     sync.Mutex
	config fixtureSubmissionConfig
	calls  int
	queued []int64
}

func newFixtureSubmissions() *fixtureSubmissions { return &fixtureSubmissions{} }

func fixtureHasID(ids []int64, id int64) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

func (f *fixtureSubmissions) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path != "/__fixture/submissions" && path != "/api/killmail/post" && !strings.HasPrefix(path, "/api/killmail/") {
			next.ServeHTTP(w, r)
			return
		}
		// Only recognized synthetic IDs are intercepted for detail/existence.
		idPath := strings.TrimPrefix(path, "/api/killmail/")
		existsPath := strings.HasSuffix(idPath, "/exists")
		id, _ := strconv.ParseInt(strings.TrimSuffix(idPath, "/exists"), 10, 64)
		f.mu.Lock()
		config := f.config
		known := fixtureHasID(f.queued, id) || fixtureHasID(config.Existing, id) || fixtureHasID(config.Ready, id)
		f.mu.Unlock()
		if path != "/__fixture/submissions" && path != "/api/killmail/post" && (!known || id <= 0 || r.Method != http.MethodGet) {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "private, no-store")
		if path == "/__fixture/submissions" {
			if r.Method == http.MethodGet {
				f.mu.Lock()
				defer f.mu.Unlock()
				_ = json.NewEncoder(w).Encode(map[string]any{"calls": f.calls, "queued": f.queued, "config": f.config})
				return
			}
			if r.Method != http.MethodPost {
				w.WriteHeader(405)
				return
			}
			var value fixtureSubmissionConfig
			if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384)).Decode(&value) != nil {
				http.Error(w, "Invalid submission fixture", 400)
				return
			}
			for _, ids := range [][]int64{value.Existing, value.Ready} {
				for _, id := range ids {
					if id <= 0 || id > 9007199254740991 {
						http.Error(w, "Invalid fixture ID", 400)
						return
					}
				}
			}
			if len(value.Detail) > 0 && (!json.Valid(value.Detail) || value.Detail[0] != '{') {
				http.Error(w, "Invalid fixture detail", 400)
				return
			}
			f.mu.Lock()
			f.config = value
			f.calls = 0
			f.queued = nil
			f.mu.Unlock()
			w.WriteHeader(204)
			return
		}
		if path == "/api/killmail/post" {
			if r.Method != http.MethodPost {
				w.WriteHeader(405)
				return
			}
			var value struct {
				Text string `json:"text"`
			}
			if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&value) != nil || strings.TrimSpace(value.Text) == "" {
				http.Error(w, "No text provided", 400)
				return
			}
			ids, existing, queued := []int64{}, []int64{}, []int64{}
			for _, match := range fixtureSubmissionLink.FindAllStringSubmatch(value.Text, -1) {
				id, err := strconv.ParseInt(match[1], 10, 64)
				if err != nil || id <= 0 || id > 9007199254740991 || fixtureHasID(ids, id) {
					continue
				}
				ids = append(ids, id)
				if fixtureHasID(config.Existing, id) {
					existing = append(existing, id)
				} else if !config.Reject {
					queued = append(queued, id)
				}
			}
			f.mu.Lock()
			f.calls++
			f.queued = append(f.queued, queued...)
			f.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"total": len(ids), "accepted": len(queued), "existing": len(existing), "rejected": len(ids) - len(existing) - len(queued), "killmails": ids, "existingIds": existing, "message": "No valid ESI killmail links found"})
			return
		}
		ready := fixtureHasID(config.Ready, id) || fixtureHasID(config.Existing, id)
		if existsPath {
			_ = json.NewEncoder(w).Encode(map[string]bool{"exists": ready})
			return
		}
		if !ready {
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"error":"Fixture killmail is pending"}`))
			return
		}
		if len(config.Detail) > 0 {
			_, _ = w.Write(config.Detail)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"killmail_id": id, "total_value": 2340000000, "victim": map[string]any{"ship_type_id": 587, "ship_name": "Rifter", "character_name": "Fixture Pilot"}})
	})
}
