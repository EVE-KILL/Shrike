package queue

import (
	"context"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/retry"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog"
)

type deferredWorker struct{ river.WorkerDefaults[testArgs] }

func (*deferredWorker) Work(context.Context, *river.Job[testArgs]) error {
	return &retry.Deferred{Operation: "esi", Reason: "admission", After: time.Minute}
}

func TestDeferralPreservesRetryBudgetInRiver(t *testing.T) {
	pool := testPool(t)
	clearTestJobs(t, pool)
	workers := river.NewWorkers()
	river.AddWorker(workers, &deferredWorker{})
	client, err := river.NewClient[pgx.Tx](riverpgxv5.New(pool), &river.Config{
		Workers: workers, Queues: map[string]river.QueueConfig{testQueue: {MaxWorkers: 1}},
		Middleware: []rivertype.Middleware{newJobLogMiddleware(zerolog.Nop())},
	})
	if err != nil {
		t.Fatal(err)
	}
	events, unsubscribe := client.Subscribe(river.EventKindJobSnoozed, river.EventKindJobFailed)
	defer unsubscribe()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		stop, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		if err := client.Stop(stop); err != nil {
			t.Error(err)
		}
	}()
	inserted, err := client.Insert(ctx, testArgs{ID: 24497}, &river.InsertOpts{Queue: testQueue, MaxAttempts: 1})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event.Kind != river.EventKindJobSnoozed || event.Job.ID != inserted.Job.ID {
			t.Fatalf("unexpected event: %+v", event)
		}
		job, err := client.JobGet(ctx, inserted.Job.ID)
		if err != nil {
			t.Fatal(err)
		}
		if job.Attempt != 0 || len(job.Errors) != 0 || job.State == rivertype.JobStateDiscarded || time.Until(job.ScheduledAt) < 55*time.Second {
			t.Fatalf("deferral consumed retry budget or lost wait: %+v", job)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for deferred job")
	}
}
