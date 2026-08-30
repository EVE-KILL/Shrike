package workers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/eve-kill/shrike/internal/cron"
	"github.com/eve-kill/shrike/internal/esi"
	"github.com/eve-kill/shrike/internal/everef"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/status"
	"github.com/eve-kill/shrike/internal/ticker"
	"github.com/riverqueue/river"
)

// The scheduled jobs.
//
// Only the ones whose dependencies are ported are registered. That is a
// deliberate and visible subset rather than a silent one: `cron:list` prints
// what is implemented and what is not, and the scheduler refuses to schedule
// anything with no implementation, so a job that has not been ported cannot be
// mistaken for one that runs and finds nothing to do.

// RegisterCrons builds the cron registry for the given dependencies.
func RegisterCrons(d *Deps) (*cron.Registry, error) {
	r := cron.NewRegistry()

	// One collector for the life of the registry — see cronStatusUpdate.
	d.statusCollector = status.NewCollector(d.Pool, d.Redis)

	register := func(name string, fn cron.RunFunc) {
		if err := r.Register(name, fn); err != nil {
			panic(err) // a wiring bug, caught by the registry tests
		}
	}

	register("tq_status", d.cronTQStatus)
	register("sovereignty", d.cronSovereignty)
	register("insurance", d.cronInsurance)
	register("price_update", d.cronPriceUpdate)
	register("missed_killmails", d.cronMissedKillmails)
	register("analyze", d.cronAnalyze)
	register("alliance_update", d.cronAllianceUpdate)
	register("affiliation_update", d.cronAffiliationUpdate)

	// Phase 1 — jobs that needed no subsystem beyond what was already ported.
	register("wars", d.cronWars)
	register("corporation_update", d.cronCorporationUpdate)
	register("entity_history_backfill", d.cronEntityHistoryBackfill)
	register("find_new_characters", d.cronFindNewCharacters)
	register("killmail_delayed", d.cronKillmailDelayed)
	register("fw_update", d.cronFwUpdate)
	register("fw_stats", d.cronFwStats)
	register("system_activity", d.cronSystemActivity)
	register("feed_purge", d.cronFeedPurge)
	register("fittings_purge", d.cronFittingsPurge)
	register("price_compaction", d.cronPriceCompaction)
	register("entity_snapshot", d.cronEntitySnapshot)
	register("kills_daily_count_reconcile", d.cronKillsDailyCountReconcile)

	// Phase 2 — event publishing.
	register("status_update", d.cronStatusUpdate)
	register("announcement_schedule", d.cronAnnouncementSchedule)

	// Phase 3 — stats.
	register("stats_pipeline", d.cronStatsPipeline)

	// Phase 4 — tokens.
	register("esi_token_sync", d.cronTokenSync)
	register("corporation_wallet_sync", d.cronCorporationWalletSync)

	// Phase 5.
	register("sde_update", d.cronSDEUpdate)
	register("battle_detection", d.cronBattleDetection)
	register("graph_purge", d.cronGraphPurge)
	register("campaign_pending", d.cronCampaignPending)
	register("campaign_processing", d.cronCampaignProcessing)
	register("ek_wallet_reservation_expiry", d.cronEkWalletReservationExpiry)
	register("image_type_sync", d.cronImageTypeSync)

	return r, nil
}

// cronTQStatus checks whether Tranquility is up and records the answer.
//
// This is the job every other TQ-dependent thing depends on, and it is the one
// job that must never be gated on TQ itself. It writes the flag the queue gate
// and the cron worker both read.
func (d *Deps) cronTQStatus(ctx context.Context) (string, error) {
	res, err := esi.FetchStatus(ctx, d.ESI)
	observation, err := interpretTQStatus(res, err)
	if err != nil {
		// A local timeout, exhausted error budget, or malformed response says
		// nothing authoritative about TQ. Keep the last known state and retry
		// instead of manufacturing an offline transition.
		return "", err
	}

	status := queue.TQOffline
	if observation.online {
		status = "online"
	}

	// The previous value, read before it is overwritten, is what makes this a
	// transition rather than a state. Held in Redis rather than in memory
	// because the cron has leader election: a different process may run the
	// next tick, and in-process state would re-announce on every handover.
	var previous string
	if d.Redis != nil {
		previous, _ = d.Redis.Get(ctx, queue.TQStatusKey).Result()

		if err := d.Redis.Set(ctx, queue.TQStatusKey, status, 0).Err(); err != nil {
			return "", err
		}
		if observation.online {
			if err := d.Redis.Set(ctx, tqPlayersKey, observation.players, 0).Err(); err != nil {
				return "", err
			}
			// The in-process ESI client uses the same pause key as the
			// TypeScript gateway. Clearing it here lets direct requests resume
			// immediately instead of waiting for an old TTL.
			if err := d.Redis.Del(ctx, esiPausedKey).Err(); err != nil {
				return "", err
			}
		} else {
			if err := d.Redis.Del(ctx, tqPlayersKey).Err(); err != nil {
				return "", err
			}
			// Refreshed on every offline tick. The TTL is a failsafe: if the
			// status job dies, a stale pause cannot wedge ESI forever.
			if err := d.Redis.Set(ctx, esiPausedKey, "tq_offline", tqPauseTTL).Err(); err != nil {
				return "", err
			}
		}
	}

	changed := previous != "" && previous != status

	if !observation.online {
		if changed {
			d.Ticker.TQOffline(ctx, observation.reason)
		}
		return "Tranquility is offline: " + observation.reason, nil
	}
	if changed {
		d.Ticker.TQOnline(ctx, fmt.Sprintf("%s players connected", ticker.FormatCount(observation.players)))
	}
	return fmt.Sprintf("Tranquility is online, %d players", observation.players), nil
}

