package frontend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type testUniverseReader struct{}

func (testUniverseReader) UniverseSystem(_ context.Context, id int64) (api.UniverseSystemData, bool, error) {
	if id != 30002813 {
		return api.UniverseSystemData{}, false, nil
	}
	var data api.UniverseSystemData
	data.System.SolarSystemID = id
	data.System.SystemName = "Tama"
	data.System.RegionID = 10000033
	data.System.RegionName = "The Citadel"
	data.System.ConstellationID = 20000396
	data.System.ConstellationName = "Isoma"
	return data, true, nil
}
func (testUniverseReader) UniverseRegion(_ context.Context, id int64) (api.UniverseRegionData, bool, error) {
	if id != 10000042 {
		return api.UniverseRegionData{}, false, nil
	}
	var data api.UniverseRegionData
	data.Region.RegionID = id
	data.Region.Name = "Metropolis"
	data.Region.ConstellationCount = 23
	data.Region.SystemCount = 158
	return data, true, nil
}
func (testUniverseReader) UniverseConstellation(_ context.Context, id int64) (api.UniverseConstellationData, bool, error) {
	if id != 20000396 {
		return api.UniverseConstellationData{}, false, nil
	}
	var data api.UniverseConstellationData
	data.Constellation.ConstellationID = id
	data.Constellation.ConstellationName = "Agiesseson"
	data.Constellation.RegionID = 10000032
	data.Constellation.RegionName = "Sinq Laison"
	return data, true, nil
}
func (testUniverseReader) UniverseLocationKills(context.Context, string, int64, int, *int64) (api.KilllistPage, error) {
	return api.KilllistPage{}, nil
}
func (testUniverseReader) UniverseLocationValuable(context.Context, string, int64, string) ([]api.ValuableKill, error) {
	return nil, nil
}

func (testUniverseReader) UniverseGroup(_ context.Context, id int64) (api.UniverseGroupData, bool, error) {
	if id == 404 {
		return api.UniverseGroupData{}, false, nil
	}
	return api.UniverseGroupData{Group: api.UniverseGroup{ID: id, Name: "Test Group", CategoryID: 6, CategoryName: "Ship", TypeCount: 2, PublishedTypeCount: 1},
		Types: []api.UniverseGroupType{{ID: 11186, Name: "Test Ship", Published: true, MetaGroupName: "Tech I"}, {ID: 99999, Name: "Hidden Type"}}}, true, nil
}

func (testUniverseReader) UniverseGroupKills(context.Context, int64, int, *int64) (api.KilllistPage, error) {
	return api.KilllistPage{Kills: []api.KilllistRow{{KillmailID: 123}}, HasMore: true}, nil
}

func (testUniverseReader) UniverseItem(_ context.Context, id int64) (api.UniverseItemData, bool, error) {
	if id == 404 {
		return api.UniverseItemData{}, false, nil
	}
	ship := id == 11186
	data := api.UniverseItemData{Item: api.UniverseItem{ID: id, Name: "Test Item", GroupID: 25, GroupName: "Frigate", IsShip: ship, Published: true, Volume: 0.01}}
	if id != 35 {
		data.MarketBreadcrumb = []api.UniverseMarketCrumb{{ID: 1, Name: "Ships", Slug: "ships"}}
	}
	return data, true, nil
}

func (testUniverseReader) UniverseItemKills(context.Context, int64, bool, int, *int64) (api.KilllistPage, error) {
	return api.KilllistPage{Kills: []api.KilllistRow{{KillmailID: 123}}, HasMore: true}, nil
}

func (testUniverseReader) UniverseMarketOrders(context.Context, int64, int32, string) (api.UniverseMarketOrders, error) {
	return api.UniverseMarketOrders{Regions: []api.UniverseMarketRegion{{ID: 10000002, Name: "The Forge"}},
		Sellers: []api.UniverseMarketOrder{{OrderID: 1, Price: 100, LocationName: "Jita", SystemID: 30000142}},
		Buyers:  []api.UniverseMarketOrder{{OrderID: 2, Price: 90, LocationName: "Jita", SystemID: 30000142}}}, nil
}

