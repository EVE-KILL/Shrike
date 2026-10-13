package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAdminPagesAuthenticateBeforeReadingPrivateData(t *testing.T) {
	for _, identity := range []string{"anonymous", "reader", "admin"} {
		var reads []string
		api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reads = append(reads, r.URL.Path)
			var data any
			switch r.URL.Path {
			case "/auth/me":
				var user *settingsUser
				if identity != "anonymous" {
					user = &settingsUser{ID: 42, Name: "Fixture admin", Admin: identity == "admin"}
				}
				data = map[string]any{"user": user}
			case "/api/admin/overview":
				data = map[string]any{"users": map[string]any{"total": 2, "recent7d": 1}, "killmails": map[string]any{"total": 100, "last24h": 7, "last7d": 49}, "esi": map[string]any{"requests24h": 8, "errorRate": 12.5}, "attention": map[string]any{"moderation": 3, "domainAssets": 2, "campaignsPaused": 1, "campaignsFailed": 4}}
			default:
				t.Fatalf("unexpected private read %s", r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(data)
		})
		handler := New(Options{Production: true, AccountAPI: api})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", "/admin", nil))
		if response.Header().Get("Cache-Control") != "private, no-store" || response.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
			t.Fatal("admin HTML or redirect could be cached/indexed")
		}
		switch identity {
		case "anonymous":
			if response.Code != 200 || !strings.Contains(response.Body.String(), "Log in with EVE Online to access the admin panel") || len(reads) != 1 {
				t.Fatal("anonymous admin gate exposed data")
			}
		case "reader":
			if response.Code != 302 || response.Header().Get("Location") != "/" || len(reads) != 1 {
				t.Fatal("non-admin redirect loaded private data")
			}
		case "admin":
			if response.Code != 200 || len(reads) != 2 {
				t.Fatal("overview added blocking ESI reads or omitted auth")
			}
			for _, marker := range []string{"Admin Panel", "Needs Attention", "12.5% error rate", "~7/day", "data-admin-badge=\"campaigns\"", "awaiting moderation", "domain assets to review", "campaigns erroring", `data-controller="admin"`, `data-controller="admin-esi-summary"`} {
				if !strings.Contains(response.Body.String(), marker) {
					t.Errorf("missing %s", marker)
				}
			}
		}
		fragment := httptest.NewRecorder()
		handler.ServeHTTP(fragment, httptest.NewRequest("GET", "/_fragments/admin/overview", nil))
		want := 401
		if identity == "reader" {
			want = 403
		}
		if identity == "admin" {
			want = 200
		}
		if fragment.Code != want || fragment.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("fragment access %s=%d", identity, fragment.Code)
		}
	}
}

func TestAdminUserRowsAndDetailReuseBoundedAPIsAndEscapeValues(t *testing.T) {
	stamp := time.Date(2026, 10, 4, 10, 0, 0, 0, time.UTC)
	user := adminUser{ID: 43, Name: "<script>reader</script>", OwnerHash: "fixture", LastLogin: &stamp, CreatedAt: &stamp}
	var query string
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data any
		switch r.URL.Path {
		case "/auth/me":
			data = map[string]any{"user": settingsUser{ID: 42, Name: "Owner", Admin: true}}
		case "/api/admin/users":
			query = r.URL.RawQuery
			data = adminUsersData{Users: []adminUser{user}, Total: 55, Pages: 2, Page: 2}
		case "/api/admin/users/43":
			data = adminUserDetailData{User: &user, Stats: map[string]any{"total_requests": 0, "total_errors": 0, "total_new_items": 0}}
		case "/api/admin/users/999":
			w.WriteHeader(404)
			return
		default:
			t.Fatalf("unexpected read %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(data)
	})
	handler := New(Options{AccountAPI: api})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/_fragments/admin/users?sort=sql-injection&dir=other&page=2&search=reader&limit=100000", nil))
	if response.Code != 200 || !strings.Contains(query, "limit=50") || !strings.Contains(query, "sort=last_login") || !strings.Contains(query, "dir=desc") || !strings.Contains(query, "page=2") || !strings.Contains(query, "search=reader") {
		t.Fatal("user fragment changed API bounds or filters")
	}
	if strings.Contains(response.Body.String(), "<script>reader</script>") || !strings.Contains(response.Body.String(), "&lt;script&gt;reader&lt;/script&gt;") || !strings.Contains(response.Body.String(), "/admin/users/43") {
		t.Fatal("user rows failed escaping or detail navigation")
	}
	if !strings.Contains(response.Body.String(), `data-controller="admin-users"`) {
		t.Fatal("user directory fragment lost its browser component")
	}
	detail := httptest.NewRecorder()
	handler.ServeHTTP(detail, httptest.NewRequest("GET", "/admin/users/43", nil))
	for _, marker := range []string{"Back to Users", "Discord ID", "Make Admin", "No ESI token found.", "View in ESI logs", "2026-10-04 10:00", `data-controller="admin-user-detail"`} {
		if !strings.Contains(detail.Body.String(), marker) {
			t.Errorf("detail missing %s", marker)
		}
	}
	if detail.Code != 200 || strings.Contains(detail.Body.String(), "<script>reader</script>") {
		t.Fatal("private detail failed")
	}
	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest("GET", "/admin/users/999", nil))
	if missing.Code != 404 {
		t.Fatal("missing admin user did not return 404")
	}
	bad := httptest.NewRecorder()
	handler.ServeHTTP(bad, httptest.NewRequest("GET", "/admin/users/not-a-number", nil))
	if bad.Code != 400 {
		t.Fatal("invalid user ID did not return 400")
	}
}
