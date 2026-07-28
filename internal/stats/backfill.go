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
	EntityTypes     []EntityType
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
	var err error
	opts.EntityTypes, err = normalizeEntityTypes(opts.EntityTypes)
	if err != nil {
		return out, err
	}
	if onlyEntityType(opts.EntityTypes, EntityFaction) && !opts.WantStats {
		return out, fmt.Errorf("faction aggregation has headline stats but no breakdowns")
	}

	if opts.Reset {
		for _, table := range selectedStatsTables(opts) {
			if len(opts.EntityTypes) == 0 {
				if _, err := pool.Exec(ctx, `TRUNCATE `+table); err != nil {
					return out, err
				}
				continue
			}
			if _, err := pool.Exec(ctx,
				`DELETE FROM `+table+` WHERE entity_type = ANY($1::smallint[])`,
				entityTypeValues(opts.EntityTypes)); err != nil {
				return out, err
			}
		}
	} else if !opts.SkipAggregation {
		for _, table := range selectedStatsTables(opts) {
			var exists bool
			query := `SELECT EXISTS (SELECT 1 FROM ` + table
			args := []any{}
			if len(opts.EntityTypes) > 0 {
				query += ` WHERE entity_type = ANY($1::smallint[])`
				args = append(args, entityTypeValues(opts.EntityTypes))
			}
			query += ` LIMIT 1)`
			if err := pool.QueryRow(ctx, query, args...).Scan(&exists); err != nil {
				return out, err
			}
			if exists {
				return out, fmt.Errorf(
					"refusing aggregation without reset: %s already has selected rows",
					table,
				)
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
					opts.WantStats, opts.WantBreakdowns, opts.EntityTypes,
				)
				if err != nil {
					return out, fmt.Errorf("aggregate %s: %w", month.Format("2006-01"), err)
				}
				out.Killmails += kills
				out.Stats += written.Stats
				out.Breakdowns += written.Breakdowns
				if opts.WantBreakdowns {
					if err := capBreakdowns(
						ctx, pool, PeriodMonthly, month, opts.EntityTypes,
					); err != nil {
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
					opts.WantStats, opts.WantBreakdowns, opts.EntityTypes,
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
			if out.MonthlyStats, err = rebuildAllMonthlyStats(
				ctx, pool, opts.EntityTypes,
			); err != nil {
				return out, err
			}
			if out.YearlyStats, err = rebuildAllYearlyStats(
				ctx, pool, opts.EntityTypes,
			); err != nil {
				return out, err
			}
		}
		if opts.WantBreakdowns {
			if out.MonthlyBreakdowns, err = rebuildAllMonthlyBreakdowns(
				ctx, pool, opts.EntityTypes,
			); err != nil {
				return out, err
			}
			if out.YearlyBreakdowns, err = rebuildAllYearlyBreakdowns(
				ctx, pool, opts.EntityTypes,
			); err != nil {
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

func normalizeEntityTypes(values []EntityType) ([]EntityType, error) {
	if len(values) == 0 {
		return nil, nil
	}
	seen := make(map[EntityType]struct{}, len(values))
	out := make([]EntityType, 0, len(values))
	for _, value := range values {
		if value < EntityCharacter || value > EntityFaction {
			return nil, fmt.Errorf("invalid stats entity type %d", value)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	slices.Sort(out)
	return out, nil
}

func entityTypeValues(values []EntityType) []int16 {
	out := make([]int16, len(values))
	for i, value := range values {
		out[i] = int16(value)
	}
	return out
}

func onlyEntityType(values []EntityType, want EntityType) bool {
	return len(values) == 1 && values[0] == want
}

func aggregateRange(
	ctx context.Context,
	pool *pgxpool.Pool,
	from, to, periodStart time.Time,
	period PeriodType,
	wantStats, wantBreakdowns bool,
	entityTypes []EntityType,
) (WriteResult, int64, error) {
	if period == PeriodMonthly {
		return aggregateRangeStaged(
			ctx, pool, from, to, periodStart, period, wantStats, wantBreakdowns,
			entityTypes,
		)
	}

	acc := NewAccumulator()
	var kills int64
	err := streamBackfillRange(
		ctx, pool, from, to, CatchupBatch, entityTypes,
		func(batch []dayItem) error {
			for _, item := range batch {
				addBackfillItem(acc, item, entityTypes)
				kills++
			}
			return nil
		},
	)
	if err != nil {
		return WriteResult{}, kills, err
	}
	acc.KeepEntityTypes(entityTypes)

	written, err := WritePeriod(ctx, pool, acc, periodStart, period, wantStats, wantBreakdowns)
	return written, kills, err
}

// aggregateRangeStaged bounds historical monthly aggregation by one input
// batch. The final cardinality of stats_breakdowns for a busy month can be far
// larger than the killmail batch itself, so keeping the accumulator for the
// whole month defeats streaming the input.
//
// Session-local tables preserve the exact additive merge semantics without
// changing the persistent schema. They shadow the public aggregate tables only
// on this transaction's connection, letting the normal writer merge each
// bounded accumulator into them. Public rows are touched once, after the full
// month has succeeded.
func aggregateRangeStaged(
	ctx context.Context,
	pool *pgxpool.Pool,
	from, to, periodStart time.Time,
	period PeriodType,
	wantStats, wantBreakdowns bool,
	entityTypes []EntityType,
) (WriteResult, int64, error) {
	var out WriteResult

	tx, err := pool.Begin(ctx)
	if err != nil {
		return out, 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	for _, statement := range []string{
		`CREATE TEMP TABLE stats
			(LIKE public.stats INCLUDING DEFAULTS)
			ON COMMIT DROP`,
		`ALTER TABLE stats
			ADD PRIMARY KEY (entity_type, entity_id, period_type, period_start)`,
		`CREATE TEMP TABLE stats_breakdowns
			(LIKE public.stats_breakdowns INCLUDING DEFAULTS)
			ON COMMIT DROP`,
		`ALTER TABLE stats_breakdowns
			ADD PRIMARY KEY (
				entity_type, entity_id, period_type, period_start, dim_category, dim_id
			)`,
	} {
		if _, err := tx.Exec(ctx, statement); err != nil {
			return out, 0, fmt.Errorf("create monthly stats staging: %w", err)
		}
	}

	var kills int64
	err = streamBackfillRange(
		ctx, pool, from, to, CatchupBatch, entityTypes,
		func(batch []dayItem) error {
			acc := NewAccumulator()
			for _, item := range batch {
				addBackfillItem(acc, item, entityTypes)
				kills++
			}
			acc.KeepEntityTypes(entityTypes)
			if _, err := WritePeriodTx(
				ctx, tx, acc, periodStart, period, wantStats, wantBreakdowns,
			); err != nil {
				return fmt.Errorf("merge monthly stats staging: %w", err)
			}
			return nil
		},
	)
	if err != nil {
		return out, kills, err
	}

	if wantBreakdowns {
		if _, err := tx.Exec(ctx,
			capBreakdownsSQL("pg_temp.stats_breakdowns"),
			int16(period), periodStart.Format("2006-01-02"), BreakdownTopN,
			len(entityTypes) == 0, entityTypeValues(entityTypes),
		); err != nil {
			return out, kills, fmt.Errorf("cap monthly stats staging: %w", err)
		}
	}

	if wantStats {
		tag, err := tx.Exec(ctx, `
			INSERT INTO public.stats AS target (
				entity_type, entity_id, period_type, period_start,
				kills, losses, solo_kills, solo_losses, npc_losses, final_blows,
				points, isk_destroyed, isk_lost, damage_dealt, damage_taken,
				sum_attacker_count
			)
			SELECT entity_type, entity_id, period_type, period_start,
			       kills, losses, solo_kills, solo_losses, npc_losses, final_blows,
			       points, isk_destroyed, isk_lost, damage_dealt, damage_taken,
			       sum_attacker_count
			FROM pg_temp.stats
			ON CONFLICT (entity_type, entity_id, period_type, period_start)
			DO UPDATE SET
				kills              = target.kills              + EXCLUDED.kills,
				losses             = target.losses             + EXCLUDED.losses,
				solo_kills         = target.solo_kills         + EXCLUDED.solo_kills,
				solo_losses        = target.solo_losses        + EXCLUDED.solo_losses,
				npc_losses         = target.npc_losses         + EXCLUDED.npc_losses,
				final_blows        = target.final_blows        + EXCLUDED.final_blows,
				points             = target.points             + EXCLUDED.points,
				isk_destroyed      = target.isk_destroyed      + EXCLUDED.isk_destroyed,
				isk_lost           = target.isk_lost           + EXCLUDED.isk_lost,
				damage_dealt       = target.damage_dealt       + EXCLUDED.damage_dealt,
				damage_taken       = target.damage_taken       + EXCLUDED.damage_taken,
				sum_attacker_count = target.sum_attacker_count + EXCLUDED.sum_attacker_count`)
		if err != nil {
			return out, kills, fmt.Errorf("publish monthly stats staging: %w", err)
		}
		out.Stats = tag.RowsAffected()
	}

	if wantBreakdowns {
		tag, err := tx.Exec(ctx, `
			INSERT INTO public.stats_breakdowns AS target (
				entity_type, entity_id, period_type, period_start,
				dim_category, dim_id, kills, losses, isk_destroyed, isk_lost,
				last_killmail_id, last_killmail_time
			)
			SELECT entity_type, entity_id, period_type, period_start,
			       dim_category, dim_id, kills, losses, isk_destroyed, isk_lost,
			       last_killmail_id, last_killmail_time
			FROM pg_temp.stats_breakdowns
			ON CONFLICT (
				entity_type, entity_id, period_type, period_start, dim_category, dim_id
			) DO UPDATE SET
				kills         = target.kills         + EXCLUDED.kills,
				losses        = target.losses        + EXCLUDED.losses,
				isk_destroyed = target.isk_destroyed + EXCLUDED.isk_destroyed,
				isk_lost      = target.isk_lost      + EXCLUDED.isk_lost,
				last_killmail_id = CASE
					WHEN target.last_killmail_time IS NULL
					     OR EXCLUDED.last_killmail_time > target.last_killmail_time
					THEN EXCLUDED.last_killmail_id
					WHEN EXCLUDED.last_killmail_time = target.last_killmail_time
					     AND (target.last_killmail_id IS NULL
					          OR EXCLUDED.last_killmail_id > target.last_killmail_id)
					THEN EXCLUDED.last_killmail_id
					ELSE target.last_killmail_id
				END,
				last_killmail_time = greatest(
					target.last_killmail_time, EXCLUDED.last_killmail_time
				)`)
		if err != nil {
			return out, kills, fmt.Errorf("publish monthly breakdown staging: %w", err)
		}
		out.Breakdowns = tag.RowsAffected()
	}

	if err := tx.Commit(ctx); err != nil {
		return out, kills, err
	}
	return out, kills, nil
}

func streamBackfillRange(
	ctx context.Context,
	pool *pgxpool.Pool,
	from, to time.Time,
	limit int,
	entityTypes []EntityType,
	yield func([]dayItem) error,
) error {
	if onlyEntityType(entityTypes, EntityFaction) {
		return streamFactionRange(ctx, pool, from, to, limit, yield)
	}
	return streamRange(ctx, pool, from, to, limit, yield)
}

func addBackfillItem(acc *Accumulator, item dayItem, entityTypes []EntityType) {
	if onlyEntityType(entityTypes, EntityFaction) {
		acc.AddFactions(item.km, item.attackers)
		return
	}
	acc.Add(item.km, item.attackers)
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

func capBreakdowns(
	ctx context.Context,
	pool *pgxpool.Pool,
	period PeriodType,
	start time.Time,
	entityTypes []EntityType,
) error {
	_, err := pool.Exec(ctx,
		capBreakdownsSQL("stats_breakdowns"),
		int16(period), start.Format("2006-01-02"), BreakdownTopN,
		len(entityTypes) == 0, entityTypeValues(entityTypes),
	)
	return err
}

func capBreakdownsSQL(table string) string {
	return fmt.Sprintf(`
			DELETE FROM %[1]s b
			USING (
				SELECT entity_type, entity_id, period_type, period_start, dim_category, dim_id
				FROM (
				SELECT entity_type, entity_id, period_type, period_start, dim_category, dim_id,
				       row_number() OVER (
				           PARTITION BY entity_type, entity_id, period_type, period_start, dim_category
					           ORDER BY (kills + losses) DESC, dim_id
					       ) AS rn
					FROM %[1]s
					WHERE period_type = $1 AND period_start = $2::date
					  AND ($4 OR entity_type = ANY($5::smallint[]))
				) ranked
				WHERE rn > $3
		) excess
		WHERE b.entity_type = excess.entity_type
		  AND b.entity_id = excess.entity_id
		  AND b.period_type = excess.period_type
			  AND b.period_start = excess.period_start
			  AND b.dim_category = excess.dim_category
			  AND b.dim_id = excess.dim_id`, table)
}

func rebuildAllMonthlyStats(
	ctx context.Context,
	pool *pgxpool.Pool,
	entityTypes []EntityType,
) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `
		DELETE FROM stats target
		WHERE target.period_type = $1
		  AND ($3 OR target.entity_type = ANY($4::smallint[]))
		  AND period_start IN (
		      SELECT DISTINCT date_trunc('month', period_start)::date
		      FROM stats source
		      WHERE source.period_type = $2
		        AND ($3 OR source.entity_type = ANY($4::smallint[]))
		  )`,
		int16(PeriodMonthly), int16(PeriodDaily),
		len(entityTypes) == 0, entityTypeValues(entityTypes),
	); err != nil {
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
		  AND ($3 OR entity_type = ANY($4::smallint[]))
		GROUP BY entity_type, entity_id, date_trunc('month', period_start)`,
		int16(PeriodMonthly), int16(PeriodDaily),
		len(entityTypes) == 0, entityTypeValues(entityTypes))
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func rebuildAllYearlyStats(
	ctx context.Context,
	pool *pgxpool.Pool,
	entityTypes []EntityType,
) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `
		DELETE FROM stats
		WHERE period_type = $1
		  AND ($2 OR entity_type = ANY($3::smallint[]))`,
		int16(PeriodYearly), len(entityTypes) == 0, entityTypeValues(entityTypes),
	); err != nil {
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
		  AND ($3 OR entity_type = ANY($4::smallint[]))
		GROUP BY entity_type, entity_id, date_trunc('year', period_start)`,
		int16(PeriodYearly), int16(PeriodMonthly),
		len(entityTypes) == 0, entityTypeValues(entityTypes))
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func rebuildAllMonthlyBreakdowns(
	ctx context.Context,
	pool *pgxpool.Pool,
	entityTypes []EntityType,
) (int64, error) {
	return rebuildAllBreakdowns(
		ctx, pool, PeriodMonthly, PeriodDaily, "month", true, entityTypes,
	)
}

func rebuildAllYearlyBreakdowns(
	ctx context.Context,
	pool *pgxpool.Pool,
	entityTypes []EntityType,
) (int64, error) {
	return rebuildAllBreakdowns(
		ctx, pool, PeriodYearly, PeriodMonthly, "year", false, entityTypes,
	)
}

func rebuildAllBreakdowns(
	ctx context.Context,
	pool *pgxpool.Pool,
	target, source PeriodType,
	trunc string,
	onlySourceMonths bool,
	entityTypes []EntityType,
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
			  AND ($3 OR entity_type = ANY($4::smallint[]))
			  AND period_start IN (
			      SELECT DISTINCT date_trunc('%s', period_start)::date
			      FROM stats_breakdowns
			      WHERE period_type = $2
			        AND ($3 OR entity_type = ANY($4::smallint[]))
			  )`, trunc),
			int16(target), int16(source),
			len(entityTypes) == 0, entityTypeValues(entityTypes),
		); err != nil {
			return 0, err
		}
	} else if _, err := tx.Exec(ctx,
		`DELETE FROM stats_breakdowns
		 WHERE period_type = $1
		   AND ($2 OR entity_type = ANY($3::smallint[]))`,
		int16(target), len(entityTypes) == 0, entityTypeValues(entityTypes),
	); err != nil {
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
			       `+latestKillmailIDRollupSQL+` AS last_killmail_id,
			       max(last_killmail_time) AS last_killmail_time
			FROM stats_breakdowns
			WHERE period_type = $2
			  AND ($3 OR entity_type = ANY($4::smallint[]))
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
		WHERE rn <= %[2]d`, trunc, BreakdownTopN),
		int16(target), int16(source),
		len(entityTypes) == 0, entityTypeValues(entityTypes))
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
