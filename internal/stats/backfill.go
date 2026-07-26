package stats

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BackfillOptions struct {
	FromMonth       time.Time
	ToMonth         time.Time
	DailyCutoff     time.Duration
	WantStats       bool
	WantBreakdowns  bool
	Reset           bool
	Reverse         bool
	SkipAggregation bool
	SkipRollup      bool
}

type BackfillResult struct {
	Months            int   `json:"months"`
	DailyMonths       int   `json:"daily_months"`
	MonthlyMonths     int   `json:"monthly_months"`
	Killmails         int64 `json:"killmails"`
	Stats             int64 `json:"stats"`
	Breakdowns        int64 `json:"breakdowns"`
	MonthlyStats      int64 `json:"monthly_stats"`
	MonthlyBreakdowns int64 `json:"monthly_breakdowns"`
	YearlyStats       int64 `json:"yearly_stats"`
	YearlyBreakdowns  int64 `json:"yearly_breakdowns"`
}

// Backfill performs the same authoritative initial population as the
// TypeScript command. Old months are written directly at monthly granularity;
// recent months retain daily rows, then monthly and yearly rows are rebuilt
// from their lower-resolution source.
func Backfill(ctx context.Context, pool *pgxpool.Pool, opts BackfillOptions) (BackfillResult, error) {
	var out BackfillResult
	if !opts.WantStats && !opts.WantBreakdowns {
		return out, fmt.Errorf("at least one stats target is required")
	}
	if opts.FromMonth.IsZero() || opts.ToMonth.IsZero() {
		return out, fmt.Errorf("from and to months are required")
	}
	opts.FromMonth = monthStart(opts.FromMonth)
	opts.ToMonth = monthStart(opts.ToMonth)
	if opts.ToMonth.Before(opts.FromMonth) {
		return out, fmt.Errorf("to month is before from month")
	}
	if opts.DailyCutoff < 0 {
		return out, fmt.Errorf("daily cutoff cannot be negative")
	}

	if opts.Reset {
		if opts.WantStats {
			if _, err := pool.Exec(ctx, `TRUNCATE stats`); err != nil {
				return out, err
			}
		}
		if opts.WantBreakdowns {
			if _, err := pool.Exec(ctx, `TRUNCATE stats_breakdowns`); err != nil {
				return out, err
			}
		}
	} else if !opts.SkipAggregation {
		for _, table := range selectedStatsTables(opts) {
			var exists bool
			if err := pool.QueryRow(ctx,
				`SELECT EXISTS (SELECT 1 FROM `+table+` LIMIT 1)`).Scan(&exists); err != nil {
				return out, err
			}
			if exists {
				return out, fmt.Errorf("refusing aggregation without reset: %s already has rows", table)
			}
		}
	}

	months := monthsInclusive(opts.FromMonth, opts.ToMonth)
	if opts.Reverse {
		slices.Reverse(months)
	}
	out.Months = len(months)

	if !opts.SkipAggregation {
		cutoff := dayStart(time.Now().UTC().Add(-opts.DailyCutoff))
		for _, month := range months {
			nextMonth := month.AddDate(0, 1, 0)
			directMonthly := !nextMonth.After(cutoff)
			if directMonthly {
				out.MonthlyMonths++
				written, kills, err := aggregateRange(
					ctx, pool, month, nextMonth, month, PeriodMonthly,
					opts.WantStats, opts.WantBreakdowns,
				)
				if err != nil {
					return out, fmt.Errorf("aggregate %s: %w", month.Format("2006-01"), err)
				}
				out.Killmails += kills
				out.Stats += written.Stats
				out.Breakdowns += written.Breakdowns
				if opts.WantBreakdowns {
					if err := capBreakdowns(ctx, pool, PeriodMonthly, month); err != nil {
						return out, fmt.Errorf("cap %s breakdowns: %w", month.Format("2006-01"), err)
					}
				}
				continue
			}

			out.DailyMonths++
			for day := month; day.Before(nextMonth); day = day.AddDate(0, 0, 1) {
				nextDay := day.AddDate(0, 0, 1)
				written, kills, err := aggregateRange(
					ctx, pool, day, nextDay, day, PeriodDaily,
					opts.WantStats, opts.WantBreakdowns,
				)
				if err != nil {
					return out, fmt.Errorf("aggregate %s: %w", day.Format("2006-01-02"), err)
				}
				out.Killmails += kills
				out.Stats += written.Stats
				out.Breakdowns += written.Breakdowns
			}
		}
	}

	if !opts.SkipRollup {
		var err error
		if opts.WantStats {
			if out.MonthlyStats, err = rebuildAllMonthlyStats(ctx, pool); err != nil {
				return out, err
			}
			if out.YearlyStats, err = rebuildAllYearlyStats(ctx, pool); err != nil {
				return out, err
			}
		}
		if opts.WantBreakdowns {
			if out.MonthlyBreakdowns, err = rebuildAllMonthlyBreakdowns(ctx, pool); err != nil {
				return out, err
			}
			if out.YearlyBreakdowns, err = rebuildAllYearlyBreakdowns(ctx, pool); err != nil {
				return out, err
			}
		}
	}
	return out, nil
}

