package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminReviewFragmentsAuthorizeBoundQueriesAndEscapeStoredContent(t *testing.T) {
	for _, section := range []string{"moderation", "comments"} {
		for _, identity := range []string{"admin", "reader", "anonymous"} {
			t.Run(section+"/"+identity, func(t *testing.T) {
				api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "GET" {
						t.Fatal("SSR mutated review data")
					}
					var data any
					switch r.URL.Path {
					case "/auth/me":
						var user *settingsUser
						if identity != "anonymous" {
							user = &settingsUser{ID: 42, Admin: identity == "admin"}
						}
						data = map[string]any{"user": user}
					case "/api/admin/moderation/queue", "/api/admin/comments/queue":
						if identity != "admin" || r.URL.Query().Get("limit") != "100" {
							t.Fatal("unauthorized or unbounded review read")
						}
						if section == "moderation" {
							if r.URL.Query().Get("kind") != "bios" || r.URL.Query().Get("status") != "all" {
								t.Fatal("review filters changed")
							}
							data = map[string]any{"items": []map[string]any{{"id": 7, "target_kind": 1, "target_id": 42, "character_name": "<script>name</script>", "rendered_html": `<p><span style="color:#ff00ff">Safe bio</span></p><script>alert(1)</script>`, "ai_action": "flag", "ai_max_score": 0.505}}, "counts": map[string]int{"pending": 1, "pending_bios": 1}}
						} else {
							if r.URL.Query().Get("filter") != "reported" {
								t.Fatal("comment filter changed")
							}
							data = map[string]any{"comments": []map[string]any{{"id": 8, "target_type": 9, "target_slug": "article", "character_name": "<script>name</script>", "body_html": "<p>Safe comment</p><script>alert(1)</script>", "flagged": true, "reports_count": 2, "moderation_status": 2}}}
						}
					default:
						t.Fatalf("unexpected read %s", r.URL.Path)
					}
					_ = json.NewEncoder(w).Encode(data)
				})
				handler := New(Options{Production: true, AccountAPI: api})
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, httptest.NewRequest("GET", "/_fragments/admin/"+section+"?kind=bios&status=all&filter=reported&limit=100000", nil))
				want := 200
				if identity == "reader" {
					want = 403
				}
				if identity == "anonymous" {
					want = 401
				}
				if response.Code != want || response.Header().Get("Cache-Control") != "private, no-store" || response.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
					t.Fatal("private review response could be exposed")
				}
				if identity == "admin" {
					body := response.Body.String()
					if strings.Contains(body, "<script>") || !strings.Contains(body, "&lt;script&gt;name&lt;/script&gt;") {
						t.Fatal("review data became active markup")
					}
					if section == "moderation" && !strings.Contains(body, "51%") {
						t.Fatal("AI midpoint score differs from JavaScript rounding")
					}
					if section == "comments" && !strings.Contains(body, "/blog/article#comment-8") {
						t.Fatal("comment context lost its canonical target")
					}
				}
			})
		}
	}
}
