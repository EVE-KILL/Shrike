package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFixtureGlobalSiteOnlyNormalizesTheIsolatedReviewHost(t *testing.T) {
	for _, host := range []string{"frontend-parity.localhost:4470", "void.localhost:4470"} {
		request := httptest.NewRequest("GET", "http://"+host+"/settings", nil)
		request.Header.Set("Origin", "http://"+host)
		request.Header.Set("Cookie", "evelogin=disposable-session")
		handler := fixtureGlobalSite(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			want := host
			if host == "frontend-parity.localhost:4470" {
				want = "localhost"
			}
			if r.Host != want || r.Header.Get("Origin") != "http://"+host || r.Header.Get("Cookie") != "evelogin=disposable-session" {
				t.Fatal("fixture changed the wrong host or request identity")
			}
		}))
		handler.ServeHTTP(httptest.NewRecorder(), request)
		if request.Host != host {
			t.Fatal("fixture mutated the original request")
		}
	}
}

func TestComparisonUpstreamsStayOnLoopback(t *testing.T) {
	for _, raw := range []string{"http://127.0.0.1:4200", "http://localhost:4450/"} {
		if _, err := localURL(raw); err != nil {
			t.Errorf("local origin rejected: %s", raw)
		}
	}
	for _, raw := range []string{"https://eve-kill.com", "http://100.111.3.8:4200", "http://user:secret@localhost:4200", "http://localhost:4200/api", "http://localhost:4200?target=remote", "//localhost:4200"} {
		if _, err := localURL(raw); err == nil {
			t.Errorf("nonlocal or decorated origin accepted: %s", raw)
		}
	}
	for _, entry := range cases {
		if len(entry.id) != 14 {
			t.Fatalf("invalid campaign fixture id %q", entry.id)
		}
	}
}
