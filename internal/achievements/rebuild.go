package achievements

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RebuildResult describes an authoritative achievement rebuild.
type RebuildResult struct {
	Definitions int   `json:"definitions"`
	Rows        int64 `json:"rows"`
	Characters  int64 `json:"characters"`
}

// Filter returns the definitions selected by the CLI's optional id and
// category filters. Matching categories case-insensitively mirrors the
// TypeScript command.
func Filter(id, category string) []Definition {
	var out []Definition
	for _, def := range All {
		if id != "" && def.ID != id {
			continue
		}
		if category != "" && !strings.EqualFold(def.Category, category) {
			continue
		}
		out = append(out, def)
	}
	return out
}

// Rebuild replaces the counters for the selected definitions with counts
// derived from the same source tables as the TypeScript command.
//
// This is deliberately not implemented by replaying the live award path.
// Replays add to existing counters and therefore cannot be safely re-run,
// whereas an administrative rebuild must converge to the same rows every time.
func Rebuild(ctx context.Context, pool *pgxpool.Pool, definitions []Definition, syncAll bool) (RebuildResult, error) {
	out := RebuildResult{Definitions: len(definitions)}
	for _, def := range definitions {
		n, err := rebuildDefinition(ctx, pool, def)
		if err != nil {
			return out, fmt.Errorf("rebuild achievement %s: %w", def.ID, err)
		}
		out.Rows += n
	}

	// The TS command only performs the table-wide denormalized point sync for a
	// complete, unfiltered rebuild. Preserve that outcome for compatibility.
	if syncAll {
		tag, err := pool.Exec(ctx, `
			UPDATE characters c
			SET achievement_points = sums.total
			FROM (
				SELECT entity_id, COALESCE(SUM(points), 0)::int AS total
				FROM entity_achievements
				GROUP BY entity_id
			) sums
			WHERE c.character_id = sums.entity_id
			  AND c.achievement_points IS DISTINCT FROM sums.total`)
		if err != nil {
			return out, fmt.Errorf("sync character achievement points: %w", err)
		}
		out.Characters = tag.RowsAffected()
	}
	return out, nil
}

func rebuildDefinition(ctx context.Context, pool *pgxpool.Pool, def Definition) (int64, error) {
	countQuery, args, err := countQuery(def)
	if err != nil {
		return 0, err
	}

	tag, err := pool.Exec(ctx, `
		INSERT INTO entity_achievements (
			entity_id, achievement_id, current_count, threshold,
			completion_tiers, is_completed, points, completed_at, last_updated
		)
		SELECT character_id, $1, cnt, $2,
		       floor(cnt::numeric / $2)::int,
		       cnt >= $2,
		       $3 * GREATEST(1, floor(cnt::numeric / $2)::int),
		       CASE WHEN cnt >= $2 THEN now() ELSE NULL END,
		       now()
		FROM (`+countQuery+`) counted
		WHERE cnt > 0
		ON CONFLICT (entity_id, achievement_id) DO UPDATE SET
			current_count = EXCLUDED.current_count,
			completion_tiers = EXCLUDED.completion_tiers,
			is_completed = EXCLUDED.is_completed,
			points = EXCLUDED.points,
			completed_at = COALESCE(entity_achievements.completed_at, EXCLUDED.completed_at),
			last_updated = now()`,
		append([]any{def.ID, def.Threshold, def.SignedBasePoints()}, args...)...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// countQuery returns a trusted SQL fragment plus bound trigger parameters.
// Every fragment is selected from the fixed Trigger enum; user input is never
// interpolated into SQL.
func countQuery(def Definition) (string, []any, error) {
	switch def.Trigger {
	case TriggerFinalBlows:
		return `
			SELECT entity_id AS character_id, SUM(final_blows)::int AS cnt
			FROM stats
			WHERE entity_type = 0 AND period_type = 2
			GROUP BY entity_id`, nil, nil
	case TriggerSoloKills:
		return `
			SELECT entity_id AS character_id, SUM(solo_kills)::int AS cnt
			FROM stats
			WHERE entity_type = 0 AND period_type = 2
			GROUP BY entity_id`, nil, nil
	case TriggerKillsByValue:
		return `
			SELECT a.character_id, count(*)::int AS cnt
			FROM killmail_attackers a
			JOIN killmails k
			  ON k.killmail_id = a.killmail_id
			 AND k.killmail_time = a.killmail_time
			WHERE a.character_id IS NOT NULL
			  AND a.final_blow = true
			  AND k.is_npc = false
			  AND k.total_value >= $4
			GROUP BY a.character_id`, []any{def.MinValue}, nil
	case TriggerKillsBySecurity:
		return `
			SELECT a.character_id, count(*)::int AS cnt
			FROM killmail_attackers a
			JOIN killmails k
			  ON k.killmail_id = a.killmail_id
			 AND k.killmail_time = a.killmail_time
			JOIN solar_systems s ON s.solar_system_id = k.solar_system_id
			WHERE a.character_id IS NOT NULL
			  AND a.final_blow = true
			  AND k.is_npc = false
			  AND s.security >= $4
			  AND s.security < $5
			GROUP BY a.character_id`, []any{def.MinSec, def.MaxSec}, nil
	case TriggerShipKills:
		return `
			SELECT character_id, SUM(kills)::int AS cnt
			FROM character_ship_stats_daily
			WHERE ship_group_id = ANY($4::int[])
			GROUP BY character_id`, []any{def.GroupIDs}, nil
	case TriggerShipLosses:
		return `
			SELECT character_id, SUM(losses)::int AS cnt
			FROM character_ship_stats_daily
			WHERE ship_group_id = ANY($4::int[])
			GROUP BY character_id`, []any{def.GroupIDs}, nil
	default:
		return "", nil, fmt.Errorf("unsupported trigger %q", def.Trigger)
	}
}
