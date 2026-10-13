package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFixtureFaultIsolationAndRecovery(t *testing.T) {
	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	faults := &fixtureFaults{enabled: true}
	handler := faults.wrap(upstream)
	set := func(body string) int {
		r := httptest.NewRecorder()
		handler.ServeHTTP(r, httptest.NewRequest("POST", "/__fixture/faults", strings.NewReader(body)))
		return r.Code
	}
	if set(`{"path":"/api/conflicts/battles","query":{"map":"true"},"status":503}`) != 204 {
		t.Fatal("unable to configure local failure")
	}
	for path, status := range map[string]int{"/api/conflicts/battles?map=true": 503, "/api/conflicts/battles": 200, "/api/map/scope": 200} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code != status {
			t.Fatalf("%s returned %d, want %d", path, response.Code, status)
		}
		if status == 503 && response.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("injected error is cacheable")
		}
	}
	if set(`{}`) != 204 {
		t.Fatal("unable to clear fault")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/conflicts/battles?map=true", nil))
	if response.Code != 200 {
		t.Fatal("recovery still failed")
	}
	for _, body := range []string{`{"path":"https://external.example"}`, `{"path":"/auth/me"}`, `{"status":302}`, `{"delayMs":10001}`} {
		if set(body) != 400 {
			t.Fatal("accepted unsafe or unsupported fault configuration")
		}
	}
	disabled := (&fixtureFaults{}).wrap(upstream)
	response = httptest.NewRecorder()
	disabled.ServeHTTP(response, httptest.NewRequest("POST", "/__fixture/faults", strings.NewReader(`{"status":503}`)))
	if response.Code != 200 {
		t.Fatal("disabled fault control intercepted the upstream")
	}
}

func TestFixtureFaultDelayRespectsCancellation(t *testing.T) {
	faults := &fixtureFaults{enabled: true, rule: fixtureFaultRule{Path: "/api/map/scope", DelayMS: 10000}}
	handler := faults.wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("cancelled delay reached upstream") }))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest("GET", "/api/map/scope", nil).WithContext(ctx)
	start := time.Now()
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if time.Since(start) > time.Second {
		t.Fatal("cancelled test request waited for the injected delay")
	}
}

func TestFixtureReadResponseCanReplaceOnlyLocalReadHandoffs(t *testing.T) {
	calls := 0
	faults := &fixtureFaults{enabled: true}
	handler := faults.wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(201) }))
	configure := func(body string) int {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("POST", "/__fixture/faults", strings.NewReader(body)))
		return w.Code
	}
	if configure(`{"path":"/api/admin/wallet/authorize","response":{"url":"/admin/wallet?wallet_auth=ok"}}`) != 204 {
		t.Fatal("read handoff fixture rejected")
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/admin/wallet/authorize", nil))
	if w.Code != 200 || w.Header().Get("Cache-Control") != "private, no-store" || !strings.Contains(w.Body.String(), "wallet_auth=ok") || calls != 0 {
		t.Fatal("fixture handoff reached upstream or became cacheable")
	}
	if configure(`{"path":"/api/admin/wallet/sync","method":"POST","response":{"queued":true}}`) != 400 {
		t.Fatal("write response mock accepted")
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("POST", "/api/admin/wallet/authorize", nil))
	if w.Code != 201 || calls != 1 {
		t.Fatal("read mock intercepted a write")
	}
}
