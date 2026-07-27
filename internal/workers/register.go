package workers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/eve-kill/shrike/internal/cron"
	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/riverqueue/river"
	"github.com/rs/zerolog"
)

// registration pairs a queue with the worker that consumes it.
//
// One table drives both the River registration and the "what is ported"
// reporting. River's Workers type does not expose what has been registered, so
// the alternative is two hand-written lists that must agree — and a list
// claiming a queue is implemented when no worker was added produces exactly the
// failure the reporting exists to prevent: jobs fetched, no worker found, and a
// backlog draining into the failure table.
type registration struct {
	kind string
	add  func(*river.Workers, *Deps)
}

var registrations = []registration{
	{
		kind: queue.KillmailArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &KillmailWorker{Deps: d}) },
	},
	{
		kind: queue.CharacterArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &CharacterWorker{Deps: d}) },
	},
	{
		kind: queue.CorporationArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &CorporationWorker{Deps: d}) },
	},
	{
		kind: queue.AllianceArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &AllianceWorker{Deps: d}) },
	},
	{
		kind: queue.CharacterHistoryArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &CharacterHistoryWorker{Deps: d}) },
	},
	{
		kind: queue.CorporationHistoryArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &CorporationHistoryWorker{Deps: d}) },
	},
	{
		kind: queue.WarArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &WarWorker{Deps: d}) },
	},
	{
		kind: queue.AnnouncementEventArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &AnnouncementEventWorker{Deps: d}) },
	},
	{
		kind: queue.CommentEventArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &CommentEventWorker{Deps: d}) },
	},
	{
		kind: queue.StatsWriterArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &StatsWriterWorker{Deps: d}) },
	},
	{
		kind: queue.TokenRefreshArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &TokenRefreshWorker{Deps: d}) },
	},
	{
		kind: queue.CharacterKillmailArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &CharacterKillmailWorker{Deps: d}) },
	},
	{
		kind: queue.CorporationKillmailArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &CorporationKillmailWorker{Deps: d}) },
	},
	{
		kind: queue.CorporationWalletArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &CorporationWalletWorker{Deps: d}) },
	},
	{
		kind: queue.AchievementsArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &AchievementsWorker{Deps: d}) },
	},
	{
		kind: queue.FitExtractArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &FitExtractWorker{Deps: d}) },
	},
	{
		kind: queue.BattleDetectionArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &BattleDetectionWorker{Deps: d}) },
	},
	{
		kind: queue.GraphIngestArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &GraphIngestWorker{Deps: d}) },
	},
	{
		kind: queue.CampaignProcessingArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &CampaignProcessingWorker{Deps: d}) },
	},
	{
		kind: queue.ImageRefreshArgs{}.Kind(),
		add:  func(w *river.Workers, d *Deps) { river.AddWorker(w, &ImageRefreshWorker{Deps: d}) },
	},
}

// Register builds the worker set.
//
// The returned registry is the cron dispatch table; the workers themselves go
// into River. Both are needed by a process that consumes jobs, and neither by
// one that only enqueues.
func Register(d *Deps) (*river.Workers, *cron.Registry, error) {
	w := river.NewWorkers()
	for _, r := range registrations {
		r.add(w, d)
	}

	registry, err := RegisterCrons(d)
	if err != nil {
		return nil, nil, err
	}
	river.AddWorker(w, newCronWorker(d, registry))

	return w, registry, nil
}

func newCronWorker(d *Deps, registry *cron.Registry) *cron.Worker {
	cronLog := d.Log.With().Str("component", "cron").Logger()
	return &cron.Worker{
		Registry: registry,
		Redis:    d.Redis,
		OnStart: func(name string) {
			cronLog.Info().Str("cron", name).Msg("cron started")
		},
		OnRun: func(run cron.Run) {
			logCronRun(cronLog, run)
		},
	}
}

func logCronRun(logger zerolog.Logger, run cron.Run) {
	var event *zerolog.Event
	if run.Err != nil {
		event = logger.Error().Err(run.Err)
	} else {
		event = logger.Info()
	}
	event.Str("cron", run.Name).Dur("duration", run.Elapsed)
	if report := strings.TrimSpace(run.Report); report != "" {
		event.Str("report", report)
	}

	switch {
	case run.Skipped:
		event.Msg("cron skipped")
	case run.Err != nil:
		event.Msg("cron failed")
	default:
		event.Msg("cron completed")
	}
}

// ImplementedQueues returns the declared queues that have a worker.
func ImplementedQueues() []string {
	var out []string
	for _, r := range registrations {
		if jobs.QueueByName(r.kind) != nil {
			out = append(out, r.kind)
		}
	}
	sort.Strings(out)
	return out
}

// UnimplementedQueues returns the declared queues with no Go worker.
//
// Reported rather than hidden. A worker process that consumes twenty queues and
// implements six looks, from the outside, exactly like one that implements all
// twenty and finds nothing to do — right up until someone notices weeks later
// that no achievements have been awarded since the cutover.
func UnimplementedQueues() []string {
	have := map[string]bool{}
	for _, r := range registrations {
		have[r.kind] = true
	}

	var out []string
	for _, q := range jobs.Queues {
		if !have[q.Name] && !q.ConsumerElsewhere {
			out = append(out, q.Name)
		}
	}
	sort.Strings(out)
	return out
}

// ConsumableQueues returns the queues a worker process should consume.
//
// The cron queue is deliberately not among them: scheduled jobs run on the cron
// process, which both schedules and works them. Keeping them off the general
// workers is the whole reason the cron queue is separate — a nightly rebuild
// that runs for twenty minutes must not occupy a slot that killmails need.
//
// Consuming a declared-but-unported queue would be worse than not consuming it.
// River fetches the job, finds no worker for the kind, and the job fails — so
// the backlog would drain into the failure table instead of waiting for the
// port to land.
func ConsumableQueues() []string {
	return ImplementedQueues()
}

// CronQueues is what the cron process consumes.
//
// River will not start a client that has periodic jobs but nothing to work, and
// that restriction is right: a process that only inserted rows and relied on
// something else to run them would report itself healthy while nothing
// scheduled ever happened.
func CronQueues() []string {
	return []string{queue.CronQueue}
}

// Describe renders a one-line summary of what is and is not ported.
func Describe(registry *cron.Registry) string {
	return fmt.Sprintf("%d/%d queues, %d/%d crons",
		len(ImplementedQueues()), len(jobs.Queues),
		len(registry.Implemented()), len(jobs.Crons))
}
