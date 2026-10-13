package frontend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
)

type testEntityReader struct{}

func (testEntityReader) EntityDetail(_ context.Context, kind string, _ int64) (map[string]any, error) {
	return map[string]any{
		kind:          map[string]any{"name": "Test Entity", "ticker": "TEST", "palette": map[string]string{"main_color": "#dd8822"}},
		"stats":       map[string]any{"kills": 42, "losses": 2, "efficiency": 95},
		"recentStats": map[string]any{"kills": 4},
		"rankings":    map[string]any{"all_time": map[string]any{"eve_kill_rating": 8, "overall_rank": 9, "population": 100}},
	}, nil
}

func (testEntityReader) EntityKilllist(context.Context, string, int64, string, int, int, *int64, int64) (api.KilllistPage, error) {
	return api.KilllistPage{TotalPages: 2, HasMore: true}, nil
}

func (testEntityReader) EntityTopLists(context.Context, string, int64, int) (api.EntityTopLists, error) {
	return api.EntityTopLists{Killed: api.EntityTopCollection{Characters: []api.TopEntry{{ID: 123, Name: "Target", Count: 4}}}}, nil
}

func (testEntityReader) EntityDashboard(context.Context, string, int64, int) (api.EntityDashboard, error) {
	return api.EntityDashboard{Stats: api.EntityDashboardStats{Kills: 10, Losses: 1, Efficiency: 91}, HeatMap: map[string]int64{"0": 5}}, nil
}

func (testEntityReader) EntityTop(context.Context, string, int64, string, string) (api.EntityTopData, error) {
	return api.EntityTopData{ShipsUsed: []api.TopEntry{{ID: 11186, Name: "Ship", Count: 5}}}, nil
}

func (testEntityReader) EntityAchievements(context.Context, int64) ([]api.EntityAchievement, error) {
	return []api.EntityAchievement{{ID: "test", Name: "Test Achievement", Category: "Combat", Rarity: "common", Threshold: 1}}, nil
}

func (testEntityReader) EntityMembers(_ context.Context, _ string, _ int64, page int, _ string, _ int, _ int64) (api.EntityMembersData, error) {
	return api.EntityMembersData{Members: []api.EntityMember{{CharacterID: 42, Name: "Test Pilot", Kills90D: 3}}, Total: 120, Page: page, Limit: 100}, nil
}

func (testEntityReader) EntityCorporations(context.Context, int64, string, string) (api.EntityCorporationsData, error) {
	return api.EntityCorporationsData{Corporations: []api.EntityCorporation{{CorporationID: 7, Name: "Test Corp", Ticker: "TC", MemberCount: 42}}, Total: 1}, nil
}

func (testEntityReader) EntityBattles(context.Context, string, int64, int) (api.EntityBattlesData, error) {
	return api.EntityBattlesData{Battles: []api.EntityBattleEntry{{BattleID: 12, SolarSystemName: "Test System", RegionName: "Test Region"}}, Page: 1}, nil
}

func (testEntityReader) EntityWars(context.Context, string, int64, int, string) (api.EntityWarsData, error) {
	return api.EntityWarsData{Wars: []api.EntityWarEntry{{WarID: 23, Aggressor: api.EntityWarSide{Name: "Test Aggressor"}, Defender: api.EntityWarSide{Name: "Test Defender"}}}, Page: 1}, nil
}

func (testEntityReader) EntityCampaigns(context.Context, string, int64, int) (api.EntityCampaignsData, error) {
	return api.EntityCampaignsData{Campaigns: []api.EntityCampaignEntry{{CampaignID: "test-campaign", Name: "Test Campaign"}}, Page: 1, HasMore: true}, nil
}

func (testEntityReader) EntityIntel(context.Context, string, int64, int) (api.EntityIntelData, error) {
	return api.EntityIntelData{DominantStyle: "Small Gang", ShipsFlown: []api.TopEntry{{ID: 11186, Name: "Test Ship", Count: 3}}}, nil
}

func (testEntityReader) EntityFactionOverview(context.Context, int64) (api.EntityFactionOverview, error) {
	return api.EntityFactionOverview{Matchup: "amarr-vs-minmatar", Side: 1}, nil
}
func (testEntityReader) EntityFactionMilitiaCorps(context.Context, int64) ([]api.EntityFactionMilitiaCorp, error) {
	return nil, nil
}

