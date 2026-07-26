package ingress

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/rs/zerolog"
)

// activeManager is the running Manager, if any.
//
// Caddy's runtime is process-global — caddy.Load replaces whatever was loaded
// last, wherever it was loaded from — so a second Manager would silently
// evict the first's configuration rather than running beside it. Making that
// an error at Start is better than discovering it as vanished listeners.
//
// It is also how modules find their handlers: Caddy builds them from JSON and
// has no way to pass a Go value in, so Provision loads this pointer instead.
var activeManager atomic.Pointer[Manager]

// Manager owns the embedded Caddy runtime.
type Manager struct {
	surfaces map[string]http.Handler
	log      zerolog.Logger

	// opMu serialises reconfiguration. Caddy provisions a replacement config
	// before retiring the previous one, so two concurrent loads would have
	// both live at once and the loser's listeners would be torn down under
	// the winner.
	opMu sync.Mutex

	mu         sync.RWMutex
	cfg        Config
	generation uint64
	running    bool
	startedAt  time.Time
	lastReload time.Time
	lastError  string
	listeners  []ListenerStatus
	routes     []RouteStatus
	events     []Event
}

// New creates a Manager over a set of named surfaces.
//
// The map is captured, not copied — callers must not mutate it after this
// returns. Surfaces are fixed for the process lifetime by design: they are
// wired at startup from code, and a config reload changes which hostnames
// route where, never what a surface is.
func New(surfaces map[string]http.Handler, log zerolog.Logger) *Manager {
	return &Manager{surfaces: surfaces, log: log}
}

// Start binds the listener and loads the initial configuration.
func (m *Manager) Start(ctx context.Context, cfg Config) error {
	m.opMu.Lock()
	defer m.opMu.Unlock()

	if cfg.Address == "" {
		return errors.New("ingress: address is required")
	}
	if _, _, err := net.SplitHostPort(cfg.Address); err != nil {
		return fmt.Errorf("ingress: invalid address %q: %w", cfg.Address, err)
	}
	if cfg.DataDir != "" {
		if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
			return fmt.Errorf("ingress data directory: %w", err)
		}
	}
	// Claim the process-global runtime atomically. A Load followed by Store
	// would let two concurrent Start calls both observe nil and both proceed to
	// replace Caddy's global configuration.
	if !activeManager.CompareAndSwap(nil, m) {
		if activeManager.Load() == m {
			return errors.New("ingress: manager is already running")
		}
		return errors.New("ingress: another embedded Caddy runtime is already active in this process")
	}

	m.mu.Lock()
	m.cfg = cfg
	if m.startedAt.IsZero() {
		m.startedAt = time.Now()
	}
	m.mu.Unlock()

	if err := m.reloadLocked("initial configuration"); err != nil {
		// Release the claim so a caller that retries with a corrected config
		// is not told the runtime is already taken by its own failed attempt.
		activeManager.CompareAndSwap(m, nil)
		return err
	}
	if err := m.waitReady(ctx, cfg.Address); err != nil {
		// Start still holds opMu, so calling Close here would try to acquire the
		// same non-reentrant mutex and deadlock. Stop through the locked helper
		// instead, and preserve both errors if cleanup ever fails.
		if stopErr := m.closeLocked(); stopErr != nil {
			return errors.Join(err, stopErr)
		}
		return err
	}
	return nil
}

// Reconfigure swaps the routing configuration on a running Manager.
//
// A rejected configuration leaves the previous one live: Caddy validates and
// provisions before it swaps, so a bad reload is a no-op at the network layer,
// and restoring the field here keeps Manager's own view honest about what is
// actually serving.
func (m *Manager) Reconfigure(cfg Config) error {
	m.opMu.Lock()
	defer m.opMu.Unlock()

	if activeManager.Load() != m {
		return errors.New("ingress: manager is not running")
	}

	m.mu.Lock()
	previous := m.cfg
	m.cfg = cfg
	m.mu.Unlock()

	if err := m.reloadLocked("configuration updated"); err != nil {
		m.mu.Lock()
		m.cfg = previous
		m.mu.Unlock()
		return err
	}
	return nil
}

