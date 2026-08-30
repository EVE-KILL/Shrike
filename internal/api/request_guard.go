package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// RequestGuard enforces the public HTTP etiquette documented by the API.
//
// It deliberately wraps /api only. Edge-layer infrastructure owns traffic
// shaping and rate limiting; the origin only requires clients to identify
// themselves so operators can contact a misbehaving integration.
type RequestGuard struct{}

// NewRequestGuard returns the production API request policy.
func NewRequestGuard() *RequestGuard {
	return &RequestGuard{}
}

// Wrap applies the guard without changing handlers for the other same-origin
// namespaces.
func (g *RequestGuard) Wrap(next http.Handler) http.Handler {
	if g == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api" && !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		// The guard can reject a request before it reaches crossOriginAPI.
		// Apply the same public API policy so browser clients can inspect the
		// error response.
		setCrossOriginAPIHeaders(w.Header())
		if r.Method != http.MethodOptions &&
			strings.TrimSpace(r.Header.Get("User-Agent")) == "" {
			writeRequestGuardError(
				w,
				http.StatusBadRequest,
				"An identifying User-Agent header is required",
			)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeRequestGuardError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
