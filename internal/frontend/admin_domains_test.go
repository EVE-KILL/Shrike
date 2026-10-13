package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminDomainsReadOnlyPrivatePagesAndPreviews(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatal("SSR mutated domain data")
		}
		var data any
		switch r.URL.Path {
		case "/auth/me":
			data = map[string]any{"user": settingsUser{ID: 42, Admin: true}}
		case "/api/admin/overview":
			data = map[string]any{}
		case "/api/admin/domains":
			data = map[string]any{"domains": []map[string]any{{"id": 7, "site_name": "<script>board</script>", "user_id": 43, "owner_name": "Reader", "active": true, "pending_assets": 2}}}
		case "/api/admin/domains/7":
			data = map[string]any{"domain": map[string]any{"id": 7, "site_name": "<script>board</script>", "user_id": 43}, "assets": []map[string]any{{"id": 8, "type": "logo", "status": "pending", "file_size": 1024}, {"id": 9, "type": "banner", "status": "rejected", "reject_reason": "<script>reason</script>"}}}
		case "/api/admin/domains/999":
			w.WriteHeader(404)
			return
		default:
			t.Fatalf("unexpected read %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(data)
	})
	handler := New(Options{Production: true, AccountAPI: api})
	for _, path := range []string{"/admin/domains", "/_fragments/admin/domains", "/admin/domains/7", "/_fragments/admin/domains/7"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
			t.Fatalf("private domain %s=%d", path, w.Code)
		}
		if strings.Contains(w.Body.String(), "<script>board</script>") || strings.Contains(w.Body.String(), "<script>reason</script>") {
			t.Fatal("domain values became executable markup")
		}
		if strings.HasSuffix(path, "/7") && !strings.Contains(w.Body.String(), "/api/admin/domains/7/assets/8/preview") {
			t.Fatal("private preview URL lost domain/asset scope")
		}
	}
	for _, check := range []struct {
		Path   string
		Status int
	}{{"/admin/domains/0", 400}, {"/admin/domains/-1", 400}, {"/admin/domains/invalid", 400}, {"/admin/domains/999", 404}} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", check.Path, nil))
		if w.Code != check.Status {
			t.Fatal("domain error status diverged")
		}
	}
}

func TestAdminDomainDetailAuthenticatesBeforeReadingPrivateData(t *testing.T) {
	for _, identity := range []string{"anonymous", "reader"} {
		t.Run(identity, func(t *testing.T) {
			reads := 0
			api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reads++
				if r.URL.Path != "/auth/me" {
					t.Fatalf("unauthorized domain read %s", r.URL.Path)
				}
				var user *settingsUser
				if identity == "reader" {
					user = &settingsUser{ID: 43}
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"user": user})
			})
			handler := New(Options{Production: true, AccountAPI: api})
			for _, path := range []string{"/admin/domains/7", "/_fragments/admin/domains/7"} {
				reads = 0
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
				want := 200
				if identity == "reader" {
					want = 302
				}
				if strings.HasPrefix(path, "/_fragments/") {
					want = 401
					if identity == "reader" {
						want = 403
					}
				}
				if response.Code != want || reads != 1 || response.Header().Get("Cache-Control") != "private, no-store" || response.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
					t.Fatalf("domain access %s %s=%d reads=%d", identity, path, response.Code, reads)
				}
			}
		})
	}
}
