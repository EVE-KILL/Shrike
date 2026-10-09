package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminAnnouncementFiltersPrivacyAndEscaping(t *testing.T) {
	var query string
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/me":
			_ = json.NewEncoder(w).Encode(map[string]any{"user": settingsUser{ID: 42, Name: "Owner", Admin: true}})
		case "/api/admin/overview":
			_ = json.NewEncoder(w).Encode(map[string]any{})
		case "/api/admin/announcements":
			query = r.URL.RawQuery
			_ = json.NewEncoder(w).Encode(map[string]any{"announcements": []map[string]any{{"id": 1, "tier": 2, "title": "<script>title</script>", "body_md": "<img src=x onerror=bad>", "starts_at": "2026-10-04T12:00:00Z", "expires_at": "2026-10-05T12:00:00Z"}}})
		default:
			t.Fatalf("unexpected read %s", r.URL.Path)
		}
	})
	handler := New(Options{Production: true, AccountAPI: api})
	for _, path := range []string{"/admin/announcements", "/_fragments/admin/announcements?status=archived"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
			t.Fatal("announcement route leaked private caching")
		}
		want := "status=active"
		if strings.Contains(path, "archived") {
			want = "status=archived"
		}
		if !strings.Contains(query, want) || !strings.Contains(query, "limit=100") {
			t.Fatal("announcement query bounds/status diverged")
		}
		for _, unsafe := range []string{"<script>title</script>", "<img src=x onerror=bad>"} {
			if strings.Contains(w.Body.String(), unsafe) {
				t.Fatal("announcement text became markup")
			}
		}
		for _, marker := range []string{"data-admin-announcement-id=\"1\"", "data-admin-announcement-form", "data-date-time-picker", "data-admin-announcement-archive"} {
			if !strings.Contains(w.Body.String(), marker) {
				t.Fatalf("missing %s", marker)
			}
		}
	}
}
