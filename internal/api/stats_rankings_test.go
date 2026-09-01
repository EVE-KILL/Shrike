package api

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func TestStatsRankingsRouteRegistersUnifiedRootPath(t *testing.T) {
	a := humachi.New(
		chi.NewRouter(),
		huma.DefaultConfig("stats-rankings-test", "test"),
	)
	registerStatsRankingsRoute(a, Options{})

	item := a.OpenAPI().Paths["/stats/rankings"]
	if item == nil || item.Get == nil {
		t.Fatal("GET /stats/rankings was not registered")
	}
	if item.Get.OperationID != "stats-rankings" {
		t.Errorf("operation id = %q", item.Get.OperationID)
	}
	if !reflect.DeepEqual(item.Get.Tags, []string{"stats"}) {
		t.Errorf("tags = %#v", item.Get.Tags)
	}
	if statsRankingsCacheTTL != time.Minute {
		t.Errorf("cache TTL = %s, want 1m", statsRankingsCacheTTL)
	}
}

func TestStatsRankingsHandlerValidatesSectionBeforeDatabaseWork(t *testing.T) {
	handler := statsRankingsHandler(Options{})

	_, err := handler(context.Background(), &legacyRequest{Query: url.Values{}})
	assertStatsRankingAPIError(
		t, err, http.StatusBadRequest, "Missing section parameter",
	)

	_, err = handler(context.Background(), &legacyRequest{Query: url.Values{
		"section": {"definitely-not-a-board"},
	}})
	assertStatsRankingAPIError(
		t, err, http.StatusBadRequest,
		"Unknown section: definitely-not-a-board",
	)
}

func assertStatsRankingAPIError(
	t *testing.T,
	err error,
	status int,
	message string,
) {
	t.Helper()
	var apiErr *legacyAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want API error", err)
	}
	if apiErr.Status != status || apiErr.Message != message {
		t.Fatalf("error = %#v, want %d %q", apiErr, status, message)
	}
}

func TestStatsRankingNumericParametersMatchFrontendDefaultsAndCaps(t *testing.T) {
	for _, test := range []struct {
		raw   string
		want  int
		valid bool
	}{
		{"", 10, true},
		{"0", 10, true},
		{"nope", 10, true},
		{"1", 1, true},
		{"50", 50, true},
		{"51", 50, true},
		{"50.5", 50, true},
		{"Infinity", 50, true},
		{"-1", 0, false},
		{"2.5", 0, false},
	} {
		got, valid := statsRankingLimit(test.raw)
		if got != test.want || valid != test.valid {
			t.Errorf(
				"limit %q = %d, %t; want %d, %t",
				test.raw, got, valid, test.want, test.valid,
			)
		}
	}

	for _, test := range []struct {
		raw   string
		want  int64
		valid bool
	}{
		{"", 7, true},
		{"0", 7, true},
		{"nope", 7, true},
		{"1", 1, true},
		{"30", 30, true},
		{"-1", 0, false},
		{"2.5", 0, false},
		{"Infinity", 0, false},
	} {
		got, valid := statsRankingDays(test.raw)
		if got != test.want || valid != test.valid {
			t.Errorf(
				"days %q = %d, %t; want %d, %t",
				test.raw, got, valid, test.want, test.valid,
			)
		}
	}
}

func TestRankingEntitySQLIdentifiersAreAllowlisted(t *testing.T) {
	for entityType, expected := range map[string]string{
		"alliance":    "LEFT JOIN alliances",
		"corporation": "LEFT JOIN corporations",
		"character":   "LEFT JOIN characters",
	} {
		selectSQL, joinSQL := rankingNameJoin(entityType)
		if !strings.Contains(selectSQL, "entity.name") ||
			!strings.Contains(joinSQL, expected) {
			t.Errorf(
				"%s name SQL = %q / %q",
				entityType, selectSQL, joinSQL,
			)
		}
	}

	injected := "alliance; DROP TABLE entity_snapshots"
	selectSQL, joinSQL := rankingNameJoin(injected)
	if joinSQL != "" || strings.Contains(selectSQL, injected) {
		t.Fatalf("unknown entity type reached SQL: %q / %q", selectSQL, joinSQL)
	}
}

func TestStatsRankingEntryBuildersPreserveFrontendShapes(t *testing.T) {
	largest := buildLargestRankingEntries([]map[string]any{{
		"entity_id": int32(99), "name": "",
		"member_count": int32(1000),
		"delta_1d":     int32(3),
		"delta_7d":     nil,
		"delta_30d":    int64(-10),
	}}, "alliance")
	wantLargest := []map[string]any{{
		"id": int64(99), "name": "Unknown",
		"member_count": int64(1000),
		"delta_1d":     int64(3),
		"delta_7d":     int64(0),
		"delta_30d":    int64(-10),
		"type":         "alliance",
	}}
	if !reflect.DeepEqual(largest, wantLargest) {
		t.Errorf("largest = %#v, want %#v", largest, wantLargest)
	}

	security := buildSecurityRankingEntries([]map[string]any{{
		"entity_id": int32(7), "name": "Seven",
		"member_count":   int32(50),
		"avg_sec_status": float32(-4.126),
		"weighted_score": float64(16.235),
	}}, "corporation")
	if got := security[0]["avg_sec_status"]; got != -4.13 {
		t.Errorf("security status = %v, want -4.13", got)
	}
	if got := security[0]["weighted_score"]; got != 16.24 {
		t.Errorf("weighted score = %v, want 16.24", got)
	}

	achievements := buildEntityAchievementRankingEntries(
		[]map[string]any{{
			"entity_id": int32(8), "name": "Eight",
			"member_count":             int32(12),
			"total_achievement_points": int32(100),
			"avg_achievement_points":   float32(8.26),
		}},
		"corporation",
	)
	if got := achievements[0]["avg_points"]; math.Abs(got.(float64)-8.3) > 1e-9 {
		t.Errorf("average points = %v, want 8.3", got)
	}
}

func TestNewestRankingFallbackMatchesFrontendCorporationBranch(t *testing.T) {
	for entityType, want := range map[string]string{
		"alliance":    "alliance",
		"character":   "character",
		"corporation": "corporation",
		"unknown":     "corporation",
	} {
		spec := newestRankingQuery(entityType)
		if spec.resultType != want {
			t.Errorf(
				"newest %q type = %q, want %q",
				entityType, spec.resultType, want,
			)
		}
		if strings.Contains(spec.query, entityType) && entityType == "unknown" {
			t.Errorf("unknown entity type reached SQL: %q", spec.query)
		}
	}
}

func TestStatsRankingPayloadAlwaysUsesAnArray(t *testing.T) {
	payload := statsRankingPayload(nil)
	body := payload.Body.(map[string]any)
	entries, ok := body["entries"].([]map[string]any)
	if !ok || entries == nil || len(entries) != 0 {
		t.Fatalf("entries = %#v, want stable []", body["entries"])
	}
}

func TestSystemRankingUsesSolarSystemPrimaryKey(t *testing.T) {
	source, err := os.ReadFile("stats_rankings.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source),
		"solar_systems e ON e.solar_system_id = r.entity_id") {
		t.Fatal("system ranking does not join through solar_system_id")
	}
}