func TestEntityPageRoutesAndFragments(t *testing.T) {
	handler := New(Options{Production: true, Entities: testEntityReader{}})
	for _, tt := range []struct {
		path, title, marker string
	}{
		{"/character/2121270074", "Test Entity", "Most Valuable Kills"},
		{"/corporation/98598862", "Test Entity [TEST]", "Most Valuable Kills"},
		{"/alliance/99003581", "Test Entity [TEST]", "Most Valuable Kills"},
		{"/faction/500003", "Test Entity [TEST]", "Last 90 Days"},
		{"/character/2121270074/kills", "Test Entity [TEST] (Kills)", `id="entity-kill-list"`},
		{"/character/2121270074/top", "Test Entity [TEST] (Top)", "Most Used Ships"},
		{"/character/2121270074/achievements", "Test Entity [TEST] (Achievements)", "Test Achievement"},
		{"/character/2121270074/losses?ship_group=26", "Test Entity [TEST] (Losses)", `data-ship-group="26"`},
		{"/corporation/98598862/members", "Test Entity [TEST] (Members)", "Test Pilot"},
		{"/alliance/99003581/corporations", "Test Entity [TEST] (Corporations)", "Test Corp"},
		{"/alliance/99003581/members", "Test Entity [TEST] (Members)", "All Corporations"},
		{"/character/2121270074/battles", "Test Entity [TEST] (Battles)", "Test System"},
		{"/corporation/98598862/wars", "Test Entity [TEST] (Wars)", "Test Aggressor"},
		{"/alliance/99003581/campaigns", "Test Entity [TEST] (Campaigns)", "Test Campaign"},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+tt.path, nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), tt.title) || !strings.Contains(w.Body.String(), tt.marker) {
			t.Errorf("%s: status=%d title=%v marker=%v", tt.path, w.Code,
				strings.Contains(w.Body.String(), tt.title), strings.Contains(w.Body.String(), tt.marker))
		}
	}
	for _, path := range []string{"/character/0", "/character/nope", "/character/5/unknown", "/faction/1", "/character/5/kills?ship_group=bad"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+path, nil))
		if w.Code != 404 {
			t.Errorf("%s: status=%d, want 404", path, w.Code)
		}
	}
	for _, tt := range []struct {
		path     string
		status   int
		contains string
	}{
		{"/_fragments/entity/killlist/character/5?role=kills", 200, `id="entity-kill-list"`},
		{"/_fragments/entity/killlist/character/5?role=wrong", 400, "Invalid role"},
		{"/_fragments/entity/top-lists/character/5?tab=kills", 200, "Target"},
		{"/_fragments/entity/dashboard/character/5?days=30", 200, "Activity Heat Map"},
		{"/_fragments/entity/top/character/5?slice=right&days=30", 200, "Most Used Ships"},
		{"/_fragments/entity/members/corporation/5?page=2&sort=name&activity=7", 200, "Test Pilot"},
		{"/_fragments/entity/corporations/5?sort=name&dir=asc", 200, "Test Corp"},
		{"/_fragments/entity/members/corporation/5?activity=23", 400, "Invalid activity filter"},
		{"/_fragments/entity/battles/character/5?page=2", 200, "Test System"},
		{"/_fragments/entity/wars/alliance/5?page=2&sort=isk", 200, "Test Aggressor"},
		{"/_fragments/entity/wars/alliance/5?sort=wrong", 400, "Invalid war sort"},
		{"/_fragments/entity/campaigns/alliance/5?page=2", 200, "Test Campaign"},
		{"/_fragments/entity/intel/character/5?days=90", 200, "Small Gang"},
		{"/_fragments/entity/intel/character/5?days=23", 400, "Invalid period"},
		{"/_fragments/entity/faction/500003", 200, "Warzone"},
		{"/_fragments/entity/tab/character/5/dashboard", 200, `data-entity-tab="dashboard"`},
		{"/_fragments/entity/tab/character/5/kills", 200, `id="entity-kill-list"`},
		{"/_fragments/entity/tab/character/5/kills?page=2&limit=100", 200, `data-page="2" data-limit="100"`},
		{"/_fragments/entity/tab/character/5/kills?page=0", 400, "Invalid page"},
		{"/_fragments/entity/tab/character/5/kills?limit=101", 400, "Invalid limit"},
		{"/_fragments/entity/tab/corporation/5/members", 200, "Test Pilot"},
		{"/_fragments/entity/tab/character/5/losses?ship_group=26", 200, `data-ship-group="26"`},
		{"/_fragments/entity/tab/character/5/unknown", 404, "404"},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+tt.path, nil))
		if w.Code != tt.status || !strings.Contains(w.Body.String(), tt.contains) {
			t.Errorf("%s: status=%d body=%s", tt.path, w.Code, w.Body.String())
		}
	}
}

