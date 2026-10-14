// Package retry describes expected backpressure independently of a job runner.
package retry

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Deferred means admission control asked us to wait. It is not a failed
// operation, and a queue must reschedule it without consuming failure attempts.
type Deferred struct {
	Operation string
	Reason    string
	After     time.Duration
}

func (e *Deferred) Error() string {
	return fmt.Sprintf("%s deferred for %s: %s", e.Operation, e.After, e.Reason)
}

func Wait(ctx context.Context, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// RetryAfter accepts both HTTP forms and never shortens the server's wait.
func RetryAfter(value string, now time.Time) time.Duration {
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds > 0 {
		const maximumSeconds = int64((1<<63 - 1) / time.Second)
		return time.Duration(min(seconds, maximumSeconds)) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil && date.After(now) {
		return date.Sub(now)
	}
	return 0
}
