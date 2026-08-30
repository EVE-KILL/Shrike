package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestGuardRequiresUserAgentOnlyForAPI(t *testing.T) {
	guard := NewRequestGuard()
	handler := guard.Wrap(http.HandlerFunc(func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, path := range []string{"/api/killmails", "/api/mcp", "/api"} {
		request := httptest.NewRequest(http.MethodGet, "http://example.test"+path, nil)
		request.Header.Del("User-Agent")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Errorf("%s without User-Agent returned %d: %s", path, response.Code, response.Body.String())
		}
		if got := response.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("%s rejection Cache-Control = %q", path, got)
		}
		if got := response.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("%s rejection CORS origin = %q", path, got)
		}
	}

	for _, path := range []string{"/images/types/42/icon", "/auth/eve", "/health"} {
		request := httptest.NewRequest(http.MethodGet, "http://example.test"+path, nil)
		request.Header.Del("User-Agent")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Errorf("%s without User-Agent returned %d", path, response.Code)
		}
	}

	identified := httptest.NewRequest(http.MethodGet, "http://example.test/api/killmails", nil)
	identified.Header.Set("User-Agent", "evekill-guard-test/1.0")
	identifiedResponse := httptest.NewRecorder()
	handler.ServeHTTP(identifiedResponse, identified)
	if identifiedResponse.Code != http.StatusNoContent {
		t.Fatalf("identified API request returned %d", identifiedResponse.Code)
	}
	for _, header := range []string{
		"RateLimit-Limit", "RateLimit-Remaining", "RateLimit-Reset", "Retry-After",
		"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset",
	} {
		if got := identifiedResponse.Header().Get(header); got != "" {
			t.Errorf("identified API response unexpectedly has %s = %q", header, got)
		}
	}

	preflight := httptest.NewRecorder()
	handler.ServeHTTP(preflight, httptest.NewRequest(http.MethodOptions, "http://example.test/api/killmails", nil))
	if preflight.Code != http.StatusNoContent {
		t.Fatalf("API preflight returned %d", preflight.Code)
	}
}
