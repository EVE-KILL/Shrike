package api

import (
	"encoding/json"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultAPIRequestsPerMinute = 600
	maximumRateLimitClients     = 50_000
)

type rateLimitWindow struct {
	used  int
	reset time.Time
}

// RequestGuard enforces the public HTTP etiquette documented by the API.
//
// It deliberately wraps /api only. Images have a different traffic shape and
// are intended to be absorbed by Cloudflare, browser auth needs independent
// abuse controls, and probes must never become unhealthy because a client on
// the same address queried the API heavily.
type RequestGuard struct {
	mu      sync.Mutex
	clients map[string]rateLimitWindow
	limit   int
	window  time.Duration
	now     func() time.Time
}

// NewRequestGuard returns the production API request policy.
func NewRequestGuard() *RequestGuard {
	return newRequestGuard(
		defaultAPIRequestsPerMinute,
		time.Minute,
		time.Now,
	)
}

func newRequestGuard(
	limit int,
	window time.Duration,
	now func() time.Time,
) *RequestGuard {
	return &RequestGuard{
		clients: make(map[string]rateLimitWindow),
		limit:   max(1, limit),
		window:  max(time.Second, window),
		now:     now,
	}
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
		// error and its rate-limit headers.
		setCrossOriginAPIHeaders(w.Header())
		allowed, remaining, reset := g.allow(clientAddress(r))
		setRateLimitHeaders(w.Header(), g.limit, remaining, reset)
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(reset))
			writeRequestGuardError(
				w,
				http.StatusTooManyRequests,
				"API rate limit exceeded",
			)
			return
		}
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

func (g *RequestGuard) allow(client string) (bool, int, int) {
	now := g.now()
	g.mu.Lock()
	defer g.mu.Unlock()

	current, exists := g.clients[client]
	if !exists && len(g.clients) >= maximumRateLimitClients {
		g.removeExpiredLocked(now)
		if len(g.clients) >= maximumRateLimitClients {
			client = "overflow"
			current, exists = g.clients[client]
		}
	}
	if !exists || !current.reset.After(now) {
		current = rateLimitWindow{reset: now.Add(g.window)}
	}
	current.used++
	g.clients[client] = current

	remaining := max(0, g.limit-current.used)
	reset := max(1, int(math.Ceil(current.reset.Sub(now).Seconds())))
	return current.used <= g.limit, remaining, reset
}

func (g *RequestGuard) removeExpiredLocked(now time.Time) {
	for client, window := range g.clients {
		if !window.reset.After(now) {
			delete(g.clients, client)
		}
	}
}

func clientAddress(r *http.Request) string {
	for _, value := range []string{
		r.Header.Get("CF-Connecting-IP"),
		firstForwarded(r.Header.Get("X-Forwarded-For")),
	} {
		if parsed := net.ParseIP(strings.TrimSpace(value)); parsed != nil {
			return parsed.String()
		}
	}
	host := strings.TrimSpace(r.RemoteAddr)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	if parsed := net.ParseIP(strings.Trim(host, "[]")); parsed != nil {
		return parsed.String()
	}
	if host != "" {
		return host
	}
	return "unknown"
}

func setRateLimitHeaders(
	header http.Header,
	limit int,
	remaining int,
	reset int,
) {
	values := [3]string{
		strconv.Itoa(limit),
		strconv.Itoa(remaining),
		strconv.Itoa(reset),
	}
	header.Set("RateLimit-Limit", values[0])
	header.Set("RateLimit-Remaining", values[1])
	header.Set("RateLimit-Reset", values[2])
	header.Set("X-RateLimit-Limit", values[0])
	header.Set("X-RateLimit-Remaining", values[1])
	header.Set("X-RateLimit-Reset", values[2])
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
