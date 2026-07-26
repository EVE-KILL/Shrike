package workers

import (
	"testing"

	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/wars"
)

func TestMissingWarCronRepairsWalkKillmailList(t *testing.T) {
	live, repair := warDiscoveryJobs(wars.Discover{
		New:     []int32{101},
		Active:  []int32{102},
		Missing: []int32{103},
	})

	if len(live) != 2 || len(repair) != 1 {
		t.Fatalf("live=%d repair=%d, want 2 and 1", len(live), len(repair))
	}
	got, ok := repair[0].(queue.WarArgs)
	if !ok {
		t.Fatalf("missing-war job has type %T, want queue.WarArgs", repair[0])
	}
	if got.WarID != 103 {
		t.Errorf("missing-war id = %d, want 103", got.WarID)
	}
	if got.MetadataOnly {
		t.Error("hourly missing-war repair is metadata-only; it would never discover the rest of a partially imported war")
	}
}