func TestEntityTabFragmentKeepsShellOutOfResponse(t *testing.T) {
	handler := New(Options{Production: true, Entities: testEntityReader{}})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+"/_fragments/entity/tab/character/2121270074/achievements", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `class="entity-main"`) || !strings.Contains(body, `data-page-title="Test Entity [TEST] (Achievements) - EVE-KILL.com"`) || !strings.Contains(body, "Test Achievement") {
		t.Fatal("tab fragment is missing content or navigation metadata")
	}
	if strings.Contains(body, "entity-hero") || strings.Contains(body, "entity-valuable") || strings.Contains(body, "entity-tabs") {
		t.Fatal("tab fragment unexpectedly includes the persistent shell")
	}
}

type testCorporationDashboardReader struct{ testEntityReader }

func (testCorporationDashboardReader) EntityDashboard(_ context.Context, _ string, _ int64, days int) (api.EntityDashboard, error) {
	return api.EntityDashboard{
		Stats:              api.EntityDashboardStats{Kills: int64(days), Losses: 2, Efficiency: 75},
		TopShipsUsed:       []api.EntityShipEntry{{ShipTypeID: 638, ShipName: "Raven", Count: 12}},
		TopShipsLost:       []api.EntityShipEntry{{ShipTypeID: 670, ShipName: "Capsule", Count: 3}},
		DiesToCorporations: []api.TopEntry{{ID: 8, Name: "Opponent Corp", Count: 4}},
	}, nil
}

func (testCorporationDashboardReader) EntityIntel(context.Context, string, int64, int) (api.EntityIntelData, error) {
	var intel api.EntityIntelData
	intel.ActiveMembers.Days7 = 17
	intel.Allies = []api.EntityIntelPeer{{ID: 9, Name: "Friendly Corp", SharedEnemyKills: 6}}
	intel.HuntingGrounds = []api.EntityIntelPeer{{ID: 30000142, Name: "Jita", ActiveCharacters: 5}}
	intel.RecentJoins = []api.EntityIntelPeer{{ID: 10, Name: "New Pilot", JoinedAt: "2026-09-25T08:00:00Z"}}
	return intel, nil
}

func TestOrganizationDashboardSectionsAndFragments(t *testing.T) {
	handler := New(Options{Production: true, Entities: testCorporationDashboardReader{}})
	for _, tt := range []struct {
		path     string
		sections []string
	}{
		{"/corporation/98598862", []string{`data-entity-intel-slot="active"`, "<h3>Stats</h3>", "Most Used Ships", "Most Lost Ships", `data-entity-intel-slot="relations"`, "Dies To (Corps)", `data-entity-intel-slot="recent"`}},
		{"/alliance/99003581", []string{`data-entity-intel-slot="active"`, "<h3>Stats</h3>", "Most Used Ships", "Most Lost Ships", `data-entity-intel-slot="relations"`, "Dies To (Corps)", `data-entity-intel-slot="recent"`}},
		{"/_fragments/entity/dashboard/corporation/98598862?days=14", []string{`data-days="14"`, `data-entity-period="14"`, "Most Used Ships", "Dies To (Corps)"}},
		{"/_fragments/entity/dashboard/alliance/99003581?days=14", []string{`data-days="14"`, `data-entity-period="14"`, "Most Used Ships", "Dies To (Corps)"}},
		{"/_fragments/entity/intel/corporation/98598862?days=90", []string{`data-entity-intel-section="active"`, "17", `data-entity-intel-section="relations"`, "Friendly Corp", "Jita", `data-entity-intel-section="recent"`, "New Pilot"}},
		{"/_fragments/entity/intel/alliance/99003581?days=90", []string{`data-entity-intel-section="active"`, "17", `data-entity-intel-section="relations"`, "Friendly Corp", "Jita", `data-entity-intel-section="recent"`, "New Pilot"}},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+tt.path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status %d: %s", tt.path, w.Code, w.Body.String())
		}
		body, previous := w.Body.String(), -1
		for _, section := range tt.sections {
			position := strings.Index(body, section)
			if position <= previous {
				t.Errorf("%s: section %q missing or out of order", tt.path, section)
			}
			previous = position
		}
		if strings.Contains(body, "Solo Losses") || strings.Contains(body, "Blob Factor") {
			t.Errorf("%s: corporation dashboard includes character-only stats", tt.path)
		}
	}
}
