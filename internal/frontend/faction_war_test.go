package frontend

import (
	"context"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
)

type testFactionWarReader struct {
	side    string
	after   *int64
	period  string
	members int
	fail    bool
}

func (f *testFactionWarReader) FactionWarDashboard(context.Context, string) (api.FactionWarData, error) {
	return api.FactionWarData{Matchup: "caldari-vs-gallente", Side1: api.FactionWarSideData{ID: 500001, Name: "Caldari State", CorpID: 1000035}, Side2: api.FactionWarSideData{ID: 500004, Name: "Gallente Federation", CorpID: 1000120}}, nil
}
func (f *testFactionWarReader) FactionWarOverview(_ context.Context, _ string, period string) (api.FactionWarOverview, error) {
	f.period = period
	return api.FactionWarOverview{}, nil
}
func (f *testFactionWarReader) FactionWarSystems(context.Context, string) (api.FactionWarSystems, error) {
	if f.fail {
		return api.FactionWarSystems{}, errors.New("unavailable")
	}
	return api.FactionWarSystems{Systems: []api.FactionWarSystem{{ID: 30002806, Name: "Hasmijaala", Security: .3, RegionName: "The Citadel", Kills: 10, OccupierID: 500004, Contested: "contested"}}}, nil
}
func (f *testFactionWarReader) FactionWarMembers(context.Context, string, url.Values) (api.WarMembersData, error) {
	f.members++
	return api.WarMembersData{}, nil
}
func (f *testFactionWarReader) FactionWarIntel(context.Context, string) (api.WarIntelData, error) {
	return api.WarIntelData{}, nil
}
func (f *testFactionWarReader) FactionWarKilllist(_ context.Context, _ string, side string, _ int, _ int, after *int64) (api.KilllistPage, error) {
	f.side = side
	f.after = after
	cursor := int64(123)
	return api.KilllistPage{Cursor: &cursor, HasMore: true}, nil
}

func TestFactionWarPageAndIndependentTabs(t *testing.T) {
	reader := &testFactionWarReader{}
	handler := New(Options{FactionWars: reader})
	request := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}
	w := request("/faction-war/caldari-vs-gallente")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Caldari State") || !strings.Contains(w.Body.String(), "https://images.evetech.net/corporations/1000035/logo") || !strings.Contains(w.Body.String(), "data-fw-systems hidden") || reader.members != 0 {
		t.Fatalf("page status=%d members=%d", w.Code, reader.members)
	}
	w = request("/_fragments/faction-war/caldari-vs-gallente/kills?side=side1&page=2&limit=100&after=123")
	if w.Code != 200 || reader.side != "side1" || reader.after == nil || *reader.after != 123 || strings.Contains(w.Body.String(), "<html") {
		t.Fatal("killfeed fragment lost its side/cursor")
	}
	w = request("/_fragments/faction-war/caldari-vs-gallente/sidebar?side=side2&period=yesterday")
	if w.Code != 200 || reader.period != "yesterday" {
		t.Fatal("leaderboard period not applied")
	}
	w = request("/_fragments/faction-war/caldari-vs-gallente/systems")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Hasmijaala") || !strings.Contains(w.Body.String(), "faction-war-systems-data") {
		t.Fatal("systems fragment missing map data")
	}
	reader.fail = true
	if w = request("/_fragments/faction-war/caldari-vs-gallente/systems"); w.Code != 503 {
		t.Fatal("unavailable systems should be retryable")
	}
	reader.fail = false
	if w = request("/_fragments/faction-war/caldari-vs-gallente/systems"); w.Code != 200 {
		t.Fatal("retry should succeed")
	}
	for _, path := range []string{"/faction-war/unknown", "/faction-war/caldari-vs-gallente/unknown", "/_fragments/faction-war/unknown/systems", "/_fragments/faction-war/caldari-vs-gallente/unknown"} {
		if w = request(path); w.Code != 404 {
			t.Fatalf("invalid route %s: %d", path, w.Code)
		}
	}
	for _, path := range []string{"kills?side=both", "kills?side=side1&after=-1", "kills?side=side1&limit=101", "sidebar?side=combined&period=bad", "members?side=combined&sort=bad"} {
		if w = request("/_fragments/faction-war/caldari-vs-gallente/" + path); w.Code != 400 {
			t.Fatalf("invalid selection %s: %d", path, w.Code)
		}
	}
}
func TestFactionWarCombinedLeaderboardIsStableAndDoesNotMutateCachedSides(t *testing.T) {
	sides := api.FactionWarLeaderboardSides{Side1: []api.FactionWarLeaderboardEntry{{ID: 1, Kills: 10}, {ID: 2, Kills: 9}}, Side2: []api.FactionWarLeaderboardEntry{{ID: 3, Kills: 10}, {ID: 4, Kills: 11}}}
	rows := factionWarLeaders(sides, "combined")
	if rows[0].ID != 4 || rows[1].ID != 1 || rows[2].ID != 3 || sides.Side1[0].ID != 1 || sides.Side2[0].ID != 3 {
		t.Fatal("combined leaderboard ordering or cached slice changed")
	}
}

func TestFactionWarControlIncludesOtherOccupiersAndRoundsLikeNuxt(t *testing.T) {
	overview := api.FactionWarOverview{}
	overview.Warzone.Total = 10
	overview.Warzone.Side1.Total = 4
	overview.Warzone.Side2.Total = 4
	if factionWarControlPct(overview) != 40 {
		t.Fatal("control must include systems occupied by a third faction")
	}
	if factionWarRoundedPct(api.FactionWarSystem{VP: 1, Threshold: 200}) != "1" {
		t.Fatal("half percent must round up like Math.round")
	}
}
