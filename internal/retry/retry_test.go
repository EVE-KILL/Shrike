package retry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestRetryAfterAndCancellation(t *testing.T) {
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		value string
		want  time.Duration
	}{
		{"17", 17 * time.Second},
		{now.Add(time.Minute).Format(http.TimeFormat), time.Minute},
		{now.Add(-time.Minute).Format(http.TimeFormat), 0},
		{"-1", 0}, {"invalid", 0},
	} {
		if got := RetryAfter(tc.value, now); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.value, got, tc.want)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := Wait(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestDeferralDoesNotHideJoinedFailures(t *testing.T) {
	short := &Deferred{Operation: "esi", After: time.Second}
	long := &Deferred{Operation: "sso", After: time.Minute}
	for _, tc := range []struct {
		err  error
		want *Deferred
	}{
		{nil, nil},
		{short, short},
		{fmt.Errorf("fetch: %w", short), short},
		{errors.Join(short, long), long},
		{errors.Join(short, errors.New("database write failed")), nil},
		{fmt.Errorf("wallet: %w", errors.Join(long, context.DeadlineExceeded)), nil},
	} {
		got, ok := Deferral(tc.err)
		if got != tc.want || ok != (tc.want != nil) {
			t.Fatalf("Deferral(%v) = %v, %v", tc.err, got, ok)
		}
	}
}