type tqStatusObservation struct {
	online  bool
	players int
	reason  string
}

func interpretTQStatus(res esi.Response[esi.Status], fetchErr error) (tqStatusObservation, error) {
	if fetchErr != nil {
		return tqStatusObservation{}, fmt.Errorf("check Tranquility status: %w", fetchErr)
	}
	if res.OK() && res.Data != nil {
		if res.Data.Error != "" {
			return tqStatusObservation{reason: res.Data.Error}, nil
		}
		return tqStatusObservation{online: true, players: int(res.Data.Players)}, nil
	}

	switch {
	case res.Status >= http.StatusInternalServerError:
		return tqStatusObservation{reason: fmt.Sprintf("ESI returned HTTP %d", res.Status)}, nil
	case res.Status == 0:
		return tqStatusObservation{}, fmt.Errorf("check Tranquility status: probe did not reach ESI")
	case res.Status == statusErrorLimited:
		return tqStatusObservation{}, fmt.Errorf(
			"check Tranquility status: ESI error limit reached (HTTP %d, retry after %ds)",
			res.Status, res.RetryAfter,
		)
	case res.Status == http.StatusTooManyRequests:
		return tqStatusObservation{}, fmt.Errorf(
			"check Tranquility status: ESI rate limit reached (HTTP %d, retry after %ds)",
			res.Status, res.RetryAfter,
		)
	default:
		return tqStatusObservation{}, fmt.Errorf(
			"check Tranquility status: unexpected HTTP %d",
			res.Status,
		)
	}
}

const tqPlayersKey = "esi:tq:players"

const (
	esiPausedKey       = "esi:paused"
	tqPauseTTL         = 2 * time.Minute
	statusErrorLimited = 420
)

// cronSovereignty imports the latest sovereignty snapshot.
func (d *Deps) cronSovereignty(ctx context.Context) (string, error) {
	res, err := everef.ImportSovereigntyLatest(ctx, d.Pool, d.EveRef)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d systems seen, %d changed, %d history entries",
		res.Seen, res.Rows, res.Related), nil
}

// cronInsurance imports insurance prices.
func (d *Deps) cronInsurance(ctx context.Context) (string, error) {
	res, err := everef.ImportInsurance(ctx, d.Pool, d.EveRef)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d insurance rows", res.Rows), nil
}

// cronPriceUpdate imports recent market history.
//
// The window is the last seven days rather than only yesterday, because EVE Ref
// revises a day's file after first publishing it and because a missed run
// should repair itself rather than leave a permanent hole in the price data
// that every killmail valued that day would inherit.
func (d *Deps) cronPriceUpdate(ctx context.Context) (string, error) {
	const window = 7

	var total int64
	var days int
	today := time.Now().UTC()

	var failed int
	for i := 0; i <= window; i++ {
		date := today.AddDate(0, 0, -i).Format("2006-01-02")
		res, err := everef.ImportPriceDay(ctx, d.Pool, d.EveRef, date)
		if err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			failed++
			continue
		}
		if res.Missing {
			failed++
			continue
		}
		total += res.Rows
		days++
	}

	return fmt.Sprintf("%d price rows over %d days, %d unavailable", total, days, failed), nil
}

