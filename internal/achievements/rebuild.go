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
		characters, err := syncPointsTx(ctx, tx)
		if err != nil {
			return out, fmt.Errorf("sync character achievement points: %w", err)
		}
		out.Characters = characters
	}

	if err := tx.Commit(ctx); err != nil {
		return out, err
	}
	return out, nil
}

// SyncPoints authoritatively refreshes the denormalized character totals. It
// is separate from Rebuild so production can rebuild expensive achievement
// families in bounded transactions and perform one final totals pass.
func SyncPoints(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var characters int64
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var err error
		characters, err = syncPointsTx(ctx, tx)
		return err
	})
	return characters, err
}

func syncPointsTx(ctx context.Context, tx pgx.Tx) (int64, error) {
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
		return 0, err
	}
	return tag.RowsAffected(), nil
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
					completion_tiers, is_completed, points, completed_at, last_updated,
					level_thresholds, point_unit
				)
			SELECT character_id, $1, cnt, $2::int,
			       level.value,
			       level.value >= cardinality($5::int[]),
			       $3::int * level.value * (level.value + 1) / 2,
			       CASE WHEN cnt >= $2::int THEN now() ELSE NULL END,
			       now(), $5::int[], $3::int
			FROM counted
			CROSS JOIN LATERAL (
				SELECT count(*)::int AS value FROM unnest($5::int[]) target
				WHERE target <= counted.cnt
			) level
			WHERE cnt > 0
			ON CONFLICT (entity_id, achievement_id) DO UPDATE SET
				current_count = EXCLUDED.current_count,
				threshold = EXCLUDED.threshold,
				completion_tiers = EXCLUDED.completion_tiers,
				is_completed = EXCLUDED.is_completed,
				points = EXCLUDED.points,
				level_thresholds = EXCLUDED.level_thresholds,
				point_unit = EXCLUDED.point_unit,
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
		def.ID, def.Threshold, def.SignedBasePoints(), sourceIndex, def.Levels()).
		Scan(&upserted, &removed)
	if err != nil {
		return 0, 0, err
	}
	return upserted, removed, nil
}

type rebuildSource struct {
	Index         int32
	Trigger       Trigger
	GroupIDs      []int32
	MinValue      float64
	MinSec        float64
	MaxSec        float64
	RegionID      int32
	SystemID      int32
	CorporationID int32
	Definitions   []Definition
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
			Index:         int32(len(out) + 1),
			Trigger:       def.Trigger,
			GroupIDs:      groups,
			MinValue:      def.MinValue,
			MinSec:        def.MinSec,
			MaxSec:        def.MaxSec,
			RegionID:      def.RegionID,
			SystemID:      def.SystemID,
			CorporationID: def.CorporationID,
			Definitions:   []Definition{def},
		})
	}
	return out, nil
}