// Close stops Caddy and releases the runtime claim.
//
// Safe to call on a Manager that never started, and safe to call twice, so it
// can sit in a defer next to Start without the caller tracking which error
// paths already cleaned up.
func (m *Manager) Close() error {
	m.opMu.Lock()
	defer m.opMu.Unlock()

	return m.closeLocked()
}

// closeLocked stops Caddy and releases the runtime claim. Callers hold opMu.
func (m *Manager) closeLocked() error {
	if activeManager.Load() != m {
		return nil
	}
	err := caddy.Stop()
	activeManager.CompareAndSwap(m, nil)

	m.mu.Lock()
	m.running = false
	m.addEventLocked("info", "ingress stopped")
	m.mu.Unlock()

	if err != nil {
		return fmt.Errorf("stopping embedded Caddy: %w", err)
	}
	return nil
}

// reloadLocked builds and loads a configuration. Callers hold opMu.
func (m *Manager) reloadLocked(reason string) error {
	m.mu.RLock()
	next := m.generation + 1
	m.mu.RUnlock()

	cfg, listeners, routes, err := m.buildConfig()
	if err != nil {
		m.recordReloadError(reason, err)
		return err
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		m.recordReloadError(reason, err)
		return err
	}
	if err := caddy.Load(raw, true); err != nil {
		m.recordReloadError(reason, err)
		return fmt.Errorf("loading embedded Caddy config: %w", err)
	}

	m.mu.Lock()
	m.generation = next
	m.running = true
	m.lastReload = time.Now()
	m.lastError = ""
	m.listeners = listeners
	m.routes = routes
	m.addEventLocked("info", reason)
	m.mu.Unlock()

	m.log.Info().Uint64("generation", next).Str("reason", reason).Msg("ingress configuration loaded")
	return nil
}

func (m *Manager) recordReloadError(reason string, err error) {
	m.mu.Lock()
	m.lastError = err.Error()
	m.addEventLocked("error", reason+": "+err.Error())
	m.mu.Unlock()
	m.log.Error().Err(err).Str("reason", reason).Msg("ingress configuration rejected")
}

func (m *Manager) addEventLocked(level, message string) {
	m.events = append(m.events, Event{At: time.Now().UTC(), Level: level, Message: message})
	if len(m.events) > 32 {
		m.events = append([]Event(nil), m.events[len(m.events)-32:]...)
	}
}

// waitReady blocks until the listener accepts, so Start does not return before
// the process can actually answer. Without it a readiness probe racing a fast
// container start can observe a connection refused that means nothing.
func (m *Manager) waitReady(ctx context.Context, address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		address = net.JoinHostPort("127.0.0.1", port)
	}

	readyCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var dialer net.Dialer
	var lastErr error
	for {
		conn, err := dialer.DialContext(readyCtx, "tcp", address)
		if err == nil {
			return conn.Close()
		}
		lastErr = err

		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-readyCtx.Done():
			timer.Stop()
			return fmt.Errorf("ingress did not accept connections on %s: %w (last dial: %v)", address, readyCtx.Err(), lastErr)
		case <-timer.C:
		}
	}
}

// Status reports what is currently serving.
func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Status{
		Running:         m.running,
		Generation:      m.generation,
		StartedAt:       m.startedAt,
		LastReloadAt:    m.lastReload,
		LastReloadError: m.lastError,
		Listeners:       append([]ListenerStatus(nil), m.listeners...),
		Routes:          append([]RouteStatus(nil), m.routes...),
		RecentEvents:    append([]Event(nil), m.events...),
	}
}

