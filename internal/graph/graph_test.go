package graph

import (
	"testing"
	"time"
)

func TestISOTimestampMatchesJavaScriptShape(t *testing.T) {
	got := isoTimestamp(time.Date(2026, 7, 26, 12, 34, 56, 789_999_999, time.FixedZone("offset", 2*60*60)))
	if got != "2026-07-26T10:34:56.789Z" {
		t.Fatalf("isoTimestamp() = %q, want JavaScript Date.toISOString shape", got)
	}
}

func TestNullableTimeUsesCanonicalISOString(t *testing.T) {
	if got := nullableTime(time.Time{}); got != nil {
		t.Fatalf("nullableTime(zero) = %#v, want nil", got)
	}
	got := nullableTime(time.Date(2026, 7, 26, 12, 34, 56, 0, time.UTC))
	if got != "2026-07-26T12:34:56.000Z" {
		t.Fatalf("nullableTime() = %#v, want fixed millisecond precision", got)
	}
}
