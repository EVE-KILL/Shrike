package ingress

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

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

func startTestServer(t *testing.T, cfg Config, surfaces map[string]http.Handler) string {
	t.Helper()
	log := zerolog.Nop()
	server, err := NewServer(cfg, surfaces, log)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", server.Addr)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, server, listener, log) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("serve: %v", err)
			}
		case <-time.After(12 * time.Second):
			t.Error("listener did not shut down")
		}
	})
	scheme := "http"
	if cfg.TLSCertFile != "" {
		scheme = "https"
	}
	return scheme + "://" + listener.Addr().String()
}

func TestServesEachSurface(t *testing.T) {
	base := startTestServer(t, testConfig(), testSurfaces())
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

func TestFrontendPreservesTenantHostAndPath(t *testing.T) {
	surfaces := testSurfaces()
	surfaces[SurfaceFrontend] = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frontend-Host", r.Host)
		_, _ = io.WriteString(w, "frontend:"+r.URL.RequestURI())
	})
	base := startTestServer(t, testConfig(), surfaces)
	client := &http.Client{Timeout: 5 * time.Second}
	for _, path := range []string{"/", "/kill/123?tab=items", "/api/site", "/auth/me", "/images/types/1", "/ws", "/health"} {
		req, err := http.NewRequest(http.MethodGet, base+path, nil)
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

func TestHTTPSWithCertificateFiles(t *testing.T) {
	fixture := httptest.NewTLSServer(http.NotFoundHandler())
	defer fixture.Close()
	cert := fixture.TLS.Certificates[0]
	key, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cfg := testConfig()
	cfg.TLSCertFile = filepath.Join(dir, "cert.pem")
	cfg.TLSKeyFile = filepath.Join(dir, "key.pem")
	if err := os.WriteFile(cfg.TLSCertFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.TLSKeyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}), 0600); err != nil {
		t.Fatal(err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(fixture.Certificate())
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			ForceAttemptHTTP2: true,
			TLSClientConfig:   &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
		},
	}
	t.Cleanup(client.CloseIdleConnections)
	base := startTestServer(t, cfg, testSurfaces())
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, base+"/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.TLS == nil || response.ProtoMajor != 2 {
		t.Fatalf("HTTPS health response: status %d, protocol %s, TLS %v", response.StatusCode, response.Proto, response.TLS != nil)
	}
}

func TestServerRejectsInvalidConfiguration(t *testing.T) {
	for _, cfg := range []Config{
		{Address: "invalid address"},
		{Address: "127.0.0.1:0", TLSCertFile: "cert.pem"},
		{Address: "127.0.0.1:0", TLSKeyFile: "key.pem"},
		{Address: "127.0.0.1:0", TLSCertFile: "missing-cert.pem", TLSKeyFile: "missing-key.pem"},
	} {
		if _, err := NewServer(cfg, testSurfaces(), zerolog.Nop()); err == nil {
			t.Errorf("accepted invalid config %+v", cfg)
		}
	}
	for name := range testSurfaces() {
		surfaces := testSurfaces()
		delete(surfaces, name)
		if _, err := NewServer(testConfig(), surfaces, zerolog.Nop()); err == nil {
			t.Errorf("accepted missing handler %s", name)
		}
	}
}

func TestShutdownDrainsInFlightRequest(t *testing.T) {
	started, draining, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	surfaces := testSurfaces()
	surfaces[SurfaceFrontend] = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-release:
			w.WriteHeader(http.StatusNoContent)
		case <-r.Context().Done():
		}
	})
	log := zerolog.Nop()
	server, err := NewServer(testConfig(), surfaces, log)
	if err != nil {
		t.Fatal(err)
	}
	server.RegisterOnShutdown(func() { close(draining) })
	timeout, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	ctx, stop := context.WithCancel(timeout)
	defer stop()
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", server.Addr)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, server, listener, log) }()

	req, err := http.NewRequestWithContext(timeout, http.MethodGet, "http://"+listener.Addr().String()+"/in-flight", nil)
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		status int
		err    error
	}
	response := make(chan result, 1)
	go func() {
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			response <- result{err: err}
			return
		}
		_ = res.Body.Close()
		response <- result{status: res.StatusCode}
	}()
	select {
	case <-started:
	case <-timeout.Done():
		t.Fatal("request never reached the listener")
	}
	stop()
	select {
	case <-draining:
	case <-timeout.Done():
		t.Fatal("shutdown did not start")
	}
	close(release)
	if got := <-response; got.err != nil || got.status != http.StatusNoContent {
		t.Fatalf("in-flight request ended with status %d, error %v", got.status, got.err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-timeout.Done():
		t.Fatal("listener did not finish draining")
	}
}
