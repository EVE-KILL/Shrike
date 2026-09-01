// Package rankings builds compact combat and EVE-KILL Rating snapshots from
// the canonical stats and achievement counters.
package rankings

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	WindowWeekly int16 = iota
	WindowNinetyDays
	WindowAllTime
)

// Refresh atomically replaces every ranking window. EVE-KILL Rating is
// deliberately open-ended: combat points are its base, and characters can
// earn an achievement bonus of up to 3/7 of that base. This preserves the
// intended 70/30 combat/achievement weighting without compressing every top
// entity into the same 1000-point percentile bucket.
func Refresh(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `CREATE TEMP TABLE ranking_refresh
		(LIKE entity_rankings INCLUDING DEFAULTS) ON COMMIT DROP`); err != nil {
		return 0, err
	}

	var total int64
	for _, window := range []int16{WindowWeekly, WindowNinetyDays, WindowAllTime} {
		periodType, since := int16(0), "CURRENT_DATE - 6"
		if window == WindowNinetyDays {
			since = "CURRENT_DATE - 89"
		}
		if window == WindowAllTime {
			periodType, since = 2, "DATE '2007-01-01'"
		}
		query := fmt.Sprintf(`
			WITH combat AS MATERIALIZED (
				SELECT entity_type, entity_id, sum(points)::bigint AS combat_points
				FROM stats
				WHERE period_type = $1 AND period_start >= %s AND points > 0
				  AND entity_type IN (0, 1, 2, 3, 4, 6)
				GROUP BY entity_type, entity_id
			), scored AS (
				SELECT c.*,
				       CASE WHEN c.entity_type = 0 THEN coalesce(ch.achievement_points, 0) ELSE 0 END::bigint AS achievement_points,
				       CASE WHEN c.entity_type = 0 THEN percent_rank() OVER (
				           PARTITION BY c.entity_type ORDER BY coalesce(ch.achievement_points, 0)
				       ) END AS achievement_pct
				FROM combat c
				LEFT JOIN characters ch ON c.entity_type = 0 AND ch.character_id = c.entity_id
			), rated AS (
				SELECT *, CASE WHEN entity_type = 0
					THEN combat_points + round(combat_points * achievement_pct * 3 / 7)
					ELSE combat_points
				END::integer AS rating
				FROM scored
			), ranked AS (
				SELECT *,
				       rank() OVER (PARTITION BY entity_type ORDER BY combat_points DESC)::integer AS combat_rank,
				       CASE WHEN entity_type = 0 THEN rank() OVER (PARTITION BY entity_type ORDER BY achievement_points DESC)::integer END AS achievement_rank,
				       rank() OVER (PARTITION BY entity_type ORDER BY rating DESC, combat_points DESC)::integer AS overall_rank,
				       count(*) OVER (PARTITION BY entity_type)::integer AS population
				FROM rated
			)
			INSERT INTO ranking_refresh (
				entity_type, entity_id, ranking_window, combat_points, achievement_points,
				eve_kill_rating, combat_rank, achievement_rank, overall_rank, population, updated_at
			)
			SELECT entity_type, entity_id, $2, combat_points, achievement_points,
			       rating, combat_rank, achievement_rank, overall_rank, population, now()
			FROM ranked`, since)
		tag, err := tx.Exec(ctx, query, periodType, window)
		if err != nil {
			return 0, fmt.Errorf("refresh ranking window %d: %w", window, err)
		}
		total += tag.RowsAffected()
	}
	if _, err := tx.Exec(ctx, `DELETE FROM entity_rankings`); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO entity_rankings SELECT * FROM ranking_refresh`); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return total, nil
}
