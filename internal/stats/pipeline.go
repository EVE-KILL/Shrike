package stats

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The nightly rollup.
//
// Daily rows are written incrementally as killmails arrive; monthly and yearly
// are rebuilt from them wholesale. Rebuilding rather than incrementing is what
// makes the aggregates self-correcting — a daily row that was written twice, or
// a killmail that arrived late, is absorbed on the next run instead of leaving
// a permanent discrepancy nobody can find.
//
// The order matters and is not arbitrary: rebuild before purging. Monthly is
// built from daily and yearly from monthly, so purging daily first would
// rebuild the recent months from data that had just been deleted.

// Retention and rollup windows.
const (
	// DailyRetentionDays is how long per-day rows are kept. Every rolling
	// window the site offers is 365 days or less, so a daily row older than
	// that answers no question — and daily is by far the largest slice.
	DailyRetentionDays = 365

	// MonthlyRetentionMonths keeps a year and a half, enough for
	// year-over-year comparison with a margin.
	MonthlyRetentionMonths = 18

	// MonthlyRebuildMonths is how far back monthly rows are rebuilt. Six
	// months including the current one: older months cannot change, because
	// the daily rows behind them have been purged.
	MonthlyRebuildMonths = 5

	// BreakdownTopN caps how many dimensions survive per rolled-up period.
	// Without it a busy alliance's monthly row would carry every system it has
	// ever fought in, and the breakdown table is already the largest in the
	// database.
	BreakdownTopN = 20

	// LeaderboardTopN is how deep each ranking goes.
	LeaderboardTopN = 100
)

// latestKillmailIDRollupSQL keeps the id paired with the newest timestamp.
// Taking MAX(id) and MAX(time) independently can combine two different
// killmails when old history arrives after newer data.
const latestKillmailIDRollupSQL = `(array_agg(
	last_killmail_id
	ORDER BY last_killmail_time DESC NULLS LAST,
	         last_killmail_id DESC NULLS LAST
))[1]`

// PipelineResult reports what the nightly run did.
type PipelineResult struct {
	PurgedDaily       int64 `json:"purged_daily"`
	PurgedMonthly     int64 `json:"purged_monthly"`
	MonthlyStats      int64 `json:"monthly_stats"`
	MonthlyBreakdowns int64 `json:"monthly_breakdowns"`
	YearlyStats       int64 `json:"yearly_stats"`
	YearlyBreakdowns  int64 `json:"yearly_breakdowns"`
	Leaderboards      int64 `json:"leaderboards"`
}

// RunPipeline rebuilds the rolled-up periods and the leaderboards.
func RunPipeline(ctx context.Context, pool *pgxpool.Pool) (PipelineResult, error) {
	var out PipelineResult
	var err error

	// Rebuild first — see the note above about ordering.
	if out.MonthlyStats, out.MonthlyBreakdowns, err = rebuildMonthly(ctx, pool); err != nil {
		return out, err
	}
	if out.YearlyStats, out.YearlyBreakdowns, err = rebuildYearly(ctx, pool); err != nil {
		return out, err
	}
	if out.PurgedDaily, err = purgeDaily(ctx, pool); err != nil {
		return out, err
	}
	if out.PurgedMonthly, err = purgeMonthly(ctx, pool); err != nil {
		return out, err
	}
	if out.Leaderboards, err = rebuildLeaderboards(ctx, pool); err != nil {
		return out, err
	}
	return out, nil
}

