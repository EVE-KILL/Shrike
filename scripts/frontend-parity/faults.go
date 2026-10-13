package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Opt-in, loopback-only comparison fault injection. Rules affect only this
// disposable comparison facade; they never change the upstream service.
type fixtureFaultRule struct {
	Path    string            `json:"path"`
	Method  string            `json:"method,omitempty"`
	Query   map[string]string `json:"query"`
	Status  int               `json:"status"`
	DelayMS int               `json:"delayMs"`
	// Response replaces an external handoff in local browser tests. It is
	// restricted to reads and never reaches the upstream application.
	Response json.RawMessage `json:"response,omitempty"`
}

type fixtureFaults struct {
	enabled bool
	mu      sync.RWMutex
	rule    fixtureFaultRule
}

func (f *fixtureFaults) wrap(next http.Handler) http.Handler {
	if !f.enabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__fixture/faults" {
			w.Header().Set("Cache-Control", "private, no-store")
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			var rule fixtureFaultRule
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&rule); err != nil ||
				(rule.Path != "" && ((!strings.HasPrefix(rule.Path, "/api/") && !strings.HasPrefix(rule.Path, "/_fragments/")) || strings.ContainsAny(rule.Path, "?#\r\n"))) ||
				(rule.Method != "" && rule.Method != http.MethodGet && rule.Method != http.MethodPut && rule.Method != http.MethodPost && rule.Method != http.MethodPatch && rule.Method != http.MethodDelete) ||
				(len(rule.Response) > 0 && (!json.Valid(rule.Response) || (rule.Method != "" && rule.Method != http.MethodGet))) ||
				(rule.Status != 0 && (rule.Status < 400 || rule.Status > 599)) || rule.DelayMS < 0 || rule.DelayMS > 10000 {
				http.Error(w, "Invalid fixture fault", http.StatusBadRequest)
				return
			}
			f.mu.Lock()
			f.rule = rule
			f.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
			return
		}
		f.mu.RLock()
		rule := f.rule
		f.mu.RUnlock()
		method := rule.Method
		if method == "" {
			method = http.MethodGet
		}
		match := r.Method == method && rule.Path != "" && r.URL.Path == rule.Path
		for key, value := range rule.Query {
			if r.URL.Query().Get(key) != value {
				match = false
			}
		}
		if match {
			if rule.DelayMS > 0 {
				timer := time.NewTimer(time.Duration(rule.DelayMS) * time.Millisecond)
				defer timer.Stop()
				select {
				case <-r.Context().Done():
					return
				case <-timer.C:
				}
			}
			if len(rule.Response) > 0 {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Cache-Control", "private, no-store")
				_, _ = w.Write(rule.Response)
				return
			}
			if rule.Status != 0 {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Cache-Control", "private, no-store")
				w.WriteHeader(rule.Status)
				_, _ = w.Write([]byte(`{"error":"Injected comparison API failure"}`))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
