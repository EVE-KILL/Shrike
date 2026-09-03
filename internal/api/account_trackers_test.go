package api

import (
	"strings"
	"testing"
)

func TestTrackerKilllistQueryBoundsBeforeEnrichment(t *testing.T) {
	query := trackerKilllistQuery(" AND event.killmail_id < $3", 4)

	for _, expected := range []string{
		"WITH tracked_kills AS MATERIALIZED",
		"event.tracker_id = $1",
		"tracker.character_id = $2",
		"event.killmail_id < $3",
		"LIMIT $4",
		"FROM tracked_kills tracked",
		"JOIN killmails k ON k.killmail_id = tracked.killmail_id",
	} {
		if !strings.Contains(query, expected) {
			t.Errorf("query missing %q", expected)
		}
	}
	if strings.Contains(query, "WHERE EXISTS") {
		t.Fatal("tracker query regressed to scanning killmails for event existence")
	}
}

func TestTrackedBoardKilllistQueryDeduplicatesAndFiltersBeforeLimit(t *testing.T) {
	query := trackedBoardKilllistQuery(
		" AND event.killmail_id < $2", "k.is_solo IS TRUE", 3,
	)
	for _, expected := range []string{
		"SELECT DISTINCT event.killmail_id",
		"tracker.character_id = $1",
		"event.killmail_id < $2",
		"k.is_solo IS TRUE",
		"LIMIT $3",
		"FROM tracked_kills tracked",
	} {
		if !strings.Contains(query, expected) {
			t.Errorf("query missing %q", expected)
		}
	}
}

func TestTrackedDashboardWidgetSanitizerKeepsLayoutSeparateFromTrackers(t *testing.T) {
	badRatio := "400px_1fr"
	latest := "latest"
	input := SiteDomainWidgets{
		Top: []SiteDomainWidget{
			{Type: "mostValuable", Enabled: true},
			{Type: "campaigns", Enabled: true},
		},
		Left: []SiteDomainWidget{
			{Type: "topCharacters", Enabled: true},
			{Type: "topCharacters", Enabled: true},
		},
		Right: []SiteDomainWidget{
			{Type: "killList", Enabled: true, KilllistType: &latest},
		},
		ColumnRatio: &badRatio,
	}
	got, valid := sanitizeTrackedDashboardWidgets(input)
	if !valid {
		t.Fatal("valid dashboard was rejected")
	}
	if len(got.Top) != 1 || got.Top[0].Type != "mostValuable" {
		t.Fatalf("top widgets = %#v", got.Top)
	}
	if len(got.Left) != 1 || got.Left[0].Type != "topCharacters" {
		t.Fatalf("left widgets = %#v", got.Left)
	}
	if got.ColumnRatio == nil || *got.ColumnRatio != "250px_1fr" {
		t.Fatalf("column ratio = %v", got.ColumnRatio)
	}
}

func TestTrackedDashboardWidgetSanitizerRequiresEnabledWidget(t *testing.T) {
	_, valid := sanitizeTrackedDashboardWidgets(SiteDomainWidgets{
		Right: []SiteDomainWidget{{Type: "killList", Enabled: false}},
	})
	if valid {
		t.Fatal("dashboard without an enabled widget was accepted")
	}
}
