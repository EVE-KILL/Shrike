// Package jobs is the canonical registry of every background queue and cron in
// the system.
//
// During the port this registry is the Go-side declaration of what the Bun
// implementation currently does, transcribed from backend/src/queues/*/queue.ts
// and backend/src/cronjobs/*.ts. Keeping it explicit rather than generated is
// the point: it is meant to be read and diffed against the TypeScript by hand
// before any worker is written, so a missed concurrency or retry setting is
// caught by inspection rather than by a production incident.
//
// The image refresh queue is a Go-side addition: HTTP replicas use it for
// stale-while-revalidate rather than doing origin work on the request path.
package jobs

// Queue describes one background queue.
//
// Field semantics are BullMQ's today, because that is what the values were
// transcribed from and what queue:status reads. They map onto River as:
// Concurrency → per-client queue max workers, Retries → MaxAttempts,
// BackoffDelay → the base of River's retry policy.
type Queue struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	// Concurrency is jobs processed simultaneously per worker process.
	Concurrency int `json:"concurrency"`

	// Retries is the attempt count on failure, and BackoffDelayMS the base
	// delay for exponential backoff between them.
	Retries      int `json:"retries"`
	BackoffDelay int `json:"backoff_delay_ms"`

	// RequiresTQ marks queues that must pause while Tranquility is offline —
	// every ESI-dependent queue, since ESI returns errors when TQ is down and
	// burning retries against it only wastes the rate-limit budget.
	RequiresTQ bool `json:"requires_tq"`

	// ConsumerElsewhere marks queues this process must be able to enqueue to
	// but must never consume, because another pod owns the work. Starting a
	// worker for one of these would race the real consumer.
	ConsumerElsewhere bool `json:"consumer_elsewhere"`
}

// Queues is the full set, ordered as displayed. Transcribed 2026-07-25 from
// backend/src/queues/*/queue.ts.
var Queues = []Queue{
	{
		Name:        "achievements",
		Description: "Process achievement increments for a killmail",
		Concurrency: 1, Retries: 3, BackoffDelay: 1000,
	},
	{
		Name:        "announcement_event",
		Description: "Forward announcement lifecycle events to the WebSocket relay",
		Concurrency: 4, Retries: 3, BackoffDelay: 500,
	},
	{
		Name:        "battle_detection",
		Description: "Detect and save battles for a time window",
		Concurrency: 1, Retries: 2, BackoffDelay: 10_000,
	},
	{
		Name:        "campaign_processing",
		Description: "Full recompute of a campaign's aggregates (backfill + refresh)",
		Concurrency: 1, Retries: 2, BackoffDelay: 30_000,
	},
	{
		Name:        "character_intel_rollup",
		Description: "Atomically rebuild one dirty character-intelligence day",
		Concurrency: 1, Retries: 3, BackoffDelay: 30_000,
	},
	{
		Name:        "character_killmail",
		Description: "Fetch character killmails via authenticated ESI",
		Concurrency: 10, Retries: 2, BackoffDelay: 10_000, RequiresTQ: true,
	},
	{
		Name:        "comment_event",
		Description: "Forward comment lifecycle events to the WebSocket relay",
		Concurrency: 16, Retries: 3, BackoffDelay: 500,
	},
	{
		Name:        "corporation_killmail",
		Description: "Fetch corporation killmails via authenticated ESI",
		Concurrency: 10, Retries: 2, BackoffDelay: 10_000, RequiresTQ: true,
	},
	{
		Name:        "corporation_wallet",
		Description: "Refresh corporation wallet balances and journal",
		Concurrency: 1, Retries: 2, BackoffDelay: 30_000, RequiresTQ: true,
	},
	{
		Name:        "discord_events",
		Description: "Discord bot fan-out — consumed by the discord/ pod",
		Concurrency: 1, Retries: 3, BackoffDelay: 1000, ConsumerElsewhere: true,
	},
	{
		Name:        "esi_alliance",
		Description: "Fetch/update alliance from ESI",
		Concurrency: 1, Retries: 10, BackoffDelay: 30_000, RequiresTQ: true,
	},
	{
		Name:        "esi_character",
		Description: "Fetch/update character from ESI",
		Concurrency: 2, Retries: 10, BackoffDelay: 30_000, RequiresTQ: true,
	},
	{
		Name:        "esi_character_history",
		Description: "Fetch character corporation history from ESI",
		Concurrency: 1, Retries: 10, BackoffDelay: 30_000, RequiresTQ: true,
	},
	{
		Name:        "esi_corporation",
		Description: "Fetch/update corporation from ESI",
		Concurrency: 1, Retries: 10, BackoffDelay: 30_000, RequiresTQ: true,
	},
	{
		Name:        "esi_corporation_history",
		Description: "Fetch corporation alliance history from ESI",
		Concurrency: 1, Retries: 10, BackoffDelay: 30_000, RequiresTQ: true,
	},
	{
		Name:        "esi_token_refresh",
		Description: "Refresh ESI token and dispatch scope-based jobs",
		Concurrency: 5, Retries: 2, BackoffDelay: 10_000, RequiresTQ: true,
	},
	{
		Name:        "esi_war",
		Description: "Fetch/update war details and killmails from ESI",
		Concurrency: 5, Retries: 3, BackoffDelay: 30_000, RequiresTQ: true,
	},
	{
		Name:        "fit_extract",
		Description: "Extract and store the normalized fit for a killmail",
		Concurrency: 5, Retries: 3, BackoffDelay: 1000,
	},
	{
		Name:        "graph_ingest",
		Description: "Ingest killmail relationships into Memgraph",
		Concurrency: 12, Retries: 2, BackoffDelay: 2000,
	},
	{
		Name:        "killmails",
		Description: "Process incoming killmails (parse, insert, stats, blob)",
		Concurrency: 5, Retries: 3, BackoffDelay: 1000, RequiresTQ: true,
	},
	{
		Name:        "image_refresh",
		Description: "Refresh stale entity images from CCP into image storage",
		Concurrency: 4, Retries: 3, BackoffDelay: 10_000,
	},
	{
		Name:        "stats_writer",
		Description: "Write daily stats + breakdown rows for one killmail",
		Concurrency: 8, Retries: 3, BackoffDelay: 2000,
	},
}

// QueueByName returns a queue declaration, or nil when unknown.
func QueueByName(name string) *Queue {
	for i := range Queues {
		if Queues[i].Name == name {
			return &Queues[i]
		}
	}
	return nil
}

// QueueNames returns every declared queue name in registry order.
func QueueNames() []string {
	out := make([]string, len(Queues))
	for i, q := range Queues {
		out[i] = q.Name
	}
	return out
}
