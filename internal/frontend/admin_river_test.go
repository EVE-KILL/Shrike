package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAdminRiverFragmentsArePrivateBoundedAndEscapeJobArguments(t *testing.T) {
	for _, identity := range []string{"admin", "reader", "anonymous"} {
		for _, part := range []string{"queues", "jobs"} {
			t.Run(identity+"/"+part, func(t *testing.T) {
				var reads atomic.Int32
				api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "GET" {
						t.Fatal("rendering mutated River")
					}
					if r.URL.Path == "/auth/me" {
						var user *settingsUser
						if identity != "anonymous" {
							user = &settingsUser{ID: 42, Admin: identity == "admin"}
						}
						_ = json.NewEncoder(w).Encode(map[string]any{"user": user})
						return
					}
					reads.Add(1)
					switch r.URL.Path {
					case "/api/admin/river":
						_ = json.NewEncoder(w).Encode(map[string]any{"queues": []map[string]any{{"name": "fixture", "depth": map[string]any{"queue": "fixture", "retryable": 2}, "description": "<script>description</script>", "concurrency": 4}}})
					case "/api/admin/river/jobs":
						query := r.URL.Query()
						if query.Get("queue") != "fixture" || query.Get("state") != "retryable" || query.Get("before_id") != "901" || query.Get("limit") != "100" {
							t.Fatal("unbounded or changed River query")
						}
						_ = json.NewEncoder(w).Encode(map[string]any{"jobs": []map[string]any{{"id": 900, "queue": "fixture", "kind": "<script>kind</script>", "state": "retryable", "args": map[string]string{"input": "<script>arguments</script>"}}}, "next_before_id": 800})
					default:
						t.Fatalf("unexpected River API %s", r.URL.Path)
					}
				})
				handler := New(Options{Production: true, AccountAPI: api})
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, httptest.NewRequest("GET", "/_fragments/admin/river/"+part+"?queue=fixture&state=retryable&before_id=901&limit=99999", nil))
				want := 200
				if identity == "reader" {
					want = 403
				}
				if identity == "anonymous" {
					want = 401
				}
				if w.Code != want || w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
					t.Fatal("private River fragment exposed")
				}
				if identity != "admin" {
					if reads.Load() != 0 {
						t.Fatal("River read before authorization")
					}
					return
				}
				if reads.Load() != 1 {
					t.Fatal("queue/job fragment loaded unrelated data")
				}
				body := w.Body.String()
				if strings.Contains(body, `data-failed="true"`) {
					t.Fatal("valid River depth object failed to decode")
				}
				if strings.Contains(body, "<script>") || !strings.Contains(body, "&lt;script&gt;") {
					t.Fatal("River metadata became active HTML")
				}
				if part == "jobs" && (!strings.Contains(body, `data-river-older="800"`) || !strings.Contains(body, `data-river-job-action="cancel"`)) {
					t.Fatal("job cursor/actions missing")
				}
			})
		}
	}
}

func TestAdminRiverFailedJobsDoNotClaimAnEmptyQueue(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/me" {
			_ = json.NewEncoder(w).Encode(map[string]any{"user": settingsUser{ID: 42, Admin: true}})
			return
		}
		http.Error(w, "temporary", http.StatusServiceUnavailable)
	})
	handler := New(Options{Production: true, AccountAPI: api})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/_fragments/admin/river/jobs", nil))
	if !strings.Contains(w.Body.String(), `data-failed="true"`) || strings.Contains(w.Body.String(), "No jobs match") {
		t.Fatal("failed River read misrepresented as empty")
	}
}
