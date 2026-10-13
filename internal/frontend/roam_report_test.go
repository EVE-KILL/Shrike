package frontend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

const testRoamID = "0123456789abcdef0123456789abcdef"

type testRoamReader struct {
	report func(context.Context, string) (api.RoamReport, bool, error)
	kills  func(context.Context, api.RoamReport, url.Values) (api.KilllistPage, error)
}

func (r testRoamReader) RoamReport(ctx context.Context, id string) (api.RoamReport, bool, error) {
	return r.report(ctx, id)
}
func (r testRoamReader) RoamReportKilllist(ctx context.Context, report api.RoamReport, query url.Values) (api.KilllistPage, error) {
	return r.kills(ctx, report, query)
}

func testRoamReport() api.RoamReport {
	start := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	return api.RoamReport{ID: testRoamID, StartTime: start, EndTime: start.Add(4 * time.Hour),
		Roster: []api.RoamPilot{
			{CharacterID: 1, Name: "Alpha <Pilot>", CorporationID: 10, CorporationName: "Fleet Corp", KillParticipations: 2, Engagements: 1, Ships: []api.RoamShip{{ShipTypeID: 587, ShipName: "Rifter", ShipGroupID: 25, ShipGroupName: "Frigate", Killmails: 2, Losses: 1, DamageDone: 1000}}},
			{CharacterID: 2, Name: "Unobserved Pilot", CorporationName: "Fleet Corp"},
		},
		Summary:     api.RoamSummary{Systems: 1, Regions: 1, Kills: 2, Losses: 1, Engagements: 1, ActivePilots: 1, UniqueCharacterTargets: 2, IskDestroyed: 100e6, IskLost: 25e6, Efficiency: 80},
		Systems:     []api.RoamSystem{{SolarSystemID: 30000142, SolarSystemName: "Jita", RegionID: 10000002, RegionName: "The Forge", FirstSeen: start, LastSeen: start.Add(time.Hour), Kills: 2, Losses: 1, Engagements: 1}},
		Engagements: []api.RoamEngagement{{Number: 1, SolarSystemID: 30000142, SolarSystemName: "Jita", RegionName: "The Forge", StartTime: start, EndTime: start.Add(time.Hour), Kills: 2, Losses: 1, PilotIDs: []int32{1}, Killmails: []api.RoamKillmail{{Role: "loss", VictimCharacterID: 1, VictimShipTypeID: 587, TotalValue: 25e6, FinalBlowCharacterName: "Enemy Pilot", FinalBlowCorpName: "Enemy Corp"}}}},
		Unresolved:  []string{"Missing Pilot"},
	}
}

func TestRoamPagesAndFragmentsPreserveContextAndMutableData(t *testing.T) {
	report := testRoamReport()
	type contextKey struct{}
	reader := testRoamReader{
		report: func(ctx context.Context, id string) (api.RoamReport, bool, error) {
			if id != testRoamID || ctx.Value(contextKey{}) != "request" {
				t.Error("report read lost ID or request context")
			}
			return report, true, nil
		},
		kills: func(ctx context.Context, loaded api.RoamReport, query url.Values) (api.KilllistPage, error) {
			if loaded.ID != report.ID || ctx.Value(contextKey{}) != "request" {
				t.Error("killlist read lost loaded report or context")
			}
			if query.Get("role") != "all" || query.Get("limit") != "25" || query.Get("page") != "2" || query.Get("after") != "777" {
				t.Errorf("killlist filters not preserved: %s", query.Encode())
			}
			return api.KilllistPage{Kills: []api.KilllistRow{{KillmailID: 1337, VictimCharacterID: 1, VictimCharacterName: "Alpha <Pilot>", ShipTypeID: 587, ShipName: "Rifter", KillmailTime: report.StartTime, AttackerCount: 3}}, TotalPages: 2}, nil
		},
	}
	handler := New(Options{Production: true, RoamReports: reader})
	for _, test := range []struct{ path, marker string }{
		{"/tools/roam-report/" + testRoamID, "Engagement trail"},
		{"/tools/roam-report/" + testRoamID + "/information", "Fleet roster"},
		{"/tools/roam-report/" + testRoamID + "/systems", "Roam combat map"},
		{"/tools/roam-report/" + testRoamID + "/composition", "No ship recorded"},
		{"/tools/roam-report/" + testRoamID + "/intel", "Enemy Pilot"},
		{"/tools/roam-report/" + testRoamID + "/kills?role=all&limit=25&page=2&after=777", "loss-row"},
		{"/_fragments/roam/" + testRoamID + "/systems", "data-roam-map"},
		{"/_fragments/roam/" + testRoamID + "/composition", "roam-composition-data"},
		{"/_fragments/roam/" + testRoamID + "/page?tab=intel", "data-roam-report"},
	} {
		t.Run(test.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, canonicalOrigin+test.path, nil).WithContext(context.WithValue(context.Background(), contextKey{}, "request"))
			r.Header.Set("Cookie", "session=test")
			r.Header.Set("Authorization", "Bearer test")
			handler.ServeHTTP(w, r)
			if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), test.marker) {
				t.Fatalf("status=%d missing %q", w.Code, test.marker)
			}
			if w.Header().Get("Cache-Control") != "private, no-store" {
				t.Errorf("mutable report was cacheable: %q", w.Header().Get("Cache-Control"))
			}
			if strings.Contains(w.Body.String(), "Alpha <Pilot>") {
				t.Error("pilot name was not HTML escaped")
			}
		})
	}
}

