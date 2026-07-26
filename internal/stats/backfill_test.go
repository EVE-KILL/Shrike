package stats

import (
	"testing"
	"time"
)

func TestMonthsInclusive(t *testing.T) {
	from := time.Date(2025, 11, 15, 4, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 28, 4, 0, 0, 0, time.UTC)
	got := monthsInclusive(from, to)
	if len(got) != 4 {
		t.Fatalf("months = %d, want 4", len(got))
	}
	if got[0].Format("2006-01") != "2025-11" || got[3].Format("2006-01") != "2026-02" {
		t.Fatalf("months = %v", got)
	}
}
