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
	WindowFourteenDays
	WindowThirtyDays
	WindowOneEightyDays
	WindowOneYear
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
	if _, err := tx.Exec(ctx, `CREATE TEMP TABLE ranking_achievements ON COMMIT DROP AS
		SELECT 0::smallint AS entity_type, character_id AS entity_id,
		       achievement_points::bigint AS achievement_points
		FROM characters
		WHERE character_id >= 90000000 AND achievement_points > 0
		UNION ALL
		SELECT 1, corporation_id, sum(achievement_points)::bigint
		FROM characters
		WHERE character_id >= 90000000 AND corporation_id >= 2000000
		  AND achievement_points > 0
		GROUP BY corporation_id
		UNION ALL
		SELECT 2, alliance_id, sum(achievement_points)::bigint
		FROM characters
		WHERE character_id >= 90000000 AND alliance_id IS NOT NULL
		  AND achievement_points > 0
		GROUP BY alliance_id`); err != nil {
		return 0, fmt.Errorf("build ranking achievement totals: %w", err)
	}
	if _, err := tx.Exec(ctx, `CREATE UNIQUE INDEX ON ranking_achievements (entity_type, entity_id)`); err != nil {
		return 0, err
	}

	var total int64
	windows := []struct {
		id         int16
		periodType int16
		since      string
	}{
		{WindowWeekly, 0, "CURRENT_DATE - 6"},
		{WindowNinetyDays, 0, "CURRENT_DATE - 89"},
		{WindowAllTime, 2, "DATE '2007-01-01'"},
		{WindowFourteenDays, 0, "CURRENT_DATE - 13"},
		{WindowThirtyDays, 0, "CURRENT_DATE - 29"},
		{WindowOneEightyDays, 0, "CURRENT_DATE - 179"},
		{WindowOneYear, 0, "CURRENT_DATE - 364"},
	}
	for _, window := range windows {
		query := fmt.Sprintf(`
			WITH combat AS MATERIALIZED (
				SELECT entity_type, entity_id, sum(points)::bigint AS combat_points
				FROM stats
				WHERE period_type = $1 AND period_start >= %s AND points > 0
				  AND entity_type IN (0, 1, 2, 3, 4, 6)
				  AND NOT (entity_type = 0 AND entity_id < 90000000)
				  AND NOT (entity_type = 1 AND entity_id < 2000000)
				GROUP BY entity_type, entity_id
			), scored AS (
				SELECT c.*,
				       CASE WHEN c.entity_type IN (0, 1, 2) THEN coalesce(a.achievement_points, 0) ELSE 0 END::bigint AS achievement_points,
				       CASE WHEN c.entity_type IN (0, 1, 2) THEN percent_rank() OVER (
				           PARTITION BY c.entity_type ORDER BY coalesce(a.achievement_points, 0)
				       ) END AS achievement_pct
				FROM combat c
				LEFT JOIN ranking_achievements a USING (entity_type, entity_id)
			), rated AS (
				SELECT *, CASE WHEN entity_type IN (0, 1, 2)
					THEN combat_points + round(combat_points * achievement_pct * 3 / 7)
					ELSE combat_points
				END::integer AS rating
				FROM scored
			), ranked AS (
				SELECT *,
				       rank() OVER (PARTITION BY entity_type ORDER BY combat_points DESC)::integer AS combat_rank,
				       CASE WHEN entity_type IN (0, 1, 2) THEN rank() OVER (PARTITION BY entity_type ORDER BY achievement_points DESC)::integer END AS achievement_rank,
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
			FROM ranked`, window.since)
		tag, err := tx.Exec(ctx, query, window.periodType, window.id)
		if err != nil {
			return 0, fmt.Errorf("refresh ranking window %d: %w", window.id, err)
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
