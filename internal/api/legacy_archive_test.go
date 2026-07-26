package api

import (
	"reflect"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func TestLegacyArchiveRoutesAreOneDomainRegistrar(t *testing.T) {
	a := humachi.New(
		chi.NewRouter(),
		huma.DefaultConfig("legacy-archive-test", "test"),
	)
	registerLegacyArchiveRoutes(a, Options{})

	for _, path := range []string{
		"/legacy/autocomplete",
		"/legacy/kills",
		"/legacy/stats",
		"/legacy/top",
		"/legacy/kill/{id}",
	} {
		item := a.OpenAPI().Paths[path]
		if item == nil || item.Get == nil {
			t.Errorf("GET %s was not registered", path)
		}
	}
}

func TestLegacyArchiveSortIsAllowlistedAndStable(t *testing.T) {
	for _, test := range []struct {
		raw        string
		column     string
		descending bool
	}{
		{"", "killmail_id", true},
		{"id_desc", "killmail_id", true},
		{"id_asc", "killmail_id", false},
		{"time_asc", "killmail_time", false},
		{"value_desc", "total_value", true},
		{"not-a-column_asc", "killmail_id", false},
		{"time_sideways", "killmail_time", true},
	} {
		column, descending := legacyArchiveSort(test.raw)
		if column != test.column || descending != test.descending {
			t.Errorf("sort %q = %q, %t; want %q, %t",
				test.raw, column, descending, test.column, test.descending)
		}
	}
}

func TestLegacyArchiveFieldMapsMatchFrontendNames(t *testing.T) {
	wantAutocomplete := map[string][3]string{
		"victim":   {"old_killmails", "victim_name", "victim_character_id"},
		"attacker": {"old_killmail_attackers", "name", "character_id"},
		"corp":     {"old_killmails", "victim_corp", "victim_corporation_id"},
		"alliance": {"old_killmails", "victim_alliance", "victim_alliance_id"},
		"system":   {"old_killmails", "system_name", "solar_system_id"},
	}
	if len(legacyAutocompleteFields) != len(wantAutocomplete) {
		t.Fatalf("autocomplete fields = %d, want %d",
			len(legacyAutocompleteFields), len(wantAutocomplete))
	}
	for name, want := range wantAutocomplete {
		got := legacyAutocompleteFields[name]
		if !reflect.DeepEqual(
			[3]string{got.Table, got.Name, got.ID},
			want,
		) {
			t.Errorf("autocomplete %s = %#v", name, got)
		}
	}

	wantTopTypes := map[string]string{
		"characters": "character", "corporations": "corporation",
		"alliances": "alliance", "ships": "ship", "systems": "system",
	}
	if len(legacyTopSpecs) != len(wantTopTypes) {
		t.Fatalf("top specs = %d, want %d", len(legacyTopSpecs), len(wantTopTypes))
	}
	for name, wantType := range wantTopTypes {
		if got := legacyTopSpecs[name].Type; got != wantType {
			t.Errorf("top type %s = %q, want %q", name, got, wantType)
		}
	}
}

func TestLegacyArchiveKillSelectResolvesListShapeInOneQuery(t *testing.T) {
	for _, fragment := range []string{
		"old_killmail_attackers count_attacker",
		"final_blow IS TRUE",
		"LEFT JOIN inv_types ship",
		"LEFT JOIN inv_types final_ship",
		"LEFT JOIN solar_systems system",
		"LEFT JOIN regions region",
	} {
		if !strings.Contains(legacyArchiveKillSelect, fragment) {
			t.Errorf("kill select missing %q", fragment)
		}
	}
}
