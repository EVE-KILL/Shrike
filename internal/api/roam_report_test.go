package api

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func TestRoamReportRoutes(t *testing.T) {
	mux := http.NewServeMux()
	a := humago.New(mux, huma.DefaultConfig("test", "test"))
	registerRoamReportRoutes(a, Options{})
	if path := a.OpenAPI().Paths["/tools/roam-report"]; path == nil || path.Post == nil || path.Post.RequestBody == nil {
		t.Fatal("create route or request schema is missing")
	}
	if path := a.OpenAPI().Paths["/tools/roam-report/{id}"]; path == nil || path.Get == nil {
		t.Fatal("saved report route is missing")
	}
	if path := a.OpenAPI().Paths["/tools/roam-report/{id}"]; path == nil || path.Put == nil || path.Put.RequestBody == nil {
		t.Fatal("edit report route or request schema is missing")
	}
	if path := a.OpenAPI().Paths["/tools/roam-report/{id}/killlist"]; path == nil || path.Get == nil {
		t.Fatal("report killlist route is missing")
	}
}

func TestRoamShipCompositionTracksMultipleHulls(t *testing.T) {
	report := roamReport{
		Roster: []roamPilot{{CharacterID: 1, Name: "Pilot One"}, {CharacterID: 2, Name: "Pilot Two"}},
		Engagements: []roamEngagement{{Killmails: []roamKillmail{{
			KillmailID: 3, Role: "loss", VictimCharacterID: 1,
			VictimShipTypeID: 20, VictimShipName: "Ship Two", VictimShipGroupID: 200,
		}}}},
	}
	addRoamShips(&report, []roamAttacker{
		{KillmailID: 1, CharacterID: 1, ShipTypeID: 10, ShipName: "Ship One", DamageDone: 100},
		{KillmailID: 2, CharacterID: 1, ShipTypeID: 20, ShipName: "Ship Two", DamageDone: 200},
		{KillmailID: 3, CharacterID: 1, ShipTypeID: 20, ShipName: "Ship Two", DamageDone: 50},
	})
	ships := report.Roster[0].Ships
	if len(ships) != 2 || ships[0].ShipTypeID != 20 || ships[0].Killmails != 2 ||
		ships[0].Losses != 1 || ships[0].DamageDone != 250 ||
		ships[1].ShipTypeID != 10 || ships[1].Killmails != 1 {
		t.Fatalf("ships = %+v", ships)
	}
	if report.Roster[1].Ships == nil || len(report.Roster[1].Ships) != 0 {
		t.Fatalf("pilot without observations = %+v", report.Roster[1].Ships)
	}
}

func TestParseRoamNames(t *testing.T) {
	names, err := parseRoamNames(" boostimail\r\nDionisius77\nW0rld'EXE\nBOOSTIMAIL\n\n")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"boostimail", "Dionisius77", "W0rld'EXE"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %#v, want %#v", names, want)
	}
	for _, input := range []string{" \n ", strings.Repeat("x", roamMaxNameLength+1)} {
		if _, err := parseRoamNames(input); err == nil {
			t.Errorf("expected invalid fleet list %q to fail", input)
		}
	}
	var manyNames []string
	for index := range roamMaxPilots + 1 {
		manyNames = append(manyNames, fmt.Sprintf("Pilot %d", index))
	}
	if _, err := parseRoamNames(strings.Join(manyNames, "\n")); err == nil {
		t.Fatal("expected too many pilots to fail")
	}
}

func TestValidateRoamWindow(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	for _, window := range []struct {
		start, end time.Time
		valid      bool
	}{
		{now.Add(-3 * time.Hour), now, true},
		{now, now, false},
		{now.Add(-73 * time.Hour), now, false},
		{now, now.Add(10 * time.Minute), false},
		{time.Time{}, now, false},
	} {
		err := validateRoamWindow(window.start, window.end, now)
		if (err == nil) != window.valid {
			t.Errorf("window %v to %v: error = %v, valid = %v", window.start, window.end, err, window.valid)
		}
	}
}

func TestBuildRoamReportGroupsCombatAndCountsLossesOnce(t *testing.T) {
	start := time.Date(2026, 9, 27, 9, 0, 0, 0, time.UTC)
	at := func(minutes int) time.Time { return start.Add(time.Duration(minutes) * time.Minute) }
	roster := []roamPilot{{CharacterID: 1, Name: "Pilot One"}, {CharacterID: 2, Name: "Pilot Two"}}
	kill := func(id int32, minutes int, system int32, victim int32, value float64) roamKillmail {
		return roamKillmail{
			KillmailID: id, KillmailTime: at(minutes), SolarSystemID: system,
			SolarSystemName: "System", RegionID: system, RegionName: "Region",
			VictimCharacterID: victim, VictimCorporationID: 900,
			VictimCorporationName: "Opponents", TotalValue: value,
		}
	}
	// Intentionally out of order: grouping must use killmail time.
	killmails := []roamKillmail{
		kill(5, 60, 10, 2, 500),
		kill(1, 0, 10, 101, 100),
		kill(4, 20, 10, 103, 400),
		kill(2, 5, 10, 2, 200),
		kill(3, 15, 20, 102, 300),
	}
	attackers := []roamAttacker{
		{KillmailID: 1, CharacterID: 1, DamageDone: 80, FinalBlow: true},
		{KillmailID: 1, CharacterID: 2, DamageDone: 20},
		{KillmailID: 3, CharacterID: 1},
		{KillmailID: 4, CharacterID: 1},
		{KillmailID: 5, CharacterID: 1}, // friendly fire: a loss, not a fleet kill
	}
	report := buildRoamReport("id", at(70), start, at(70),
		[]string{"Pilot One", "Pilot Two", "Unknown"}, roster, []string{"Unknown"},
		killmails, attackers, nil)
	if report.Summary.Kills != 3 || report.Summary.Losses != 2 || report.Summary.FriendlyFireLosses != 1 {
		t.Errorf("outcomes = %+v", report.Summary)
	}
	if report.Summary.IskDestroyed != 800 || report.Summary.IskLost != 700 || report.Summary.Efficiency != 53.33 {
		t.Errorf("ISK = %+v", report.Summary)
	}
	if report.Summary.Engagements != 4 || report.Summary.Systems != 2 || report.Summary.CoordinatedKills != 1 {
		t.Errorf("activity = %+v", report.Summary)
	}
	if !reflect.DeepEqual([]int32{
		report.Engagements[0].SolarSystemID,
		report.Engagements[1].SolarSystemID,
		report.Engagements[2].SolarSystemID,
		report.Engagements[3].SolarSystemID,
	}, []int32{10, 20, 10, 10}) {
		t.Errorf("combat trail = %+v", report.Engagements)
	}
	if report.Roster[0].KillParticipations != 3 || report.Roster[0].FinalBlows != 1 || report.Roster[0].Engagements != 4 {
		t.Errorf("pilot one = %+v", report.Roster[0])
	}
	if report.Roster[1].KillParticipations != 1 || report.Roster[1].Losses != 2 {
		t.Errorf("pilot two = %+v", report.Roster[1])
	}
	if report.Engagements[3].Killmails[0].Role != "loss" || !report.Engagements[3].Killmails[0].FriendlyFire {
		t.Errorf("friendly fire = %+v", report.Engagements[3].Killmails[0])
	}
	if len(report.Unresolved) != 1 || report.Unresolved[0] != "Unknown" {
		t.Errorf("unresolved = %#v", report.Unresolved)
	}
}
