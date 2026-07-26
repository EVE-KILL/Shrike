package api

import (
	"reflect"
	"testing"
)

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
