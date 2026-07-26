package api

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func TestFactionWarRoutesRegisterLegacyAndConsolidatedPaths(t *testing.T) {
	mux := http.NewServeMux()
	a := humago.New(mux, huma.DefaultConfig("test", "test"))
	registerFactionWarDashboardRoutes(a, Options{})

	for _, path := range []string{
		"/faction-wars",
		"/faction-war/{matchup}",
		"/faction-war/{matchup}/dashboard",
		"/faction-war/{matchup}/overview",
		"/faction-war/{matchup}/systems",
		"/faction-war/{matchup}/members",
		"/faction-war/{matchup}/intel",
	} {
		item := a.OpenAPI().Paths[path]
		if item == nil || item.Get == nil {
			t.Errorf("GET %s was not registered", path)
		}
	}
}

func TestFactionWarMatchupsHaveOneCompleteDefinition(t *testing.T) {
	if got, want := factionWarMatchupOrder, []string{
		"caldari-vs-gallente",
		"amarr-vs-minmatar",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("matchup order = %v, want %v", got, want)
	}

	caldari := factionWarMatchups["caldari-vs-gallente"]
	if caldari.Side1 != (factionWarSide{
		ID: 500001, Name: "Caldari State", CorpID: 1000035, Key: "caldari",
	}) {
		t.Errorf("Caldari definition = %#v", caldari.Side1)
	}
	if caldari.Side2 != (factionWarSide{
		ID: 500004, Name: "Gallente Federation", CorpID: 1000120, Key: "gallente",
	}) {
		t.Errorf("Gallente definition = %#v", caldari.Side2)
	}
	if caldari.ListingKey != "caldariVsGallente" {
		t.Errorf("Caldari/Gallente listing key = %q", caldari.ListingKey)
	}

	amarr := factionWarMatchups["amarr-vs-minmatar"]
	if amarr.Side1.ID != 500003 || amarr.Side1.CorpID != 1000084 ||
		amarr.Side2.ID != 500002 || amarr.Side2.CorpID != 1000051 {
		t.Errorf("Amarr/Minmatar definition = %#v", amarr)
	}
}

func TestParseFactionWarDaysClampsAndDefaults(t *testing.T) {
	tests := map[string]int{
		"":       30,
		"nope":   30,
		"0":      1,
		"-20":    1,
		"1":      1,
		"7.9":    7,
		"365":    365,
		"999999": 365,
		"1e100":  365,
	}
	for raw, want := range tests {
		t.Run(raw, func(t *testing.T) {
			got := parseFactionWarDays(url.Values{"days": {raw}})
			if got != want {
				t.Fatalf("days %q = %d, want %d", raw, got, want)
			}
		})
	}
}

func TestParseFactionWarMemberOptionsHonorsDaysLimitAndFilters(t *testing.T) {
	options := parseFactionWarMemberOptions(url.Values{
		"days":          {"45"},
		"limit":         {"900"},
		"side":          {"defender"},
		"sort":          {"isk"},
		"corporationId": {"123"},
		"allianceId":    {"456"},
	})
	if options.Days != 45 || options.Limit != factionWarMaximumLimit {
		t.Fatalf("days/limit = %d/%d", options.Days, options.Limit)
	}
	if options.Side != "defender" || options.Sort != "isk" {
		t.Fatalf("side/sort = %q/%q", options.Side, options.Sort)
	}
	if options.CorporationID == nil || *options.CorporationID != 123 ||
		options.AllianceID == nil || *options.AllianceID != 456 {
		t.Fatalf("member IDs = %v/%v", options.CorporationID, options.AllianceID)
	}

	defaults := parseFactionWarMemberOptions(url.Values{
		"limit":         {"0"},
		"side":          {"some-faction"},
		"sort":          {"random()"},
		"corporationId": {"-1"},
		"allianceId":    {"not-a-number"},
	})
	if defaults.Limit != 1 || defaults.Side != "combined" ||
		defaults.Sort != "activity" ||
		defaults.CorporationID != nil || defaults.AllianceID != nil {
		t.Fatalf("normalized member options = %#v", defaults)
	}
}

func TestFactionWarMembersQueryParameterizesUserFilters(t *testing.T) {
	matchup := factionWarMatchups["caldari-vs-gallente"]
	corporationID := int32(123)
	allianceID := int32(456)
	since := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)
	query, args := factionWarMembersQuery(
		matchup,
		factionWarMemberOptions{
			Days:          30,
			Limit:         42,
			Side:          "defender",
			Sort:          "isk",
			CorporationID: &corporationID,
			AllianceID:    &allianceID,
		},
		since,
	)

	for _, fragment := range []string{
		"COALESCE(k.side, l.side) = 2",
		"ch.corporation_id = $4",
		"ch.alliance_id = $5",
		"LIMIT $6",
		"COALESCE(k.isk_destroyed, 0) + COALESCE(l.isk_lost, 0)",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("query does not contain %q", fragment)
		}
	}
	if strings.Contains(query, "corporation_id = 123") ||
		strings.Contains(query, "alliance_id = 456") {
		t.Fatal("member filters were interpolated into SQL")
	}

	wantArgs := []any{
		int32(500001),
		int32(500004),
		since,
		int32(123),
		int32(456),
		42,
	}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", args, wantArgs)
	}
}

func TestFactionWarListingTreatsOpponentLossesAsKills(t *testing.T) {
	side := factionWarListingSide(
		factionWarSide{
			ID: 500001, Name: "Caldari State", CorpID: 1000035, Key: "caldari",
		},
		factionWarLossStats{Losses: 7, ISKLost: 70},
		factionWarLossStats{Losses: 12, ISKLost: 120},
		factionWarESIStats{Pilots: 99, SystemsControlled: 17},
	)
	for key, want := range map[string]any{
		"faction_id":         int32(500001),
		"kills":              int64(12),
		"isk_destroyed":      float64(120),
		"losses":             int64(7),
		"isk_lost":           float64(70),
		"pilots":             int64(99),
		"systems_controlled": int64(17),
	} {
		if got := side[key]; got != want {
			t.Errorf("%s = %#v, want %#v", key, got, want)
		}
	}
}

func TestFactionWarIntelMappersPreserveFallbacksNullsAndEmptyArrays(t *testing.T) {
	systems := factionWarIntelSystems([]map[string]any{{
		"system_id":     int32(30000142),
		"system_name":   nil,
		"security":      nil,
		"region_id":     nil,
		"region_name":   nil,
		"kills":         int64(5),
		"isk_destroyed": float64(123.5),
	}})
	if got := systems[0]["system_name"]; got != "System 30000142" {
		t.Errorf("system fallback = %#v", got)
	}
	for _, key := range []string{"security", "region_id", "region_name"} {
		if systems[0][key] != nil {
			t.Errorf("%s = %#v, want null", key, systems[0][key])
		}
	}

	if got := factionWarIntelShips(nil, true); got == nil || len(got) != 0 {
		t.Errorf("empty ships = %#v, want non-nil empty slice", got)
	}
}
