package stats

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Writing accumulated counters.
//
// Both merges are single statements built with UNNEST, so a killmail that
// produces two thousand counter updates costs two round trips rather than two
// thousand. The rows are sorted by primary key first, which is not cosmetic:
// concurrent writers that take unique-index locks in a consistent global order
// cannot deadlock against each other, and two workers processing kills that
// share a large alliance is the normal case rather than an unlucky one.

// WriteResult reports what a merge wrote.
type WriteResult struct {
	Stats      int64 `json:"stats"`
	Breakdowns int64 `json:"breakdowns"`
}

// Write merges the accumulated counters for one day.
//
// The period is always daily here. Monthly and yearly rows are rebuilt from
// these by the pipeline rather than written incrementally — incrementing three
// period types per counter would triple the write cost for aggregates nobody
// reads until the next day.
func Write(ctx context.Context, pool *pgxpool.Pool, a *Accumulator, day time.Time) (WriteResult, error) {
	return WritePeriod(ctx, pool, a, day, PeriodDaily, true, true)
}

// WritePeriod merges an accumulator into one explicit period. It is used by
// the historical backfill to write old months directly as monthly rows rather
// than creating years of daily rows that the retention job would immediately
// purge.
func WritePeriod(
	ctx context.Context,
	pool *pgxpool.Pool,
	a *Accumulator,
	periodStart time.Time,
	period PeriodType,
	wantStats bool,
	wantBreakdowns bool,
) (WriteResult, error) {
	var out WriteResult
	if a.Empty() {
		return out, nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	out, err = WritePeriodTx(ctx, tx, a, periodStart, period, wantStats, wantBreakdowns)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

// WritePeriodTx is WritePeriod inside a caller-owned transaction.
//
// The stats worker uses this so the additive counter updates and the
// killmail_processing ledger bit commit atomically. A River retry after the
// counters committed but before the bit did would otherwise count the same
// killmail twice.
func WritePeriodTx(
	ctx context.Context,
	tx pgx.Tx,
	a *Accumulator,
	periodStart time.Time,
	period PeriodType,
	wantStats bool,
	wantBreakdowns bool,
) (WriteResult, error) {
	var out WriteResult
	if a.Empty() {
		return out, nil
	}

	var err error
	if wantStats {
		if out.Stats, err = writeStats(ctx, tx, a, periodStart, period); err != nil {
			return out, err
		}
	}
	if wantBreakdowns {
		if out.Breakdowns, err = writeBreakdowns(ctx, tx, a, periodStart, period); err != nil {
			return out, err
		}
	}
	return out, nil
}

func writeStats(
	ctx context.Context,
	tx pgx.Tx,
	a *Accumulator,
	periodStart time.Time,
	period PeriodType,
) (int64, error) {
	if len(a.Stats) == 0 {
		return 0, nil
	}

	keys := make([]StatsKey, 0, len(a.Stats))
	for k := range a.Stats {
		keys = append(keys, k)
	}
	sortStatsKeys(keys)

	n := len(keys)
	var (
		entityTypes = make([]int16, n)
		entityIDs   = make([]int32, n)
		kills       = make([]int64, n)
		losses      = make([]int64, n)
		soloKills   = make([]int64, n)
		soloLosses  = make([]int64, n)
		npcLosses   = make([]int64, n)
		finalBlows  = make([]int64, n)
		points      = make([]int64, n)
		iskD        = make([]float64, n)
		iskL        = make([]float64, n)
		dmgDealt    = make([]int64, n)
		dmgTaken    = make([]int64, n)
		sumAtk      = make([]int64, n)
	)

	for i, k := range keys {
		r := a.Stats[k]
		entityTypes[i], entityIDs[i] = int16(k.EntityType), k.EntityID
		kills[i], losses[i] = r.Kills, r.Losses
		soloKills[i], soloLosses[i], npcLosses[i] = r.SoloKills, r.SoloLosses, r.NPCLosses
		finalBlows[i], points[i] = r.FinalBlows, r.Points
		iskD[i], iskL[i] = r.IskDestroyed, r.IskLost
		dmgDealt[i], dmgTaken[i], sumAtk[i] = r.DamageDealt, r.DamageTaken, r.SumAttackerCount
	}

	tag, err := tx.Exec(ctx, `
        INSERT INTO stats (
            entity_type, entity_id, period_type, period_start,
            kills, losses, solo_kills, solo_losses, npc_losses, final_blows,
            points, isk_destroyed, isk_lost, damage_dealt, damage_taken, sum_attacker_count
        )
        SELECT * FROM unnest(
            $1::smallint[], $2::int[], $3::smallint[], $4::date[],
            $5::int[], $6::int[], $7::int[], $8::int[], $9::int[], $10::int[],
            $11::int[], $12::float8[], $13::float8[], $14::bigint[], $15::bigint[], $16::bigint[]
        )
        ON CONFLICT (entity_type, entity_id, period_type, period_start) DO UPDATE SET
            kills              = stats.kills              + EXCLUDED.kills,
            losses             = stats.losses             + EXCLUDED.losses,
            solo_kills         = stats.solo_kills         + EXCLUDED.solo_kills,
            solo_losses        = stats.solo_losses        + EXCLUDED.solo_losses,
            npc_losses         = stats.npc_losses         + EXCLUDED.npc_losses,
            final_blows        = stats.final_blows        + EXCLUDED.final_blows,
            points             = stats.points             + EXCLUDED.points,
            isk_destroyed      = stats.isk_destroyed      + EXCLUDED.isk_destroyed,
            isk_lost           = stats.isk_lost           + EXCLUDED.isk_lost,
            damage_dealt       = stats.damage_dealt       + EXCLUDED.damage_dealt,
            damage_taken       = stats.damage_taken       + EXCLUDED.damage_taken,
            sum_attacker_count = stats.sum_attacker_count + EXCLUDED.sum_attacker_count`,
		entityTypes, entityIDs, repeatInt16(int16(period), n), repeatDate(periodStart, n),
		kills, losses, soloKills, soloLosses, npcLosses, finalBlows,
		points, iskD, iskL, dmgDealt, dmgTaken, sumAtk)
	if err != nil {
		return 0, fmt.Errorf("merge stats: %w", err)
	}
	return tag.RowsAffected(), nil
}

func writeBreakdowns(
	ctx context.Context,
	tx pgx.Tx,
	a *Accumulator,
	periodStart time.Time,
	period PeriodType,
) (int64, error) {
	if len(a.Breakdowns) == 0 {
		return 0, nil
	}

	keys := make([]BreakdownKey, 0, len(a.Breakdowns))
	for k := range a.Breakdowns {
		keys = append(keys, k)
	}
	sortBreakdownKeys(keys)

	n := len(keys)
	var (
		entityTypes = make([]int16, n)
		entityIDs   = make([]int32, n)
		dimCats     = make([]int16, n)
		dimIDs      = make([]int32, n)
		kills       = make([]int64, n)
		losses      = make([]int64, n)
		iskD        = make([]float64, n)
		iskL        = make([]float64, n)
		lastIDs     = make([]int64, n)
		lastTimes   = make([]time.Time, n)
	)

	for i, k := range keys {
		b := a.Breakdowns[k]
		entityTypes[i], entityIDs[i] = int16(k.EntityType), k.EntityID
		dimCats[i], dimIDs[i] = int16(k.DimCategory), k.DimID
		kills[i], losses[i] = b.Kills, b.Losses
		iskD[i], iskL[i] = b.IskDestroyed, b.IskLost
		lastIDs[i] = b.LastKillmailID
		lastTimes[i] = time.Unix(b.LastKillmailTime, 0).UTC()
	}

	// The last-killmail marker takes the greater of the two rather than the
	// incoming one: a replay or a late-arriving older killmail must not move
	// "last seen" backwards.
	tag, err := tx.Exec(ctx, `
        INSERT INTO stats_breakdowns (
            entity_type, entity_id, period_type, period_start,
            dim_category, dim_id,
            kills, losses, isk_destroyed, isk_lost,
            last_killmail_id, last_killmail_time
        )
        SELECT * FROM unnest(
            $1::smallint[], $2::int[], $3::smallint[], $4::date[],
            $5::smallint[], $6::int[],
            $7::int[], $8::int[], $9::float8[], $10::float8[],
            $11::int[], $12::timestamptz[]
        )
        ON CONFLICT (entity_type, entity_id, period_type, period_start, dim_category, dim_id)
        DO UPDATE SET
            kills         = stats_breakdowns.kills         + EXCLUDED.kills,
            losses        = stats_breakdowns.losses        + EXCLUDED.losses,
            isk_destroyed = stats_breakdowns.isk_destroyed + EXCLUDED.isk_destroyed,
            isk_lost      = stats_breakdowns.isk_lost      + EXCLUDED.isk_lost,
            last_killmail_id = CASE
                WHEN stats_breakdowns.last_killmail_time IS NULL
                     OR EXCLUDED.last_killmail_time > stats_breakdowns.last_killmail_time
                THEN EXCLUDED.last_killmail_id
                WHEN EXCLUDED.last_killmail_time = stats_breakdowns.last_killmail_time
                     AND (stats_breakdowns.last_killmail_id IS NULL
                          OR EXCLUDED.last_killmail_id > stats_breakdowns.last_killmail_id)
                THEN EXCLUDED.last_killmail_id
                ELSE stats_breakdowns.last_killmail_id END,
            last_killmail_time = greatest(
                coalesce(stats_breakdowns.last_killmail_time, '-infinity'::timestamptz),
                EXCLUDED.last_killmail_time)`,
		entityTypes, entityIDs, repeatInt16(int16(period), n), repeatDate(periodStart, n),
		dimCats, dimIDs, kills, losses, iskD, iskL, lastIDs, lastTimes)
	if err != nil {
		return 0, fmt.Errorf("merge stats breakdowns: %w", err)
	}
	return tag.RowsAffected(), nil
}

// sortStatsKeys orders by primary key, so concurrent writers take locks in the
// same global order and cannot deadlock.
func sortStatsKeys(keys []StatsKey) {
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].EntityType != keys[j].EntityType {
			return keys[i].EntityType < keys[j].EntityType
		}
		return keys[i].EntityID < keys[j].EntityID
	})
}

func sortBreakdownKeys(keys []BreakdownKey) {
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		switch {
		case a.EntityType != b.EntityType:
			return a.EntityType < b.EntityType
		case a.EntityID != b.EntityID:
			return a.EntityID < b.EntityID
		case a.DimCategory != b.DimCategory:
			return a.DimCategory < b.DimCategory
		default:
			return a.DimID < b.DimID
		}
	})
}

func repeatInt16(v int16, n int) []int16 {
	out := make([]int16, n)
	for i := range out {
		out[i] = v
	}
	return out
}

// repeatDate truncates a timestamp to its UTC day.
//
// The .UTC() is load-bearing. pgx returns a timestamptz in the connection's
// local zone, so calling Year/Month/Day on it directly takes the *local* day —
// and every kill in the hours either side of midnight lands in the wrong day's
// row. The totals still balance, which is what makes it so easy to miss: only a
// per-day comparison shows it.
func repeatDate(t time.Time, n int) []time.Time {
	t = t.UTC()
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	out := make([]time.Time, n)
	for i := range out {
		out[i] = day
	}
	return out
}
