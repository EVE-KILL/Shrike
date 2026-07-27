package queue

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
)

// CronConcurrency is how many scheduled jobs may run at once.
//
// Low on purpose. Crons are mostly serial maintenance — importers, rebuilds,
// reconciliations — and several of them touch the same tables. Running a
// handful concurrently is useful so a slow hourly job does not delay a job that
// runs every thirty seconds; running many is a way to have two rebuilds of the
// same aggregate racing each other.
const CronConcurrency = 4

// Options configure a client.
type Options struct {
	Pool *pgxpool.Pool

	// Workers is the set of registered job handlers. Nil makes an
	// insert-only client — which is what every process that enqueues work but
	// does not consume it should use, because a client with no workers cannot
	// accidentally start processing a queue another pod owns.
	Workers *river.Workers

	// Queues limits which queues this process consumes. Empty means every
	// declared queue that is not marked as consumed elsewhere.
	Queues []string

	// PeriodicJobs are the scheduled jobs this client will run if elected
	// leader. Only the cron process supplies these.
	PeriodicJobs []*river.PeriodicJob

	Logger *slog.Logger
}

// Client wraps a River client with the registry's settings applied.
type Client struct {
	*river.Client[pgx.Tx]

	pool *pgxpool.Pool

	// consuming records which queues this client actually works, which the
	// status output needs and which is not readable back off the River client.
	consuming []string
}

// New builds a client.
//
// Every queue's concurrency, retry count and backoff comes from
// internal/jobs.Queues rather than being repeated here, so the declarations
// stay the single source for queue behavior.
func New(opts Options) (*Client, error) {
	if opts.Pool == nil {
		return nil, errors.New("queue: a database pool is required")
	}

	cfg := &river.Config{
		Logger:      opts.Logger,
		RetryPolicy: &RegistryRetryPolicy{},

		// The default is three; the registry says what each queue actually
		// wants and every insert sets it explicitly. This is the floor for
		// anything that does not.
		MaxAttempts: 3,

		// A job that has been running for an hour is wedged, not slow. The
		// longest legitimate unit of work here is a single killmail or a single
		// ESI fetch, both of which are seconds.
		JobTimeout: time.Hour,

		PeriodicJobs: opts.PeriodicJobs,
	}

	var consuming []string
	if opts.Workers != nil {
		cfg.Workers = opts.Workers
		cfg.Queues = map[string]river.QueueConfig{}

		for _, q := range selectQueues(opts.Queues) {
			cfg.Queues[q.Name] = river.QueueConfig{MaxWorkers: q.Concurrency}
			consuming = append(consuming, q.Name)
		}

		// The cron queue is not in the registry — it holds scheduled jobs
		// rather than dispatched ones — but a process with workers still needs
		// to consume it, or nothing scheduled ever runs.
		if wantsQueue(opts.Queues, CronQueue) {
			cfg.Queues[CronQueue] = river.QueueConfig{MaxWorkers: CronConcurrency}
			consuming = append(consuming, CronQueue)
		}

		if len(cfg.Queues) == 0 {
			return nil, fmt.Errorf("queue: no consumable queues matched %v", opts.Queues)
		}
	}

	client, err := river.NewClient(riverpgxv5.New(opts.Pool), cfg)
	if err != nil {
		return nil, err
	}
	return &Client{Client: client, pool: opts.Pool, consuming: consuming}, nil
}

// Consuming reports the queues this client works, in registry order.
func (c *Client) Consuming() []string { return c.consuming }

// selectQueues resolves the requested names against the registry.
//
// Queues marked ConsumerElsewhere are never included when the selection is
// implicit: the Discord bot owns discord_events, and a backend worker that
// picked those jobs up would race the real consumer and deliver nothing. An
// explicit request for one is honoured, because that is someone who means it.
func selectQueues(requested []string) []jobs.Queue {
	if len(requested) == 0 {
		var out []jobs.Queue
		for _, q := range jobs.Queues {
			if q.ConsumerElsewhere {
				continue
			}
			out = append(out, q)
		}
		return out
	}

	var out []jobs.Queue
	for _, name := range requested {
		if q := jobs.QueueByName(name); q != nil {
			out = append(out, *q)
		}
	}
	return out
}

func wantsQueue(requested []string, name string) bool {
	if len(requested) == 0 {
		return true
	}
	for _, r := range requested {
		if r == name {
			return true
		}
	}
	return false
}

// InsertOptsFor builds the insert options for a queue at a priority.
//
// Centralised so that MaxAttempts is never taken from River's default when the
// registry has an answer, and so uniqueness is on by default. Deduplication
// being opt-out rather than opt-in is the safer direction: a duplicate that
// slips through processes a killmail twice and double-counts every statistic
// it touches, while an unwanted collapse merely defers work by one run.
func InsertOptsFor(queueName string, priority Priority) *river.InsertOpts {
	opts := &river.InsertOpts{
		Queue:    queueName,
		Priority: int(priority),
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: activeUniqueStates,
		},
	}
	if !priority.Valid() {
		opts.Priority = int(Live)
	}
	if q := jobs.QueueByName(queueName); q != nil && q.Retries > 0 {
		opts.MaxAttempts = q.Retries
	}
	return opts
}

// activeUniqueStates reproduce BullMQ's non-TTL deduplication lifecycle.
//
// A duplicate is suppressed only while the original can still run. Completed,
// cancelled and discarded jobs are retained for diagnostics, but must not keep
// a stable entity id from ever being dispatched again. River's default unique
// state set includes completed, which turns a ten-minute token refresh into a
// once-per-job-retention refresh (normally once a day).
var activeUniqueStates = []rivertype.JobState{
	rivertype.JobStateAvailable,
	rivertype.JobStatePending,
	rivertype.JobStateRetryable,
	rivertype.JobStateRunning,
	rivertype.JobStateScheduled,
}
