package sde

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// solar_system_jumps is the system adjacency graph, derived from stargates.
//
// The archive has no jump list. Each stargate record names its own system and,
// under "destination", the system on the other side — so one pass over
// mapStargates produces every edge. Both directions appear naturally, because
// each end of a gate pair is its own record.
//
// The constellation and region columns are denormalised onto the edge so that
// "is this jump a regional border" is answerable without two joins, which is the
// common question when drawing a route.

// ImportSystemJumps builds the adjacency table.
func ImportSystemJumps(ctx context.Context, pool *pgxpool.Pool, src *Source) (LoadResult, error) {
	res := LoadResult{Table: "solar_system_jumps", Member: "mapStargates"}
	start := time.Now()

	if !src.Has("mapStargates") {
		res.Missing = true
		res.Elapsed = time.Since(start).Round(time.Millisecond).String()
		return res, nil
	}

	// Gate ID -> owning system, so a destination expressed as a stargate ID can
	// be resolved. Records name the destination system directly, but the map
	// also lets a gate-only destination resolve.
	gateSystem := make(map[int32]int32, 14000)
	if err := src.Stream(ctx, "mapStargates", func(r Row) error {
		id, ok := r.Key()
		if !ok {
			return nil
		}
		if sys := r.Int("solarSystemID"); sys != nil {
			gateSystem[int32(id)] = *sys
		}
		return nil
	}); err != nil {
		return res, err
	}

	// System -> its constellation and region.
	type place struct{ constellation, region *int32 }
	systems := make(map[int32]place, 8600)
	rows, err := pool.Query(ctx, `SELECT solar_system_id, constellation_id, region_id FROM solar_systems`)
	if err != nil {
		return res, err
	}
	for rows.Next() {
		var id int32
		var p place
		if err := rows.Scan(&id, &p.constellation, &p.region); err != nil {
			rows.Close()
			return res, err
		}
		systems[id] = p
	}
	rows.Close()
	if rows.Err() != nil {
		return res, rows.Err()
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return res, err
	}
	defer conn.Release()

	const staging = "sde_staging_solar_system_jumps"
	if _, err := conn.Exec(ctx, fmt.Sprintf(
		`CREATE TEMP TABLE %s (LIKE public.solar_system_jumps INCLUDING DEFAULTS) ON COMMIT PRESERVE ROWS`,
		staging)); err != nil {
		return res, fmt.Errorf("create staging: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(context.WithoutCancel(ctx), "DROP TABLE IF EXISTS "+staging)
	}()

	columns := []string{
		"from_solar_system_id", "to_solar_system_id",
		"from_constellation_id", "to_constellation_id",
		"from_region_id", "to_region_id",
	}
	w := &stagingWriter{ctx: ctx, conn: conn.Conn(), table: staging, columns: columns}

	// Deduplicate on the UNORDERED pair. Every gate has a counterpart at the far
	// end, so iterating gates yields each connection twice; production stores it
	// once, keeping whichever direction was seen first (6,659 rows read low-to-
	// high, 330 the other way — an artefact of archive order, not a rule).
	//
	// Consequence worth knowing for anything that reads this table: a query
	// filtering only on from_solar_system_id will miss roughly half of a
	// system's neighbours. The edge is undirected and both columns must be
	// checked.
	seen := make(map[[2]int32]bool, 14000)
	unordered := func(a, b int32) [2]int32 {
		if a > b {
			return [2]int32{b, a}
		}
		return [2]int32{a, b}
	}

	if err := src.Stream(ctx, "mapStargates", func(r Row) error {
		res.Read++

		from := r.Int("solarSystemID")
		if from == nil {
			return nil
		}

		dest := r.Map("destination")
		if dest == nil {
			return nil
		}
		d := Row(dest)
		to := d.Int("solarSystemID")
		if to == nil {
			// Fall back to resolving the far gate's owner.
			if gate := d.Int("stargateID"); gate != nil {
				if sys, ok := gateSystem[*gate]; ok {
					to = &sys
				}
			}
		}
		if to == nil || *to == *from {
			return nil
		}

		key := unordered(*from, *to)
		if seen[key] {
			return nil
		}
		seen[key] = true

		f, t := systems[*from], systems[*to]
		return w.add([]any{*from, *to, f.constellation, t.constellation, f.region, t.region})
	}); err != nil {
		return res, err
	}

	if err := w.flush(); err != nil {
		return res, err
	}
	res.Written = w.written

	tbl := Table{
		Name:        "solar_system_jumps",
		PK:          []string{"from_solar_system_id", "to_solar_system_id"},
		Columns:     columns,
		PruneAbsent: true,
	}

	// This table is a pure function of the archive: every run computes the
	// complete edge set, so anything not in staging is stale and must go. Most
	// SDE tables only ever upsert, which is why production still carries rows
	// CCP removed years ago — but here that would be actively wrong, because a
	// changed dedup rule or a removed gate leaves a phantom connection in the
	// map. Merge and prune together so the table is never inconsistent.
	tx, err := conn.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	if _, err := tx.Exec(ctx, mergeSQL(tbl, staging)); err != nil {
		return res, fmt.Errorf("merge jumps: %w", err)
	}
	pruned, err := tx.Exec(ctx, fmt.Sprintf(`
        DELETE FROM public.solar_system_jumps j
        WHERE NOT EXISTS (
            SELECT 1 FROM %s s
            WHERE s.from_solar_system_id = j.from_solar_system_id
              AND s.to_solar_system_id = j.to_solar_system_id
        )`, staging))
	if err != nil {
		return res, fmt.Errorf("prune jumps: %w", err)
	}
	res.Pruned = pruned.RowsAffected()

	if err := tx.Commit(ctx); err != nil {
		return res, err
	}

	res.Duration = time.Since(start)
	res.Elapsed = res.Duration.Round(time.Millisecond).String()
	return res, nil
}
