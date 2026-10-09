package api

import (
	"testing"
)

func TestFactionWarKillfeedUsesOpponentVictimsAndBoundedCursors(t *testing.T) {
	for _, tc := range []struct{ slug, side, factions string }{
		{"caldari-vs-gallente", "combined", "500001,500004"},
		{"caldari-vs-gallente", "side1", "500004"},
		{"caldari-vs-gallente", "side2", "500001"},
		{"amarr-vs-minmatar", "combined", "500003,500002"},
		{"amarr-vs-minmatar", "side1", "500002"},
		{"amarr-vs-minmatar", "side2", "500003"},
	} {
		q, err := factionWarKillQuery(tc.slug, tc.side, 1, 50, nil)
		if err != nil || q.Get("victimFactions") != tc.factions || q.Get("page") != "" || q.Get("after") != "" {
			t.Fatalf("first page %s/%s = %v, %v", tc.slug, tc.side, q, err)
		}
		id := int64(123)
		q, err = factionWarKillQuery(tc.slug, tc.side, 2, 50, &id)
		if err != nil || q.Get("after") != "123" || q.Get("page") != "" {
			t.Fatalf("second page = %v, %v", q, err)
		}
	}
	if _, err := factionWarKillQuery("caldari-vs-gallente", "combined", 2, 50, nil); err == nil {
		t.Fatal("deep pagination requires its cursor")
	}
	if _, err := factionWarKillQuery("unknown", "combined", 1, 50, nil); err == nil {
		t.Fatal("unknown factions must not become an unfiltered global killlist")
	}
}