func (testUniverseReader) UniverseMarketHistory(context.Context, int64, int32, int) (api.UniverseMarketHistory, error) {
	first, second := 100.0, 110.0
	volume := int64(10)
	return api.UniverseMarketHistory{Days: 30, History: []api.UniverseMarketHistoryPoint{
		{Date: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Average: &first, Volume: &volume},
		{Date: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), Average: &second, Volume: &volume},
	}}, nil
}

func (testUniverseReader) UniverseFitFamilies(context.Context, int64, url.Values) (api.UniverseFitFamilies, error) {
	return api.UniverseFitFamilies{ShipTypeID: 11186, WindowDays: 90, Families: []api.UniverseFitFamily{{FamilyHash: "abc", TotalUses: 12, VariantCount: 2, Modules: []api.UniverseFitModule{{SlotGroup: 1, TypeID: 42, Name: "Rocket Launcher"}}}}}, nil
}
func (testUniverseReader) UniverseFitMeta(context.Context, int64, url.Values) (api.UniverseFitMeta, error) {
	return api.UniverseFitMeta{TotalKills: 12, Groups: []api.UniverseFitMetaGroup{{ID: 52, Name: "Warp Scrambler", Pct: 90}}}, nil
}
func (testUniverseReader) UniverseFitDistributions(context.Context, int64, url.Values) (api.UniverseFitDistributions, error) {
	return api.UniverseFitDistributions{WindowDays: 90, Metrics: []api.UniverseFitDistributionMetric{{Metric: "ehp", Median: 5000, P10: 3000, P90: 8000, Buckets: []api.UniverseFitDistributionBucket{{Bucket: 1, LowerBound: 3000, UpperBound: 4000, ObservationCount: 2}}}}}, nil
}

func (testUniverseReader) UniverseMarketTree(context.Context) ([]api.UniverseMarketTreeNode, error) {
	return []api.UniverseMarketTreeNode{{ID: 4, Name: "Ships", Slug: "ships", Children: []api.UniverseMarketTreeNode{{ID: 5, Name: "Frigates", Slug: "frigates", HasTypes: true}, {ID: 6, Name: "Unique", Slug: "unique", HasTypes: true}}}}, nil
}
func (testUniverseReader) UniverseMarketGroupItems(_ context.Context, id int64) ([]api.UniverseMarketGroupItem, error) {
	if id == 6 {
		return []api.UniverseMarketGroupItem{{ID: 11186, Name: "Test Item", MarketGroupID: 6, IsShip: true}}, nil
	}
	return []api.UniverseMarketGroupItem{{ID: 11186, Name: "Test Item", MarketGroupID: 5, IsShip: true}, {ID: 11187, Name: "Other Item", MarketGroupID: 5}}, nil
}

func TestGroupPagesAndFragments(t *testing.T) {
	handler := New(Options{Production: true, Universe: testUniverseReader{}})
	for _, tc := range []struct {
		path   string
		status int
		marker string
	}{
		{"/group/25", 200, "Test Ship"},
		{"/group/25/kills", 200, `id="kill-list"`},
		{"/_fragments/group/tab/25/dashboard", 200, `data-group-tab="dashboard"`},
		{"/_fragments/group/tab/25/kills", 200, `data-group-tab="kills"`},
		{"/_fragments/group/kills/25?page=2&limit=10&after=42", 200, `data-page="2"`},
		{"/_fragments/group/kills/25?page=2", 400, "Invalid page"},
		{"/group/404", 404, "Page not found"},
		{"/group/25/other", 404, "Page not found"},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+tc.path, nil))
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.marker) {
			t.Errorf("%s: status %d, marker %q present = %v", tc.path, w.Code, tc.marker, strings.Contains(w.Body.String(), tc.marker))
		}
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+"/_fragments/group/tab/25/dashboard", nil))
	if strings.Contains(w.Body.String(), "group-hero") || strings.Contains(w.Body.String(), "group-hull-strip") {
		t.Fatal("group tab fragment includes persistent header")
	}
}
