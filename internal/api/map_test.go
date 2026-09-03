package api

import (
	"reflect"
	"testing"
)

func TestParseMapActivityHours(t *testing.T) {
	for _, test := range []struct {
		raw  string
		want int
		ok   bool
	}{
		{"", 24, true},
		{"1", 1, true},
		{"6", 6, true},
		{"24", 24, true},
		{"168", 168, true},
		{"12", 0, false},
		{"nope", 0, false},
	} {
		got, err := parseMapActivityHours(test.raw)
		if (err == nil) != test.ok || got != test.want {
			t.Errorf("parseMapActivityHours(%q) = %d, %v; want %d, ok=%v", test.raw, got, err, test.want, test.ok)
		}
	}
}

func TestRegionInMapScopeMatchesFrontendBuckets(t *testing.T) {
	tests := []struct {
		scope string
		id    int64
		want  bool
	}{
		{"new-eden", 10000070, true},
		{"new-eden", 10001004, true},
		{"new-eden", 10001000, false},
		{"zarzakh", 10001000, true},
		{"wormhole", 11000001, true},
		{"wormhole", 11000034, false},
		{"abyssal", 12000005, true},
		{"proving", 14000001, true},
		{"nope", 10000001, false},
	}
	for _, test := range tests {
		if got := regionInMapScope(test.scope, test.id); got != test.want {
			t.Errorf("%s/%d = %v, want %v", test.scope, test.id, got, test.want)
		}
	}
}

func TestGroupMapRegionsKeepsStableBuckets(t *testing.T) {
	rows := []map[string]any{
		{"region_id": int32(10000070), "name": "Pochven"},
		{"region_id": int32(10001000), "name": "Zarzakh"},
		{"region_id": int32(10000002), "name": "The Forge"},
		{"region_id": int32(11000001), "name": "A-R00001"},
	}
	grouped := groupMapRegions(rows)
	for _, name := range []string{
		"kspace", "pochven", "zarzakh", "wormhole", "abyssal", "proving",
	} {
		if grouped[name] == nil {
			t.Errorf("%s bucket is nil", name)
		}
	}
	if got := grouped["kspace"].([]map[string]any); !reflect.DeepEqual(got, rows[2:3]) {
		t.Errorf("kspace = %#v", got)
	}
}

func TestParseMapSystemIDs(t *testing.T) {
	got, err := parseMapSystemIDs("30000142, 30002187,30000142")
	if err != nil {
		t.Fatalf("parseMapSystemIDs returned %v", err)
	}
	want := []int32{30000142, 30002187}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("systems = %#v, want %#v", got, want)
	}
	if _, err := parseMapSystemIDs("Jita"); err == nil {
		t.Fatal("expected a non-numeric system to fail")
	}
	tooMany := "1,2,3,4,5,6,7,8,9"
	if _, err := parseMapSystemIDs(tooMany); err == nil {
		t.Fatalf("expected more than %d systems to fail", mapAIIDMaxAnchors)
	}
}
