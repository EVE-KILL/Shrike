package achievements

import "testing"

func TestFilterMatchesIDAndCategory(t *testing.T) {
	got := Filter("", "combat")
	if len(got) != 5 {
		t.Fatalf("combat definitions = %d, want 5", len(got))
	}

	got = Filter("veteran_killer", "COMBAT")
	if len(got) != 1 || got[0].ID != "veteran_killer" {
		t.Fatalf("filtered definitions = %#v", got)
	}

	if got := Filter("veteran_killer", "Locations"); len(got) != 0 {
		t.Fatalf("mismatched filters returned %d definitions", len(got))
	}
}

func TestEveryDefinitionHasARebuildQuery(t *testing.T) {
	for _, def := range All {
		query, _, err := countQuery(def)
		if err != nil {
			t.Errorf("%s: %v", def.ID, err)
		}
		if query == "" {
			t.Errorf("%s has an empty rebuild query", def.ID)
		}
	}
}
