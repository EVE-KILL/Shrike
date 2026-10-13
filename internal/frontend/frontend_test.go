package frontend

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPageAndAssetRouting(t *testing.T) {
	handler := New(Options{})
	for _, tt := range []struct {
		method, path      string
		status            int
		contentType, body string
	}{
		{"GET", "/", 200, "text/html", "EVE-KILL"},
		{"GET", "/post", 200, "text/html", "Post from clipboard"},
		{"HEAD", "/", 200, "text/html", ""},
		{"GET", "/kill/123", 404, "text/html", "Page not found"},
		{"GET", "/assets/site.css", 200, "text/css", "color-scheme"},
		{"GET", "/assets/missing.css", 404, "text/plain", "404"},
		{"GET", "/_nuxt/old.js", 404, "text/html", "Page not found"},
		{"POST", "/", 405, "text/plain", "Method Not Allowed"},
	} {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, nil))
			if w.Code != tt.status || !strings.HasPrefix(w.Header().Get("Content-Type"), tt.contentType) || !strings.Contains(w.Body.String(), tt.body) {
				t.Fatalf("response = %d %s %q", w.Code, w.Header().Get("Content-Type"), w.Body.String())
			}
			if strings.Contains(w.Body.String(), "_nuxt") {
				t.Fatal("page unexpectedly loads the Nuxt runtime")
			}
			if tt.contentType == "text/html" && w.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatal("migration placeholder must not be cached")
			}
		})
	}
}
