// Package workers is where the job implementations live and where everything
// else is wired together.
//
// It sits at the top of the dependency graph on purpose. The queue knows how to
// enqueue and the entity, killmail and importer packages know how to do the
// work, but none of them know about each other — entities returns the follow-up
// work a refresh implies rather than enqueuing it, and killmail parses rather
// than dispatching. Joining them is this package's only job, which keeps the
// cycle that would otherwise form (queue → entities → queue) from existing.
package workers

import (
	"context"

	"github.com/eve-kill/shrike/internal/entities"
	"github.com/eve-kill/shrike/internal/esi"
	"github.com/eve-kill/shrike/internal/eve"
	"github.com/eve-kill/shrike/internal/everef"
	"github.com/eve-kill/shrike/internal/graph"
	"github.com/eve-kill/shrike/internal/images"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/relay"
	"github.com/eve-kill/shrike/internal/sso"
	"github.com/eve-kill/shrike/internal/status"
	"github.com/eve-kill/shrike/internal/ticker"
	"github.com/eve-kill/shrike/internal/zkb"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Deps is everything the jobs need.
//
// Passed as one struct rather than as arguments because almost every job needs
// most of it, and because a worker that needs something not listed here is a
// worker whose dependency was never wired — a compile error rather than a nil
// pointer at three in the morning.
type Deps struct {
	Pool  *pgxpool.Pool
	Redis *redis.Client

	ESI *esi.Client
	SSO *sso.Client

	// Graph is the Memgraph client. Nil when Memgraph is unreachable or not
	// configured — the graph is entirely derived, so running without it is a
	// degraded mode rather than a failure.
	Graph  *graph.Client
	EveRef *everef.Client
	ZKB    *zkb.Client

	// MarketPaths resolves a hull to its /market/... link for killlist rows.
	// Materialised once with the cache, for the same reason: it is a thousand
	// SDE rows that never change while the process is running.
	MarketPaths eve.MarketPaths

	// Relay publishes live events. Nil in CLI paths that run a job once, so
	// every use has to tolerate that — Publisher's methods are nil-safe.
	Relay *relay.Publisher

	// Ticker emits the ephemeral site announcements. Like Relay it is nil-safe
	// and is genuinely nil in one-shot CLI paths: detecting a battle from the
	// command line should not require a relay to announce it to.
	Ticker *ticker.Emitter

	// Cache and Prices are the static-data snapshot and the price resolver.
	// Both are loaded once at startup: the cache because it is immutable after
	// load, and Prices because it memoises per day and a worker sharing one
	// resolver across a thousand killmails does a fraction of the queries a
	// per-job one would.
	Cache  *eve.Cache
	Prices *eve.Prices

	// Queue is how a job enqueues follow-up work. Nil in the CLI paths that run
	// a job once without a running queue, so every use has to tolerate that.
	Queue *queue.Client

	// Images serves and refreshes entity images. ImageStore is kept separately
	// for the daily bulk TurtleTools import, which does not belong in the HTTP
	// service's response cache.
	Images      *images.Service
	ImageStore  images.ObjectStore
	GitHubToken string

	// UserAgent identifies us to CCP, which their acceptable-use policy requires.
	UserAgent string

	Log zerolog.Logger

	// statusCollector caches the expensive halves of the status payload between
	// ticks. Built by RegisterCrons rather than lazily, so it is never nil and
	// there is no shared mutable package state to race on.
	statusCollector *status.Collector
}

// refresher builds the entity fetcher.
func (d *Deps) refresher() *entities.Refresher {
	return &entities.Refresher{Pool: d.Pool, ESI: d.ESI}
}

// dispatchCascade enqueues follow-up work, tolerating the absence of a queue.
//
// The CLI runs jobs with no queue client so an operator can refresh one
// character without a worker process running. In that case the cascade is
// reported and dropped rather than silently retried into nothing, which is
// honest: the follow-up work genuinely did not happen.
func (d *Deps) dispatchCascade(ctx context.Context, c entities.Cascade, parent queue.Priority) (int, error) {
	if d.Queue == nil || c.Empty() {
		return 0, nil
	}
	return queue.DispatchCascade(ctx, d.Queue, queue.Cascade{
		Characters:           c.Characters,
		Corporations:         c.Corporations,
		Alliances:            c.Alliances,
		CharacterHistories:   c.CharacterHistories,
		CorporationHistories: c.CorporationHistories,
	}, parent)
}
