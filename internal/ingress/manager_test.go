package ingress

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
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
		SurfaceFrontend:   h(SurfaceFrontend),
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
		{Match: "path /health, /ready", Surface: SurfaceSameOrigin},
		{Match: "path /ws, /ws/*", Surface: SurfaceWS},
		{
			Match:   "path /api, /api/*, /auth, /auth/*, /images, /images/*",
			Surface: SurfaceSameOrigin,
		},
		{Match: "(default)", Surface: SurfaceFrontend},
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

func TestAPIPathsPrecedeFrontend(t *testing.T) {
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

func TestCatchAllUsesFrontendInProcess(t *testing.T) {
	m := newTestManager(t, testConfig())
	cfg, _, _, err := m.buildConfig()
	if err != nil {
		t.Fatal(err)
	}
	handler := originHandler(t, lastRoute(t, cfg))
	if handler["handler"] != "shrike" || handler["surface"] != SurfaceFrontend {
		t.Fatalf("catch-all = %#v, want in-process frontend", handler)
	}
}

func TestMissingFrontendFailsStartup(t *testing.T) {
	surfaces := testSurfaces()
	delete(surfaces, SurfaceFrontend)
	m := New(surfaces, zerolog.Nop())
	m.cfg = testConfig()
	_, _, _, err := m.buildConfig()
	if err == nil || !strings.Contains(err.Error(), SurfaceFrontend) {
		t.Fatalf("error = %v, want missing frontend", err)
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

func TestEveryRouteStartsWithAccessLogger(t *testing.T) {
	m := newTestManager(t, testConfig())

	cfg, _, _, err := m.buildConfig()
	if err != nil {
		t.Fatalf("buildConfig: %v", err)
	}
	for i, raw := range allRoutes(t, cfg) {
		route := raw.(map[string]any)
		handlers := route["handle"].([]any)
		if got := handlers[0].(map[string]any)["handler"]; got != "shrike_access_log" {
			t.Errorf("route %d first handler = %v, want shrike_access_log", i, got)
		}
		if len(handlers) < 2 {
			t.Errorf("route %d has an access logger but no origin handler", i)
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

func TestLocalHTTPSConfigUsesInternalCAAndScopedStorage(t *testing.T) {
	cfg := testConfig()
	cfg.DataDir = t.TempDir()
	cfg.LocalHTTPS = true
	cfg.SkipLocalTrustInstall = true

	m := newTestManager(t, cfg)
	built, listeners, _, err := m.buildConfig()
	if err != nil {
		t.Fatalf("buildConfig: %v", err)
	}

	server := serverConfig(t, built)
	policies, ok := server["tls_connection_policies"].([]any)
	if !ok || len(policies) != 1 {
		t.Fatalf("TLS connection policies = %#v, want one policy", server["tls_connection_policies"])
	}

	apps := built["apps"].(map[string]any)
	tlsApp := apps["tls"].(map[string]any)
	certificates := tlsApp["certificates"].(map[string]any)
	names, ok := certificates["automate"].([]string)
	if !ok {
		t.Fatalf("automated certificate names = %#v, want []string", certificates["automate"])
	}
	wantNames := []string{"localhost"}
	if strings.Join(names, ",") != strings.Join(wantNames, ",") {
		t.Errorf("automated certificate names = %v, want %v", names, wantNames)
	}

	automation := tlsApp["automation"].(map[string]any)
	automationPolicies := automation["policies"].([]any)
	if len(automationPolicies) != 2 {
		t.Fatalf("automation policies = %#v, want localhost and tenant boards", automationPolicies)
	}
	for _, policy := range automationPolicies {
		issuers := policy.(map[string]any)["issuers"].([]any)
		if got := issuers[0].(map[string]any)["module"]; got != "internal" {
			t.Errorf("certificate issuer = %v, want internal", got)
		}
	}

	// Board hostnames live in the database, so the tenant policy issues on
	// demand rather than naming each board up front.
	tenant := automationPolicies[1].(map[string]any)
	tenantNames, _ := tenant["subjects"].([]string)
	if strings.Join(tenantNames, ",") != "*.localhost" {
		t.Errorf("tenant policy subjects = %v, want [*.localhost]", tenant["subjects"])
	}
	if onDemand, _ := tenant["on_demand"].(bool); !onDemand {
		t.Error("tenant policy does not issue certificates on demand")
	}

	pkiApp := apps["pki"].(map[string]any)
	authorities := pkiApp["certificate_authorities"].(map[string]any)
	localCA := authorities["local"].(map[string]any)
	if install, _ := localCA["install_trust"].(bool); install {
		t.Error("test configuration unexpectedly installs the local CA into host trust stores")
	}

	storage := built["storage"].(map[string]any)
	if got := storage["module"]; got != "file_system" {
		t.Errorf("storage module = %v, want file_system", got)
	}
	if got, want := storage["root"], filepath.Join(cfg.DataDir, "caddy"); got != want {
		t.Errorf("storage root = %v, want %v", got, want)
	}

	if len(listeners) != 1 || !listeners[0].TLS {
		t.Errorf("listener status does not report TLS: %+v", listeners)
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

func TestStartRejectsLocalHTTPSWithoutDataDirectory(t *testing.T) {
	m := New(testSurfaces(), zerolog.Nop())
	err := m.Start(context.Background(), Config{
		Address:    "127.0.0.1:0",
		LocalHTTPS: true,
	})
	if err == nil {
		_ = m.Close()
		t.Fatal("Start accepted local HTTPS without a data directory")
	}
	if !strings.Contains(err.Error(), "data directory") {
		t.Errorf("error does not explain the missing data directory: %v", err)
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
		{"readiness has no host requirement", "", "/ready", SurfaceSameOrigin, 200},
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
		{"unknown host falls through", "tenant.example.com", "/", SurfaceFrontend, 200},
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

func TestServesLocalHTTPSWithGeneratedCertificate(t *testing.T) {
	port := freePort(t)
	cfg := testConfig()
	cfg.Address = fmt.Sprintf("127.0.0.1:%d", port)
	cfg.DataDir = t.TempDir()
	cfg.LocalHTTPS = true
	// A live TLS test must never modify the developer or CI trust stores.
	cfg.SkipLocalTrustInstall = true

	m := New(testSurfaces(), zerolog.Nop())
	if err := m.Start(context.Background(), cfg); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })

	rootPEM, err := os.ReadFile(filepath.Join(
		cfg.DataDir, "caddy", "pki", "authorities", "local", "root.crt",
	))
	if err != nil {
		t.Fatalf("read generated root CA: %v", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(rootPEM) {
		t.Fatal("generated root CA is not valid PEM")
	}

	client := &http.Client{Transport: &http.Transport{
		ForceAttemptHTTP2: true,
		TLSClientConfig: &tls.Config{ //nolint:gosec // trusted test CA, verification remains enabled
			RootCAs:    roots,
			ServerName: "localhost",
			MinVersion: tls.VersionTLS12,
		},
	}}
	res, err := client.Get(fmt.Sprintf("https://127.0.0.1:%d/health", port)) //nolint:noctx // test request
	if err != nil {
		t.Fatalf("HTTPS request: %v", err)
	}
	defer res.Body.Close() //nolint:errcheck // response body of a test request

	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", res.StatusCode, body)
	}
	if got := string(body); got != SurfaceSameOrigin {
		t.Errorf("served by %q, want %q", got, SurfaceSameOrigin)
	}
	if res.TLS == nil || res.TLS.Version < tls.VersionTLS12 {
		t.Errorf("response was not served over the configured TLS listener: %+v", res.TLS)
	}
}

func TestFrontendPreservesTenantHostAndPath(t *testing.T) {
	port := freePort(t)
	cfg := testConfig()
	cfg.Address = fmt.Sprintf("127.0.0.1:%d", port)
	surfaces := testSurfaces()
	surfaces[SurfaceFrontend] = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frontend-Host", r.Host)
		_, _ = io.WriteString(w, "frontend:"+r.URL.RequestURI())
	})
	m := New(surfaces, zerolog.Nop())
	if err := m.Start(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Close() })
	client := &http.Client{Timeout: 5 * time.Second}
	for _, path := range []string{"/", "/kill/123?tab=items", "/api/site", "/auth/me", "/images/types/1", "/ws", "/health"} {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d%s", port, path), nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "tenant.example.com"
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		want := SurfaceSameOrigin
		if path == "/ws" {
			want = SurfaceWS
		}
		if path == "/" || strings.HasPrefix(path, "/kill/") {
			want = "frontend:" + path
			if got := res.Header.Get("X-Frontend-Host"); got != req.Host {
				t.Errorf("host = %q", got)
			}
		}
		if res.StatusCode != 200 || string(body) != want {
			t.Errorf("%s: status %d, body %q, want %q", path, res.StatusCode, body, want)
		}
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
	server := serverConfig(t, cfg)
	routes, ok := server["routes"].([]any)
	if !ok || len(routes) == 0 {
		t.Fatalf("no routes in the built config")
	}
	return routes
}

func serverConfig(t *testing.T, cfg map[string]any) map[string]any {
	t.Helper()
	apps := cfg["apps"].(map[string]any)
	httpApp := apps["http"].(map[string]any)
	servers := httpApp["servers"].(map[string]any)
	return servers["shrike"].(map[string]any)
}

func lastRoute(t *testing.T, cfg map[string]any) map[string]any {
	t.Helper()
	routes := allRoutes(t, cfg)
	return routes[len(routes)-1].(map[string]any)
}

func originHandler(t *testing.T, route map[string]any) map[string]any {
	t.Helper()
	handlers, ok := route["handle"].([]any)
	if !ok || len(handlers) < 2 {
		t.Fatalf("route has no access and origin handlers: %+v", route)
	}
	return handlers[len(handlers)-1].(map[string]any)
}
