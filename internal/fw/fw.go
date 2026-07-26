// Package fw imports faction warfare state from ESI.
//
// Two datasets with different shapes: system occupancy, which is current state
// plus a history of who took what from whom, and faction statistics, which are
// a snapshot with no history at all.
//
// The occupancy importer is written the way it is because of a production bug
// it exists to avoid. The TypeScript FwUpdateCronJob builds its upsert as
// `set: { owner_faction_id: fwSystems.owner_faction_id, ... }`, which renders
// as `SET owner_faction_id = fw_systems.owner_faction_id` — every column
// assigned to itself. The row is touched and nothing changes, so the current
// state never advances, and every run therefore re-detects flips that happened
// months ago and appends them to the history again. Production shows the
// result: fw_systems.updated_at frozen at 2026-04-11 while fw_system_history
// holds 88,376 rows describing 53 actual transitions, one of them recorded
// 5,079 times, still growing at about 1,400 rows a day.
//
// This is the same bug as SovereigntyCronJob's, and the same fix: compare
// against what is stored, write only what moved, and prove it with a test that
// applies one snapshot twice.
package fw

import (
	"context"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/esi"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Result reports one import.
type Result struct {
	// Seen is how many systems ESI described.
	Seen int64 `json:"seen"`
	// Rows is how many current-state rows actually changed.
	Rows int64 `json:"rows"`
	// Flips is how many genuine occupier changes were recorded.
	Flips int64 `json:"flips"`
}

// system is one row of stored occupancy.
type system struct {
	owner     int32
	occupier  int32
	contested string
	points    int32
	threshold int32
}

// ImportSystems refreshes faction warfare occupancy.
func ImportSystems(ctx context.Context, pool *pgxpool.Pool, client *esi.Client) (Result, error) {
	var out Result

	res, err := esi.FetchFwSystems(ctx, client)
	if err != nil {
		return out, err
	}
	if !res.OK() || res.Data == nil {
		return out, fmt.Errorf("ESI returned %d for faction warfare systems", res.Status)
	}
	entries := *res.Data
	out.Seen = int64(len(entries))

	stored, err := loadSystems(ctx, pool)
	if err != nil {
		return out, err
	}

	now := time.Now().UTC()
	tx, err := pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	for _, e := range entries {
		if e.SolarSystemID == 0 {
			continue
		}

		prev, known := stored[e.SolarSystemID]
		unchanged := known &&
			prev.owner == e.OwnerFactionID &&
			prev.occupier == e.OccupierFactionID &&
			prev.contested == e.Contested &&
			prev.points == e.VictoryPoints &&
			prev.threshold == e.VictoryPointsThreshold

		// Victory points move constantly, so most runs do change something and
		// this is not the same no-op-heavy workload sovereignty has. The check
		// still matters: it keeps updated_at meaningful and, more importantly,
		// it is what the flip detection below is compared against.
		if unchanged {
			continue
		}

		if _, err := tx.Exec(ctx, `
            INSERT INTO fw_systems (
                solar_system_id, owner_faction_id, occupier_faction_id,
                contested, victory_points, victory_points_threshold, updated_at
            ) VALUES ($1,$2,$3,$4,$5,$6,$7)
            ON CONFLICT (solar_system_id) DO UPDATE SET
                owner_faction_id = EXCLUDED.owner_faction_id,
                occupier_faction_id = EXCLUDED.occupier_faction_id,
                contested = EXCLUDED.contested,
                victory_points = EXCLUDED.victory_points,
                victory_points_threshold = EXCLUDED.victory_points_threshold,
                updated_at = EXCLUDED.updated_at`,
			// Every column on both tables is NOT NULL, so the zero-means-absent
			// convention used elsewhere does not apply — an unoccupied system
			// genuinely stores 0 rather than NULL, and substituting NULL here
			// would fail the insert outright.
			e.SolarSystemID, e.OwnerFactionID, e.OccupierFactionID,
			e.Contested, e.VictoryPoints, e.VictoryPointsThreshold, now); err != nil {
			return out, fmt.Errorf("upsert fw system %d: %w", e.SolarSystemID, err)
		}
		out.Rows++

		// A flip is only a flip against a system we already knew about. The
		// first time a system is seen there is nothing it changed from, and
		// recording that as a transition from NULL would put 160 phantom events
		// in the history on a fresh install.
		if known && prev.occupier != e.OccupierFactionID {
			if _, err := tx.Exec(ctx, `
                INSERT INTO fw_system_history (
                    solar_system_id, old_occupier_faction_id, new_occupier_faction_id, flipped_at
                ) VALUES ($1,$2,$3,$4)`,
				e.SolarSystemID, prev.occupier, e.OccupierFactionID, now); err != nil {
				return out, fmt.Errorf("record fw flip for %d: %w", e.SolarSystemID, err)
			}
			out.Flips++
		}
	}

	return out, tx.Commit(ctx)
}

// loadSystems reads the stored occupancy.
func loadSystems(ctx context.Context, pool *pgxpool.Pool) (map[int32]system, error) {
	rows, err := pool.Query(ctx, `
        SELECT solar_system_id, coalesce(owner_faction_id, 0), coalesce(occupier_faction_id, 0),
               coalesce(contested, ''), coalesce(victory_points, 0), coalesce(victory_points_threshold, 0)
        FROM fw_systems`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int32]system, 256)
	for rows.Next() {
		var id int32
		var s system
		if err := rows.Scan(&id, &s.owner, &s.occupier, &s.contested, &s.points, &s.threshold); err != nil {
			return nil, err
		}
		out[id] = s
	}
	return out, rows.Err()
}

// PurgeHistoryDuplicates removes the artefacts the TypeScript cron left behind.
//
// Consecutive identical transitions for one system are not events — they are
// the same flip re-detected because the current-state table never advanced.
// Only the first of each run survives.
//
// Offered as a maintenance operation rather than run automatically: it deletes
// tens of thousands of rows, and that should be somebody's deliberate decision.
func PurgeHistoryDuplicates(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	tag, err := pool.Exec(ctx, `
        DELETE FROM fw_system_history h
        USING (
            SELECT id,
                   lag(new_occupier_faction_id) OVER w AS prev_new,
                   lag(old_occupier_faction_id) OVER w AS prev_old
            FROM fw_system_history
            WINDOW w AS (PARTITION BY solar_system_id ORDER BY flipped_at, id)
        ) d
        WHERE h.id = d.id
          AND h.new_occupier_faction_id IS NOT DISTINCT FROM d.prev_new
          AND h.old_occupier_faction_id IS NOT DISTINCT FROM d.prev_old`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
