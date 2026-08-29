package api

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func TestGlobalStatsRegistrarRegistersEstablishedStatsRoute(t *testing.T) {
	a := humachi.New(
		chi.NewRouter(),
		huma.DefaultConfig("global-stats-test", "test"),
	)
	registerGlobalStatsRoute(a, Options{})
	item := a.OpenAPI().Paths["/stats"]
	if item == nil || item.Get == nil {
		t.Error("GET /stats was not registered")
	}
	if item := a.OpenAPI().Paths["/stats/rankings"]; item != nil {
		t.Error("rankings leaked into the established response-schema registrar")
	}
}

func TestGlobalStatsDaysCapsAtFrontendNinetyDays(t *testing.T) {
	for _, test := range []struct {
		raw  string
		want float64
	}{
		{"", 7},
		{"0", 7},
		{"0.5", 0.5},
		{"30", 30},
		{"90", 90},
		{"365", 90},
		{"Infinity", 90},
	} {
		if got := globalStatsDays(test.raw); got != test.want {
			t.Errorf("days %q = %v, want %v", test.raw, got, test.want)
		}
	}
}

func TestRealtimeGlobalStatsWindowUsesRoundedMinutes(t *testing.T) {
	minutes, err := validateRealtimeGlobalStatsWindow(0.5)
	if err != nil || minutes != 30 {
		t.Fatalf("30 minute window = %d, %v", minutes, err)
	}
	minutes, err = validateRealtimeGlobalStatsWindow(12)
	if err != nil || minutes != 720 {
		t.Fatalf("12 hour window = %d, %v", minutes, err)
	}
	if _, err := validateRealtimeGlobalStatsWindow(9.0 / 60); err == nil {
		t.Error("nine minute realtime window was accepted")
	}
	if _, err := validateRealtimeGlobalStatsWindow(25); err == nil {
		t.Error("25 hour realtime window was accepted")
	}
}

func TestRealtimeGlobalStatsDispatchMatchesFrontend(t *testing.T) {
	want := []string{
		"characters", "corporations", "alliances",
		"isk_destroyers_chars", "isk_destroyers_corps",
		"isk_destroyers_alliances", "biggest_losers",
		"solo_killers", "top_points", "systems", "regions",
		"dangerous_systems", "deadliest_regions", "ships",
		"most_used_ships", "most_destroyed_ships",
	}
	for _, dataType := range want {
		if _, ok := realtimeGlobalStatsQueries[dataType]; !ok {
			t.Errorf("missing realtime query for %s", dataType)
		}
	}
	if len(realtimeGlobalStatsQueries) != len(want) {
		t.Errorf(
			"realtime dispatch has %d entries, want %d",
			len(realtimeGlobalStatsQueries), len(want),
		)
	}
}

func TestNormalizeRealtimeGlobalStatsPreservesFrontendShape(t *testing.T) {
	rows := []map[string]any{{
		"id": int32(9), "name": nil, "metric": float64(123.5),
	}}
	got := normalizeRealtimeGlobalStats(rows, "corporation", true)
	want := []map[string]any{{
		"id": int32(9), "name": "Unknown",
		"count": float64(123.5), "isk": float64(123.5),
		"type": "corporation",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalized rows = %#v, want %#v", got, want)
	}
}

func TestApplyGlobalStatsPalettesUsesEntityInheritanceResults(t *testing.T) {
	entries := []map[string]any{
		{"id": int32(1), "type": "character"},
		{"id": int32(2), "type": "corporation"},
		{"id": int32(3), "type": "alliance"},
		{"id": int32(4), "type": "ship"},
	}
	palette := map[string]any{"primary": "#112233"}
	got := applyGlobalStatsPalettes(entries, []map[string]any{
		{"entity_type": "character", "id": int32(1), "palette": palette},
		{"entity_type": "corporation", "id": int32(2), "palette": nil},
	})
	if !reflect.DeepEqual(got[0]["palette"], palette) {
		t.Errorf("character palette = %#v", got[0]["palette"])
	}
	if value, exists := got[1]["palette"]; !exists || value != nil {
		t.Errorf("corporation palette = %#v, exists=%t", value, exists)
	}
	if value, exists := got[2]["palette"]; !exists || value != nil {
		t.Errorf("alliance palette = %#v, exists=%t", value, exists)
	}
	if _, exists := got[3]["palette"]; exists {
		t.Error("ship entry unexpectedly received a palette")
	}
}

func TestGlobalStatsLimitRetainsFrontendCap(t *testing.T) {
	if got := globalStatsLimit("500"); got != 100 {
		t.Errorf("limit cap = %d, want 100", got)
	}
	if got := globalStatsLimit("0"); got != 10 {
		t.Errorf("zero limit = %d, want default 10", got)
	}
	if math.IsNaN(globalStatsDays("not-a-number")) {
		t.Error("invalid days produced NaN")
	}
}

func TestGlobalStatsIncludesFactionLeaderboard(t *testing.T) {
	if _, ok := validGlobalStatsTypes["factions"]; !ok {
		t.Fatal("factions are not an accepted global stats type")
	}
	if got := globalStatsEntityFilter("factions"); !strings.Contains(
		got, "factions.militia_corporation_id IS NOT NULL",
	) {
		t.Fatalf("faction filter = %q, want militia filter", got)
	}
	if got := globalStatsEntityFilter("characters"); got != "" {
		t.Fatalf("character filter = %q, want none", got)
	}

	document := New(Options{}).document
	operation := document.Paths["/stats"].Get
	for _, parameter := range operation.Parameters {
		if parameter.Name != "dataType" {
			continue
		}
		for _, value := range parameter.Schema.Enum {
			if value == "factions" {
				return
			}
		}
		t.Fatal("global stats OpenAPI dataType enum omits factions")
	}
	t.Fatal("global stats OpenAPI operation has no dataType parameter")
}
