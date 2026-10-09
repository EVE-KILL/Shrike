package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

func TestTrackersPrivateSSRAndOwnedKilllistFragments(t *testing.T) {
	var calls []string
	account := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		cookie, _ := r.Cookie("evelogin")
		authenticated := cookie != nil && cookie.Value == "fixture-session"
		if r.URL.Path == "/auth/me" {
			if authenticated {
				_, _ = w.Write([]byte(`{"user":{"characterId":42,"characterName":"Private fixture"}}`))
			} else {
				_, _ = w.Write([]byte(`{"user":null}`))
			}
			return
		}
		if !authenticated {
			w.WriteHeader(401)
			return
		}
		switch r.URL.Path {
		case "/api/me/trackers":
			_, _ = w.Write([]byte(`{"trackers":[{"id":1,"target_type":"corporation","target_id":10,"target_name":"Private <script>fixture</script>","enabled":true,"notifications_enabled":false,"created_at":"2026-10-03T12:00:00Z","updated_at":"2026-10-03T12:00:00Z","event_count":1}],"limit":50}`))
		case "/api/me/trackers/1/killmails":
			if r.URL.Query().Get("limit") != "10" || r.URL.Query().Get("after") != "1234" {
				t.Errorf("lost cursor or limit: %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(api.KilllistPage{Kills: []api.KilllistRow{{KillmailID: 100, ShipName: "Rifter", VictimCorporationID: 10, KillmailTime: time.Now()}}, HasMore: true})
		default:
			w.WriteHeader(404)
		}
	})
	h := New(Options{AccountAPI: account})
	call := func(path string, authenticated bool) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("GET", path, nil)
		if authenticated {
			r.AddCookie(&http.Cookie{Name: "evelogin", Value: "fixture-session"})
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	guest := call("/trackers", false)
	if guest.Code != 200 || !strings.Contains(guest.Body.String(), "Log in to create and manage entity trackers.") || strings.Contains(guest.Body.String(), "Private fixture") {
		t.Fatal("anonymous private page escaped its login gate")
	}
	page := call("/trackers", true)
	if page.Code != 200 || page.Header().Get("Cache-Control") != "private, no-store" || !strings.Contains(page.Body.String(), `noindex, nofollow`) || strings.Contains(page.Body.String(), "Private <script>") || !strings.Contains(page.Body.String(), "Private &lt;script&gt;") {
		t.Fatal("private SSR status, policy or escaping failed")
	}
	for _, tc := range []struct {
		path          string
		authenticated bool
		status        int
	}{
		{"/_fragments/trackers/1/killmails?limit=10&after=1234&lossType=corporation&lossId=10&live=1", true, 200},
		{"/_fragments/trackers/2/killmails", true, 404},
		{"/_fragments/trackers/1/killmails", false, 401},
		{"/_fragments/trackers/1/killmails?limit=999", true, 400},
	} {
		w := call(tc.path, tc.authenticated)
		if w.Code != tc.status || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Errorf("fragment %s = %d %s", tc.path, w.Code, w.Header())
		}
		if tc.status == 200 && (!strings.Contains(w.Body.String(), "loss-row") || strings.Contains(w.Body.String(), "<header")) {
			t.Fatal("killlist fragment lost corporate-loss coloring or rendered a whole page")
		}
	}
	if strings.Count(strings.Join(calls, ","), "/api/me/trackers,") > 1 {
		t.Fatal("paginated fragments unnecessarily reloaded every tracker")
	}
}
