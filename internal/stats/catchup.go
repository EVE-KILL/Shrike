package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Re-aggregating a recent date range.
//
// The live path counts each killmail once as it arrives, which is correct as
// long as it runs. When it does not — a worker outage, a deploy gap, a queue
// that was drained by hand — the days it missed are permanently short. This
// rebuilds them.
//
// It replaces rather than adds: the daily rows for each day are deleted and
// recomputed from the killmails. That is what makes it safe to re-run, and it
// is the only safe shape for a counter — a catchup that merely added would
// double every day it had already covered.
//
// The aggregation itself is the same Accumulator the live path uses, not a
// second set-based implementation. Two implementations of the same counters
// would eventually disagree, and the one that runs nightly disagreeing with the
// one that runs per killmail is precisely the bug this command exists to fix.

// CatchupResult reports what a catchup replaced.
type CatchupResult struct {
	Days       int   `json:"days"`
	Killmails  int64 `json:"killmails"`
	Deleted    int64 `json:"deleted"`
	Stats      int64 `json:"stats"`
	Breakdowns int64 `json:"breakdowns"`
}

// CatchupBatch is how many killmails are held in memory at once.
//
// The accumulator for a whole day of a busy killboard is large but bounded;
// the killmails and their attackers are not, so they are streamed in batches
// rather than read in one slice.
const CatchupBatch = 5_000

// Catchup rebuilds the daily rows for [from, to).
//
// One day at a time, each in its own transaction. A range that fails partway
// leaves the days it finished correct rather than rolling back the lot —
// re-running covers the rest, and the days already done are replaced again to
// the same values.
func Catchup(ctx context.Context, pool *pgxpool.Pool, from, to time.Time) (CatchupResult, error) {
	return CatchupTargets(ctx, pool, from, to, true, true)
}

// CatchupTargets rebuilds either or both daily aggregate tables for [from, to).
// The selector exists for CLI parity with the TypeScript command, where
// operators can repair stats and breakdowns independently.
func CatchupTargets(
	ctx context.Context,
	pool *pgxpool.Pool,
	from, to time.Time,
	wantStats, wantBreakdowns bool,
) (CatchupResult, error) {
	var out CatchupResult
	if !wantStats && !wantBreakdowns {
		return out, fmt.Errorf("at least one stats target is required")
	}

	for day := from; day.Before(to); day = day.AddDate(0, 0, 1) {
		deleted, written, err := catchupDay(ctx, pool, day, wantStats, wantBreakdowns)
		if err != nil {
			return out, fmt.Errorf("catchup %s: %w", day.Format("2006-01-02"), err)
		}
		out.Days++
		out.Deleted += deleted
		out.Stats += written.Stats
		out.Breakdowns += written.Breakdowns
		out.Killmails += written.killmails
	}

	return out, nil
}

type dayResult struct {
	WriteResult
	killmails int64
}

func catchupDay(
	ctx context.Context,
	pool *pgxpool.Pool,
	day time.Time,
	wantStats, wantBreakdowns bool,
) (int64, dayResult, error) {
	var out dayResult

	day = day.UTC().Truncate(24 * time.Hour)
	next := day.AddDate(0, 0, 1)

	// Accumulate before deleting. If the aggregation fails the existing rows
	// are still there — stale, but present — which is a better failure than a
	// day of counters deleted and not replaced.
	acc := NewAccumulator()
	var lastID int64
	for {
		batch, err := loadDay(ctx, pool, day, next, lastID, CatchupBatch)
		if err != nil {
			return 0, out, err
		}
		if len(batch) == 0 {
			break
		}
		for _, item := range batch {
			acc.Add(item.km, item.attackers)
			out.killmails++
			lastID = item.km.KillmailID
		}
		if len(batch) < CatchupBatch {
			break
		}
	}

	deleted, err := deleteDaily(ctx, pool, day, wantStats, wantBreakdowns)
	if err != nil {
		return 0, out, err
	}

	written, err := WritePeriod(
		ctx, pool, acc, day, PeriodDaily, wantStats, wantBreakdowns,
	)
	if err != nil {
		return deleted, out, err
	}
	out.WriteResult = written
	return deleted, out, nil
}

