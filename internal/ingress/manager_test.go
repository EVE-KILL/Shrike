package ingress

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// The route table is ordered, and the order is the routing policy rather than
// an implementation detail — "which surface answers this request" is decided
// by position at least as often as by matcher. These tests assert the order
// directly so that reordering routes has to be a deliberate act with a failing
// test attached, not a tidy-up someone does while moving code around.

func testSurfaces() map[string]http.Handler {
	h := func(name string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, name)
		})
	}
	return map[string]http.Handler{
		SurfaceSameOrigin: h(SurfaceSameOrigin),
		SurfaceWS:         h(SurfaceWS),
	}
}

func testConfig() Config {
	return Config{Address: "127.0.0.1:0"}
}

// newTestManager builds a Manager without starting Caddy, so buildConfig can be
// exercised without binding a port or claiming the process-global runtime.
func newTestManager(t *testing.T, cfg Config) *Manager {
	t.Helper()
	m := New(testSurfaces(), zerolog.Nop())
	m.cfg = cfg
	return m
}

func TestRouteOrder(t *testing.T) {
	m := newTestManager(t, testConfig())

	_, _, routes, err := m.buildConfig()
	if err != nil {
		t.Fatalf("buildConfig: %v", err)
	}

	want := []RouteStatus{
		{Match: "path /health", Surface: SurfaceSameOrigin},
		{Match: "path /ws, /ws/*", Surface: SurfaceWS},
		{
			Match:   "path /api, /api/*, /auth, /auth/*, /images, /images/*",
			Surface: SurfaceSameOrigin,
		},
		{Match: "(default)", Surface: "404 — no renderer configured"},
	}
	if len(routes) != len(want) {
		t.Fatalf("got %d routes, want %d: %+v", len(routes), len(want), routes)
	}
	for i := range want {
		if routes[i] != want[i] {
			t.Errorf("route %d = %+v, want %+v", i, routes[i], want[i])
		}
	}
}

func TestGoOwnedPathsPrecedeNuxt(t *testing.T) {
	m := newTestManager(t, testConfig())

	_, _, routes, err := m.buildConfig()
	if err != nil {
		t.Fatalf("buildConfig: %v", err)
	}
	site, fallback := -1, -1
	for i, r := range routes {
		if strings.Contains(r.Match, "/images/*") {
			site = i
		}
		if r.Match == "(default)" {
			fallback = i
		}
	}
	if site == -1 || fallback == -1 || site >= fallback {
		t.Fatalf("site route %d and fallback %d are misordered: %+v", site, fallback, routes)
	}
}

// A surface named in the route table but absent from the handler map must fail
// the build. Caddy would otherwise reject the config at provision time with a
// message about a module, which is a much longer walk to the same answer.
func TestUnregisteredSurfaceIsAnError(t *testing.T) {
	m := New(map[string]http.Handler{
		// same-origin is deliberately missing.
		SurfaceWS: http.NotFoundHandler(),
	}, zerolog.Nop())
	m.cfg = testConfig()

	_, _, _, err := m.buildConfig()
	if err == nil {
		t.Fatal("buildConfig succeeded with an unregistered surface")
	}
	if !strings.Contains(err.Error(), SurfaceSameOrigin) {
		t.Errorf("error does not name the missing surface: %v", err)
	}
}

// Without a renderer socket the catch-all must be an explicit 404. Falling off
// the end of Caddy's route chain instead produces an empty 200, which tells a
// crawler the page exists and a developer nothing.
func TestCatchAllWithoutSocketIsAnExplicit404(t *testing.T) {
	m := newTestManager(t, testConfig())

	cfg, _, _, err := m.buildConfig()
	if err != nil {
		t.Fatalf("buildConfig: %v", err)
	}

	last := lastRoute(t, cfg)
	handler := firstHandler(t, last)
	if got := handler["handler"]; got != "static_response" {
		t.Fatalf("catch-all handler = %v, want static_response", got)
	}
	if got := handler["status_code"]; got != 404 {
		t.Errorf("catch-all status = %v, want 404", got)
	}
}

func TestCatchAllWithSocketProxiesOverUnix(t *testing.T) {
	cfg := testConfig()
	cfg.NuxtSocket = "/run/shrike/nuxt.sock"

	m := newTestManager(t, cfg)
	built, _, _, err := m.buildConfig()
	if err != nil {
		t.Fatalf("buildConfig: %v", err)
	}

	handler := firstHandler(t, lastRoute(t, built))
	if got := handler["handler"]; got != "reverse_proxy" {
		t.Fatalf("catch-all handler = %v, want reverse_proxy", got)
	}
	upstreams, ok := handler["upstreams"].([]any)
	if !ok || len(upstreams) != 1 {
		t.Fatalf("upstreams = %#v, want exactly one", handler["upstreams"])
	}
	// The unix// spelling is what caddy.ParseNetworkAddress needs to treat the
	// remainder as a filesystem path rather than a host:port.
	if got := upstreams[0].(map[string]any)["dial"]; got != "unix//run/shrike/nuxt.sock" {
		t.Errorf("dial = %v, want unix//run/shrike/nuxt.sock", got)
	}
}

