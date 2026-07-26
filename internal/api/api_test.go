package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthAndSchemaPathsForEachHumaSurface(t *testing.T) {
	opts := Options{Version: "test-version", Commit: "test-commit"}

	for _, tc := range []struct {
		name       string
		handler    http.Handler
		schemaPath string
	}{
		{"public", Public(opts), "/schemas/Health.json"},
		{"private", Private(opts), "/api/schemas/Health.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			health := httptest.NewRecorder()
			tc.handler.ServeHTTP(
				health,
				httptest.NewRequest(http.MethodGet, "http://example.com/health", nil),
			)
			if health.Code != http.StatusOK {
				t.Fatalf("health status = %d, want 200 (body %q)", health.Code, health.Body.String())
			}
			if got := health.Header().Get("Link"); !strings.Contains(got, "<"+tc.schemaPath+">") {
				t.Fatalf("health Link = %q, want path %q", got, tc.schemaPath)
			}
			if body := health.Body.String(); !strings.Contains(body, `"version":"test-version"`) ||
				!strings.Contains(body, `"commit":"test-commit"`) {
				t.Errorf("health body does not identify the build: %s", body)
			}

			schema := httptest.NewRecorder()
			tc.handler.ServeHTTP(
				schema,
				httptest.NewRequest(http.MethodGet, "http://example.com"+tc.schemaPath, nil),
			)
			if schema.Code != http.StatusOK {
				t.Fatalf("schema status = %d, want 200 (body %q)", schema.Code, schema.Body.String())
			}
		})
	}
}
