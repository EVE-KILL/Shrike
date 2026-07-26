package api

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestUniverseRouteInventoryUsesRootPaths(t *testing.T) {
	want := []universeRoute{
		{
			Name: "region", Canonical: "/universe/regions/{id}",
			Alias: "/region/{id}", Summary: "Region page data",
			NotFound: "Region not found",
		},
		{
			Name: "constellation", Canonical: "/universe/constellations/{id}",
			Alias: "/constellation/{id}", Summary: "Constellation page data",
			NotFound: "Constellation not found",
		},
		{
			Name: "system", Canonical: "/universe/systems/{id}",
			Alias: "/system/{id}", Summary: "Solar system page data",
			NotFound: "System not found",
		},
		{
			Name: "type", Canonical: "/universe/types/{id}",
			Alias: "/item/{id}", Summary: "Inventory type page data",
			NotFound: "Item not found",
		},
	}
	if len(universeRoutes) != len(want) {
		t.Fatalf("routes = %d, want %d", len(universeRoutes), len(want))
	}
	for i, route := range universeRoutes {
		if route.Name != want[i].Name ||
			route.Canonical != want[i].Canonical ||
			route.Alias != want[i].Alias ||
			route.Summary != want[i].Summary ||
			route.NotFound != want[i].NotFound {
			t.Errorf("route %d = %#v, want %#v", i, route, want[i])
		}
		if route.Load == nil {
			t.Errorf("route %s has no loader", route.Name)
		}
	}
}

func TestParseUniverseIDMatchesPostgresIntegerDomain(t *testing.T) {
	for _, test := range []struct {
		raw  string
		want int64
	}{
		{"1", 1},
		{" 30000142 ", 30000142},
		{"2147483647", 2147483647},
	} {
		got, err := parseUniverseID(test.raw)
		if err != nil || got != test.want {
			t.Errorf("parse %q = %d, %v; want %d", test.raw, got, err, test.want)
		}
	}

	for _, raw := range []string{
		"", "0", "-1", "1.5", "NaN", "2147483648", "not-an-id",
	} {
		_, err := parseUniverseID(raw)
		var apiErr *legacyAPIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("parse %q error = %v, want API error", raw, err)
		}
		if apiErr.Status != 400 || apiErr.Message != "Invalid id" {
			t.Errorf("parse %q error = %#v", raw, apiErr)
		}
	}
}

func TestBuildUniverseCelestialsMatchesFrontendCategories(t *testing.T) {
	rows := []map[string]any{
		{"item_id": int32(1), "group_id": int32(6), "item_name": "Sun"},
		{"item_id": int32(2), "group_id": int32(7), "item_name": "Planet"},
		{"item_id": int32(3), "group_id": int32(8), "item_name": "Moon"},
		{"item_id": int32(4), "group_id": int32(9), "item_name": "Belt"},
		{"item_id": int32(5), "group_id": int32(10), "item_name": "Gate"},
		{"item_id": int32(6), "group_id": int32(99), "item_name": "Other"},
	}
	counts, list := buildUniverseCelestials(rows)
	wantCounts := map[string]int{
		"stars": 1, "planets": 1, "moons": 1, "belts": 1, "stargates": 1,
	}
	if !reflect.DeepEqual(counts, wantCounts) {
		t.Errorf("counts = %v, want %v", counts, wantCounts)
	}
	if len(list) != 4 {
		t.Fatalf("list = %#v, want four visible celestial categories", list)
	}
	if got := []any{
		list[0]["category"], list[1]["category"],
		list[2]["category"], list[3]["category"],
	}; !reflect.DeepEqual(got, []any{"star", "planet", "moon", "belt"}) {
		t.Errorf("categories = %v", got)
	}
	if _, exists := rows[0]["category"]; exists {
		t.Error("source row was mutated")
	}
}

func TestBuildUniverseSystemActivityPreservesHistoryAndSummaries(t *testing.T) {
	now := time.Date(2026, 7, 26, 4, 0, 0, 0, time.UTC)
	rows := []map[string]any{
		{
			"timestamp": now, "ship_kills": int32(2), "npc_kills": nil,
			"pod_kills": int32(1), "ship_jumps": int32(10),
		},
		{
			"timestamp": now.Add(-time.Hour), "ship_kills": int32(3),
			"npc_kills": int32(4), "pod_kills": int32(0),
			"ship_jumps": int32(20),
		},
	}
	got := buildUniverseSystemActivity(rows)
	latest := got["latest"].(map[string]any)
	if latest["npc_kills"] != int64(0) || latest["ship_kills"] != int32(2) {
		t.Errorf("latest = %#v", latest)
	}
	wantSummary := map[string]int64{
		"ship_kills": 5, "npc_kills": 4, "pod_kills": 1, "ship_jumps": 30,
	}
	if !reflect.DeepEqual(got["summary_24h"], wantSummary) {
		t.Errorf("summary = %#v, want %#v", got["summary_24h"], wantSummary)
	}
	if !reflect.DeepEqual(got["history"], rows) {
		t.Error("activity history order changed")
	}

	empty := buildUniverseSystemActivity(nil)
	if empty["latest"] != nil || empty["summary_24h"] != nil {
		t.Errorf("empty activity = %#v", empty)
	}
	history := empty["history"].([]map[string]any)
	if history == nil || len(history) != 0 {
		t.Errorf("empty history = %#v, want []", history)
	}
}

