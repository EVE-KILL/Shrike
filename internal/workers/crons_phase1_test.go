package workers

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/esi"
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

func TestInterpretTQStatus(t *testing.T) {
	t.Run("online is authoritative", func(t *testing.T) {
		status := esi.Status{Players: 27_168}
		got, err := interpretTQStatus(esi.Response[esi.Status]{
			Status: http.StatusOK,
			Data:   &status,
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !got.online || got.players != 27_168 {
			t.Fatalf("observation = %+v", got)
		}
	})

	t.Run("server failure is authoritative offline", func(t *testing.T) {
		got, err := interpretTQStatus(esi.Response[esi.Status]{
			Status: http.StatusServiceUnavailable,
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got.online || !strings.Contains(got.reason, "503") {
			t.Fatalf("observation = %+v", got)
		}
	})

	t.Run("transport failure preserves last known state", func(t *testing.T) {
		_, err := interpretTQStatus(esi.Response[esi.Status]{}, errors.New("dial timeout"))
		if err == nil || !strings.Contains(err.Error(), "dial timeout") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("local admission failure is not downtime", func(t *testing.T) {
		for _, status := range []int{0, 420, http.StatusTooManyRequests} {
			if _, err := interpretTQStatus(esi.Response[esi.Status]{Status: status}, nil); err == nil {
				t.Errorf("HTTP %d was interpreted as TQ downtime", status)
			}
		}
	})
}
