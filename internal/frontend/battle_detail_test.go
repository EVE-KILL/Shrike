package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
)

func TestBattleDetailRoutesFragmentsAndKillmailRedirect(t *testing.T) {
	data := battleDetailData{BattleID: 42, SystemID: 30000142, SystemName: "Jita", Start: "2026-04-06T16:00:00Z", End: "2026-04-06T16:03:00Z", Minutes: 3, Teams: []battleTeam{{Index: 0}, {Index: 1}}, Sides: []battleSide{{Corps: []int64{1}}, {Corps: []int64{2}}}}
	var lastQuery url.Values
	handler := New(Options{Production: true, ConflictsAPI: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "eve-kill.com" || r.Header.Get("Cookie") != "session=test" {
			t.Errorf("scope lost: host=%q", r.Host)
		}
		switch r.URL.Path {
		case "/api/battle/42":
			json.NewEncoder(w).Encode(data)
		case "/api/battle/42/replay":
			json.NewEncoder(w).Encode(battleReplayData{Kills: []battleReplayKill{}})
		case "/api/battle/42/killlist":
			lastQuery = r.URL.Query()
			json.NewEncoder(w).Encode(api.KilllistPage{HasMore: true, Cursor: battleInt64(123)})
		case "/api/battle/42/composition":
			json.NewEncoder(w).Encode(battleCompositionData{Teams: []battleCompositionTeam{{Index: 0}, {Index: 1}}})
		case "/api/battle/42/intel":
			json.NewEncoder(w).Encode(battleIntelData{Teams: []battleIntelTeam{{Index: 0}, {Index: 1}}})
		case "/api/battle/killmail/99":
			json.NewEncoder(w).Encode(map[string]any{"redirect": true, "battle_id": 42})
		default:
			http.NotFound(w, r)
		}
	})})
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", canonicalOrigin+path, nil)
		r.Header.Set("Cookie", "session=test")
		handler.ServeHTTP(w, r)
		return w
	}
	for _, tab := range []string{"", "summary", "kills", "timeline", "replay", "composition", "intel", "comments"} {
		path := "/battle/42"
		if tab != "" {
			path += "/" + tab
		}
		w := get(path)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "Battle in Jita") || !strings.Contains(w.Body.String(), `href="https://eve-kill.com/battle/42"`) {
			t.Errorf("%s status=%d", path, w.Code)
		}
	}
	fragment := get("/_fragments/battle/42/comments")
	if fragment.Code != 200 || strings.Contains(fragment.Body.String(), "hero-surface") || !strings.Contains(fragment.Body.String(), `data-target-type="7" data-target-id="42"`) {
		t.Fatalf("comments fragment status=%d", fragment.Code)
	}
	feed := get("/_fragments/battle/42/killfeed?page=2&limit=10&after=123&side=1&group=30&minIsk=100000000&from=1775491200000&to=1775491260000")
	if feed.Code != 200 || !strings.Contains(feed.Body.String(), `data-page="2"`) || !strings.Contains(feed.Body.String(), `data-limit="10"`) {
		t.Fatalf("killfeed status=%d", feed.Code)
	}
	for key, want := range map[string]string{"side": "1", "group": "30", "minIsk": "100000000", "from": "1775491200000", "to": "1775491260000", "after": "123", "limit": "10"} {
		if lastQuery.Get(key) != want {
			t.Errorf("%s=%q want %q", key, lastQuery.Get(key), want)
		}
	}
	redirect := get("/battle/1?killmail=99")
	if redirect.Code != 302 || redirect.Header().Get("Location") != "/battle/42" {
		t.Fatalf("saved redirect status=%d location=%q", redirect.Code, redirect.Header().Get("Location"))
	}
	failed := get("/battle/1?killmail=100")
	if failed.Code != 200 || !strings.Contains(failed.Body.String(), "Could not generate a battle report") || !strings.Contains(failed.Body.String(), `href="/kill/100"`) {
		t.Fatalf("generated failure status=%d", failed.Code)
	}
	for _, path := range []string{"/battle/0", "/battle/2147483648", "/battle/42/other", "/battle/42/comments?killmail=99", "/battle/404"} {
		if got := get(path); got.Code != 404 {
			t.Errorf("%s status=%d", path, got.Code)
		}
	}
	for _, path := range []string{"/_fragments/battle/42/killfeed?page=2", "/_fragments/battle/42/killfeed?limit=500", "/_fragments/battle/42/killfeed?after=-1"} {
		if got := get(path); got.Code != 400 {
			t.Errorf("%s status=%d", path, got.Code)
		}
	}
}
func battleInt64(n int64) *int64 { return &n }

func TestBattleLossWindowsIncludeFinalBoundaryAndKeepQuietIntervals(t *testing.T) {
	start := "2026-04-06T16:00:00Z"
	end := "2026-04-06T16:03:00Z"
	v := battleDetailView{Data: battleDetailData{Start: start, End: end, Sides: []battleSide{{Corps: []int64{1}, Alliances: []int64{10}}, {Corps: []int64{2}, Alliances: []int64{20}}}}, Replay: battleReplayData{Kills: []battleReplayKill{{ID: 1, Time: start, CorpID: 1, AllianceID: 20, GroupID: 30, Value: 1e11}, {ID: 2, Time: "2026-04-06T16:01:00Z", CorpID: 3, GroupID: 29, Value: 10000}, {ID: 3, Time: end, CorpID: 2, GroupID: 659, Value: 1e11}}}}
	buildBattleLosses(&v)
	if len(v.Bins) != 3 || v.Bins[1].Sides[2].Count != 1 || v.Bins[1].Sides[1].Count != 0 || v.Bins[2].Big != 1 || v.Bins[2].End != battleTimeMS(end)+1 {
		t.Fatalf("unexpected boundary bins: %+v", v.Bins)
	}
	if v.Bins[0].Sides[0].Count != 1 {
		t.Error("explicit corporation must win over alliance side")
	}
	filtered := battleDetailView{Data: v.Data, Replay: v.Replay, Filter: battleLossFilter{From: &v.Bins[2].Start, To: &v.Bins[2].End}}
	buildBattleLosses(&filtered)
	if len(filtered.Filtered) != 1 || filtered.Filtered[0].ID != 3 {
		t.Fatalf("final interval losses %+v", filtered.Filtered)
	}
	quiet := battleDetailView{Data: v.Data, Replay: battleReplayData{Kills: v.Replay.Kills[:1]}}
	buildBattleLosses(&quiet)
	if quiet.Bins[1].Count != 0 {
		t.Error("quiet interval disappeared")
	}
}
func TestBattleLinksPreserveGenerationAndFiltersWithoutMutatingQuery(t *testing.T) {
	v := battleDetailView{ID: 1, Query: url.Values{"killmail": {"99"}, "side": {"1"}, "from": {"100"}}}
	before := v.Query.Encode()
	link := v.path("kills", url.Values{"group": {"30"}, "from": nil})
	u, _ := url.Parse(link)
	if u.Path != "/battle/1/kills" || u.Query().Get("killmail") != "99" || u.Query().Get("side") != "1" || u.Query().Has("from") || !reflect.DeepEqual(before, v.Query.Encode()) {
		t.Fatalf("link %s query %s", link, v.Query.Encode())
	}
}
