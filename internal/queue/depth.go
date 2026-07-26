package queue

import (
	"context"

	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Reading the backlog.
//
// This is one of the quieter wins of moving off Redis: the queue is a table, so
// "what is stuck and why" is a query rather than a set of Redis key
// conventions, and anybody with psql can answer it without this tool.

// Depth is one queue's job counts by state.
type Depth struct {
	Queue string `json:"queue"`

	// Available is ready to run now.
	Available int64 `json:"available"`
	// Running is leased by a worker.
	Running int64 `json:"running"`
	// Scheduled is waiting for a future time — a delay or a retry backoff.
	Scheduled int64 `json:"scheduled"`
	// Retryable failed and will be tried again.
	Retryable int64 `json:"retryable"`
	// Discarded exhausted its attempts. These need a human.
	Discarded int64 `json:"discarded"`
	// Cancelled was cancelled deliberately, usually because the work was
	// impossible rather than because it failed.
	Cancelled int64 `json:"cancelled"`
	// Completed succeeded and has not yet been cleaned up.
	Completed int64 `json:"completed"`
}

// Total is every job in any state.
func (d Depth) Total() int64 {
	return d.Available + d.Running + d.Scheduled + d.Retryable +
		d.Discarded + d.Cancelled + d.Completed
}

// Pending is the work still owing. Completed, discarded and cancelled are
// history and deliberately excluded — a queue with a million completed jobs and
// nothing waiting is idle, not busy.
func (d Depth) Pending() int64 {
	return d.Available + d.Running + d.Scheduled + d.Retryable
}

// Depths reads live counts for every queue that has jobs, plus every declared
// queue, so a queue with nothing in it still appears rather than vanishing.
func Depths(ctx context.Context, pool *pgxpool.Pool) ([]Depth, error) {
	// One grouped scan rather than a count per state per queue, which at seven
	// states and twenty queues would be 140 round trips.
	return queryDepths(ctx, pool, false)
}

// StatusDepths reads only states that contribute to the published backlog.
//
// Completed and cancelled jobs are retained as diagnostics, but neither is in
// the BullMQ-shaped status contract. Excluding them is significant at live
// throughput because completed jobs remain in River for a day.
func StatusDepths(ctx context.Context, pool *pgxpool.Pool) ([]Depth, error) {
	return queryDepths(ctx, pool, true)
}

func queryDepths(ctx context.Context, pool *pgxpool.Pool, statusOnly bool) ([]Depth, error) {
	query := `
		SELECT queue, state, count(*)
		FROM river_job`
	if statusOnly {
		query += `
		WHERE state IN (
			'available', 'running', 'scheduled', 'retryable', 'pending', 'discarded'
		)`
	}
	query += `
		GROUP BY queue, state`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byQueue := map[string]*Depth{}
	for rows.Next() {
		var name, state string
		var n int64
		if err := rows.Scan(&name, &state, &n); err != nil {
			return nil, err
		}

		d, ok := byQueue[name]
		if !ok {
			d = &Depth{Queue: name}
			byQueue[name] = d
		}

		switch state {
		case "available":
			d.Available = n
		case "running":
			d.Running = n
		case "scheduled":
			d.Scheduled = n
		case "retryable":
			d.Retryable = n
		case "discarded":
			d.Discarded = n
		case "cancelled":
			d.Cancelled = n
		case "completed":
			d.Completed = n
		case "pending":
			// River's "pending" is a job held back by a workflow dependency,
			// which nothing here uses. Counted as scheduled so it cannot go
			// missing from the totals if that ever changes.
			d.Scheduled += n
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Declared queues first, in registry order, then the cron queue, then
	// anything else found in the table — which would be a queue enqueued to but
	// never declared, and worth seeing.
	var out []Depth
	seen := map[string]bool{}

	appendQueue := func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		if d, ok := byQueue[name]; ok {
			out = append(out, *d)
			return
		}
		out = append(out, Depth{Queue: name})
	}

	for _, name := range jobs.QueueNames() {
		appendQueue(name)
	}
	appendQueue(CronQueue)

	for name := range byQueue {
		appendQueue(name)
	}
	return out, nil
}
