package zkb

import (
	"context"
	"sync"
	"time"
)

// WindowLimiter allows at most n requests per sliding window.
//
// It reproduces the TypeScript client's behaviour rather than using a token
// bucket: timestamps of recent requests are kept, anything older than the
// window is discarded, and a caller that would exceed the limit waits until the
// oldest timestamp ages out. The difference from a bucket matters at the
// boundary — a bucket refilling at 10/s permits a smooth 10 per second, whereas
// this permits 10 immediately and then nothing until the window slides. R2Z2
// counts requests per second, so matching its own accounting is the safer
// reading.
//
// The clock is injectable so tests can assert the pacing without spending real
// seconds doing it.
type WindowLimiter struct {
	limit  int
	window time.Duration

	mu    sync.Mutex
	times []time.Time

	// now and sleep are the clock. Nil means the real one.
	now   func() time.Time
	sleep func(context.Context, time.Duration) error
}

// NewWindowLimiter builds a limiter permitting limit requests per window.
func NewWindowLimiter(limit int, window time.Duration) *WindowLimiter {
	return &WindowLimiter{
		limit:  limit,
		window: window,
		times:  make([]time.Time, 0, limit),
	}
}

func (l *WindowLimiter) clock() time.Time {
	if l.now != nil {
		return l.now()
	}
	return time.Now()
}

func (l *WindowLimiter) pause(ctx context.Context, d time.Duration) error {
	if l.sleep != nil {
		return l.sleep(ctx, d)
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Wait blocks until another request is permitted.
func (l *WindowLimiter) Wait(ctx context.Context) error {
	for {
		l.mu.Lock()
		now := l.clock()
		l.evict(now)

		if len(l.times) < l.limit {
			l.times = append(l.times, now)
			l.mu.Unlock()
			return nil
		}

		// Full. The earliest slot frees up when the oldest request leaves the
		// window; sleep exactly that long rather than polling.
		wait := l.window - now.Sub(l.times[0])
		l.mu.Unlock()

		if wait <= 0 {
			// The oldest entry has already aged out but the clock moved between
			// the two reads. Retry immediately rather than sleeping a negative
			// duration, which would busy-loop.
			continue
		}
		if err := l.pause(ctx, wait); err != nil {
			return err
		}
	}
}

// evict drops timestamps that have left the window. Called with the lock held.
func (l *WindowLimiter) evict(now time.Time) {
	cutoff := now.Add(-l.window)
	keep := 0
	for _, t := range l.times {
		if t.After(cutoff) {
			l.times[keep] = t
			keep++
		}
	}
	l.times = l.times[:keep]
}
