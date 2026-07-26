package queue

import (
	"context"
	"errors"
	"time"

	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/redis/go-redis/v9"
	"github.com/riverqueue/river"
)

// Pausing work while Tranquility is down.
//
// When the cluster goes offline every ESI endpoint starts answering with errors
// rather than data. Jobs that keep asking do not merely waste effort: ESI's
// error limit is global and shared across the whole application, so a few
// thousand failing requests during a downtime exhaust the budget and get the
// client banned for a window that outlasts the downtime itself. The queues that
// need ESI therefore stop entirely, and the ones that do not carry on.
//
// The status flag is written by the tq_status cron, and the key is the one the
// TypeScript services already read and write — during the changeover both
// implementations must agree about whether TQ is up.

// TQStatusKey holds "online" or "offline".
const TQStatusKey = "esi:tq:status"

// TQOffline is the value that means stop.
const TQOffline = "offline"

// TQPollInterval is how often the flag is checked.
//
// Ten seconds is a compromise the other way round from most polling: the cost
// of being late to pause is real (wasted error budget) but the cost of being
// late to resume is only latency, and the flag itself is only refreshed every
// thirty seconds by the cron that writes it.
const TQPollInterval = 10 * time.Second

// TQGate pauses and resumes the ESI-dependent queues.
type TQGate struct {
	Client *Client
	Redis  *redis.Client

	// OnChange is called when the gate flips, for logging. Optional.
	OnChange func(offline bool)

	paused bool
}

// Watch blocks, polling the status flag until the context is cancelled.
//
// Redis being unreachable is deliberately not treated as "TQ is offline". A
// Redis outage would otherwise halt all killmail processing, which is both
// wrong and the opposite of what an operator wants during an incident: the
// safer failure is to keep working and let the ESI client's own error-limit
// handling apply the brakes if it turns out TQ really is down.
func (g *TQGate) Watch(ctx context.Context) error {
	// Check once immediately so a worker starting during a downtime does not
	// spend the first ten seconds hammering a dead ESI.
	g.check(ctx)

	ticker := time.NewTicker(TQPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			g.check(ctx)
		}
	}
}

// check reads the flag and applies it.
//
// The desired state is re-applied on every tick rather than only when it
// changes. Pausing is a single idempotent UPDATE per queue, so re-applying
// costs almost nothing, and tracking "already paused" in memory does not
// survive contact with reality: a pause issued before River has created the
// queue rows silently does nothing, and a gate that had recorded the pause
// would never try again. The cached value is only used to decide when to log.
func (g *TQGate) check(ctx context.Context) {
	offline, err := TQIsOffline(ctx, g.Redis)
	if err != nil {
		return
	}

	g.apply(ctx, offline)

	if offline != g.paused {
		g.paused = offline
		if g.OnChange != nil {
			g.OnChange(offline)
		}
	}
}

// apply pauses or resumes every queue that needs ESI.
func (g *TQGate) apply(ctx context.Context, offline bool) {
	for _, q := range jobs.Queues {
		if !q.RequiresTQ {
			continue
		}

		var err error
		if offline {
			err = g.Client.QueuePause(ctx, q.Name, nil)
		} else {
			err = g.Client.QueueResume(ctx, q.Name, nil)
		}

		// A queue with no row yet is one no client in the cluster has ever
		// consumed — including the unported ones, which have no Go worker at
		// all. There is nothing running to pause, so this is expected rather
		// than an error.
		if err != nil && !errors.Is(err, river.ErrNotFound) {
			// Left to the next tick rather than retried here: the flag is
			// re-read every ten seconds anyway, and a tight retry loop against
			// a struggling database helps nobody.
			continue
		}
	}
}

// TQIsOffline reads the status flag.
//
// An unset key means online. The flag is only ever written to say "offline", so
// its absence — on a fresh Redis, before the first tq_status run — must not
// stop every ESI queue in the cluster.
func TQIsOffline(ctx context.Context, rdb *redis.Client) (bool, error) {
	if rdb == nil {
		return false, nil
	}
	v, err := rdb.Get(ctx, TQStatusKey).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return v == TQOffline, nil
}
