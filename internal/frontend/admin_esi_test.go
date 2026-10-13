package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminESIPageAndFragmentKeepSeededEntityAndLogsPrivate(t *testing.T) {
	var logQuery string
	reads := []string{}
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads = append(reads, r.URL.Path)
		var data any
		switch r.URL.Path {
		case "/auth/me":
			data = map[string]any{"user": settingsUser{ID: 42, Name: "Owner", Admin: true}}
		case "/api/admin/overview":
			data = map[string]any{"esi": map[string]any{"requests24h": 77, "errors24h": 2, "errorRate": 2.6}}
		case "/api/admin/esi-logs":
			logQuery = r.URL.RawQuery
			data = map[string]any{"rows": []map[string]any{{"id": 1, "endpoint": "/characters/43/killmails/recent/", "character_name": "<script>safe</script>"}}, "total": 1, "pages": 1, "sources": []string{"fixture"}}
		default:
			t.Fatalf("unexpected blocking read %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(data)
	})
	handler := New(Options{Production: true, AccountAPI: api})
	for _, path := range []string{"/admin/esi?character_id=43&character_name=Fixture%20Reader", "/_fragments/admin/esi?character_id=43&character_name=Fixture%20Reader"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code != 200 || response.Header().Get("Cache-Control") != "private, no-store" || response.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
			t.Fatalf("private ESI route %s=%d", path, response.Code)
		}
		if !strings.Contains(logQuery, "character_id=43") || !strings.Contains(logQuery, "limit=50") {
			t.Fatal("ESI seed/filter bounds were not forwarded")
		}
		for _, marker := range []string{"Requests (24h)", "77", "data-admin-esi-count", "data-esi-log-search", "data-show-entity=\"true\"", "Fixture Reader", "ESI request details"} {
			if !strings.Contains(response.Body.String(), marker) {
				t.Errorf("missing ESI marker %s", marker)
			}
		}
		if strings.Contains(response.Body.String(), "<script>safe</script>") {
			t.Fatal("ESI row data escaped its HTML attribute")
		}
	}
	for _, read := range reads {
		if read == "/api/admin/esi" {
			t.Fatal("lazy ESI summary became a blocking page fetch")
		}
	}
	if adminESIEntity(map[string][]string{"character_id": {"-1"}}) != nil || adminESIEntity(map[string][]string{"character_id": {"not-an-id"}}) != nil {
		t.Fatal("invalid entity seeded a filter")
	}
}