func TestBuildUniverseTypeAttributesPreservesShipGroupingAndFlatItems(t *testing.T) {
	values := map[int64]float64{
		263: 1000, 12: 8, 182: 3300, 277: 5, 422: 2, 633: 4,
	}
	groupedValue, flat := buildUniverseTypeAttributes(nil, values, true)
	grouped := groupedValue.(map[string][]map[string]any)
	if len(flat) != 0 {
		t.Errorf("ship flat attributes = %#v, want []", flat)
	}
	if got := grouped["defense"]; len(got) != 1 ||
		got[0]["name"] != "Shield HP" || got[0]["value"] != float64(1000) {
		t.Errorf("defense = %#v", got)
	}
	if got := grouped["fitting"]; len(got) != 1 || got[0]["id"] != int64(12) {
		t.Errorf("fitting = %#v", got)
	}
	if _, exists := grouped["navigation"]; exists {
		t.Error("empty ship attribute group was emitted")
	}

	rows := []map[string]any{
		{"attribute_id": int32(30), "value": float64(15)},
		{"attribute_id": int32(50), "value": float64(0)},
		{"attribute_id": int32(182), "value": float64(3300)},
		{"attribute_id": int32(633), "value": float64(4)},
	}
	groupedValue, flat = buildUniverseTypeAttributes(rows, nil, false)
	if groupedValue != nil {
		t.Errorf("non-ship grouped attributes = %#v, want null", groupedValue)
	}
	wantFlat := []map[string]any{{"id": int64(30), "value": float64(15)}}
	if !reflect.DeepEqual(flat, wantFlat) {
		t.Errorf("flat = %#v, want %#v", flat, wantFlat)
	}
}

func TestUniverseTypeDerivedReferenceData(t *testing.T) {
	skills := buildUniverseRequiredSkills(
		map[int64]float64{182: 3300, 277: 5, 183: 4400, 278: 3},
		map[int64]any{3300: "Spaceship Command"},
	)
	wantSkills := []map[string]any{
		{"type_id": int64(3300), "name": "Spaceship Command", "level": float64(5)},
		{"type_id": int64(4400), "name": nil, "level": float64(3)},
	}
	if !reflect.DeepEqual(skills, wantSkills) {
		t.Errorf("skills = %#v, want %#v", skills, wantSkills)
	}

	breadcrumb := buildUniverseMarketBreadcrumb([]map[string]any{
		{"market_group_id": int32(1), "name": "Ammunition & Charges"},
		{"market_group_id": int32(2), "name": nil},
	})
	if breadcrumb[0]["slug"] != "ammunition-and-charges" ||
		breadcrumb[1]["name"] != "" || breadcrumb[1]["slug"] != "" {
		t.Errorf("breadcrumb = %#v", breadcrumb)
	}
}

func TestSummarizeUniversePricesMatchesJavaScriptMath(t *testing.T) {
	rows := []map[string]any{
		{
			"date": "2026-07-26", "average": float64(10.4),
			"highest": float64(12), "lowest": float64(9), "volume": int64(100),
		},
		{
			"date": "2026-07-25", "average": nil,
			"highest": float64(20), "lowest": float64(0), "volume": int64(200),
		},
	}
	got := summarizeUniverseMarketPrices(rows)
	want := map[string]any{
		"latest": float64(10.4), "latest_date": "2026-07-26",
		"average_90d": float64(5), "highest_90d": float64(20),
		"lowest_90d": float64(9), "avg_volume_90d": float64(150),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("market summary = %#v, want %#v", got, want)
	}
	if got := summarizeUniverseMarketPrices([]map[string]any{
		{"average": float64(1), "highest": nil, "lowest": nil, "volume": nil},
	}); got["lowest_90d"] != nil {
		t.Errorf("empty positive low became %v, want JSON null", got["lowest_90d"])
	}
	if summarizeUniverseMarketPrices(nil) != nil {
		t.Error("empty market history should have a null summary")
	}

	custom := summarizeUniverseCustomPrices([]map[string]any{
		{"date": "9999-12-31", "price": float64(10)},
		{"date": "2026-07-01", "price": float64(21)},
	})
	wantCustom := map[string]any{
		"latest": float64(10), "latest_date": "9999-12-31",
		"average_90d": float64(16), "highest_90d": float64(21),
		"lowest_90d": float64(10),
	}
	if !reflect.DeepEqual(custom, wantCustom) {
		t.Errorf("custom summary = %#v, want %#v", custom, wantCustom)
	}
}
