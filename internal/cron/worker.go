package cron

import (
	"context"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/redis/go-redis/v9"
	"github.com/riverqueue/river"
)

// Run is one completed cron run, reported for display.
type Run struct {
	Name    string
	Report  string
	Elapsed time.Duration

	// Skipped is set when the job did not run because Tranquility is offline.
	Skipped bool
	Err     error
}

// Worker runs scheduled jobs.
//
// One worker handles every cron, dispatching on the name in the job args. See
// the note on queue.CronArgs for why they share a job kind.
type Worker struct {
	river.WorkerDefaults[queue.CronArgs]

	Registry *Registry
	Redis    *redis.Client

	// OnRun is called after each run, for logging. Optional.
	OnRun func(Run)
}

// Work executes one scheduled job.
func (w *Worker) Work(ctx context.Context, job *river.Job[queue.CronArgs]) error {
	name := job.Args.Name

	declared := jobs.CronByName(name)
	if declared == nil {
		// The schedule and the registry disagree, which is a wiring bug rather
		// than a transient failure. Returning a plain error would have River
		// retry it three times before giving up on something that will never
		// succeed; cancelling says so once.
		return river.JobCancel(fmt.Errorf("cron %q is not declared", name))
	}

	handler, ok := w.Registry.Lookup(name)
	if !ok {
		return river.JobCancel(fmt.Errorf("cron %q has no implementation", name))
	}

	// A TQ-dependent job during a downtime is skipped, not failed. Failing it
	// would burn the retry budget on a condition that has nothing to do with
	// the job and will clear on its own; the next tick picks it up.
	if declared.RequiresTQ {
		offline, err := queue.TQIsOffline(ctx, w.Redis)
		if err == nil && offline {
			if w.OnRun != nil {
				w.OnRun(Run{Name: name, Skipped: true, Report: "Tranquility is offline"})
			}
			return nil
		}
	}

	start := time.Now()
	report, err := handler(ctx)
	elapsed := time.Since(start)

	if w.OnRun != nil {
		w.OnRun(Run{Name: name, Report: report, Elapsed: elapsed, Err: err})
	}
	return err
}

// Timeout gives scheduled jobs longer than the client default.
//
// Several crons are importers that legitimately run for many minutes — a day of
// killmails, a full sovereignty snapshot — and the default would kill them
// partway through, repeatedly, without ever completing one.
func (w *Worker) Timeout(*river.Job[queue.CronArgs]) time.Duration { return 2 * time.Hour }

// RunOnce executes a cron immediately, outside the scheduler.
//
// This is `cron:run`, and it deliberately bypasses River: an operator running a
// job by hand wants it to run now, in the foreground, with its output on their
// terminal — not to insert a row and hope a worker somewhere picks it up.
func RunOnce(ctx context.Context, r *Registry, name string) (Run, error) {
	declared := jobs.CronByName(name)
	if declared == nil {
		return Run{Name: name}, fmt.Errorf("cron %q is not declared", name)
	}

	handler, ok := r.Lookup(name)
	if !ok {
		return Run{Name: declared.Name}, fmt.Errorf("cron %q is declared but not implemented", declared.Name)
	}

	start := time.Now()
	report, err := handler(ctx)
	return Run{
		Name:    declared.Name,
		Report:  report,
		Elapsed: time.Since(start),
		Err:     err,
	}, err
}
