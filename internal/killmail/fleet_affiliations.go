package killmail

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Each affiliation contributes at most once per kill. Its sole character, if
// any, lets the dashboard exclude the viewer without storing a copy of the
// affiliation list for every attacker.
const insertFleetAffiliationsSQL = `
	WITH source AS MATERIALIZED (
		SELECT killmail_id FROM killmails
		WHERE killmail_id = ANY($1::integer[])
		ORDER BY killmail_id FOR KEY SHARE
	), affiliations AS (
		SELECT a.killmail_id, f.kind, f.entity_id,
		       CASE WHEN min(a.character_id) = max(a.character_id) THEN min(a.character_id) END AS sole_character_id
		FROM source s
		JOIN killmail_attackers a USING (killmail_id)
		CROSS JOIN LATERAL (VALUES (0::smallint, a.corporation_id), (1::smallint, a.alliance_id)) f(kind, entity_id)
		WHERE a.character_id IS NOT NULL AND f.entity_id IS NOT NULL
		GROUP BY a.killmail_id, f.kind, f.entity_id
	)
	INSERT INTO killmail_fleet_affiliations (killmail_id, kinds, entity_ids, sole_character_ids)
	SELECT s.killmail_id,
	       COALESCE(array_agg(a.kind ORDER BY a.kind, a.entity_id) FILTER (WHERE a.kind IS NOT NULL), '{}'),
	       COALESCE(array_agg(a.entity_id ORDER BY a.kind, a.entity_id) FILTER (WHERE a.kind IS NOT NULL), '{}'),
	       COALESCE(array_agg(a.sole_character_id ORDER BY a.kind, a.entity_id) FILTER (WHERE a.kind IS NOT NULL), '{}')
	FROM source s LEFT JOIN affiliations a USING (killmail_id)
	GROUP BY s.killmail_id
	ON CONFLICT (killmail_id) DO NOTHING`

// BackfillFleetAffiliations prepares a bounded time range using the live
// projection. A retry skips killmails already prepared.
func BackfillFleetAffiliations(ctx context.Context, pool *pgxpool.Pool, from, to time.Time) (int64, error) {
	var prepared int64
	for start := from; start.Before(to); start = start.Add(time.Hour) {
		end := start.Add(time.Hour)
		if end.After(to) {
			end = to
		}
		rows, err := pool.Query(ctx, `
			SELECT k.killmail_id FROM killmails k
			WHERE k.killmail_time >= $1 AND k.killmail_time < $2
			  AND NOT EXISTS (SELECT 1 FROM killmail_fleet_affiliations f WHERE f.killmail_id = k.killmail_id)`, start, end)
		if err != nil {
			return prepared, fmt.Errorf("select fleet affiliations for %s: %w", start, err)
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[int32])
		if err != nil {
			return prepared, fmt.Errorf("read fleet affiliation IDs for %s: %w", start, err)
		}
		for batch := range slices.Chunk(ids, 1000) {
			tag, err := pool.Exec(ctx, insertFleetAffiliationsSQL, batch)
			if err != nil {
				return prepared, fmt.Errorf("prepare fleet affiliations for %s: %w", start, err)
			}
			prepared += tag.RowsAffected()
		}
	}
	return prepared, nil
}
