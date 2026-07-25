package jobs

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Reading BullMQ's Redis keys directly is a transition-period affordance: it
// lets the Go CLI report on the queues the Bun workers are running so the
// registry above can be checked against live reality before any worker is
// ported. Once the queues move to River this is replaced by queries against
// river_job.
//
// The key layout was verified against the production instance on 2026-07-25
// rather than taken from documentation:
//
//	bull:<queue>:wait          list   — ready, FIFO
//	bull:<queue>:prioritized   zset   — ready, priority-ordered
//	bull:<queue>:active        list   — leased by a worker
//	bull:<queue>:delayed       zset   — scheduled for later
//	bull:<queue>:failed        zset   — exhausted retries
//	bull:<queue>:completed     zset   — retained briefly for dedupe
//	bull:<queue>:meta          hash   — exists once a Queue has been constructed
//	bull:<queue>:paused        list   — present only while paused
//
// No custom prefix is configured (backend/src/queue/connection.ts passes bare
// redisOptions()), so the default "bull" prefix applies.
const bullPrefix = "bull"

// Depth is the live state of one queue.
type Depth struct {
	Name        string `json:"name"`
	Waiting     int64  `json:"waiting"`
	Prioritized int64  `json:"prioritized"`
	Active      int64  `json:"active"`
	Delayed     int64  `json:"delayed"`
	Failed      int64  `json:"failed"`
	Completed   int64  `json:"completed"`
	Paused      bool   `json:"paused"`

	// Known is false when Redis holds no metadata for the queue, meaning no
	// producer or worker has ever constructed it.
	Known bool `json:"known"`
}

// Pending is the work still owing: ready plus scheduled. Completed and failed
// are history and deliberately excluded.
func (d Depth) Pending() int64 {
	return d.Waiting + d.Prioritized + d.Delayed
}

// ReadDepths fetches live counts for the named queues in one pipeline round
// trip, which matters over a 27ms Tailscale hop: 20 queues x 7 keys would
// otherwise be 140 sequential round trips, about four seconds.
func ReadDepths(ctx context.Context, rdb *redis.Client, names []string) ([]Depth, error) {
	pipe := rdb.Pipeline()

	type cmds struct {
		wait, prioritized, active, delayed, failed, completed *redis.IntCmd
		meta                                                  *redis.IntCmd
		paused                                                *redis.IntCmd
	}
	staged := make([]cmds, len(names))

	for i, n := range names {
		k := func(suffix string) string { return fmt.Sprintf("%s:%s:%s", bullPrefix, n, suffix) }
		staged[i] = cmds{
			wait:        pipe.LLen(ctx, k("wait")),
			prioritized: pipe.ZCard(ctx, k("prioritized")),
			active:      pipe.LLen(ctx, k("active")),
			delayed:     pipe.ZCard(ctx, k("delayed")),
			failed:      pipe.ZCard(ctx, k("failed")),
			completed:   pipe.ZCard(ctx, k("completed")),
			meta:        pipe.Exists(ctx, k("meta")),
			paused:      pipe.Exists(ctx, k("paused")),
		}
	}

	// Exec reports the first command error; missing keys are not errors (LLen and
	// ZCard both return 0), so anything here is a genuine connection failure.
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, err
	}

	out := make([]Depth, len(names))
	for i, n := range names {
		s := staged[i]
		out[i] = Depth{
			Name:        n,
			Waiting:     s.wait.Val(),
			Prioritized: s.prioritized.Val(),
			Active:      s.active.Val(),
			Delayed:     s.delayed.Val(),
			Failed:      s.failed.Val(),
			Completed:   s.completed.Val(),
			Paused:      s.paused.Val() > 0,
			Known:       s.meta.Val() > 0,
		}
	}
	return out, nil
}

// DiscoverQueues returns queue names that exist in Redis, found by scanning for
// metadata keys. Used to detect queues live in Redis but absent from the
// registry — drift the registry would otherwise hide.
func DiscoverQueues(ctx context.Context, rdb *redis.Client) ([]string, error) {
	var (
		names  []string
		cursor uint64
	)
	pattern := bullPrefix + ":*:meta"
	for {
		keys, next, err := rdb.Scan(ctx, cursor, pattern, 500).Result()
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			// bull:<name>:meta — the name may itself contain colons, so trim the
			// fixed prefix and suffix rather than splitting.
			if len(k) > len(bullPrefix)+len(":meta")+2 {
				names = append(names, k[len(bullPrefix)+1:len(k)-len(":meta")])
			}
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	return names, nil
}
