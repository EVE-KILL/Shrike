package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
)

func TestTrackerDashboardPrivateSSRIndependentWidgetsAndValidation(t *testing.T) {
	var mutex sync.Mutex
	calls := map[string]int{}
	account := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		calls[r.URL.Path]++
		mutex.Unlock()
		cookie, _ := r.Cookie("evelogin")
		authenticated := cookie != nil && cookie.Value == "fixture"
		if r.URL.Path == "/auth/me" {
			if authenticated {
				w.Write([]byte(`{"user":{"characterId":42}}`))
			} else {
				w.Write([]byte(`{"user":null}`))
			}
			return
		}
		if !authenticated {
			w.WriteHeader(401)
			return
		}
		switch r.URL.Path {
		case "/api/me/tracked/config":
			w.Write([]byte(`{"configured":true,"widgets":{"top":[{"type":"mostValuable","enabled":true},{"type":"textBlock","enabled":true,"content":"Private <script>note</script>"}],"left":[{"type":"topCharacters","enabled":true},{"type":"topRegions","enabled":false}],"right":[{"type":"killList","enabled":true,"killlistType":"npc"}],"columnRatio":"1fr_2fr"}}`))
		case "/api/me/trackers":
			w.Write([]byte(`{"trackers":[],"limit":50}`))
		case "/api/me/tracked/summary":
			w.Write([]byte(`{"tracker_count":2,"active_tracker_count":1,"kill_count":1234,"total_value":2000000000}`))
		case "/api/me/tracked/stats":
			if r.URL.Query().Get("dataType") == "most_valuable_ships" {
				if r.URL.Query().Get("limit") != "8" || r.URL.Query().Get("days") != "7" {
					t.Error("valuable query lost its bounded scope")
				}
				json.NewEncoder(w).Encode(map[string]any{"entries": []api.ValuableKill{{KillmailID: 100, ShipName: "Rifter"}}})
			} else {
				if r.URL.Query().Get("dataType") != "characters" || r.URL.Query().Get("limit") != "10" || r.URL.Query().Get("days") != "7" {
					t.Error("top box query lost its scope")
				}
				json.NewEncoder(w).Encode(map[string]any{"entries": []api.TopEntry{{ID: 42, Name: "Owner", Count: 1}}})
			}
		case "/api/me/tracked/killmails":
			if r.URL.Query().Get("after") != "100" || r.URL.Query().Get("limit") != "10" || r.URL.Query().Get("type") != "npc" {
				t.Error("lost private cursor/type")
			}
			json.NewEncoder(w).Encode(api.KilllistPage{Kills: []api.KilllistRow{{KillmailID: 90, ShipName: "Rifter"}}})
		default:
			w.WriteHeader(404)
		}
	})
	h := New(Options{AccountAPI: account, Production: true})
	call := func(path string, auth bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "https://eve-kill.com"+path, nil)
		if auth {
			r.AddCookie(&http.Cookie{Name: "evelogin", Value: "fixture"})
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	guest := call("/tracker", false)
	if guest.Code != 200 || !strings.Contains(guest.Body.String(), "Log in to build your personal Tracker dashboard.") {
		t.Fatal("dashboard escaped login gate")
	}
	page := call("/tracker", true)
	body := page.Body.String()
	if page.Code != 200 || page.Header().Get("Cache-Control") != "private, no-store" || page.Header().Get("X-Robots-Tag") != "noindex, nofollow" || !strings.Contains(body, "1,234") || !strings.Contains(body, "Private &lt;script&gt;note&lt;/script&gt;") || !strings.Contains(body, "--dashboard-columns:1fr 2fr") || !strings.Contains(body, `data-dashboard-killtype="npc"`) {
		t.Fatal("SSR policy, summary, layout or escaped note failed")
	}
	if calls["/api/me/tracked/stats"] != 0 || calls["/api/me/tracked/killmails"] != 0 {
		t.Fatal("SSR waited on aggregate widgets")
	}
	if strings.Contains(body, `data-dashboard-remote="topRegions"`) {
		t.Fatal("disabled widget loaded")
	}
	for _, tc := range []struct {
		path   string
		auth   bool
		status int
	}{
		{"/_fragments/tracker/stats/topCharacters", true, 200}, {"/_fragments/tracker/valuable?kind=ships", true, 200},
		{"/_fragments/tracker/killmails?type=npc&limit=10&after=100", true, 200},
		{"/_fragments/tracker/stats/topCharacters", false, 401}, {"/_fragments/tracker/valuable?kind=ships", false, 401}, {"/_fragments/tracker/killmails", false, 401}, {"/_fragments/tracker", false, 401},
		{"/_fragments/tracker/stats/campaigns", true, 404}, {"/_fragments/tracker/valuable?kind=anything", true, 400}, {"/_fragments/tracker/killmails?type=unknown", true, 400}, {"/_fragments/tracker/killmails?limit=10000", true, 400},
	} {
		w := call(tc.path, tc.auth)
		if w.Code != tc.status || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Errorf("%s=%d", tc.path, w.Code)
		}
	}
}