func rebuildSourceKey(def Definition) (string, []int32, error) {
	switch def.Trigger {
	case TriggerFinalBlows, TriggerKills, TriggerLosses:
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
	case TriggerKillsByRegion, TriggerLossesByRegion, TriggerTournament:
		return fmt.Sprintf("%s:%d", def.Trigger, def.RegionID), nil, nil
	case TriggerKillsBySystem, TriggerLossesBySystem:
		return fmt.Sprintf("%s:%d", def.Trigger, def.SystemID), nil, nil
	case TriggerConcorded, TriggerKilledByCorp, TriggerKilledCorp:
		return fmt.Sprintf("%s:%d", def.Trigger, def.CorporationID), nil, nil
	case TriggerAwox, TriggerAwoxed, TriggerGank:
		return string(def.Trigger), nil, nil
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

	var statsSources, criteriaSources, shipKillSources, shipLossSources, specialSources []rebuildSource
	for _, source := range sources {
		switch source.Trigger {
		case TriggerFinalBlows, TriggerSoloKills, TriggerKills, TriggerLosses:
			statsSources = append(statsSources, source)
		case TriggerKillsByValue, TriggerKillsBySecurity:
			criteriaSources = append(criteriaSources, source)
		case TriggerShipKills:
			shipKillSources = append(shipKillSources, source)
		case TriggerShipLosses:
			shipLossSources = append(shipLossSources, source)
		default:
			specialSources = append(specialSources, source)
		}
	}
	if len(shipKillSources)+len(shipLossSources) > 0 {
		if err := createShipSourceMapping(ctx, tx, shipKillSources, shipLossSources); err != nil {
			return fmt.Errorf("prepare ship achievement mapping: %w", err)
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
		{"special", specialSources, loadSpecialCounts},
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
	var finalBlowsIndex, soloKillsIndex, killsIndex, lossesIndex int32
	for _, source := range sources {
		switch source.Trigger {
		case TriggerFinalBlows:
			finalBlowsIndex = source.Index
		case TriggerSoloKills:
			soloKillsIndex = source.Index
		case TriggerKills:
			killsIndex = source.Index
		case TriggerLosses:
			lossesIndex = source.Index
		}
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO achievement_rebuild_counts (source_index, character_id, cnt)
		WITH totals AS MATERIALIZED (
			SELECT entity_id AS character_id,
			       sum(final_blows)::int AS final_blows,
			       sum(solo_kills)::int AS solo_kills,
			       sum(kills)::int AS kills,
			       sum(losses)::int AS losses
			FROM stats
			WHERE entity_type = 0 AND period_type = 2
			GROUP BY entity_id
		)
		SELECT value.source_index, totals.character_id, value.cnt
		FROM totals
		CROSS JOIN LATERAL (VALUES
			($1::int, totals.final_blows),
			($2::int, totals.solo_kills),
			($3::int, totals.kills),
			($4::int, totals.losses)
		) value(source_index, cnt)
		WHERE value.source_index <> 0 AND value.cnt > 0`,
		finalBlowsIndex, soloKillsIndex, killsIndex, lossesIndex)
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

func loadShipKillCounts(ctx context.Context, tx pgx.Tx, _ []rebuildSource) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO achievement_rebuild_counts (source_index, character_id, cnt)
		WITH relevant_kills AS MATERIALIZED (
			SELECT source.source_index, k.killmail_id, k.killmail_time
			FROM killmails k
			JOIN achievement_rebuild_ship_groups source
			  ON source.group_id = k.victim_ship_group_id
			WHERE source.is_loss = false
		)
		SELECT source.source_index, a.character_id, count(*)::int
			FROM relevant_kills source
			JOIN killmail_attackers a
			  ON a.killmail_id = source.killmail_id
			 AND a.killmail_time = source.killmail_time
			WHERE a.character_id IS NOT NULL
			GROUP BY source.source_index, a.character_id`)
	return err
}

func loadShipLossCounts(ctx context.Context, tx pgx.Tx, _ []rebuildSource) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO achievement_rebuild_counts (source_index, character_id, cnt)
		WITH relevant_losses AS MATERIALIZED (
			SELECT source.source_index, k.victim_character_id
			FROM killmails k
			JOIN achievement_rebuild_ship_groups source ON source.group_id = k.victim_ship_group_id
			WHERE victim_character_id IS NOT NULL
			  AND source.is_loss = true
		)
		SELECT source_index, victim_character_id, count(*)::int
			FROM relevant_losses
			GROUP BY source_index, victim_character_id`)
	return err
}

func loadSpecialCounts(ctx context.Context, tx pgx.Tx, sources []rebuildSource) error {
	for _, source := range sources {
		var query string
		var args []any
		switch source.Trigger {
		case TriggerKillsByRegion:
			regionWhere := "k.region_id = $2"
			if source.RegionID == -1 {
				regionWhere = "k.region_id >= 11000000 AND k.region_id < 12000000"
			}
			query = `SELECT a.character_id, count(*)::int AS cnt
				FROM killmail_attackers a JOIN killmails k USING (killmail_id, killmail_time)
				WHERE a.character_id IS NOT NULL AND a.final_blow AND NOT k.is_npc AND ` + regionWhere + `
				GROUP BY a.character_id`
			args = []any{source.Index, source.RegionID}
			if source.RegionID == -1 {
				args = []any{source.Index}
			}
		case TriggerKillsBySystem:
			query = `SELECT a.character_id, count(*)::int AS cnt
				FROM killmail_attackers a JOIN killmails k USING (killmail_id, killmail_time)
				WHERE a.character_id IS NOT NULL AND a.final_blow AND NOT k.is_npc AND k.solar_system_id = $2
				GROUP BY a.character_id`
			args = []any{source.Index, source.SystemID}
		case TriggerLossesByRegion:
			query = `SELECT victim_character_id AS character_id, count(*)::int AS cnt FROM killmails
				WHERE victim_character_id IS NOT NULL AND region_id = $2 GROUP BY victim_character_id`
			args = []any{source.Index, source.RegionID}
		case TriggerLossesBySystem:
			query = `SELECT victim_character_id AS character_id, count(*)::int AS cnt FROM killmails
				WHERE victim_character_id IS NOT NULL AND solar_system_id = $2 GROUP BY victim_character_id`
			args = []any{source.Index, source.SystemID}
		case TriggerConcorded, TriggerKilledByCorp:
			query = `SELECT k.victim_character_id AS character_id, count(DISTINCT k.killmail_id)::int AS cnt
				FROM killmail_attackers a JOIN killmails k USING (killmail_id, killmail_time)
				WHERE k.victim_character_id IS NOT NULL AND a.corporation_id = $2
				GROUP BY k.victim_character_id`
			args = []any{source.Index, source.CorporationID}
		case TriggerKilledCorp:
			query = `SELECT a.character_id, count(*)::int AS cnt
				FROM killmail_attackers a JOIN killmails k USING (killmail_id, killmail_time)
				WHERE a.character_id IS NOT NULL AND a.final_blow AND k.victim_corporation_id = $2
				GROUP BY a.character_id`
			args = []any{source.Index, source.CorporationID}
		case TriggerTournament:
			query = `SELECT character_id, count(DISTINCT killmail_id)::int AS cnt FROM (
				SELECT a.character_id, k.killmail_id
				FROM killmail_attackers a JOIN killmails k USING (killmail_id, killmail_time)
				WHERE a.character_id IS NOT NULL AND k.region_id = $2
				UNION ALL
				SELECT victim_character_id, killmail_id FROM killmails
				WHERE victim_character_id IS NOT NULL AND region_id = $2
			) participants GROUP BY character_id`
			args = []any{source.Index, source.RegionID}
		case TriggerAwox:
			query = `SELECT a.character_id, count(*)::int AS cnt
				FROM killmail_attackers a JOIN killmails k USING (killmail_id, killmail_time)
				WHERE a.character_id IS NOT NULL AND a.final_blow
				  AND ((a.corporation_id IS NOT NULL AND a.corporation_id = k.victim_corporation_id)
				    OR (a.alliance_id IS NOT NULL AND a.alliance_id = k.victim_alliance_id))
				GROUP BY a.character_id`
			args = []any{source.Index}
		case TriggerAwoxed:
			query = `SELECT k.victim_character_id AS character_id, count(DISTINCT k.killmail_id)::int AS cnt
				FROM killmail_attackers a JOIN killmails k USING (killmail_id, killmail_time)
				WHERE k.victim_character_id IS NOT NULL
				  AND ((a.corporation_id IS NOT NULL AND a.corporation_id = k.victim_corporation_id)
				    OR (a.alliance_id IS NOT NULL AND a.alliance_id = k.victim_alliance_id))
				GROUP BY k.victim_character_id`
			args = []any{source.Index}
		case TriggerGank:
			query = `SELECT a.character_id, count(*)::int AS cnt
				FROM killmail_attackers a
				JOIN killmails k USING (killmail_id, killmail_time)
				JOIN solar_systems s ON s.solar_system_id = k.solar_system_id
				WHERE a.character_id IS NOT NULL AND NOT k.is_npc
				  AND s.security >= 0.5 AND a.security_status < -5
				GROUP BY a.character_id`
			args = []any{source.Index}
		default:
			return fmt.Errorf("unsupported special trigger %q", source.Trigger)
		}

		_, err := tx.Exec(ctx, `INSERT INTO achievement_rebuild_counts (source_index, character_id, cnt)
			SELECT $1::int, counted.character_id, counted.cnt FROM (`+query+`) counted
			WHERE counted.cnt > 0`, args...)
		if err != nil {
			return fmt.Errorf("%s: %w", source.Trigger, err)
		}
	}
	return nil
}

func createShipSourceMapping(
	ctx context.Context,
	tx pgx.Tx,
	kills, losses []rebuildSource,
) error {
	if _, err := tx.Exec(ctx, `CREATE TEMP TABLE achievement_rebuild_ship_groups (
		source_index integer NOT NULL,
		group_id integer NOT NULL,
		is_loss boolean NOT NULL,
		PRIMARY KEY (source_index, group_id)
	) ON COMMIT DROP`); err != nil {
		return err
	}
	rows := make([][]any, 0, (len(kills)+len(losses))*2)
	for _, family := range []struct {
		sources []rebuildSource
		isLoss  bool
	}{{kills, false}, {losses, true}} {
		for _, source := range family.sources {
			for _, groupID := range source.GroupIDs {
				rows = append(rows, []any{source.Index, groupID, family.isLoss})
			}
		}
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"achievement_rebuild_ship_groups"},
		[]string{"source_index", "group_id", "is_loss"}, pgx.CopyFromRows(rows)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `CREATE INDEX ON achievement_rebuild_ship_groups (group_id)`); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `ANALYZE achievement_rebuild_ship_groups`)
	return err
}