// Every route must be terminal. Shrike's handler does not call the next element
// in the chain, so a non-terminal route happens to work today — but a future
// handler that does call next would fall through into the renderer, and a
// request meant for the API would come back as a rendered HTML page.
func TestEveryRouteIsTerminal(t *testing.T) {
	m := newTestManager(t, testConfig())

	cfg, _, _, err := m.buildConfig()
	if err != nil {
		t.Fatalf("buildConfig: %v", err)
	}
	for i, raw := range allRoutes(t, cfg) {
		route := raw.(map[string]any)
		if terminal, _ := route["terminal"].(bool); !terminal {
			t.Errorf("route %d is not terminal: %+v", i, route)
		}
	}
}

func TestAdminAPIIsDisabled(t *testing.T) {
	m := newTestManager(t, testConfig())

	cfg, _, _, err := m.buildConfig()
	if err != nil {
		t.Fatalf("buildConfig: %v", err)
	}
	admin, ok := cfg["admin"].(map[string]any)
	if !ok {
		t.Fatal("no admin block; Caddy would expose its default admin socket")
	}
	if disabled, _ := admin["disabled"].(bool); !disabled {
		t.Error("admin API is not disabled")
	}
}

func TestStartRejectsBadAddress(t *testing.T) {
	m := New(testSurfaces(), zerolog.Nop())
	if err := m.Start(context.Background(), Config{Address: "not-an-address"}); err == nil {
		t.Fatal("Start accepted a malformed address")
		_ = m.Close()
	}
	if activeManager.Load() != nil {
		t.Error("a failed Start left the runtime claimed")
	}
}

// Start holds opMu while loading and checking the listener. Its error cleanup
// must not call the public Close method, which acquires that same mutex.
func TestCanceledStartReturnsAndReleasesRuntime(t *testing.T) {
	port := freePort(t)
	cfg := testConfig()
	cfg.Address = fmt.Sprintf("127.0.0.1:%d", port)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	m := New(testSurfaces(), zerolog.Nop())
	done := make(chan error, 1)
	go func() { done <- m.Start(ctx, cfg) }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Start succeeded with an already-canceled context")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start deadlocked while cleaning up a canceled readiness check")
	}
	if activeManager.Load() != nil {
		t.Fatal("canceled Start left the Caddy runtime claimed")
	}
}

// Live routing through Caddy. The JSON-shape tests above assert what we asked
// for; this one asserts what Caddy actually does with it, which is the part
// that would break silently on a Caddy upgrade.
func TestServesEachSurface(t *testing.T) {
	port := freePort(t)

	cfg := testConfig()
	cfg.Address = fmt.Sprintf("127.0.0.1:%d", port)

	m := New(testSurfaces(), zerolog.Nop())
	if err := m.Start(context.Background(), cfg); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	for _, tc := range []struct {
		name string
		host string
		path string
		want string
		code int
	}{
		{"health has no host requirement", "", "/health", SurfaceSameOrigin, 200},
		{"websocket path health is not special", "eve-kill.test", "/health", SurfaceSameOrigin, 200},
		{"websocket root path", "eve-kill.test", "/ws", SurfaceWS, 200},
		{"websocket endpoint path", "eve-kill.test", "/ws/killlist", SurfaceWS, 200},
		{"frontend api root", "eve-kill.test", "/api", SurfaceSameOrigin, 200},
		{"frontend api prefix", "eve-kill.test", "/api/killlist", SurfaceSameOrigin, 200},
		{"frontend auth root", "eve-kill.test", "/auth", SurfaceSameOrigin, 200},
		{"frontend auth prefix", "eve-kill.test", "/auth/callback", SurfaceSameOrigin, 200},
		{"frontend images root", "eve-kill.test", "/images", SurfaceSameOrigin, 200},
		{"frontend images prefix", "eve-kill.test", "/images/types/42/icon", SurfaceSameOrigin, 200},
		// An unknown host is a frontend request by definition, which is what
		// makes tenant custom domains work without being enumerated.
		{"unknown host falls through", "tenant.example.com", "/", "", 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, base+tc.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tc.host != "" {
				req.Host = tc.host
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer res.Body.Close() //nolint:errcheck // response body of a test request

			body, _ := io.ReadAll(res.Body)
			if res.StatusCode != tc.code {
				t.Errorf("status = %d, want %d (body %q)", res.StatusCode, tc.code, body)
			}
			if tc.want != "" && string(body) != tc.want {
				t.Errorf("served by %q, want %q", body, tc.want)
			}
		})
	}
}

func TestCatchAllProxiesToUnixSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix sockets are unavailable on Windows")
	}

	// Darwin's sockaddr_un path limit is short enough that t.TempDir's full
	// nested path can exceed it. Keep the socket name directly under /tmp.
	socket := filepath.Join(
		"/tmp",
		fmt.Sprintf("shrike-nuxt-%d-%d.sock", os.Getpid(), time.Now().UnixNano()),
	)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("listen on Unix socket: %v", err)
	}

	backend := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Renderer-Host", r.Host)
		_, _ = io.WriteString(w, "nuxt:"+r.URL.Path)
	})}
	backendDone := make(chan error, 1)
	go func() { backendDone <- backend.Serve(listener) }()
	t.Cleanup(func() {
		_ = backend.Close()
		if err := <-backendDone; err != nil && err != http.ErrServerClosed {
			t.Errorf("renderer server: %v", err)
		}
	})

	port := freePort(t)
	cfg := testConfig()
	cfg.Address = fmt.Sprintf("127.0.0.1:%d", port)
	cfg.NuxtSocket = socket

	m := New(testSurfaces(), zerolog.Nop())
	if err := m.Start(context.Background(), cfg); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })

	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/rendered", port),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "tenant.example.com"
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request through renderer fallback: %v", err)
	}
	defer res.Body.Close() //nolint:errcheck // response body of a test request

	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", res.StatusCode, body)
	}
	if got, want := string(body), "nuxt:/rendered"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
	if got, want := res.Header.Get("X-Renderer-Host"), "tenant.example.com"; got != want {
		t.Errorf("renderer Host = %q, want %q", got, want)
	}
}

// Caddy's runtime is process-global, so a second Manager would evict the
// first's configuration rather than running beside it.
func TestSecondManagerIsRefused(t *testing.T) {
	port := freePort(t)

	cfg := testConfig()
	cfg.Address = fmt.Sprintf("127.0.0.1:%d", port)

	first := New(testSurfaces(), zerolog.Nop())
	if err := first.Start(context.Background(), cfg); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	t.Cleanup(func() { _ = first.Close() })

	second := New(testSurfaces(), zerolog.Nop())
	if err := second.Start(context.Background(), cfg); err == nil {
		_ = second.Close()
		t.Fatal("a second Manager was allowed to claim the Caddy runtime")
	}

	// The first must still be serving: a refused second Start has to be inert,
	// not something that tears down the runtime on its way out.
	res, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port)) //nolint:noctx // test request
	if err != nil {
		t.Fatalf("first manager stopped serving after the refused Start: %v", err)
	}
	_ = res.Body.Close()
}

func TestCloseIsIdempotent(t *testing.T) {
	port := freePort(t)

	cfg := testConfig()
	cfg.Address = fmt.Sprintf("127.0.0.1:%d", port)

	m := New(testSurfaces(), zerolog.Nop())
	if err := m.Start(context.Background(), cfg); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := m.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := m.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}

	// The port must actually be released, not merely stop being routed.
	deadline := time.Now().Add(2 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", cfg.Address, 100*time.Millisecond)
		if err != nil {
			return
		}
		_ = conn.Close()
		if time.Now().After(deadline) {
			t.Fatal("listener still accepting after Close")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot bind a local port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		t.Fatalf("releasing probe listener: %v", err)
	}
	return port
}

func allRoutes(t *testing.T, cfg map[string]any) []any {
	t.Helper()
	apps := cfg["apps"].(map[string]any)
	httpApp := apps["http"].(map[string]any)
	servers := httpApp["servers"].(map[string]any)
	server := servers["shrike"].(map[string]any)
	routes, ok := server["routes"].([]any)
	if !ok || len(routes) == 0 {
		t.Fatalf("no routes in the built config")
	}
	return routes
}

func lastRoute(t *testing.T, cfg map[string]any) map[string]any {
	t.Helper()
	routes := allRoutes(t, cfg)
	return routes[len(routes)-1].(map[string]any)
}

func firstHandler(t *testing.T, route map[string]any) map[string]any {
	t.Helper()
	handlers, ok := route["handle"].([]any)
	if !ok || len(handlers) == 0 {
		t.Fatalf("route has no handlers: %+v", route)
	}
	return handlers[0].(map[string]any)
}
