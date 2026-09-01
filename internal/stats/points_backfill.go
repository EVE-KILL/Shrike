package stats

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

// RebuildPointOptions controls the targeted point-only stats rebuild. Unlike a
// full stats reset it leaves kills, losses, ISK, damage, and breakdowns live.
type RebuildPointOptions struct {
	FromMonth time.Time
	ToMonth   time.Time
	Workers   int
}

// RebuildPoints replaces combat points in the canonical daily/monthly source
// rows, then uses the normal pipeline to regenerate rollups and leaderboards.
func RebuildPoints(ctx context.Context, pool *pgxpool.Pool, opts RebuildPointOptions) (int64, PipelineResult, error) {
	months := monthsInclusive(monthStart(opts.FromMonth), monthStart(opts.ToMonth))
	workers := max(1, opts.Workers)
	group, groupCtx := errgroup.WithContext(ctx)
	jobs := make(chan time.Time)
	var rows atomic.Int64
	for range workers {
		group.Go(func() error {
			for month := range jobs {
				written, err := rebuildMonthPoints(groupCtx, pool, month)
				if err != nil {
					return fmt.Errorf("rebuild points for %s: %w", month.Format("2006-01"), err)
				}
				rows.Add(written)
			}
			return nil
		})
	}
	group.Go(func() error {
		defer close(jobs)
		for _, month := range months {
			select {
			case jobs <- month:
			case <-groupCtx.Done():
				return groupCtx.Err()
			}
		}
		return nil
	})
	if err := group.Wait(); err != nil {
		return rows.Load(), PipelineResult{}, err
	}
	pipeline, err := RunPipeline(ctx, pool)
	return rows.Load(), pipeline, err
}

func rebuildMonthPoints(ctx context.Context, pool *pgxpool.Pool, month time.Time) (int64, error) {
	next := month.AddDate(0, 1, 0)
	cutoff := dayStart(time.Now().UTC().Add(-DailyRetentionDays * 24 * time.Hour))
	periodType := PeriodMonthly
	periodExpression := "$1::date"
	if next.After(cutoff) {
		periodType = PeriodDaily
		periodExpression = "a.killmail_time::date"
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `
		UPDATE stats SET points = 0
		WHERE entity_type IN (0, 1, 2, 3, 4, 5, 6, 7)
		  AND period_type = $1
		  AND period_start >= $2::date AND period_start < $3::date`,
		int16(periodType), month, next); err != nil {
		return 0, err
	}
	query := fmt.Sprintf(`
		WITH scored AS MATERIALIZED (
			SELECT character_id, corporation_id, alliance_id, faction_id,
			       ship_type_id, k.solar_system_id, k.constellation_id,
			       k.region_id, a.points, %s AS period_start
			FROM killmail_attackers a
			JOIN killmails k ON k.killmail_id = a.killmail_id
			WHERE a.killmail_time >= $1 AND a.killmail_time < $2
			  AND a.character_id IS NOT NULL AND a.points > 0
		), per_entity AS (
			SELECT 0::smallint AS entity_type, character_id AS entity_id, period_start, sum(points)::bigint AS points
			FROM scored GROUP BY character_id, period_start
			UNION ALL
			SELECT 1, corporation_id, period_start, sum(points)::bigint FROM scored
			WHERE corporation_id IS NOT NULL GROUP BY corporation_id, period_start
			UNION ALL
			SELECT 2, alliance_id, period_start, sum(points)::bigint FROM scored
			WHERE alliance_id IS NOT NULL GROUP BY alliance_id, period_start
			UNION ALL
			SELECT 3, ship_type_id, period_start, sum(points)::bigint FROM scored
			WHERE ship_type_id IS NOT NULL GROUP BY ship_type_id, period_start
			UNION ALL
			SELECT 4, solar_system_id, period_start, sum(points)::bigint FROM scored
			WHERE solar_system_id IS NOT NULL GROUP BY solar_system_id, period_start
			UNION ALL
			SELECT 5, constellation_id, period_start, sum(points)::bigint FROM scored
			WHERE constellation_id IS NOT NULL GROUP BY constellation_id, period_start
			UNION ALL
			SELECT 6, region_id, period_start, sum(points)::bigint FROM scored
			WHERE region_id IS NOT NULL GROUP BY region_id, period_start
			UNION ALL
			SELECT 7, faction_id, period_start, sum(points)::bigint FROM scored
			WHERE faction_id IS NOT NULL GROUP BY faction_id, period_start
		)
		INSERT INTO stats (entity_type, entity_id, period_type, period_start, points)
		SELECT entity_type, entity_id, $3, period_start, points FROM per_entity
		ON CONFLICT (entity_type, entity_id, period_type, period_start)
		DO UPDATE SET points = EXCLUDED.points`, periodExpression)
	tag, err := tx.Exec(ctx, query, month, next, int16(periodType))
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