// buildConfig assembles the Caddy JSON configuration.
//
// The route table is ordered, and the order carries the routing policy:
//
//  1. /health on the public API host goes through the public Huma surface. That
//     keeps the response's generated schema link on the public /schemas path.
//  2. /health on every other host goes through the private Huma surface.
//     Kubernetes probes by pod IP with no useful Host header, so this route
//     matches on path alone rather than relying on DNS.
//  3. The two dedicated hostnames, each to its own surface.
//  4. /ws, /api and /auth on the frontend origin. After the hostnames so that a request
//     to api.eve-kill.com/api/... resolves as the public surface rather than
//     being captured by the private path prefix.
//  5. Everything else to the Nuxt renderer. Unmatched is the frontend by
//     definition, which is what lets tenant custom domains work without ever
//     being enumerated here.
//
// Every route is terminal. Shrike's handler does not call the next element in
// the chain, so this is belt and braces — but the belt is what stops a future
// handler that does call next from quietly falling through into the renderer.
func (m *Manager) buildConfig() (map[string]any, []ListenerStatus, []RouteStatus, error) {
	m.mu.RLock()
	cfg := m.cfg
	m.mu.RUnlock()

	var routes []any
	var status []RouteStatus

	surfaceRoute := func(match map[string]any, describe, surface string) error {
		if _, ok := m.surfaces[surface]; !ok {
			return fmt.Errorf("surface %q is routed but not registered", surface)
		}
		route := map[string]any{
			"handle":   []any{map[string]any{"handler": "shrike", "surface": surface}},
			"terminal": true,
		}
		if match != nil {
			route["match"] = []any{match}
		}
		routes = append(routes, route)
		status = append(status, RouteStatus{Match: describe, Surface: surface})
		return nil
	}

	// claimed guards against one hostname being listed under two surfaces.
	// Caddy would not object — the matchers are separate, so the earlier route
	// would simply win — and a silently shadowed surface is a much harder
	// thing to notice than a refused startup.
	claimed := map[string]string{}

	hostRoutes := []struct {
		hosts   []string
		surface string
	}{
		{cfg.PublicHosts, SurfacePublic},
		{cfg.ImagesHosts, SurfaceImages},
	}
	for i := range hostRoutes {
		hosts, err := normalizeHosts(hostRoutes[i].hosts, hostRoutes[i].surface, claimed)
		if err != nil {
			return nil, nil, nil, err
		}
		hostRoutes[i].hosts = hosts
	}

	// Huma generates response schema links from the surface's SchemasPath.
	// Public health must therefore reach the public surface; sending every
	// /health request to private would emit /api/schemas links which the public
	// hostname does not serve.
	if publicHosts := hostRoutes[0].hosts; len(publicHosts) > 0 {
		if err := surfaceRoute(
			map[string]any{"host": publicHosts, "path": []string{"/health"}},
			"host "+strings.Join(publicHosts, ", ")+" and path /health", SurfacePublic,
		); err != nil {
			return nil, nil, nil, err
		}
	}

	// The path-only fallback keeps liveness independent of DNS and Host. It
	// also gives the WS and images hostnames the same process health endpoint
	// while those surfaces remain non-Huma handlers.
	if err := surfaceRoute(
		map[string]any{"path": []string{"/health"}},
		"path /health", SurfacePrivate,
	); err != nil {
		return nil, nil, nil, err
	}

	for _, h := range hostRoutes {
		hosts := h.hosts
		if len(hosts) == 0 {
			// No configured hostname means a deployment that does not serve
			// that surface. Omitting the route is what keeps it from
			// answering on a name nobody asked it to answer on.
			continue
		}
		if err := surfaceRoute(
			map[string]any{"host": hosts},
			"host "+strings.Join(hosts, ", "), h.surface,
		); err != nil {
			return nil, nil, nil, err
		}
	}

	// WebSockets share the frontend origin but bypass the Nuxt process. A
	// dedicated prefix avoids stealing the real /comments and /status pages,
	// while still letting custom domains use their own origin for live data.
	if err := surfaceRoute(
		map[string]any{"path": []string{"/ws", "/ws/*"}},
		"path /ws, /ws/*", SurfaceWS,
	); err != nil {
		return nil, nil, nil, err
	}

	if err := surfaceRoute(
		map[string]any{"path": []string{"/api", "/api/*", "/auth", "/auth/*"}},
		"path /api, /api/*, /auth, /auth/*", SurfacePrivate,
	); err != nil {
		return nil, nil, nil, err
	}

	// The catch-all is always emitted, in one of two forms. Leaving the route
	// list open-ended instead would fall off the end of Caddy's chain, and an
	// unhandled request there is answered with an empty 200 — the single worst
	// response available, since it tells a crawler the page exists and tells a
	// developer nothing at all.
	if socket := strings.TrimSpace(cfg.NuxtSocket); socket != "" {
		routes = append(routes, map[string]any{
			"handle": []any{map[string]any{
				"handler": "reverse_proxy",
				"upstreams": []any{map[string]any{
					"dial": "unix/" + socket,
				}},
			}},
			"terminal": true,
		})
		status = append(status, RouteStatus{Match: "(default)", Surface: "nuxt:" + socket})
	} else {
		routes = append(routes, map[string]any{
			"handle": []any{map[string]any{
				"handler":     "static_response",
				"status_code": 404,
				"headers":     map[string]any{"Content-Type": []string{"text/plain; charset=utf-8"}},
				"body":        "no renderer configured\n",
			}},
			"terminal": true,
		})
		status = append(status, RouteStatus{Match: "(default)", Surface: "404 — no renderer configured"})
	}

	server := map[string]any{
		"listen": []string{"tcp/" + cfg.Address},
		// Nothing here terminates TLS yet, and automatic HTTPS would otherwise
		// try to provision certificates and bind a redirect listener for every
		// hostname named above.
		"automatic_https": map[string]any{"disable": true},
		"routes":          routes,
		// Slowloris guard, matching the timeout the plain net/http server used
		// before Caddy took over the listener.
		"read_header_timeout": "10s",
		"read_timeout":        "2m",
		"idle_timeout":        "5m",
		"max_header_bytes":    64 << 10,
	}

	config := map[string]any{
		// No admin API. Shrike's configuration comes from Manager and nowhere
		// else; an admin socket would be a second, unauthenticated way to
		// reconfigure the edge. Persistence off for the same reason — the
		// running config should always be the one this code built.
		"admin": map[string]any{
			"disabled": true,
			"config":   map[string]any{"persist": false},
		},
		"logging": map[string]any{
			"logs": map[string]any{"default": map[string]any{"level": caddyLogLevel(cfg.LogLevel)}},
		},
		"apps": map[string]any{
			"http": map[string]any{
				"servers": map[string]any{"shrike": server},
			},
			// caddytls is linked whether or not we ask for it — caddyhttp needs
			// it for connection policies — and its storage sweeper runs on a
			// timer against the default storage path, which on a developer's
			// machine is somewhere under their home directory. Shrike stores no
			// certificates, so the sweep can only find other software's, and
			// writing outside our own data directory is not ours to do.
			"tls": map[string]any{"disable_storage_clean": true},
		},
	}
	// No "storage" block yet. Caddy's default file storage needs no module
	// linked, whereas naming file_system here would reference a module that is
	// deliberately not imported — and there is nothing to store until Shrike
	// terminates TLS. DataDir is carried in Config for that day.

	listeners := []ListenerStatus{{
		Name:        "shrike",
		Address:     cfg.Address,
		Description: "Embedded Caddy listener serving every Shrike surface",
	}}
	return config, listeners, status, nil
}

