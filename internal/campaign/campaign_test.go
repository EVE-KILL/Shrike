package campaign

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeLocationMatchesCampaignLimits(t *testing.T) {
	location := Location{
		SystemIDs:        []int32{1, 1, 0, -1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		ConstellationIDs: []int32{20, 20, 21, 22, 23, 24, 25},
		RegionIDs:        []int32{30, 31, 30, 32, 33, 34, 35},
	}

	got := normalizeLocation(location)
	if len(got.SystemIDs) != maxLocationSystems ||
		got.SystemIDs[0] != 1 ||
		got.SystemIDs[len(got.SystemIDs)-1] != 10 {
		t.Errorf("systems = %v, want ten unique positive ids in input order", got.SystemIDs)
	}
	if len(got.ConstellationIDs) != maxLocationConstellations ||
		got.ConstellationIDs[0] != 20 ||
		got.ConstellationIDs[len(got.ConstellationIDs)-1] != 24 {
		t.Errorf("constellations = %v, want five unique positive ids", got.ConstellationIDs)
	}
	if len(got.RegionIDs) != maxLocationRegions ||
		got.RegionIDs[0] != 30 ||
		got.RegionIDs[len(got.RegionIDs)-1] != 34 {
		t.Errorf("regions = %v, want five unique positive ids", got.RegionIDs)
	}
}

func TestMostValuableTimestampWireShape(t *testing.T) {
	encoded, err := json.Marshal(campaignMostValuable{
		KillmailTime: "2026-07-26T12:34:56.000Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"killmailTime":"2026-07-26T12:34:56.000Z"`) {
		t.Fatalf("most valuable JSON = %s, want JavaScript toISOString shape", encoded)
	}
}
