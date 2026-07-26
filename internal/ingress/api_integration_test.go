package ingress

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/rs/zerolog"
)

// Exercise the actual Huma surfaces through Caddy. The generic live routing
// test deliberately uses marker handlers to make route ownership observable,
// but marker handlers cannot catch a surface-specific generated schema link.
func TestPublicAndPrivateHealthSchemaLinksResolve(t *testing.T) {
	port := freePort(t)
	cfg := testConfig()
	cfg.Address = fmt.Sprintf("127.0.0.1:%d", port)

	opts := api.Options{Version: "test-version", Commit: "test-commit"}
	m := New(map[string]http.Handler{
		SurfacePrivate: api.Private(opts),
		SurfacePublic:  api.Public(opts),
		SurfaceWS:      api.WS(opts),
		SurfaceImages:  api.Images(opts),
	}, zerolog.Nop())
	if err := m.Start(context.Background(), cfg); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	for _, tc := range []struct {
		name       string
		host       string
		schemaPath string
	}{
		{"public", "api.example.com", "/schemas/Health.json"},
		{"private", "tenant.example.com", "/api/schemas/Health.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := requestWithHost(t, base+"/health", tc.host)
			body, _ := io.ReadAll(res.Body)
			_ = res.Body.Close()
			if res.StatusCode != http.StatusOK {
				t.Fatalf("health status = %d, want 200 (body %q)", res.StatusCode, body)
			}
			if !strings.Contains(res.Header.Get("Link"), "<"+tc.schemaPath+">") {
				t.Fatalf("health Link = %q, want path %q", res.Header.Get("Link"), tc.schemaPath)
			}
			if !strings.Contains(string(body), `"version":"test-version"`) ||
				!strings.Contains(string(body), `"commit":"test-commit"`) {
				t.Errorf("health body does not identify the build: %s", body)
			}

			schema := requestWithHost(t, base+tc.schemaPath, tc.host)
			schemaBody, _ := io.ReadAll(schema.Body)
			_ = schema.Body.Close()
			if schema.StatusCode != http.StatusOK {
				t.Fatalf("schema status = %d, want 200 (body %q)", schema.StatusCode, schemaBody)
			}
		})
	}
}

func requestWithHost(t *testing.T, url, host string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = host
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s with Host %s: %v", url, host, err)
	}
	return res
}
