package ingress

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
)

type healthyDB struct{}

func (healthyDB) Ping(context.Context) error {
	return nil
}

func (healthyDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("unexpected Query")
}

func (healthyDB) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("unexpected QueryRow")
}

// Exercise the actual Huma surfaces through Caddy. The generic live routing
// test deliberately uses marker handlers to make route ownership observable,
// but marker handlers cannot catch a surface-specific generated schema link.
func TestPublicAndPrivateHealthSchemaLinksResolve(t *testing.T) {
	port := freePort(t)
	cfg := testConfig()
	cfg.Address = fmt.Sprintf("127.0.0.1:%d", port)

	opts := api.Options{Version: "test-version", Commit: "test-commit", DB: healthyDB{}}
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
		public     bool
	}{
		{"public", "api.example.com", "/schemas/health-response.json", true},
		{"private", "tenant.example.com", "/api/schemas/Health.json", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := requestWithHost(t, base+"/health", tc.host)
			body, _ := io.ReadAll(res.Body)
			_ = res.Body.Close()
			if res.StatusCode != http.StatusOK {
				t.Fatalf("health status = %d, want 200 (body %q)", res.StatusCode, body)
			}
			if tc.public {
				if !strings.Contains(string(body), `"ok":true`) ||
					!strings.Contains(string(body), `"timestamp":"`) {
					t.Errorf("public health does not preserve legacy shape: %s", body)
				}
				var payload map[string]any
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatal(err)
				}
				schemaURL, err := url.Parse(payload["$schema"].(string))
				if err != nil {
					t.Fatal(err)
				}
				if schemaURL.Path != tc.schemaPath {
					t.Fatalf("$schema path = %q, want %q", schemaURL.Path, tc.schemaPath)
				}
			} else if !strings.Contains(string(body), `"version":"test-version"`) ||
				!strings.Contains(string(body), `"commit":"test-commit"`) {
				t.Errorf("private health body does not identify the build: %s", body)
			}

			if !strings.Contains(res.Header.Get("Link"), "<"+tc.schemaPath+">") {
				t.Fatalf("health Link = %q, want path %q", res.Header.Get("Link"), tc.schemaPath)
			}
			schema := requestWithHost(t, base+tc.schemaPath, tc.host)
			schemaBody, _ := io.ReadAll(schema.Body)
			_ = schema.Body.Close()
			if schema.StatusCode != http.StatusOK {
				t.Fatalf("schema status = %d, want 200 (body %q)", schema.StatusCode, schemaBody)
			}
			if string(schemaBody) == "null" ||
				!strings.Contains(string(schemaBody), `"$schema"`) {
				t.Fatalf("schema body is not usable: %s", schemaBody)
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