// cronAnalyze refreshes the planner statistics on the large tables.
//
// Postgres' autovacuum thresholds are proportional, so on a table with hundreds
// of millions of rows it waits for tens of millions of changes before analysing
// — by which time the planner has been choosing badly for hours. Doing it on a
// schedule is cheaper than the sequential scans it prevents.
func (d *Deps) cronAnalyze(ctx context.Context) (string, error) {
	tables := []string{
		"killmails", "killmail_attackers", "killmail_items",
		"characters", "corporations", "alliances",
		"stats", "stats_breakdowns", "entity_achievements", "prices",
	}

	analysed := 0
	for _, t := range tables {
		// The identifier is from this fixed list, never from input.
		if _, err := d.Pool.Exec(ctx, "ANALYZE "+t); err != nil {
			return "", fmt.Errorf("analyze %s: %w", t, err)
		}
		analysed++
	}
	return fmt.Sprintf("%d tables analysed", analysed), nil
}

// cronMissedKillmails repairs gaps using zKillboard's daily history index.
//
// The ephemeral feed retains hours, so anything missed during a longer outage
// is not recoverable by following the sequence. This compares the last two
// weeks of zKillboard's own id/hash index against what is stored and enqueues
// the difference.
func (d *Deps) cronMissedKillmails(ctx context.Context) (string, error) {
	const daysBack = 14

	if d.Queue == nil {
		return "", fmt.Errorf("missed_killmails needs a queue to dispatch into")
	}

	var checked, missing int
	now := time.Now().UTC()

	for i := 1; i <= daysBack; i++ {
		day := now.AddDate(0, 0, -i)

		index, err := d.ZKB.History(ctx, day.Format("20060102"))
		if err != nil {
			// A day zKillboard has not published is routine and must not stop
			// the other thirteen from being checked.
			continue
		}
		checked++
		if len(index) == 0 {
			continue
		}

		gaps, err := d.missingFrom(ctx, day, index)
		if err != nil {
			return "", err
		}
		if len(gaps) == 0 {
			continue
		}

		// Repair work is the lowest tier by definition: it is about the past,
		// and must never delay a killmail arriving now.
		if _, err := queue.DispatchMany(ctx, d.Queue, gaps, queue.RecentBackfill); err != nil {
			return "", err
		}
		missing += len(gaps)
	}

	return fmt.Sprintf("%d days checked, %d missing killmails dispatched", checked, missing), nil
}

// missingFrom returns the killmails in the index that are not stored.
//
// Bounded by killmail_time so it is one indexed range scan per day rather than
// a lookup per killmail, which at twenty thousand a day would be the whole cost
// of the job.
func (d *Deps) missingFrom(ctx context.Context, day time.Time, index map[int64]string) ([]river.JobArgs, error) {
	ids := make([]int64, 0, len(index))
	for id := range index {
		ids = append(ids, id)
	}

	rows, err := d.Pool.Query(ctx, `
        SELECT killmail_id FROM killmails
        WHERE killmail_time >= $1::date
          AND killmail_time < ($1::date + interval '1 day')
          AND killmail_id = ANY($2::bigint[])`,
		day.Format("2006-01-02"), ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stored := make(map[int64]bool, len(ids))
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		stored[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var out []river.JobArgs
	for id, hash := range index {
		if !stored[id] {
			out = append(out, queue.KillmailArgs{KillmailID: id, KillmailHash: hash})
		}
	}
	return out, nil
}

// cronAllianceUpdate discovers new alliances and refreshes stale ones.
func (d *Deps) cronAllianceUpdate(ctx context.Context) (string, error) {
	if d.Queue == nil {
		return "", fmt.Errorf("alliance_update needs a queue to dispatch into")
	}

	res, err := esi.FetchAllianceList(ctx, d.ESI)
	if err != nil {
		return "", err
	}
	if !res.OK() || res.Data == nil {
		return "", fmt.Errorf("ESI returned %d for the alliance list", res.Status)
	}

	// The full list is every alliance in the game, so the interesting question
	// is which of them we do not already have fresh. Asking the database once
	// with the whole list beats asking per alliance by three orders of
	// magnitude.
	unknown, err := staleAlliances(ctx, d, *res.Data)
	if err != nil {
		return "", err
	}

	batch := make([]river.JobArgs, 0, len(unknown))
	for _, id := range unknown {
		batch = append(batch, queue.AllianceArgs{AllianceID: id})
	}

	n, err := queue.DispatchMany(ctx, d.Queue, batch, queue.DormantBackfill)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d alliances known, %d queued for refresh", len(*res.Data), n), nil
}

// cronAffiliationUpdate notices characters who have changed corporation.
//
// The bulk affiliation endpoint answers for a thousand characters in one
// request, which is the only reason keeping affiliations current is affordable
// at all: asking per character would be a million requests a day.
func (d *Deps) cronAffiliationUpdate(ctx context.Context) (string, error) {
	return d.runAffiliationUpdate(ctx)
}