func selectedStatsTables(opts BackfillOptions) []string {
	var tables []string
	if opts.WantStats {
		tables = append(tables, "stats")
	}
	if opts.WantBreakdowns {
		tables = append(tables, "stats_breakdowns")
	}
	return tables
}

func aggregateRange(
	ctx context.Context,
	pool *pgxpool.Pool,
	from, to, periodStart time.Time,
	period PeriodType,
	wantStats, wantBreakdowns bool,
) (WriteResult, int64, error) {
	acc := NewAccumulator()
	var lastID, kills int64
	for {
		batch, err := loadDay(ctx, pool, from, to, lastID, CatchupBatch)
		if err != nil {
			return WriteResult{}, kills, err
		}
		if len(batch) == 0 {
			break
		}
		for _, item := range batch {
			acc.Add(item.km, item.attackers)
			lastID = item.km.KillmailID
			kills++
		}
		if len(batch) < CatchupBatch {
			break
		}
	}

	written, err := WritePeriod(ctx, pool, acc, periodStart, period, wantStats, wantBreakdowns)
	return written, kills, err
}

func monthStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func dayStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func monthsInclusive(from, to time.Time) []time.Time {
	var out []time.Time
	for month := monthStart(from); !month.After(monthStart(to)); month = month.AddDate(0, 1, 0) {
		out = append(out, month)
	}
	return out
}

func capBreakdowns(ctx context.Context, pool *pgxpool.Pool, period PeriodType, start time.Time) error {
	_, err := pool.Exec(ctx, `
		DELETE FROM stats_breakdowns b
		USING (
			SELECT entity_type, entity_id, period_type, period_start, dim_category, dim_id
			FROM (
				SELECT entity_type, entity_id, period_type, period_start, dim_category, dim_id,
				       row_number() OVER (
				           PARTITION BY entity_type, entity_id, period_type, period_start, dim_category
				           ORDER BY (kills + losses) DESC, dim_id
				       ) AS rn
				FROM stats_breakdowns
				WHERE period_type = $1 AND period_start = $2::date
			) ranked
			WHERE rn > $3
		) excess
		WHERE b.entity_type = excess.entity_type
		  AND b.entity_id = excess.entity_id
		  AND b.period_type = excess.period_type
		  AND b.period_start = excess.period_start
		  AND b.dim_category = excess.dim_category
		  AND b.dim_id = excess.dim_id`,
		int16(period), start.Format("2006-01-02"), BreakdownTopN)
	return err
}

