package retry

import (
	"context"
	"errors"
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
