package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestViewportFixtureCannotEmbedExternalDestinations(t *testing.T) {
	for _, raw := range []string{"https://eve-kill.com", "//eve-kill.com", "/\\evil.example", "/\n/evil.example"} {
		if _, err := fixtureViewportPath(raw); err == nil {
			t.Fatalf("external or malformed frame accepted: %q", raw)
		}
	}
	if got, err := fixtureViewportPath("/campaign/PrizeWait00001/comments?side=1"); err != nil || !strings.Contains(got, "side=1") || !strings.Contains(got, "__fixture_frame=1") {
		t.Fatalf("local query not preserved: %q %v", got, err)
	}
}
func TestFramePermissionChangesOnlyTheMarkedLocalFixtureRequest(t *testing.T) {
	handler := fixtureFrameHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
		w.Write([]byte("page"))
	}))
	for _, marked := range []bool{false, true} {
		path := "/campaign/PrizeWait00001"
		if marked {
			path += "?__fixture_frame=1"
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		wanted := "frame-ancestors 'none'"
		if marked {
			wanted = "frame-ancestors 'self'"
		}
		if !strings.Contains(response.Header().Get("Content-Security-Policy"), wanted) || response.Body.String() != "page" {
			t.Fatalf("fixture frame altered other content: %v %s", response.Header(), response.Body.String())
		}
	}
}
