package battle

import (
	"testing"
	"time"
)

func TestMergeHotspotWindowsMatchesBackfillGrouping(t *testing.T) {
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	hours := []HotspotWindow{
		{SolarSystemID: 1, Start: base, End: base.Add(time.Hour), Kills: 10},
		// One empty hour is still merged by the TS condition.
		{SolarSystemID: 1, Start: base.Add(2 * time.Hour), End: base.Add(3 * time.Hour), Kills: 12},
		// Two empty hours starts a new candidate.
		{SolarSystemID: 1, Start: base.Add(5 * time.Hour), End: base.Add(6 * time.Hour), Kills: 14},
		{SolarSystemID: 2, Start: base, End: base.Add(time.Hour), Kills: 20},
	}

	got := mergeHotspotWindows(hours)
	if len(got) != 3 {
		t.Fatalf("got %d windows, want 3: %#v", len(got), got)
	}
	if got[0].Start != base || got[0].End != base.Add(3*time.Hour) || got[0].Kills != 22 {
		t.Fatalf("first merged window = %#v", got[0])
	}
	if got[1].Start != base.Add(5*time.Hour) || got[1].Kills != 14 {
		t.Fatalf("second window = %#v", got[1])
	}
	if got[2].SolarSystemID != 2 {
		t.Fatalf("third window = %#v", got[2])
	}
}
