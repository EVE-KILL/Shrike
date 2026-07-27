package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestGuardRequiresUserAgentOnlyForAPI(t *testing.T) {
	guard := newRequestGuard(10, time.Minute, time.Now)
	handler := guard.Wrap(http.HandlerFunc(func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		w.WriteHeader(http.StatusNoContent)
	}))

	apiRequest := httptest.NewRequest(
		http.MethodGet,
		"http://example.test/api/killmails",
		nil,
	)
	apiResponse := httptest.NewRecorder()
	handler.ServeHTTP(apiResponse, apiRequest)
	if apiResponse.Code != http.StatusBadRequest {
		t.Fatalf(
			"API without User-Agent returned %d: %s",
			apiResponse.Code,
			apiResponse.Body.String(),
		)
	}
	if got := apiResponse.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("API rejection Cache-Control = %q", got)
	}
	if got := apiResponse.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("API rejection CORS origin = %q", got)
	}

	for _, path := range []string{"/images/types/42/icon", "/auth/eve", "/health"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(
			response,
			httptest.NewRequest(http.MethodGet, "http://example.test"+path, nil),
		)
		if response.Code != http.StatusNoContent {
			t.Errorf("%s without User-Agent returned %d", path, response.Code)
		}
		if response.Header().Get("RateLimit-Limit") != "" {
			t.Errorf("%s unexpectedly has API rate-limit headers", path)
		}
	}

	preflight := httptest.NewRecorder()
	handler.ServeHTTP(
		preflight,
		httptest.NewRequest(
			http.MethodOptions,
			"http://example.test/api/killmails",
			nil,
		),
	)
	if preflight.Code != http.StatusNoContent {
		t.Fatalf("API preflight returned %d", preflight.Code)
	}
}

func TestRequestGuardLimitsEachClientAddressIndependently(t *testing.T) {
	now := time.Date(2026, 7, 27, 14, 0, 0, 0, time.UTC)
	guard := newRequestGuard(2, time.Minute, func() time.Time {
		return now
	})
	handler := guard.Wrap(http.HandlerFunc(func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := func(address string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(
			http.MethodGet,
			"http://example.test/api/killmails",
			nil,
		)
		r.Header.Set("User-Agent", "evekill-guard-test/1.0")
		r.Header.Set("CF-Connecting-IP", address)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}

	first := request("192.0.2.10")
	if first.Code != http.StatusNoContent ||
		first.Header().Get("RateLimit-Limit") != "2" ||
		first.Header().Get("RateLimit-Remaining") != "1" ||
		first.Header().Get("RateLimit-Reset") != "60" {
		t.Fatalf("first response = %d, headers %v", first.Code, first.Header())
	}
	second := request("192.0.2.10")
	if second.Code != http.StatusNoContent ||
		second.Header().Get("RateLimit-Remaining") != "0" {
		t.Fatalf("second response = %d, headers %v", second.Code, second.Header())
	}
	limited := request("192.0.2.10")
	if limited.Code != http.StatusTooManyRequests ||
		limited.Header().Get("Retry-After") != "60" ||
		limited.Header().Get("X-RateLimit-Remaining") != "0" ||
		limited.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("limited response = %d, headers %v", limited.Code, limited.Header())
	}
	if got := limited.Header().Get("Access-Control-Expose-Headers"); !strings.Contains(got, "Retry-After") ||
		!strings.Contains(got, "RateLimit-Remaining") {
		t.Fatalf("limited response does not expose rate headers: %q", got)
	}

	otherClient := request("192.0.2.11")
	if otherClient.Code != http.StatusNoContent {
		t.Fatalf("other client returned %d", otherClient.Code)
	}

	now = now.Add(time.Minute)
	reset := request("192.0.2.10")
	if reset.Code != http.StatusNoContent ||
		reset.Header().Get("RateLimit-Remaining") != "1" {
		t.Fatalf("reset response = %d, headers %v", reset.Code, reset.Header())
	}
}

func TestClientAddressPrefersValidatedForwardingHeaders(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"http://example.test/api",
		nil,
	)
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("X-Forwarded-For", "198.51.100.8, 127.0.0.1")
	if got := clientAddress(request); got != "198.51.100.8" {
		t.Fatalf("forwarded address = %q", got)
	}
	request.Header.Set("CF-Connecting-IP", "2001:db8::42")
	if got := clientAddress(request); got != "2001:db8::42" {
		t.Fatalf("Cloudflare address = %q", got)
	}
	request.Header.Set("CF-Connecting-IP", "not-an-ip")
	if got := clientAddress(request); got != "198.51.100.8" {
		t.Fatalf("invalid Cloudflare fallback = %q", got)
	}
}
