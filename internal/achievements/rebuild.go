package achievements

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RebuildResult describes an authoritative achievement rebuild.
type RebuildResult struct {
	Definitions int   `json:"definitions"`
	Rows        int64 `json:"rows"`
	Removed     int64 `json:"removed"`
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

	sources, err := rebuildSources(definitions)
	if err != nil {
		return out, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	if len(sources) > 0 {
		if err := loadRebuildCounts(ctx, tx, sources); err != nil {
			return out, err
		}
	}

	for _, source := range sources {
		for _, def := range source.Definitions {
			upserted, removed, err := rebuildDefinition(ctx, tx, def, source.Index)
			if err != nil {
				return out, fmt.Errorf("rebuild achievement %s: %w", def.ID, err)
			}
			out.Rows += upserted
			out.Removed += removed
		}
	}

	// The TS command only performs the table-wide denormalized point sync for a
	// complete, unfiltered rebuild. Preserve that outcome for compatibility.
	if syncAll {
		tag, err := tx.Exec(ctx, `
			WITH totals AS MATERIALIZED (
				SELECT c.character_id,
				       coalesce(sum(a.points), 0)::int AS points
				FROM characters c
				LEFT JOIN entity_achievements a ON a.entity_id = c.character_id
				GROUP BY c.character_id
			)
			UPDATE characters c
			SET achievement_points = totals.points
			FROM totals
			WHERE c.character_id = totals.character_id
			  AND c.achievement_points IS DISTINCT FROM totals.points`)
		if err != nil {
			return out, fmt.Errorf("sync character achievement points: %w", err)
		}
		out.Characters = tag.RowsAffected()
	}

	if err := tx.Commit(ctx); err != nil {
		return out, err
	}
	return out, nil
}

func rebuildDefinition(
	ctx context.Context,
	tx pgx.Tx,
	def Definition,
	sourceIndex int32,
) (int64, int64, error) {
	var upserted, removed int64
	err := tx.QueryRow(ctx, `
			WITH counted AS MATERIALIZED (
				SELECT character_id, cnt
				FROM achievement_rebuild_counts
				WHERE source_index = $4
			),
			upserted AS (
				INSERT INTO entity_achievements (
				entity_id, achievement_id, current_count, threshold,
				completion_tiers, is_completed, points, completed_at, last_updated
			)
			SELECT character_id, $1, cnt, $2::int,
			       floor(cnt::numeric / $2::numeric)::int,
			       cnt >= $2::int,
			       $3::int * GREATEST(1, floor(cnt::numeric / $2::numeric)::int),
			       CASE WHEN cnt >= $2::int THEN now() ELSE NULL END,
			       now()
			FROM counted
			WHERE cnt > 0
			ON CONFLICT (entity_id, achievement_id) DO UPDATE SET
				current_count = EXCLUDED.current_count,
				completion_tiers = EXCLUDED.completion_tiers,
				is_completed = EXCLUDED.is_completed,
				points = EXCLUDED.points,
				completed_at = COALESCE(entity_achievements.completed_at, EXCLUDED.completed_at),
				last_updated = now()
			RETURNING 1
		),
		removed AS (
			DELETE FROM entity_achievements existing
			WHERE existing.achievement_id = $1
			  AND NOT EXISTS (
				SELECT 1
				FROM counted
				WHERE counted.character_id = existing.entity_id
				  AND counted.cnt > 0
			  )
			RETURNING 1
		)
			SELECT (SELECT count(*) FROM upserted),
			       (SELECT count(*) FROM removed)`,
		def.ID, def.Threshold, def.SignedBasePoints(), sourceIndex).
		Scan(&upserted, &removed)
	if err != nil {
		return 0, 0, err
	}
	return upserted, removed, nil
}

type rebuildSource struct {
	Index       int32
	Trigger     Trigger
	GroupIDs    []int32
	MinValue    float64
	MinSec      float64
	MaxSec      float64
	Definitions []Definition
}

// rebuildSources collapses definitions that count the same source rows. Five
// escalating "frigates killed" thresholds, for example, need one count set and
// five cheap threshold applications rather than five scans of killmail history.
func rebuildSources(definitions []Definition) ([]rebuildSource, error) {
	var out []rebuildSource
	byKey := make(map[string]int, len(definitions))

	for _, def := range definitions {
		key, groups, err := rebuildSourceKey(def)
		if err != nil {
			return nil, fmt.Errorf("achievement %s: %w", def.ID, err)
		}
		if i, ok := byKey[key]; ok {
			out[i].Definitions = append(out[i].Definitions, def)
			continue
		}

		byKey[key] = len(out)
		out = append(out, rebuildSource{
			Index:       int32(len(out) + 1),
			Trigger:     def.Trigger,
			GroupIDs:    groups,
			MinValue:    def.MinValue,
			MinSec:      def.MinSec,
			MaxSec:      def.MaxSec,
			Definitions: []Definition{def},
		})
	}
	return out, nil
}

func rebuildSourceKey(def Definition) (string, []int32, error) {
	switch def.Trigger {
	case TriggerFinalBlows:
		return string(def.Trigger), nil, nil
	case TriggerSoloKills:
		return string(def.Trigger), nil, nil
	case TriggerKillsByValue:
		return fmt.Sprintf("%s:%g", def.Trigger, def.MinValue), nil, nil
	case TriggerKillsBySecurity:
		return fmt.Sprintf("%s:%g:%g", def.Trigger, def.MinSec, def.MaxSec), nil, nil
	case TriggerShipKills:
		groups := slices.Clone(def.GroupIDs)
		slices.Sort(groups)
		return fmt.Sprintf("%s:%v", def.Trigger, groups), groups, nil
	case TriggerShipLosses:
		groups := slices.Clone(def.GroupIDs)
		slices.Sort(groups)
		return fmt.Sprintf("%s:%v", def.Trigger, groups), groups, nil
	default:
		return "", nil, fmt.Errorf("unsupported trigger %q", def.Trigger)
	}
}

func loadRebuildCounts(ctx context.Context, tx pgx.Tx, sources []rebuildSource) error {
	if _, err := tx.Exec(ctx, `
		CREATE TEMP TABLE achievement_rebuild_counts (
			source_index integer NOT NULL,
			character_id integer NOT NULL,
			cnt integer NOT NULL,
			PRIMARY KEY (source_index, character_id)
		) ON COMMIT DROP`); err != nil {
		return fmt.Errorf("create achievement rebuild staging: %w", err)
	}

	var statsSources, criteriaSources, shipKillSources, shipLossSources []rebuildSource
	for _, source := range sources {
		switch source.Trigger {
		case TriggerFinalBlows, TriggerSoloKills:
			statsSources = append(statsSources, source)
		case TriggerKillsByValue, TriggerKillsBySecurity:
			criteriaSources = append(criteriaSources, source)
		case TriggerShipKills:
			shipKillSources = append(shipKillSources, source)
		case TriggerShipLosses:
			shipLossSources = append(shipLossSources, source)
		}
	}

	for _, load := range []struct {
		name    string
		sources []rebuildSource
		run     func(context.Context, pgx.Tx, []rebuildSource) error
	}{
		{"stats", statsSources, loadStatsCounts},
		{"value/security", criteriaSources, loadCriteriaCounts},
		{"ship kills", shipKillSources, loadShipKillCounts},
		{"ship losses", shipLossSources, loadShipLossCounts},
	} {
		if len(load.sources) == 0 {
			continue
		}
		if err := load.run(ctx, tx, load.sources); err != nil {
			return fmt.Errorf("count achievement %s: %w", load.name, err)
		}
	}
	return nil
}

func loadStatsCounts(ctx context.Context, tx pgx.Tx, sources []rebuildSource) error {
	var finalBlowsIndex, soloKillsIndex int32
	for _, source := range sources {
		switch source.Trigger {
		case TriggerFinalBlows:
			finalBlowsIndex = source.Index
		case TriggerSoloKills:
			soloKillsIndex = source.Index
		}
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO achievement_rebuild_counts (source_index, character_id, cnt)
		WITH totals AS MATERIALIZED (
			SELECT entity_id AS character_id,
			       sum(final_blows)::int AS final_blows,
			       sum(solo_kills)::int AS solo_kills
			FROM stats
			WHERE entity_type = 0 AND period_type = 2
			GROUP BY entity_id
		)
		SELECT value.source_index, totals.character_id, value.cnt
		FROM totals
		CROSS JOIN LATERAL (VALUES
			($1::int, totals.final_blows),
			($2::int, totals.solo_kills)
		) value(source_index, cnt)
		WHERE value.source_index <> 0 AND value.cnt > 0`,
		finalBlowsIndex, soloKillsIndex)
	return err
}

func loadCriteriaCounts(ctx context.Context, tx pgx.Tx, sources []rebuildSource) error {
	var counts strings.Builder
	var values strings.Builder
	args := make([]any, 0, len(sources)*3)
	for i, source := range sources {
		if i > 0 {
			counts.WriteString(",\n")
			values.WriteString(", ")
		}
		alias := fmt.Sprintf("count_%d", i)
		switch source.Trigger {
		case TriggerKillsByValue:
			minValue := len(args) + 1
			args = append(args, source.MinValue)
			fmt.Fprintf(&counts,
				"count(*) FILTER (WHERE k.total_value >= $%d::float8)::int AS %s",
				minValue, alias)
		case TriggerKillsBySecurity:
			minSec := len(args) + 1
			maxSec := minSec + 1
			args = append(args, source.MinSec, source.MaxSec)
			fmt.Fprintf(&counts,
				"count(*) FILTER (WHERE s.security >= $%d::float8 AND s.security < $%d::float8)::int AS %s",
				minSec, maxSec, alias)
		}
		sourceIndex := len(args) + 1
		args = append(args, source.Index)
		fmt.Fprintf(&values, "($%d::int, totals.%s)", sourceIndex, alias)
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO achievement_rebuild_counts (source_index, character_id, cnt)
		WITH totals AS MATERIALIZED (
			SELECT a.character_id,
			       `+counts.String()+`
			FROM killmail_attackers a
			JOIN killmails k
			  ON k.killmail_id = a.killmail_id
			 AND k.killmail_time = a.killmail_time
			LEFT JOIN solar_systems s ON s.solar_system_id = k.solar_system_id
			WHERE a.character_id IS NOT NULL
			  AND a.final_blow = true
			  AND k.is_npc = false
			GROUP BY a.character_id
		)
		SELECT value.source_index, totals.character_id, value.cnt
		FROM totals
		CROSS JOIN LATERAL (VALUES `+values.String()+`) value(source_index, cnt)
		WHERE value.cnt > 0`, args...)
	return err
}

func loadShipKillCounts(ctx context.Context, tx pgx.Tx, sources []rebuildSource) error {
	counts, values, args := shipCountProjection(sources, "k.victim_ship_group_id")
	_, err := tx.Exec(ctx, `
		INSERT INTO achievement_rebuild_counts (source_index, character_id, cnt)
		WITH totals AS MATERIALIZED (
			SELECT a.character_id,
			       `+counts+`
			FROM killmail_attackers a
			JOIN killmails k
			  ON k.killmail_id = a.killmail_id
			 AND k.killmail_time = a.killmail_time
			WHERE a.character_id IS NOT NULL
			GROUP BY a.character_id
		)
		SELECT value.source_index, totals.character_id, value.cnt
		FROM totals
		CROSS JOIN LATERAL (VALUES `+values+`) value(source_index, cnt)
		WHERE value.cnt > 0`, args...)
	return err
}

func loadShipLossCounts(ctx context.Context, tx pgx.Tx, sources []rebuildSource) error {
	counts, values, args := shipCountProjection(sources, "victim_ship_group_id")
	_, err := tx.Exec(ctx, `
		INSERT INTO achievement_rebuild_counts (source_index, character_id, cnt)
		WITH totals AS MATERIALIZED (
			SELECT victim_character_id AS character_id,
			       `+counts+`
			FROM killmails
			WHERE victim_character_id IS NOT NULL
			GROUP BY victim_character_id
		)
		SELECT value.source_index, totals.character_id, value.cnt
		FROM totals
		CROSS JOIN LATERAL (VALUES `+values+`) value(source_index, cnt)
		WHERE value.cnt > 0`, args...)
	return err
}

func shipCountProjection(sources []rebuildSource, groupColumn string) (string, string, []any) {
	var counts strings.Builder
	var values strings.Builder
	args := make([]any, 0, len(sources)*2)
	for i, source := range sources {
		if i > 0 {
			counts.WriteString(",\n")
			values.WriteString(", ")
		}
		groupIDs := len(args) + 1
		args = append(args, source.GroupIDs)
		alias := fmt.Sprintf("count_%d", i)
		fmt.Fprintf(&counts,
			"count(*) FILTER (WHERE %s = ANY($%d::int[]))::int AS %s",
			groupColumn, groupIDs, alias)

		sourceIndex := len(args) + 1
		args = append(args, source.Index)
		fmt.Fprintf(&values, "($%d::int, totals.%s)", sourceIndex, alias)
	}
	return counts.String(), values.String(), args
}
