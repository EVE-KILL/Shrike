// Package queue is the job system: what work exists, how it is enqueued, and
// how a worker process consumes it.
//
// It replaces BullMQ with River, which moves the queue from Redis into
// Postgres. That is the substantive change, not an implementation detail. The
// BullMQ arrangement lost jobs whenever Redis was flushed or evicted under
// memory pressure, had no transactional relationship with the rows the jobs
// were about, and needed a separate deduplication mechanism bolted on top. A
// Postgres-backed queue makes "insert the killmail and enqueue its follow-up
// work" a single atomic act, and makes the backlog a table anybody can query.
//
// The declarations in internal/jobs stay authoritative for concurrency, retry
// and backoff. This package turns them into a running client.
package queue

import (
	"encoding/json"
	"time"

	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// Job argument types.
//
// Each corresponds to one queue in internal/jobs.Queues, and Kind() must return
// that queue's name — River routes on the kind string, and the name is also
// what `queue:status` and every operational habit already refer to.
//
// The `river:"unique"` tags are the deduplication key. They matter more than
// they look: without them River hashes the entire encoded args, so the same
// killmail arriving once with an embedded ESI body and once without would be
// two different jobs and the kill would be processed twice. Tagging only the
// identifying field reproduces what the TypeScript dispatcher expressed as a
// `dedupe: "km-<id>"` string, and does it without a second key to keep in sync.

// KillmailArgs processes one killmail.
type KillmailArgs struct {
	KillmailID   int64  `json:"killmail_id" river:"unique"`
	KillmailHash string `json:"killmail_hash"`

	// WarID is supplied out of band by the war queue; the public ESI killmail
	// endpoint does not return it.
	WarID int32 `json:"war_id,omitempty"`

	// Killmail is the ESI document, present when the source already had it.
	// R2Z2 embeds it in every feed entry, which is what makes that ingest path
	// cost nothing from the ESI budget — carrying it here is the difference
	// between a fetch per killmail and no fetch at all.
	Killmail *killmail.ESIKillmail `json:"killmail,omitempty"`
}

func (KillmailArgs) Kind() string { return "killmails" }

// CharacterArgs fetches and stores one character.
type CharacterArgs struct {
	CharacterID int32 `json:"character_id" river:"unique"`
}

func (CharacterArgs) Kind() string { return "esi_character" }

// CorporationArgs fetches and stores one corporation.
type CorporationArgs struct {
	CorporationID int32 `json:"corporation_id" river:"unique"`
}

func (CorporationArgs) Kind() string { return "esi_corporation" }

// AllianceArgs fetches and stores one alliance.
type AllianceArgs struct {
	AllianceID int32 `json:"alliance_id" river:"unique"`
}

func (AllianceArgs) Kind() string { return "esi_alliance" }

// CharacterHistoryArgs fetches one character's corporation history.
type CharacterHistoryArgs struct {
	CharacterID int32 `json:"character_id" river:"unique"`

	// Force overrides the sync marker that otherwise suppresses a fetch whose
	// answer is already known. Deliberately outside the uniqueness key: a
	// forced refresh should collapse against a pending ordinary one rather than
	// queue alongside it.
	Force bool `json:"force,omitempty"`
}

func (CharacterHistoryArgs) Kind() string { return "esi_character_history" }

// CorporationHistoryArgs fetches one corporation's alliance history.
type CorporationHistoryArgs struct {
	CorporationID int32 `json:"corporation_id" river:"unique"`
	Force         bool  `json:"force,omitempty"`
}

func (CorporationHistoryArgs) Kind() string { return "esi_corporation_history" }

// WarArgs fetches one war.
type WarArgs struct {
	WarID int32 `json:"war_id" river:"unique"`

	// MetadataOnly skips walking the war's killmail list. Set by the repair
	// sweep, which fills in wars referenced by killmails already stored — so
	// paging ESI for those killmails would spend the request budget to discover
	// nothing, across hundreds of thousands of finished wars.
	MetadataOnly bool `json:"metadata_only,omitempty"`
}

func (WarArgs) Kind() string { return "esi_war" }

// TokenRefreshArgs refreshes one character's SSO token.
type TokenRefreshArgs struct {
	CharacterID int32 `json:"character_id" river:"unique"`
}

func (TokenRefreshArgs) Kind() string { return "esi_token_refresh" }

// CharacterKillmailArgs reads a character's killmails through their own token.
//
// Worth having even though the public feed carries almost everything: a
// character's own token returns kills that never reached zKillboard, which is
// the only way to see losses nobody else reported.
type CharacterKillmailArgs struct {
	CharacterID int32 `json:"character_id" river:"unique"`
}

func (CharacterKillmailArgs) Kind() string { return "character_killmail" }

// CorporationKillmailArgs reads a corporation's killmails.
//
// Keyed on the corporation rather than the character: several members may have
// granted the scope, and their tokens all return the same list. CharacterID is
// outside the uniqueness key so whichever member's token triggered the job
// collapses onto one job for the corporation.
type CorporationKillmailArgs struct {
	CorporationID int32 `json:"corporation_id" river:"unique"`
	CharacterID   int32 `json:"character_id"`
}

func (CorporationKillmailArgs) Kind() string { return "corporation_killmail" }

// CorporationWalletArgs syncs one corporation's wallet.
type CorporationWalletArgs struct {
	CorporationID int32 `json:"corporation_id" river:"unique"`

	// Force bypasses the staleness gate. Outside the uniqueness key so a forced
	// sync collapses against a pending scheduled one rather than queueing
	// alongside it and doing the same seven paginated walks twice.
	Force bool `json:"force,omitempty"`
}

func (CorporationWalletArgs) Kind() string { return "corporation_wallet" }

// CampaignProcessingArgs recomputes one campaign's statistics.
type CampaignProcessingArgs struct {
	CampaignID string `json:"campaign_id" river:"unique"`
}

func (CampaignProcessingArgs) Kind() string { return "campaign_processing" }

// GraphIngestArgs records one killmail's relationships in the graph.
type GraphIngestArgs struct {
	KillmailID int64 `json:"killmail_id" river:"unique"`
}

func (GraphIngestArgs) Kind() string { return "graph_ingest" }

// BattleDetectionArgs finds the battles in one time window.
//
// Both bounds are in the uniqueness key: two windows are the same job only when
// they cover the same span, and the hourly and daily scans deliberately overlap.
type BattleDetectionArgs struct {
	From time.Time `json:"from" river:"unique"`
	To   time.Time `json:"to" river:"unique"`
}

func (BattleDetectionArgs) Kind() string { return "battle_detection" }

// FitExtractArgs records the fit a killmail's victim was flying.
//
// Only the id: the extractor re-reads the killmail's items, which keeps the
// payload small and makes a re-run produce exactly what the first run did.
type FitExtractArgs struct {
	KillmailID int64 `json:"killmail_id" river:"unique"`
}

func (FitExtractArgs) Kind() string { return "fit_extract" }

// AchievementsArgs awards the badges one killmail earned.
//
// Carries the killmail's shape rather than its id: the fields are a small fixed
// set the caller already has, so re-reading the mail would be two queries to
// rebuild what was just in hand.
type AchievementsArgs struct {
	KillmailID int64 `json:"killmail_id" river:"unique"`

	TotalValue        float64 `json:"total_value"`
	SystemSecurity    float64 `json:"system_security"`
	HasSecurity       bool    `json:"has_security"`
	IsNPC             bool    `json:"is_npc"`
	IsSolo            bool    `json:"is_solo"`
	VictimShipGroupID int32   `json:"victim_ship_group_id"`
	VictimCharacterID int32   `json:"victim_character_id"`

	Attackers []AchievementAttacker `json:"attackers"`
}

// AchievementAttacker is one participant, reduced to what badges care about.
type AchievementAttacker struct {
	CharacterID int32 `json:"character_id"`
	ShipGroupID int32 `json:"ship_group_id"`
	FinalBlow   bool  `json:"final_blow"`
}

func (AchievementsArgs) Kind() string { return "achievements" }

// StatsWriterArgs folds one killmail into the aggregate counters.
//
// Only the id: the worker reads the killmail back from the database, so the
// payload stays tiny and a replay produces exactly the counters the original
// run did rather than whatever was in flight at dispatch time.
type StatsWriterArgs struct {
	KillmailID int64 `json:"killmail_id" river:"unique"`
}

func (StatsWriterArgs) Kind() string { return "stats_writer" }

// ImageRefreshArgs refreshes one cached character, corporation, or alliance
// image from CCP. Kind and ID form the uniqueness key so a popular stale image
// produces one refresh regardless of how many HTTP replicas noticed it.
type ImageRefreshArgs struct {
	EntityKind string `json:"entity_kind" river:"unique"`
	EntityID   int64  `json:"entity_id" river:"unique"`
}

func (ImageRefreshArgs) Kind() string { return "image_refresh" }

// AnnouncementEventArgs forwards an announcement lifecycle event.
//
// The payload is opaque: it is produced by the frontend and consumed by the
// relay's clients, and nothing in between has any business reshaping it.
// Uniqueness keys on the whole payload rather than an id, because the same
// announcement legitimately produces several events — new, then updated, then
// expired — and collapsing those would drop all but the first.
type AnnouncementEventArgs struct {
	Payload json.RawMessage `json:"payload"`
}

func (AnnouncementEventArgs) Kind() string { return "announcement_event" }

// CommentEventArgs forwards a comment lifecycle event.
type CommentEventArgs struct {
	Payload json.RawMessage `json:"payload"`
}

func (CommentEventArgs) Kind() string { return "comment_event" }

// CronArgs runs one scheduled job.
//
// Every cron shares a single job kind rather than having one kind each. River
// routes on the kind returned by a zero-valued args struct, so a per-cron kind
// would need 32 near-identical types and 32 worker registrations to express
// what is really one dispatch table. The cron's name is in the args instead,
// where `river_job.args->>'name'` still makes the backlog readable per job.
type CronArgs struct {
	Name string `json:"name" river:"unique"`
}

func (CronArgs) Kind() string { return "cron" }

// InsertOpts sets the cron queue, its retry budget, and its overlap guard.
//
// A cron that fails is retried a couple of times and then left alone until its
// next tick, which is the natural repair: a job that runs hourly does not need
// ten retries, because the eleventh attempt would land after the next run has
// already started.
//
// The uniqueness is what stops a slow cron from piling up on itself. Several
// run far more often than they are guaranteed to finish — status_update every
// second, esi_token_sync every thirty — and without this a run that overruns
// its interval gets a second copy started underneath it, then a third. The
// TypeScript runner guarded this in memory ("previous run still in progress");
// doing it through the queue makes the guarantee cluster-wide instead of per
// process, which matters now that any number of schedulers can be running.
func (CronArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       CronQueue,
		MaxAttempts: 3,
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: cronUniqueStates,
		},
	}
}

// cronUniqueStates are the states that block a new run of the same cron.
//
// This must be set explicitly, because River's default set includes completed —
// which is right for ordinary jobs and completely wrong for a recurring one. A
// cron whose last run had completed would be blocked from starting again until
// the job cleaner removed that row, and with the default 24-hour retention that
// means every cron in the system runs exactly once per day regardless of its
// schedule. Cancelled and discarded are excluded for the same reason: a run
// that failed permanently must not stop the next one from being attempted.
var cronUniqueStates = []rivertype.JobState{
	rivertype.JobStateAvailable,
	rivertype.JobStatePending,
	rivertype.JobStateRetryable,
	rivertype.JobStateRunning,
	rivertype.JobStateScheduled,
}

// CronQueue is where scheduled jobs run.
//
// Separate from the work queues so a slow hourly job cannot occupy the workers
// that keep the killmail feed moving, and so its concurrency can stay low
// without throttling anything else.
const CronQueue = "cron"
