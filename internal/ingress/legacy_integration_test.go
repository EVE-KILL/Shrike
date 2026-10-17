package ingress

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/rs/zerolog"
)

func TestLegacySubdomainsThroughCaddy(t *testing.T) {
	type received struct {
		Surface, Host, URI, Body, Method, ForwardedHost, Scheme, Cookie string
	}
	surfaces := map[string]http.Handler{}
	for _, surface := range []string{SurfaceSameOrigin, SurfaceWS, SurfaceFrontend} {
		surfaces[surface] = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.NewEncoder(w).Encode(received{
				Surface: surface, Host: r.Host, URI: r.URL.RequestURI(),
				Body: string(body), Method: r.Method,
				ForwardedHost: r.Header.Get("X-Forwarded-Host"),
				Scheme:        r.Header.Get("X-Forwarded-Proto"), Cookie: r.Header.Get("Cookie"),
			})
		})
	}
	cfg := testConfig()
	cfg.Address = fmt.Sprintf("127.0.0.1:%d", freePort(t))
	cfg.LegacyDomain = "example.test"
	m := New(surfaces, zerolog.Nop())
	if err := m.Start(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Close() })

	for _, tc := range []struct{ host, path, surface, want string }{
		{"api.example.test", "/characters/123?x=a%2Fb&x=c", SurfaceSameOrigin, "/api/characters/123?x=a%2Fb&x=c"},
		{"api.example.test:443", "/api/characters/123", SurfaceSameOrigin, "/api/characters/123"},
		{"api.example.test", "/apiary/a%2Fb", SurfaceSameOrigin, "/api/apiary/a%2Fb"},
		{"images.example.test", "/types/587/icon?size=64", SurfaceSameOrigin, "/images/types/587/icon?size=64"},
		{"images.example.test", "/images/types/587/icon", SurfaceSameOrigin, "/images/types/587/icon"},
		{"ws.example.test", "/status", SurfaceWS, "/ws/status"},
		{"ws.example.test", "/", SurfaceWS, "/ws/"},
		{"ws.example.test", "/ws/status", SurfaceWS, "/ws/status"},
		{"mcp.example.test", "/?token=redacted", SurfaceSameOrigin, "/api/mcp?token=redacted"},
		{"mcp.example.test", "/mcp", SurfaceSameOrigin, "/api/mcp"},
		{"mcp.example.test", "/api/mcp", SurfaceSameOrigin, "/api/mcp"},
		{"api.example.test", "/health", SurfaceSameOrigin, "/health"},
		{"example.test", "/api/characters/123", SurfaceSameOrigin, "/api/characters/123"},
		{"void.example.test", "/status", SurfaceFrontend, "/status"},
		{"api.other.test", "/characters/123", SurfaceFrontend, "/characters/123"},
	} {
		t.Run(tc.host+tc.path, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "http://"+cfg.Address+tc.path, strings.NewReader("original body"))
			if err != nil {
				t.Fatal(err)
			}
			req.Host = tc.host
			req.Header.Set("X-Forwarded-Host", "spoofed.example")
			req.Header.Set("X-Forwarded-Proto", "https")
			req.Header.Set("Cookie", "session=test")
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = res.Body.Close() }()
			var got received
			if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			want := received{Surface: tc.surface, Host: tc.host, URI: tc.want,
				Body: "original body", Method: http.MethodPost,
				ForwardedHost: tc.host, Scheme: "https", Cookie: "session=test"}
			if res.StatusCode != http.StatusOK || got != want {
				t.Fatalf("status=%d received=%+v, want %+v", res.StatusCode, got, want)
			}
		})
	}
}

func TestLegacyMCPInitializesThroughCaddy(t *testing.T) {
	service := api.New(api.Options{Version: "test", RequestGuard: api.NewRequestGuard()})
	t.Cleanup(service.Close)
	surfaces := testSurfaces()
	surfaces[SurfaceSameOrigin] = service.Site()
	cfg := testConfig()
	cfg.Address = fmt.Sprintf("127.0.0.1:%d", freePort(t))
	cfg.LegacyDomain = "eve-kill.com"
	m := New(surfaces, zerolog.Nop())
	if err := m.Start(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Close() })
	for _, path := range []string{"/", "/mcp", "/api/mcp"} {
		body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"ingress-test","version":"1"}}}`
		req, err := http.NewRequest(http.MethodPost, "http://"+cfg.Address+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "mcp.eve-kill.com"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		payload, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if res.StatusCode != http.StatusOK || !strings.Contains(string(payload), `"name":"evekill-mcp"`) {
			t.Fatalf("MCP %s: status %d body %s", path, res.StatusCode, payload)
		}
	}
}

func TestLegacyDomainRejectsURLAndWildcard(t *testing.T) {
	for _, domain := range []string{"https://example.test", "*.example.test", "example.test:4000"} {
		cfg := testConfig()
		cfg.LegacyDomain = domain
		_, _, _, err := newTestManager(t, cfg).buildConfig()
		if err == nil {
			t.Errorf("accepted invalid legacy domain %q", domain)
		}
	}
}
