package jobs

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Cron describes one scheduled job.
type Cron struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	// Schedule is a human interval string — "30s", "5m", "6h", "1d" — matching
	// parseSchedule() in backend/src/cron/CronJob.ts. These are fixed intervals,
	// not cron expressions: there is no way to express "daily at 11:05 UTC" and
	// jobs wanting that check the clock inside their own run().
	Schedule string `json:"schedule"`

	// RunOnStart runs the job immediately at boot instead of waiting a full
	// interval.
	RunOnStart bool `json:"run_on_start"`

	// RequiresTQ skips the run while Tranquility is offline.
	RequiresTQ bool `json:"requires_tq"`

	// LegacyName records a differently-spelled name in the Bun implementation.
	// Only system_activity has one — it is the single hyphenated name among 32,
	// and normalising it matters because these strings become River job kinds.
	LegacyName string `json:"legacy_name,omitempty"`
}

// Crons is the full set. Transcribed 2026-07-25 from backend/src/cronjobs/*.ts.
var Crons = []Cron{
	{Name: "affiliation_update", Schedule: "1m", RequiresTQ: true,
		Description: "Check character affiliations via ESI and queue updates for changes"},
	{Name: "alliance_update", Schedule: "6h", RequiresTQ: true,
		Description: "Discover new alliances from ESI and refresh stale ones"},
	{Name: "analyze", Schedule: "12h",
		Description: "Update PostgreSQL planner statistics for large tables"},
	{Name: "announcement_schedule", Schedule: "30s", RunOnStart: true,
		Description: "Emit WebSocket events for scheduled announcement starts/expiries"},
	{Name: "battle_detection", Schedule: "1h",
		Description: "Queue battle detection jobs (hourly + daily)"},
	{Name: "campaign_pending", Schedule: "1m", RunOnStart: true,
		Description: "Dispatch first-compute jobs for pending campaigns"},
	{Name: "campaign_processing", Schedule: "1h",
		Description: "Gate-check + recompute active campaigns, archive dead ones"},
	{Name: "corporation_update", Schedule: "1h", RequiresTQ: true,
		Description: "Refresh stale corporations from ESI"},
	{Name: "corporation_wallet_sync", Schedule: "1h", RunOnStart: true, RequiresTQ: true,
		Description: "Refresh authorized corporation wallets"},
	{Name: "ek_wallet_reservation_expiry", Schedule: "1m", RunOnStart: true,
		Description: "Release expired EK Wallet reservations"},
	{Name: "entity_history_backfill", Schedule: "1m", RequiresTQ: true,
		Description: "Trickle missing entity histories into low-priority queues"},
	{Name: "entity_snapshot", Schedule: "1d",
		Description: "Daily snapshot of corporation/alliance member stats"},
	{Name: "esi_token_sync", Schedule: "30s", RunOnStart: true, RequiresTQ: true,
		Description: "Refresh stale ESI tokens and dispatch scope-based jobs"},
	{Name: "feed_purge", Schedule: "1d",
		Description: "Purge feed_queue rows older than 1 year"},
	{Name: "find_new_characters", Schedule: "10m", RequiresTQ: true,
		Description: "Probe IDs above MAX(character_id) to discover newly-allocated chars"},
	{Name: "fittings_purge", Schedule: "1d",
		Description: "Purge fittings, fitting_items, killmail_fittings older than 90 days"},
	{Name: "fitting_distributions", Schedule: "1d",
		Description: "Rebuild 90-day fitting-stat histograms and percentiles"},
	{Name: "fw_stats", Schedule: "6h", RunOnStart: true, RequiresTQ: true,
		Description: "Fetch faction warfare stats and leaderboards from ESI (11:05 UTC)"},
	{Name: "fw_update", Schedule: "30m", RunOnStart: true, RequiresTQ: true,
		Description: "Fetch faction warfare system occupancy from ESI"},
	{Name: "graph_purge", Schedule: "1d",
		Description: "Prune graph edges older than the retention window and orphaned nodes"},
	{Name: "insurance", Schedule: "1d",
		Description: "Fetch insurance prices from EVE Ref"},
	{Name: "image_type_sync", Schedule: "1d",
		Description: "Mirror the latest TurtleTools type-ID image export to image storage"},
	{Name: "killmail_delayed", Schedule: "1m",
		Description: "Dispatch killmails whose ESI delay has expired"},
	{Name: "kills_daily_count_reconcile", Schedule: "1d",
		Description: "Rebuild the last few days of kills_daily_count to correct drift"},
	{Name: "character_intel_rollup", Schedule: "1d",
		Description: "Repair recent/dirty character-intelligence days and purge expired facts"},
	{Name: "missed_killmails", Schedule: "1h",
		Description: "Check zKB history for missing killmails (last 14 days)"},
	{Name: "price_compaction", Schedule: "1d",
		Description: "Compact old daily price data into weekly averages"},
	{Name: "price_update", Schedule: "1h",
		Description: "Fetch last 7 days of market history from EVE Ref"},
	{Name: "sde_update", Schedule: "1d",
		Description: "Check for SDE updates and run the importer if needed"},
	{Name: "sovereignty", Schedule: "6h",
		Description: "Fetch latest sovereignty snapshot from EVE Ref"},
	{Name: "stats_pipeline", Schedule: "1d",
		Description: "Nightly stats maintenance (purge daily, rebuild periods + leaderboards)"},
	{Name: "status_update", Schedule: "1s", RunOnStart: true,
		Description: "Publish system status to the WebSocket relay"},
	{Name: "system_activity", Schedule: "1h", LegacyName: "system-activity",
		Description: "Fetch hourly system kills and jumps from ESI"},
	{Name: "tq_status", Schedule: "30s", RunOnStart: true,
		Description: "Check TQ/ESI status and pause the gateway if offline"},
	{Name: "wars", Schedule: "1h", RequiresTQ: true,
		Description: "Fetch new wars from ESI and re-check active wars"},
}

// CronByName resolves a cron by its canonical name, falling back to a legacy
// spelling so `cron:run system-activity` keeps working.
func CronByName(name string) *Cron {
	for i := range Crons {
		if Crons[i].Name == name || Crons[i].LegacyName == name {
			return &Crons[i]
		}
	}
	return nil
}

// ParseSchedule converts an interval string to a duration. It mirrors
// parseSchedule() in backend/src/cron/CronJob.ts and deliberately does not use
// time.ParseDuration, which rejects the "d" unit that several jobs use.
func ParseSchedule(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid schedule %q", s)
	}

	unit := s[len(s)-1]
	n, err := strconv.Atoi(s[:len(s)-1])
	if err != nil {
		return 0, fmt.Errorf("invalid schedule %q: %w", s, err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("invalid schedule %q: interval must be positive", s)
	}

	switch unit {
	case 's':
		return time.Duration(n) * time.Second, nil
	case 'm':
		return time.Duration(n) * time.Minute, nil
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid schedule %q: unknown unit %q", s, string(unit))
	}
}
