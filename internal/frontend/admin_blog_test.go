package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminBlogFragmentPrivacyBoundedFilterAndStoredText(t *testing.T) {
	for _, identity := range []string{"admin", "reader", "anonymous"} {
		t.Run(identity, func(t *testing.T) {
			api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Fatal("blog rendering wrote data")
				}
				if r.URL.Path == "/auth/me" {
					var user *settingsUser
					if identity != "anonymous" {
						user = &settingsUser{ID: 42, Admin: identity == "admin"}
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"user": user})
					return
				}
				if identity != "admin" {
					t.Fatal("read blog before authorization")
				}
				if r.URL.Path != "/api/admin/blog" || r.URL.Query().Get("status") != "draft" || r.URL.Query().Get("limit") != "100" || len(r.URL.Query()) != 2 {
					t.Fatal("unbounded or changed blog query")
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"posts": []map[string]any{{"id": 7, "slug": "draft", "status": 0, "title": "<script>title</script>", "body_md": "<script>literal draft</script>", "body_html": "<script>not trusted</script>", "author_name": "<script>author</script>", "tags": []string{"<script>tag</script>"}, "updated_at": "2026-10-05T15:30:00+02:00"}}})
			})
			handler := New(Options{Production: true, AccountAPI: api})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/_fragments/admin/blog?status=draft&limit=999999&title=ignored", nil))
			want := 200
			if identity == "reader" {
				want = 403
			}
			if identity == "anonymous" {
				want = 401
			}
			if w.Code != want || w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
				t.Fatal("blog fragment could expose drafts")
			}
			if identity == "admin" {
				body := w.Body.String()
				if strings.Contains(body, "<script>") || !strings.Contains(body, "&lt;script&gt;title&lt;/script&gt;") || !strings.Contains(body, "Updated: 2026-10-05 13:30") || !strings.Contains(body, `href="/blog/preview/draft"`) || strings.Contains(body, `href="/blog/draft"`) {
					t.Fatal("stored text, UTC date or draft links diverged")
				}
			}
		})
	}
}

func TestAdminBlogFailedReadHasRetryWithoutFalseEmptyList(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/me" {
			_ = json.NewEncoder(w).Encode(map[string]any{"user": settingsUser{ID: 42, Admin: true}})
			return
		}
		http.Error(w, "offline", http.StatusServiceUnavailable)
	})
	w := httptest.NewRecorder()
	New(Options{Production: true, AccountAPI: api}).ServeHTTP(w, httptest.NewRequest("GET", "/_fragments/admin/blog?status=invalid", nil))
	body := w.Body.String()
	if w.Code != 200 || !strings.Contains(body, "Could not load blog posts.") || !strings.Contains(body, "data-admin-blog-retry") || strings.Contains(body, "No blog posts found.") || !strings.Contains(body, `data-status="all"`) {
		t.Fatal("failed blog read masqueraded as an empty list")
	}
}