// normalizeHosts cleans one surface's hostname list and records each name in
// claimed, so a second surface asking for the same one is an error.
//
// Lowercased because hostnames are case-insensitive and Caddy compares them
// that way; without it, "API.localhost" and "api.localhost" would reach Caddy
// as two distinct names, and Caddy would reject the pair as a duplicate after
// normalising them itself.
func normalizeHosts(hosts []string, surface string, claimed map[string]string) ([]string, error) {
	out := make([]string, 0, len(hosts))
	for _, raw := range hosts {
		host := strings.ToLower(strings.TrimSpace(raw))
		if host == "" {
			continue
		}
		if owner, ok := claimed[host]; ok {
			if owner == surface {
				// Repeated within one surface: harmless intent, but Caddy
				// rejects a repeated name outright, so drop it here.
				continue
			}
			return nil, fmt.Errorf(
				"hostname %q is claimed by both the %s and %s surfaces", host, owner, surface)
		}
		claimed[host] = surface
		out = append(out, host)
	}
	return out, nil
}

// caddyLogLevel maps a zerolog level name onto the zap levels Caddy accepts.
func caddyLogLevel(level string) string {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "TRACE", "DEBUG":
		// Caddy has no TRACE. Debug is the closest thing that still says
		// "tell me everything".
		return "DEBUG"
	case "WARN", "WARNING":
		return "WARN"
	case "ERROR", "FATAL", "PANIC", "DISABLED":
		// Caddy's ERROR is the floor worth keeping: silencing the edge
		// entirely would hide bind failures.
		return "ERROR"
	default:
		return "INFO"
	}
}