// rollup rebuilds one period type from another.
//
// Monthly from daily and yearly from monthly are the same operation with a
// different truncation and source, so they share this. Delete-then-insert
// inside one transaction, so a reader never sees a period half rebuilt.
func rollup(ctx context.Context, pool *pgxpool.Pool, target, source PeriodType, trunc, window string) (int64, int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	// The window is computed by Postgres rather than in Go so the boundary is
	// the database's idea of the current date, which is the same one every
	// other query in the pipeline uses.
	var from string
	if err := tx.QueryRow(ctx, fmt.Sprintf(
		`SELECT to_char(date_trunc('%s', CURRENT_DATE - interval '%s'), 'YYYY-MM-DD')`,
		trunc, window)).Scan(&from); err != nil {
		return 0, 0, err
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM stats WHERE period_type = $1 AND period_start >= $2::date`,
		int16(target), from); err != nil {
		return 0, 0, fmt.Errorf("clear %s stats: %w", trunc, err)
	}

	statsTag, err := tx.Exec(ctx, fmt.Sprintf(`
        INSERT INTO stats (
            entity_type, entity_id, period_type, period_start,
            kills, losses, solo_kills, solo_losses, npc_losses, final_blows, points,
            isk_destroyed, isk_lost, damage_dealt, damage_taken, sum_attacker_count
        )
        SELECT entity_type, entity_id, $1, date_trunc('%s', period_start)::date,
               sum(kills), sum(losses), sum(solo_kills), sum(solo_losses), sum(npc_losses),
               sum(final_blows), sum(points),
               sum(isk_destroyed), sum(isk_lost),
               sum(damage_dealt), sum(damage_taken), sum(sum_attacker_count)
        FROM stats
        WHERE period_type = $2 AND period_start >= $3::date
        GROUP BY entity_type, entity_id, date_trunc('%s', period_start)`, trunc, trunc),
		int16(target), int16(source), from)
	if err != nil {
		return 0, 0, fmt.Errorf("rebuild %s stats: %w", trunc, err)
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM stats_breakdowns WHERE period_type = $1 AND period_start >= $2::date`,
		int16(target), from); err != nil {
		return 0, 0, fmt.Errorf("clear %s breakdowns: %w", trunc, err)
	}

	// Only the top N dimensions per category survive the rollup, ranked by
	// total activity. dim_id breaks ties so the choice is deterministic — two
	// runs over the same data must keep the same rows.
	breakdownTag, err := tx.Exec(ctx, fmt.Sprintf(`
        INSERT INTO stats_breakdowns (
            entity_type, entity_id, period_type, period_start, dim_category, dim_id,
            kills, losses, isk_destroyed, isk_lost, last_killmail_id, last_killmail_time
        )
        WITH aggregated AS (
            SELECT entity_type, entity_id,
                   date_trunc('%s', period_start)::date AS period_start,
                   dim_category, dim_id,
                   sum(kills) AS kills, sum(losses) AS losses,
                   sum(isk_destroyed) AS isk_destroyed, sum(isk_lost) AS isk_lost,
                   `+latestKillmailIDRollupSQL+` AS last_killmail_id,
                   max(last_killmail_time) AS last_killmail_time
            FROM stats_breakdowns
            WHERE period_type = $2 AND period_start >= $3::date
            GROUP BY entity_type, entity_id, date_trunc('%s', period_start), dim_category, dim_id
        )
        SELECT entity_type, entity_id, $1, period_start, dim_category, dim_id,
               kills, losses, isk_destroyed, isk_lost, last_killmail_id, last_killmail_time
        FROM (
            SELECT *, row_number() OVER (
                PARTITION BY entity_type, entity_id, period_start, dim_category
                ORDER BY (kills + losses) DESC, dim_id ASC
            ) AS rn
            FROM aggregated
        ) capped
        WHERE rn <= %d`, trunc, trunc, BreakdownTopN),
		int16(target), int16(source), from)
	if err != nil {
		return 0, 0, fmt.Errorf("rebuild %s breakdowns: %w", trunc, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	return statsTag.RowsAffected(), breakdownTag.RowsAffected(), nil
}

func rebuildMonthly(ctx context.Context, pool *pgxpool.Pool) (int64, int64, error) {
	return rollup(ctx, pool, PeriodMonthly, PeriodDaily, "month",
		fmt.Sprintf("%d months", MonthlyRebuildMonths))
}

// rebuildYearly covers this year and the previous one, so a run in January
// still finalises the year that just ended.
func rebuildYearly(ctx context.Context, pool *pgxpool.Pool) (int64, int64, error) {
	return rollup(ctx, pool, PeriodYearly, PeriodMonthly, "year", "1 year")
}

func purgeDaily(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	return purge(ctx, pool, PeriodDaily,
		fmt.Sprintf("CURRENT_DATE - %d", DailyRetentionDays))
}

func purgeMonthly(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	return purge(ctx, pool, PeriodMonthly,
		fmt.Sprintf("date_trunc('month', CURRENT_DATE - interval '%d months')::date",
			MonthlyRetentionMonths))
}

// purge deletes rows of one period type older than a cutoff.
//
// The cutoff is a SQL expression rather than a bound parameter, and that is
// deliberate. Binding it as a parameter is what broke this in the TypeScript:
// the driver sent it untyped, Postgres resolved `CURRENT_DATE - $1` as
// date-minus-date returning an integer, and the comparison became `date <
// integer` — which throws and aborted the whole pipeline before any rollup ran.
// Both expressions here are built from constants in this file.
func purge(ctx context.Context, pool *pgxpool.Pool, period PeriodType, cutoff string) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	statsTag, err := tx.Exec(ctx, fmt.Sprintf(
		`DELETE FROM stats WHERE period_type = $1 AND period_start < %s`, cutoff), int16(period))
	if err != nil {
		return 0, fmt.Errorf("purge stats: %w", err)
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf(
		`DELETE FROM stats_breakdowns WHERE period_type = $1 AND period_start < %s`, cutoff),
		int16(period)); err != nil {
		return 0, fmt.Errorf("purge breakdowns: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return statsTag.RowsAffected(), nil
}

// leaderboardMetrics pairs each metric with the column it ranks by.
var leaderboardMetrics = []struct {
	metric LeaderboardMetric
	column string
}{
	{MetricKills, "kills"},
	{MetricLosses, "losses"},
	{MetricIskDestroyed, "isk_destroyed"},
	{MetricIskLost, "isk_lost"},
	{MetricSoloKills, "solo_kills"},
	{MetricPoints, "points"},
	{MetricFinalBlows, "final_blows"},
}

// rebuildLeaderboards recomputes every ranking.
//
// One transaction per metric: delete and reinsert together, so a reader never
// sees a metric half rebuilt. Across metrics it does not matter — they are
// independent rankings and nothing reads two at once expecting consistency.
func rebuildLeaderboards(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var total int64

	for _, m := range leaderboardMetrics {
		n, err := rebuildLeaderboard(ctx, pool, m.metric, m.column)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

func rebuildLeaderboard(ctx context.Context, pool *pgxpool.Pool, metric LeaderboardMetric, column string) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	if _, err := tx.Exec(ctx,
		`DELETE FROM stats_leaderboards WHERE metric = $1`, int16(metric)); err != nil {
		return 0, fmt.Errorf("clear leaderboard %d: %w", metric, err)
	}

	// The cast to smallint happens after the rank filter, not inside the inner
	// select. Casting inside would have Postgres evaluate row_number()::smallint
	// for every row in each partition before filtering — and a partition with
	// more than 32,767 entities, which a yearly character partition comfortably
	// exceeds, overflows.
	//
	// The column name comes from the fixed table above, never from input.
	tag, err := tx.Exec(ctx, fmt.Sprintf(`
        INSERT INTO stats_leaderboards (entity_type, period_type, period_start, metric, rank, entity_id, value)
        SELECT entity_type, period_type, period_start, $1, rank::smallint, entity_id, value
        FROM (
            SELECT entity_type, period_type, period_start, entity_id,
                   %[1]s::float8 AS value,
                   row_number() OVER (
                       PARTITION BY entity_type, period_type, period_start
                       ORDER BY %[1]s DESC, entity_id
                   ) AS rank
            FROM stats
            WHERE %[1]s > 0
			  AND NOT (entity_type = 0 AND entity_id < 90000000)
			  AND NOT (entity_type = 1 AND entity_id < 2000000)
        ) ranked
        WHERE rank <= %[2]d`, column, LeaderboardTopN), int16(metric))
	if err != nil {
		return 0, fmt.Errorf("rebuild leaderboard %d: %w", metric, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
