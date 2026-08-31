package killtype

import (
	"os"
	"strings"
	"testing"
)

func TestLabelsMatchPublicKillTypes(t *testing.T) {
	want := map[string]bool{}
	for _, id := range Types {
		if id != "latest" {
			want[id] = true
		}
	}
	seen := map[string]bool{}
	for _, label := range Labels {
		if seen[label.ID] {
			t.Errorf("duplicate label %q", label.ID)
		}
		seen[label.ID] = true
		if label.Name == "" || label.Description == "" || label.Category == "" {
			t.Errorf("label %q has incomplete public metadata", label.ID)
		}
		if _, ok := Predicates()[label.ID]; !ok {
			t.Errorf("label %q has no authoritative predicate", label.ID)
		}
	}
	for id := range want {
		if !seen[id] {
			t.Errorf("public kill type %q has no label metadata", id)
		}
	}
}

func TestEveryPublicLabelIsInDefaultKillsDropdown(t *testing.T) {
	menu, err := os.ReadFile("../../web/app/composables/useDomainConfig.ts")
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range Labels {
		path := "/kills/" + label.ID
		if !strings.Contains(string(menu), path) {
			t.Errorf("public label %q is missing from the default Kills dropdown", label.ID)
		}
	}
}

func TestLabelSearchFiltersUseAdvancedSearchVocabulary(t *testing.T) {
	allowed := map[string]bool{
		"location": true, "attackerCount": true, "attackerType": true,
		"iskValue": true, "shipCategory": true, "techLevel": true,
	}
	for _, label := range Labels {
		for key := range label.Search {
			if !allowed[key] {
				t.Errorf("label %q uses unknown advanced-search key %q", label.ID, key)
			}
		}
	}
}
