package frontend

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStatusPageLoadsLiveDashboard(t *testing.T) {
	handler := New(Options{})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/status", nil))
	if response.Code != 200 {
		t.Fatalf("status = %d", response.Code)
	}
	for _, want := range []string{"System Status", "data-status-panels", "status.css", "status.js", "Queues, workers, databases"} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("status page missing %q", want)
		}
	}
	if got := response.Header().Get("X-Robots-Tag"); got != "noindex, nofollow" {
		t.Errorf("robots = %q", got)
	}
	for _, path := range []string{"/assets/status.css", "/assets/status.js"} {
		asset := httptest.NewRecorder()
		handler.ServeHTTP(asset, httptest.NewRequest("GET", path, nil))
		if asset.Code != 200 || asset.Body.Len() == 0 {
			t.Errorf("%s status = %d, bytes = %d", path, asset.Code, asset.Body.Len())
		}
	}
}
