// Package ingress embeds Caddy as Shrike's HTTP front door.
//
// Caddy is a library here, not a sidecar. There is no Caddyfile: Manager
// builds a JSON configuration and hands it to caddy.Load. Production origins
// stay on plain HTTP behind Cloudflare, which owns public TLS, HTTP/2 and
// HTTP/3. Development enables Caddy's internal CA on the same listener, giving
// localhost a trusted certificate and exercising HTTP/2 and HTTP/3 locally
// without a second process or config file.
//
// The part that makes it worth the dependency is that Shrike's own handlers
// are Caddy modules rather than a proxy target. Caddy calls them directly, in
// process. Only the Nuxt renderer, which is genuinely a separate Bun process,
// is reached over a socket.
//
// This package exposes Shrike-shaped configuration and status. Caddy's JSON
// schema stays an implementation detail behind Manager so that pinning a new
// Caddy version does not ripple into the rest of the codebase.
package ingress

import "time"

// Surfaces are the names Manager routes to. They are the vocabulary shared
// between the route table here and the handler map the caller supplies; a name
// in one and not the other is a startup error rather than a runtime 404.
const (
	// SurfaceSameOrigin backs /api, /auth, and /images on the main-site origin.
	SurfaceSameOrigin = "same-origin"
	// SurfaceWS backs /ws and /ws/* on the frontend origin.
	SurfaceWS = "ws"
)

// Config describes the listener and where each surface lives.
//
// The hostnames are configuration rather than constants because the same
// binary runs on a laptop, and a config builder with eve-kill.com baked into
// it would be untestable anywhere else.
type Config struct {
	// Address is the listener, as host:port.
	Address string

	// DataDir is where Caddy keeps its own state, including the development CA
	// and localhost certificates when LocalHTTPS is enabled.
	DataDir string

	// LogLevel is a zerolog level name, translated to Caddy's zap levels.
	LogLevel string

	// LocalHTTPS enables an explicitly managed localhost certificate from
	// Caddy's internal CA. It is for local development only; public TLS is
	// terminated by Cloudflare before requests reach the production origin.
	LocalHTTPS bool

	// SkipLocalTrustInstall leaves the generated development CA out of system,
	// Java, and Firefox trust stores. Tests use this to exercise real TLS
	// without changing the host; normal local development leaves it false.
	SkipLocalTrustInstall bool

	// NuxtSocket is the Unix socket the Nitro renderer listens on. Every
	// request that matches no surface is proxied there, which is what makes
	// tenant custom domains work without enumerating them: an unrecognised
	// host is a frontend request by definition.
	//
	// Empty disables the fallback, and unmatched requests get a plain 404
	// instead. That is the right shape for an API-only deployment and for
	// tests, which should not need a renderer to assert on routing.
	NuxtSocket string
}

// ListenerStatus describes one bound listener for the status endpoint.
type ListenerStatus struct {
	Name        string `json:"name"`
	Address     string `json:"address"`
	TLS         bool   `json:"tls"`
	Description string `json:"description,omitempty"`
}

// RouteStatus describes one entry in the resolved route table, in the order
// Caddy evaluates it. Worth surfacing because "which surface answered this"
// is the first question when a request lands somewhere unexpected, and the
// ordering is the answer more often than the matcher is.
type RouteStatus struct {
	Match   string `json:"match"`
	Surface string `json:"surface"`
}

// Event is one entry in Manager's bounded history.
type Event struct {
	At      time.Time `json:"at"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

// Status is the ingress view for operators.
type Status struct {
	Running         bool             `json:"running"`
	Generation      uint64           `json:"generation"`
	StartedAt       time.Time        `json:"started_at"`
	LastReloadAt    time.Time        `json:"last_reload_at,omitzero"`
	LastReloadError string           `json:"last_reload_error,omitempty"`
	Listeners       []ListenerStatus `json:"listeners"`
	Routes          []RouteStatus    `json:"routes"`
	RecentEvents    []Event          `json:"recent_events,omitempty"`
}
