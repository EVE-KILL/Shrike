package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/eve-kill/shrike/internal/cron"
	"github.com/eve-kill/shrike/internal/esi"
	"github.com/eve-kill/shrike/internal/eve"
	"github.com/eve-kill/shrike/internal/everef"
	"github.com/eve-kill/shrike/internal/graph"
	"github.com/eve-kill/shrike/internal/images"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/redisx"
	"github.com/eve-kill/shrike/internal/relay"
	"github.com/eve-kill/shrike/internal/sso"
	"github.com/eve-kill/shrike/internal/ticker"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/eve-kill/shrike/internal/workers"
	"github.com/eve-kill/shrike/internal/zkb"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var workCmd = &cobra.Command{
	Use:   "work",
	Short: "Long-running worker processes",
	Long: `The three processes that keep the killboard current.

  work:zkb     follows zKillboard's live feed and enqueues what arrives
  work:queues  consumes the job queues
  work:cron    schedules and runs the recurring jobs

They are separate processes deliberately. The feed reader must never be blocked
by a slow job, and the cron process holds a leader election that only one of any
number of replicas wins — so queues can be scaled out while scheduling stays
singular. Scheduled jobs run on their own queue, so a twenty-minute nightly
rebuild cannot occupy a worker slot that killmails need.`,
}

// deps builds the dependency bundle the workers need.
//
// The static-data cache and the price resolver are loaded once here rather than
// per job: the cache is immutable after load, and Prices memoises per day, so a
// long-lived worker sharing one resolver issues a fraction of the queries a
// per-job one would.
func deps(ctx context.Context, pool *pgxpool.Pool, withQueue bool) (*workers.Deps, error) {
	cache, err := eve.Load(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("load static data: %w", err)
	}
	paths, err := eve.LoadMarketPaths(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("load market paths: %w", err)
	}

	coordination := redisx.Coordination(cfg)

	// Memgraph is optional. The graph is entirely derived from the killmails,
	// so a worker that cannot reach it runs without the graph rather than
	// refusing to start — and says so once rather than per job.
	var graphClient *graph.Client
	if g, err := graph.Connect(ctx, cfg.MemgraphURL); err != nil {
		log.Warn().Err(err).Str("url", cfg.MemgraphURL).
			Msg("memgraph is unreachable — relationship graph will not be updated")
	} else {
		graphClient = g
	}

	d := &workers.Deps{
		Pool:  pool,
		Redis: coordination,
		ESI:   esi.New(cfg),
		SSO: &sso.Client{
			ClientID:     cfg.EVEClientID,
			ClientSecret: cfg.EVEClientSecret,
			UserAgent:    userAgent(),
		},
		EveRef:      everef.NewClient(userAgent()),
		ZKB:         zkb.New(userAgent()),
		Graph:       graphClient,
		Cache:       cache,
		MarketPaths: paths,
		Prices:      eve.NewPrices(pool, cache),
		UserAgent:   userAgent(),
		Log:         log.Logger,
		Relay: &relay.Publisher{
			Redis: coordination,
			OnError: func(channel string, err error) {
				log.Warn().Str("channel", channel).Err(err).Msg("relay publish failed")
			},
		},
	}

	imageStore, err := newImageStorage(cfg)
	if err != nil {
		return nil, err
	}
	d.ImageStore = imageStore
	d.Images = images.New(images.Options{
		Store: imageStore, UserAgent: userAgent(),
	})
	d.GitHubToken = os.Getenv("GITHUB_TOKEN")

	// The ticker publishes through the relay and persists to the cache Redis,
	// which is a different instance from the coordination one: ephemeral
	// announcements are cache data, and losing them on a cache flush is fine.
	d.Ticker = &ticker.Emitter{
		Relay: d.Relay,
		Redis: redisx.Cache(cfg),
		Cache: cache,
	}

	if withQueue {
		// An insert-only client: no workers, so this process cannot start
		// consuming a queue it was not asked to consume.
		c, err := queue.New(queue.Options{Pool: pool})
		if err != nil {
			return nil, err
		}
		d.Queue = c
	}
	return d, nil
}

// --- work:zkb ---