func TestRoamRejectsInvalidRoutesAndShowsUnavailableReport(t *testing.T) {
	calls := 0
	handler := New(Options{Production: true, RoamReports: testRoamReader{report: func(context.Context, string) (api.RoamReport, bool, error) {
		calls++
		return api.RoamReport{}, false, nil
	}}})
	for _, path := range []string{"/tools/roam-report/invalid", "/tools/roam-report/" + testRoamID + "/edit", "/_fragments/roam/" + testRoamID + "/invalid"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 404 {
			t.Errorf("%s status=%d", path, w.Code)
		}
	}
	if calls != 0 {
		t.Errorf("invalid routes called API %d times", calls)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/tools/roam-report/"+testRoamID, nil))
	if w.Code != 404 || !strings.Contains(w.Body.String(), "Report unavailable") || !strings.Contains(w.Body.String(), "Create a new report") {
		t.Errorf("missing report did not render its unavailable state")
	}
}

func TestRoamDatabaseUnavailablePreservesStatusWithoutLeakingDetails(t *testing.T) {
	handler := New(Options{Production: true, RoamReports: api.NewHomepageReader(nil)})
	for _, path := range []string{"/tools/roam-report/" + testRoamID, "/_fragments/roam/" + testRoamID + "/information"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+path, nil))
		if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "database is not configured") {
			t.Errorf("%s exposed a backend failure or changed its status: %d", path, w.Code)
		}
		if w.Header().Get("Cache-Control") != "private, no-store" || !strings.Contains(w.Header().Get("X-Robots-Tag"), "noindex") {
			t.Error("unavailable report may be cached or indexed")
		}
	}
}

func TestRoamCompositionSortDoesNotChangeAggregationOrder(t *testing.T) {
	report := testRoamReport()
	report.Roster[1].Ships = []api.RoamShip{{ShipTypeID: 11192, ShipName: "Buzzard", Killmails: 5}}
	view := buildRoamComposition(report)
	if view.Rows[0].Pilot.CharacterID != 1 || view.SortedRows[0].Pilot.CharacterID != 2 || report.Roster[0].CharacterID != 1 {
		t.Fatal("SSR sort changed the input order used for stable client aggregates")
	}
}

func TestRoamIntelRecognizesOverlappingSpecialists(t *testing.T) {
	report := testRoamReport()
	report.Roster[0].Ships = []api.RoamShip{{ShipTypeID: 37604, ShipGroupName: "Force Auxiliary"}, {ShipTypeID: 37482, ShipGroupName: "Command Destroyer"}}
	view := buildRoamIntel(report)
	if len(view.Sections[0].Rows) != 1 || len(view.Sections[1].Rows) != 1 || len(view.Sections[2].Rows) != 1 || !roamHasSpecialists(view) {
		t.Fatal("specialist classifications lost command hull or logistics capital")
	}
}

func TestRoamCompositionKeepsUnobservedPilotsAndAccumulatesLosses(t *testing.T) {
	report := testRoamReport()
	report.Engagements[0].Killmails = append(report.Engagements[0].Killmails, api.RoamKillmail{Role: "loss", VictimCharacterID: 1, VictimShipTypeID: 587, TotalValue: 10e6, FriendlyFire: true}, api.RoamKillmail{Role: "kill", VictimCharacterID: 1, VictimShipTypeID: 587, TotalValue: 100e6})
	view := buildRoamComposition(report)
	if len(view.Rows) != 2 || view.ObservedPilots != 1 || view.HullTypes != 1 {
		t.Fatalf("missing roster/observations: %+v", view)
	}
	if view.Rows[0].ISKLost != 35e6 || view.Rows[1].Ship != nil || view.Rows[1].Pilot.Name != "Unobserved Pilot" {
		t.Errorf("composition lost unobserved pilot or miscounted losses: %+v", view.Rows)
	}
	encoded, _ := json.Marshal(view.Rows)
	if strings.Contains(string(encoded), `"ships"`) {
		t.Error("normalized rows duplicate the full pilot ship list")
	}
	intel := buildRoamIntel(report)
	if len(intel.Threats) != 1 || intel.Threats[0].ISK != 25e6 {
		t.Errorf("intel counted friendly fire as an opposing threat: %+v", intel.Threats)
	}
}