// deleteDaily removes one day's daily rows.
//
// Daily only. Monthly and yearly rows are derived from these by the pipeline,
// so deleting them here would leave a hole until the nightly run rebuilt it,
// and rebuilding them here would duplicate what the pipeline already does.
func deleteDaily(
	ctx context.Context,
	pool *pgxpool.Pool,
	day time.Time,
	wantStats, wantBreakdowns bool,
) (int64, error) {
	var total int64
	var tables []string
	if wantStats {
		tables = append(tables, "stats")
	}
	if wantBreakdowns {
		tables = append(tables, "stats_breakdowns")
	}
	for _, table := range tables {
		// The table name is from this fixed list, never from input.
		tag, err := pool.Exec(ctx, fmt.Sprintf(
			`DELETE FROM %s WHERE period_type = $1 AND period_start = $2::date`, table),
			PeriodDaily, day.Format("2006-01-02"))
		if err != nil {
			return total, fmt.Errorf("clear %s: %w", table, err)
		}
		total += tag.RowsAffected()
	}
	return total, nil
}

type dayItem struct {
	km                  Killmail
	attackers           []Attacker
	storedAttackerCount pgtype.Int4
}

// loadDay reads a batch of one day's killmails with their attackers.
//
// Paged by killmail id rather than by OFFSET: a busy day is hundreds of
// thousands of rows and OFFSET would re-scan everything before the page on
// every call. Two queries per batch rather than two per killmail — the
// difference between a catchup that takes seconds and one that takes hours.
func loadDay(ctx context.Context, pool *pgxpool.Pool, from, to time.Time, afterID int64, limit int) ([]dayItem, error) {
	rows, err := pool.Query(ctx, `
        SELECT killmail_id, killmail_time,
               coalesce(solar_system_id, 0), coalesce(constellation_id, 0), coalesce(region_id, 0),
               coalesce(victim_character_id, 0), coalesce(victim_corporation_id, 0),
               coalesce(victim_alliance_id, 0), coalesce(victim_ship_type_id, 0),
               coalesce(victim_damage_taken, 0),
               coalesce(total_value, 0), coalesce(points, 0), attacker_count,
               coalesce(is_npc, false), coalesce(is_solo, false)
        FROM killmails
        WHERE killmail_time >= $1 AND killmail_time < $2 AND killmail_id > $3
        ORDER BY killmail_id
        LIMIT $4`, from, to, afterID, limit)
	if err != nil {
		return nil, err
	}

	var items []dayItem
	byID := map[int64]int{}
	var ids []int64

	for rows.Next() {
		var item dayItem
		if err := rows.Scan(&item.km.KillmailID, &item.km.KillmailTime,
			&item.km.SolarSystemID, &item.km.ConstellationID, &item.km.RegionID,
			&item.km.VictimCharacterID, &item.km.VictimCorporationID,
			&item.km.VictimAllianceID, &item.km.VictimShipTypeID,
			&item.km.VictimDamageTaken,
			&item.km.TotalValue, &item.km.Points, &item.storedAttackerCount,
			&item.km.IsNPC, &item.km.IsSolo); err != nil {
			rows.Close()
			return nil, err
		}
		byID[item.km.KillmailID] = len(items)
		items = append(items, item)
		ids = append(ids, item.km.KillmailID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	arows, err := pool.Query(ctx, `
        SELECT killmail_id, coalesce(character_id, 0), coalesce(corporation_id, 0),
               coalesce(alliance_id, 0), coalesce(ship_type_id, 0),
               coalesce(damage_done, 0), coalesce(final_blow, false)
        FROM killmail_attackers WHERE killmail_id = ANY($1::bigint[])`, ids)
	if err != nil {
		return nil, err
	}
	defer arows.Close()

	for arows.Next() {
		var id int64
		var a Attacker
		if err := arows.Scan(&id, &a.CharacterID, &a.CorporationID, &a.AllianceID,
			&a.ShipTypeID, &a.DamageDone, &a.FinalBlow); err != nil {
			return nil, err
		}
		if i, ok := byID[id]; ok {
			items[i].attackers = append(items[i].attackers, a)
		}
	}
	if err := arows.Err(); err != nil {
		return nil, err
	}
	for i := range items {
		items[i].km.AttackerCount = resolvedAttackerCount(
			items[i].storedAttackerCount,
			len(items[i].attackers),
		)
	}
	return items, nil
}
