package api

import (
	"context"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func TestEntityTopRegistrarRegistersThreeLiveFrontendRoutes(t *testing.T) {
	a := humachi.New(
		chi.NewRouter(),
		huma.DefaultConfig("entity-top-test", "test"),
	)
	registerEntityTopRoutes(a, Options{})

	for _, kind := range []string{"character", "corporation", "alliance"} {
		path := "/" + kind + "/{id}/top"
		item := a.OpenAPI().Paths[path]
		if item == nil || item.Get == nil {
			t.Errorf("GET %s was not registered", path)
			continue
		}
		wantID := "entity-top-" + kind + "-compat"
		if item.Get.OperationID != wantID {
			t.Errorf(
				"GET %s operation = %q, want %q",
				path, item.Get.OperationID, wantID,
			)
		}
	}
	if item := a.OpenAPI().Paths["/entities/{type}/{id}/top"]; item != nil {
		t.Error("unrequested generic entity-top route was registered")
	}
}

func TestParseEntityTopParamsMatchesFrontendClamp(t *testing.T) {
	for _, test := range []struct {
		query   url.Values
		slice   string
		days    float64
		allTime bool
	}{
		{url.Values{}, "left", 7, false},
		{url.Values{"days": {"0.001"}}, "left", 1.0 / 24.0, false},
		{url.Values{"days": {"500"}}, "left", 365, false},
		{url.Values{"slice": {"right"}, "days": {"alltime"}}, "right", 0, true},
		{url.Values{"slice": {"left"}, "days": {"alltime"}}, "left", 7, false},
		{url.Values{"slice": {"unknown"}, "days": {"30"}}, "left", 30, false},
	} {
		slice, period := parseEntityTopParams(&legacyRequest{Query: test.query})
		if slice != test.slice ||
			period.Days != test.days ||
			period.AllTime != test.allTime {
			t.Errorf(
				"params %#v = %s %#v; want %s days=%v alltime=%t",
				test.query, slice, period,
				test.slice, test.days, test.allTime,
			)
		}
	}
}

func TestEntityTopAlwaysReturnsAllFifteenSections(t *testing.T) {
	result, err := loadEntityTop(
		context.Background(),
		nil,
		entityPageCharacter,
		entityCharacter,
		90000001,
		"left",
		entityTopPeriod{Days: 7},
	)
	if err != nil {
		t.Fatal(err)
	}
	wantKeys := []string{
		"charactersByKills", "charactersByPoints", "charactersByIsk",
		"soloKillers", "corporationsByKills", "shipsUsed", "systems",
		"constellations", "regions", "killedCorporations",
		"killedAlliances", "killedByCorporations", "killedByAlliances",
		"achievementPoints", "recentMembers",
	}
	if len(result) != len(wantKeys) {
		t.Fatalf("section count = %d, want %d: %#v",
			len(result), len(wantKeys), result)
	}
	for _, key := range wantKeys {
		rows, ok := result[key].([]map[string]any)
		if !ok || rows == nil || len(rows) != 0 {
			t.Errorf("%s = %#v, want stable empty array", key, result[key])
		}
	}
}

func TestEntityTopWindowBucketsMatchStatsContract(t *testing.T) {
	for _, test := range []struct {
		period entityTopPeriod
		want   string
	}{
		{entityTopPeriod{Days: 0.5}, "1d"},
		{entityTopPeriod{Days: 1}, "1d"},
		{entityTopPeriod{Days: 7}, "7d"},
		{entityTopPeriod{Days: 14}, "14d"},
		{entityTopPeriod{Days: 30}, "30d"},
		{entityTopPeriod{Days: 90}, "90d"},
		{entityTopPeriod{Days: 180}, "180d"},
		{entityTopPeriod{Days: 365}, "365d"},
		{entityTopPeriod{AllTime: true}, "alltime"},
	} {
		if got := entityTopStatsWindow(test.period); got != test.want {
			t.Errorf("window %#v = %q, want %q", test.period, got, test.want)
		}
	}
}

func TestEntityTopSinceRoundsSubDayMinutesAndWholeDays(t *testing.T) {
	now := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)
	if got := entityTopSince(1.0/24.0, now); !got.Equal(
		now.Add(-time.Hour),
	) {
		t.Errorf("one-hour cutoff = %s", got)
	}
	if got := entityTopSince(7.49, now); !got.Equal(
		now.AddDate(0, 0, -7),
	) {
		t.Errorf("7.49-day cutoff = %s", got)
	}
	if got := entityTopSince(7.5, now); !got.Equal(
		now.AddDate(0, 0, -8),
	) {
		t.Errorf("7.5-day cutoff = %s", got)
	}
}

func TestEntityTopSubDayInteractionSQLUsesAllowlistedColumns(t *testing.T) {
	killed := entityTopSubDayInteractionQuery(
		"corporation_id", "victim_corporation_id",
		"killed", "alliance", 42, time.Time{},
	)
	for _, fragment := range []string{
		"killmail.victim_alliance_id",
		"attacker.corporation_id = $1",
		"LEFT JOIN alliances target",
		"COUNT(DISTINCT killmail.killmail_id)",
	} {
		if !strings.Contains(killed.SQL, fragment) {
			t.Errorf("killed query missing %q:\n%s", fragment, killed.SQL)
		}
	}

	died := entityTopSubDayInteractionQuery(
		"alliance_id", "victim_alliance_id",
		"died", "corporation", 42, time.Time{},
	)
	for _, fragment := range []string{
		"attacker.corporation_id AS id",
		"killmail.victim_alliance_id = $1",
		"LEFT JOIN corporations target",
	} {
		if !strings.Contains(died.SQL, fragment) {
			t.Errorf("died query missing %q:\n%s", fragment, died.SQL)
		}
	}
}

func TestNormalizeEntityTopRowsPreservesPaletteAndNumberShape(t *testing.T) {
	palette := map[string]any{"primary": "#123456"}
	got := normalizeEntityTopRows([]map[string]any{{
		"id": int32(9), "name": nil, "count": int64(12),
		"palette": palette,
	}})
	want := []map[string]any{{
		"id": int64(9), "name": "Unknown", "count": float64(12),
		"palette": palette,
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %#v, want %#v", got, want)
	}
}