func rebuildAllMonthlyStats(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `
		DELETE FROM stats
		WHERE period_type = $1
		  AND period_start IN (
		      SELECT DISTINCT date_trunc('month', period_start)::date
		      FROM stats WHERE period_type = $2
		  )`, int16(PeriodMonthly), int16(PeriodDaily)); err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO stats (
			entity_type, entity_id, period_type, period_start,
			kills, losses, solo_kills, solo_losses, npc_losses, final_blows, points,
			isk_destroyed, isk_lost, damage_dealt, damage_taken, sum_attacker_count
		)
		SELECT entity_type, entity_id, $1, date_trunc('month', period_start)::date,
		       sum(kills), sum(losses), sum(solo_kills), sum(solo_losses), sum(npc_losses),
		       sum(final_blows), sum(points), sum(isk_destroyed), sum(isk_lost),
		       sum(damage_dealt), sum(damage_taken), sum(sum_attacker_count)
		FROM stats
		WHERE period_type = $2
		GROUP BY entity_type, entity_id, date_trunc('month', period_start)`,
		int16(PeriodMonthly), int16(PeriodDaily))
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func rebuildAllYearlyStats(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `DELETE FROM stats WHERE period_type = $1`, int16(PeriodYearly)); err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO stats (
			entity_type, entity_id, period_type, period_start,
			kills, losses, solo_kills, solo_losses, npc_losses, final_blows, points,
			isk_destroyed, isk_lost, damage_dealt, damage_taken, sum_attacker_count
		)
		SELECT entity_type, entity_id, $1, date_trunc('year', period_start)::date,
		       sum(kills), sum(losses), sum(solo_kills), sum(solo_losses), sum(npc_losses),
		       sum(final_blows), sum(points), sum(isk_destroyed), sum(isk_lost),
		       sum(damage_dealt), sum(damage_taken), sum(sum_attacker_count)
		FROM stats
		WHERE period_type = $2
		GROUP BY entity_type, entity_id, date_trunc('year', period_start)`,
		int16(PeriodYearly), int16(PeriodMonthly))
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func rebuildAllMonthlyBreakdowns(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	return rebuildAllBreakdowns(ctx, pool, PeriodMonthly, PeriodDaily, "month", true)
}

func rebuildAllYearlyBreakdowns(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	return rebuildAllBreakdowns(ctx, pool, PeriodYearly, PeriodMonthly, "year", false)
}

func rebuildAllBreakdowns(
	ctx context.Context,
	pool *pgxpool.Pool,
	target, source PeriodType,
	trunc string,
	onlySourceMonths bool,
) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if onlySourceMonths {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
			DELETE FROM stats_breakdowns
			WHERE period_type = $1
			  AND period_start IN (
			      SELECT DISTINCT date_trunc('%s', period_start)::date
			      FROM stats_breakdowns WHERE period_type = $2
			  )`, trunc), int16(target), int16(source)); err != nil {
			return 0, err
		}
	} else if _, err := tx.Exec(ctx,
		`DELETE FROM stats_breakdowns WHERE period_type = $1`, int16(target)); err != nil {
		return 0, err
	}

	tag, err := tx.Exec(ctx, fmt.Sprintf(`
		INSERT INTO stats_breakdowns (
			entity_type, entity_id, period_type, period_start, dim_category, dim_id,
			kills, losses, isk_destroyed, isk_lost, last_killmail_id, last_killmail_time
		)
		WITH aggregated AS (
			SELECT entity_type, entity_id,
			       date_trunc('%[1]s', period_start)::date AS period_start,
			       dim_category, dim_id,
			       sum(kills) AS kills, sum(losses) AS losses,
			       sum(isk_destroyed) AS isk_destroyed, sum(isk_lost) AS isk_lost,
			       max(last_killmail_id) AS last_killmail_id,
			       max(last_killmail_time) AS last_killmail_time
			FROM stats_breakdowns
			WHERE period_type = $2
			GROUP BY entity_type, entity_id, date_trunc('%[1]s', period_start), dim_category, dim_id
		)
		SELECT entity_type, entity_id, $1, period_start, dim_category, dim_id,
		       kills, losses, isk_destroyed, isk_lost, last_killmail_id, last_killmail_time
		FROM (
			SELECT *, row_number() OVER (
				PARTITION BY entity_type, entity_id, period_start, dim_category
				ORDER BY (kills + losses) DESC, dim_id
			) AS rn
			FROM aggregated
		) capped
		WHERE rn <= %[2]d`, trunc, BreakdownTopN), int16(target), int16(source))
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
