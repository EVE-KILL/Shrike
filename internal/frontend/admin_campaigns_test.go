package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminCampaignFragmentsBoundReadsAndRetainSourceStates(t *testing.T) {
	for _, identity := range []string{"admin", "anonymous", "reader"} {
		t.Run(identity, func(t *testing.T) {
			var query string
			api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Fatal("SSR mutated campaign data")
				}
				var data any
				switch r.URL.Path {
				case "/auth/me":
					var user *settingsUser
					if identity != "anonymous" {
						user = &settingsUser{ID: 42, Admin: identity == "admin"}
					}
					data = map[string]any{"user": user}
				case "/api/admin/campaigns":
					if identity != "admin" {
						t.Fatal("private list read without authorization")
					}
					query = r.URL.RawQuery
					data = map[string]any{"page": 2, "hasMore": true, "campaigns": []map[string]any{{"campaign_id": "AdminCamp00001", "name": "<script>campaign</script>", "status": 2, "processing_paused": true, "last_processing_error": "<script>error</script>", "last_processing_duration_ms": 60000, "last_processing_killmails": 0, "totals": map[string]any{"killCount": 10}}}}
				default:
					t.Fatalf("unexpected read %s", r.URL.Path)
				}
				_ = json.NewEncoder(w).Encode(data)
			})
			handler := New(Options{Production: true, AccountAPI: api})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/_fragments/admin/campaigns?state=failed&q=literal%25&page=2&limit=100000", nil))
			want := 200
			if identity == "anonymous" {
				want = 401
			}
			if identity == "reader" {
				want = 403
			}
			if w.Code != want || w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
				t.Fatalf("private campaign fragment=%d", w.Code)
			}
			if identity == "admin" {
				if query != "page=2&q=literal%25&state=failed" {
					t.Fatalf("unbounded or changed query %s", query)
				}
				for _, marker := range []string{"&lt;script&gt;campaign&lt;/script&gt;", "&lt;script&gt;error&lt;/script&gt;", "Paused", "Resume", "1.0m", "Page 2", "/campaign/AdminCamp00001"} {
					if !strings.Contains(w.Body.String(), marker) {
						t.Errorf("missing %s", marker)
					}
				}
				if strings.Contains(w.Body.String(), "<script>error</script>") {
					t.Fatal("processing error became markup")
				}
			}
		})
	}
}