var workZkbCmd = &cobra.Command{
	Use:   "zkb",
	Short: "Follow the zKillboard R2Z2 feed",
	Long: `Follows zKillboard's sequential feed and enqueues every killmail it
publishes.

The feed embeds the full ESI killmail in each entry, so a kill arriving this way
costs nothing from the ESI request or error budget — no hash lookup and no
/killmails/ round trip. That is what makes this the cheap ingest path and the
ESI fetcher the expensive one.

Position is stored in the config table under the same key the Bun listener uses,
so this resumes exactly where that process stopped. The feed is ephemeral —
entries expire after hours — so a listener that falls far enough behind cannot
catch up by following the sequence, and the missed_killmails cron repairs the
gap from zKillboard's daily history index instead.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		d, err := deps(cmd.Context(), pool, true)
		if err != nil {
			return err
		}

		store := &zkb.PostgresStore{
			Pool: pool,
			Deliver: func(ctx context.Context, r *zkb.Response) error {
				// The ESI document travels with the job, so the worker that
				// picks it up does not re-fetch what the feed already gave us.
				_, err := queue.Dispatch(ctx, d.Queue, queue.KillmailArgs{
					KillmailID:   r.KillmailID,
					KillmailHash: r.KillmailHash(),
					Killmail:     &r.ESI,
				}, queue.Live)
				return err
			},
		}

		listener := &zkb.Listener{
			Client:  d.ZKB,
			Store:   store,
			Counter: d.Redis,
			OnEvent: logZkbEvent,
		}

		ui.Section("zKillboard feed")
		return RunService(cmd, "zkb", func(ctx context.Context) error {
			stats, err := listener.Start(ctx)
			log.Info().
				Int64("accepted", stats.Accepted).
				Int64("reposts", stats.Reposts).
				Int64("sequence", stats.Sequence).
				Msg("feed stopped")
			return err
		})
	},
}

// logZkbEvent reports one feed entry.
//
// Caught-up events are not logged. The listener emits one every six seconds
// while sitting at the head of the feed, which is its normal state, and logging
// them would bury everything else at ten lines a minute.
func logZkbEvent(e zkb.Event) {
	switch e.Kind {
	case "new":
		log.Info().Int64("killmail", e.KillmailID).Int64("seq", e.Sequence).Msg("queued")
	case "repost":
		log.Debug().Int64("killmail", e.KillmailID).Int64("seq", e.Sequence).Msg("already stored")
	case "error":
		log.Warn().Int64("seq", e.Sequence).Err(e.Err).Msg("feed error")
	}
}

// --- work:queues ---

var flagWorkQueues []string

var workQueuesCmd = &cobra.Command{
	Use:   "queues",
	Short: "Consume the job queues",
	Long: `Runs workers for every queue that has a Go implementation.

Queues that are declared but not yet ported are deliberately not consumed.
River fetches a job, finds no worker registered for its kind, and fails it — so
consuming an unported queue would drain its backlog into the failure table
rather than leaving it waiting for the port to land.

ESI-dependent queues pause automatically while Tranquility is offline. ESI's
error limit is global, so continuing to ask during a downtime exhausts a budget
shared by everything else and outlasts the downtime itself.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		d, err := deps(cmd.Context(), pool, false)
		if err != nil {
			return err
		}

		riverWorkers, _, err := workers.Register(d)
		if err != nil {
			return err
		}

		consume := flagWorkQueues
		if len(consume) == 0 {
			consume = workers.ConsumableQueues()
		}

		client, err := queue.New(queue.Options{
			Pool:    pool,
			Workers: riverWorkers,
			Queues:  consume,
		})
		if err != nil {
			return err
		}
		// Jobs enqueue follow-up work through the same client that runs them.
		d.Queue = client

		gate := &queue.TQGate{
			Client: client,
			Redis:  d.Redis,
			OnChange: func(offline bool) {
				if offline {
					log.Warn().Msg("Tranquility is offline — ESI queues paused")
				} else {
					log.Info().Msg("Tranquility is online — ESI queues resumed")
				}
			},
		}

		return RunService(cmd, "queues", func(ctx context.Context) error {
			if err := client.Start(ctx); err != nil {
				return err
			}
			go func() { _ = gate.Watch(ctx) }()

			<-ctx.Done()

			// Stop drains in-flight jobs rather than abandoning them. A killmail
			// abandoned mid-insert would be retried anyway, but an ESI fetch
			// abandoned after the request has already been spent is pure waste.
			stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			return client.Stop(stopCtx)
		})
	},
}

// --- work:cron ---

var workCronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Schedule and run the recurring jobs",
	Long: `Schedules every implemented cron on its declared interval and works them.

Scheduling is leader-elected: any number of replicas can run this, and exactly
one of them schedules at a time while all of them work the resulting jobs. That
is the difference from the Bun cron runner, where a second process meant every
job ran twice.

Intervals fire on wall-clock boundaries measured from the Unix epoch, so an
hourly job runs at the top of the hour and a daily job at UTC midnight, whatever
time the process started. Restarting does not shift a schedule.

Scheduled jobs run here rather than on the general workers, on their own queue.
A nightly rebuild that runs for twenty minutes must not occupy a slot that
killmails need.

A cron already queued or running is never queued again, so a job that overruns
its own interval does not stack copies of itself.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		d, err := deps(cmd.Context(), pool, false)
		if err != nil {
			return err
		}

		riverWorkers, registry, err := workers.Register(d)
		if err != nil {
			return err
		}

		periodic, err := registry.PeriodicJobs()
		if err != nil {
			return err
		}

		client, err := queue.New(queue.Options{
			Pool:         pool,
			Workers:      riverWorkers,
			Queues:       workers.CronQueues(),
			PeriodicJobs: periodic,
		})
		if err != nil {
			return err
		}
		// Crons that exist only to enqueue work — the killmail repair job, the
		// entity refreshes — dispatch through the same client that runs them.
		d.Queue = client

		reportSchedule(registry)

		return RunService(cmd, "cron", func(ctx context.Context) error {
			if err := client.Start(ctx); err != nil {
				return err
			}
			<-ctx.Done()

			stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			return client.Stop(stopCtx)
		})
	},
}

// reportSchedule prints what will be scheduled.
func reportSchedule(registry *cron.Registry) {
	specs, err := registry.Specs()
	if err != nil {
		return
	}

	ui.Section("Schedule")
	table := ui.NewTable("CRON", "EVERY", "FLAGS", "DESCRIPTION")
	for _, s := range specs {
		var flags []string
		if s.RunOnStart {
			flags = append(flags, ui.Primary("boot"))
		}
		if s.RequiresTQ {
			flags = append(flags, ui.Warn2("tq"))
		}
		table.Row(ui.Command(s.Name), s.Schedule.Interval().String(), joinFlags(flags), s.Description)
	}
	fmt.Println(table.Render())

	if missing := registry.Unimplemented(); len(missing) > 0 {
		ui.Newline()
		ui.KV("Not scheduled", fmt.Sprintf("%d crons with no Go implementation", len(missing)))
	}
	ui.Newline()
}

func init() {
	workQueuesCmd.Flags().StringSliceVar(&flagWorkQueues, "queue", nil,
		"Consume only these queues (default: every queue with an implementation)")

	workCmd.AddCommand(workZkbCmd, workQueuesCmd, workCronCmd)
}
