package frontend

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUmamiRetainsExistingWebsiteAndHostAwareProductionTracking(t *testing.T) {
	for _, host := range []string{"eve-kill.com", "void.eve-kill.com", "killboard.example.org:443"} {
		t.Run(host, func(t *testing.T) {
			request := httptest.NewRequest("GET", "https://"+host+"/", nil)
			meta, err := resolveMetadata(request, true, 200, Metadata{NeedsWASM: true, StructuredData: map[string]string{"name": "fixture"}})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(meta.CSP, "connect-src 'self' "+analyticsOrigin) || !strings.Contains(meta.CSP, "script-src 'self' "+analyticsOrigin) || !strings.Contains(meta.CSP, "'wasm-unsafe-eval'") || !strings.Contains(meta.CSP, "'sha256-") {
				t.Fatalf("analytics must preserve the existing CSP protections: %s", meta.CSP)
			}
			var body strings.Builder
			if err := pageHead(meta, false).Render(context.Background(), &body); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(body.String(), `src="https://analytics.karbowiak.dk/script.js"`) || !strings.Contains(body.String(), `data-website-id="`+analyticsWebsiteID+`"`) || !strings.Contains(body.String(), `data-domains="`+meta.AnalyticsHost+`"`) {
				t.Fatalf("missing existing Umami integration: %s", body.String())
			}
		})
	}
}

func TestUmamiDoesNotTrackDevelopmentOrLocalOrigins(t *testing.T) {
	for _, production := range []bool{false, true} {
		for _, host := range []string{"localhost:4000", "void.localhost:4200", "127.0.0.1:4487", "[::1]:4000"} {
			meta, err := resolveMetadata(httptest.NewRequest("GET", "http://"+host+"/", nil), production, 200, Metadata{})
			if err != nil {
				t.Fatal(err)
			}
			if meta.AnalyticsHost != "" || strings.Contains(meta.CSP, analyticsOrigin) {
				t.Fatalf("local origin enabled analytics: %s", host)
			}
		}
	}
	meta, err := resolveMetadata(httptest.NewRequest("GET", "https://eve-kill.com/", nil), false, 200, Metadata{})
	if err != nil {
		t.Fatal(err)
	}
	if meta.AnalyticsHost != "" {
		t.Fatal("development enabled analytics")
	}
}
