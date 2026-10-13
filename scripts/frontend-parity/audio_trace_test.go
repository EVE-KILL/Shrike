package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestAudioObserverRequiresAnExplicitLocalComparisonFrame(t *testing.T) {
	body := `<html><head></head><body>Map</body></html>`
	handler := fixtureFrameHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'")
		_, _ = w.Write([]byte(body))
	}))
	for _, q := range []string{"", "?__fixture_audio=1", "?__fixture_frame=1", "?__fixture_frame=1&__fixture_audio=1"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/map"+q, nil))
		want := strings.Contains(q, "__fixture_frame=1") && strings.Contains(q, "__fixture_audio=1")
		if strings.Contains(w.Body.String(), "audio-trace.js") != want {
			t.Fatal("audio observer leaked outside its explicit frame")
		}
		if want && (w.Header().Get("Content-Length") != "" || !strings.Contains(w.Header().Get("Content-Security-Policy"), "'self'")) {
			t.Fatal("instrumented response has stale length or incorrect framing")
		}
	}
}
