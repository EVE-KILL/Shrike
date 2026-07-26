package sde

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The celestials table is a flat index of every named thing in space — regions,
// constellations, systems, stars, planets, moons, belts, stargates, stations —
// ~496 k rows drawn from nine sources into one table. It is what turns a
// killmail's location ID into text.
//
// It cannot use the declarative Table machinery for two reasons: several
// archive members feed one table, and the names are hierarchical. A moon is
// named from its planet, a station from whatever it orbits, so the passes must
// run in dependency order and carry names forward.
//
// Group IDs are fixed by CCP and verified against production:
//
//	 3  region            114     "Fountain"
//	 4  constellation   1,184     "0P-VMU"
//	 5  system          8,490     "Jita"
//	 6  star            8,089     "Jita - Star"
//	 7  planet         68,407     "Jita IV"
//	 8  moon          344,457     "Jita IV - Moon 4"
//	 9  asteroid belt  40,928     "Jita IV - Asteroid Belt 1"
//	10  stargate       13,978     "Stargate (Perimeter)"
//	15  station         5,210     "Jita IV - Moon 4 - Caldari Navy Assembly Plant"
//	995 secondary sun   1,038     "Unknown Anomaly"
const (
	groupRegion        = 3
	groupConstellation = 4
	groupSystem        = 5
	groupStar          = 6
	groupPlanet        = 7
	groupMoon          = 8
	groupAsteroidBelt  = 9
	groupStargate      = 10
	groupStation       = 15
	groupSecondarySun  = 995
)

// celestialColumns is the COPY column list, and the order every pass must use.
var celestialColumns = []string{
	"item_id", "item_name", "type_id", "group_id", "solar_system_id",
	"constellation_id", "region_id", "orbit_id", "x", "y", "z", "radius",
	"security", "celestial_index", "orbit_index",
}

// romanNumerals covers planet indices. EVE systems top out well below 30
// planets; anything beyond falls back to the decimal index rather than
// producing a wrong name.
var romanNumerals = []string{
	"", "I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X",
	"XI", "XII", "XIII", "XIV", "XV", "XVI", "XVII", "XVIII", "XIX", "XX",
	"XXI", "XXII", "XXIII", "XXIV", "XXV", "XXVI", "XXVII", "XXVIII", "XXIX", "XXX",
}

func toRoman(n int32) string {
	if n > 0 && int(n) < len(romanNumerals) {
		return romanNumerals[n]
	}
	return fmt.Sprintf("%d", n)
}

// systemInfo is the per-system context every celestial in that system inherits.
type systemInfo struct {
	name            string
	constellationID *int32
	regionID        *int32
	security        *float64
}

// celestialLookups holds everything the naming passes need, read from Postgres
// rather than re-streamed from the archive: those tables are already imported by
// the time this runs, and 8.5 k systems is far cheaper to query than to re-parse
// a 5 MB member.
type celestialLookups struct {
	systems      map[int32]systemInfo
	typeGroup    map[int32]int32 // type_id -> group_id, for the real group of a planet/moon
	corpName     map[int32]string
	opName       map[int32]string
	stationOrbit map[int32]int32 // station_id -> orbit_id
	// names accumulates planet and moon names as they are emitted, because
	// moons are named from planets and stations from either.
	names map[int32]string
}

func loadCelestialLookups(ctx context.Context, pool *pgxpool.Pool) (*celestialLookups, error) {
	l := &celestialLookups{
		systems:   make(map[int32]systemInfo, 8600),
		typeGroup: make(map[int32]int32, 54000),
		corpName:  make(map[int32]string, 300),
		opName:    make(map[int32]string, 80),
		names:     make(map[int32]string, 420000),
	}

	rows, err := pool.Query(ctx, `
        SELECT solar_system_id, coalesce(system_name,''), constellation_id, region_id, security
        FROM solar_systems`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int32
		var si systemInfo
		if err := rows.Scan(&id, &si.name, &si.constellationID, &si.regionID, &si.security); err != nil {
			rows.Close()
			return nil, err
		}
		l.systems[id] = si
	}
	rows.Close()
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if err := scanPairs(ctx, pool,
		`SELECT type_id, group_id FROM inv_types WHERE group_id IS NOT NULL`,
		func(k, v int32) { l.typeGroup[k] = v }); err != nil {
		return nil, err
	}

	if err := scanNames(ctx, pool,
		`SELECT corporation_id, coalesce(name,'') FROM npc_corporations`,
		func(k int32, v string) { l.corpName[k] = v }); err != nil {
		return nil, err
	}
	if err := scanNames(ctx, pool,
		`SELECT operation_id, coalesce(operation_name,'') FROM station_operations`,
		func(k int32, v string) { l.opName[k] = v }); err != nil {
		return nil, err
	}
	return l, nil
}

func scanPairs(ctx context.Context, pool *pgxpool.Pool, sql string, fn func(k, v int32)) error {
	rows, err := pool.Query(ctx, sql)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var k, v int32
		if err := rows.Scan(&k, &v); err != nil {
			return err
		}
		fn(k, v)
	}
	return rows.Err()
}

func scanNames(ctx context.Context, pool *pgxpool.Pool, sql string, fn func(k int32, v string)) error {
	rows, err := pool.Query(ctx, sql)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var k int32
		var v string
		if err := rows.Scan(&k, &v); err != nil {
			return err
		}
		fn(k, v)
	}
	return rows.Err()
}

// ImportCelestials runs every pass and merges the result.
func ImportCelestials(ctx context.Context, pool *pgxpool.Pool, src *Source) (LoadResult, error) {
	res := LoadResult{Table: "celestials", Member: "map*"}
	start := time.Now()

	lookups, err := loadCelestialLookups(ctx, pool)
	if err != nil {
		return res, fmt.Errorf("load lookups: %w", err)
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return res, err
	}
	defer conn.Release()

	const staging = "sde_staging_celestials"
	if _, err := conn.Exec(ctx, fmt.Sprintf(
		`CREATE TEMP TABLE %s (LIKE public.celestials INCLUDING DEFAULTS) ON COMMIT PRESERVE ROWS`,
		staging)); err != nil {
		return res, fmt.Errorf("create staging: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(context.WithoutCancel(ctx), "DROP TABLE IF EXISTS "+staging)
	}()

	w := &stagingWriter{ctx: ctx, conn: conn.Conn(), table: staging, columns: celestialColumns}

	// Passes run in dependency order: planets before moons before stations.
	passes := []struct {
		name string
		run  func() error
	}{
		{"map hierarchy", func() error { return celestialsFromDatabase(ctx, pool, w) }},
		{"stars", func() error { return celestialsFromMember(ctx, src, w, lookups, "mapStars", groupStar) }},
		{"planets", func() error { return celestialsFromMember(ctx, src, w, lookups, "mapPlanets", groupPlanet) }},
		{"moons", func() error { return celestialsFromMember(ctx, src, w, lookups, "mapMoons", groupMoon) }},
		{"belts", func() error { return celestialsFromMember(ctx, src, w, lookups, "mapAsteroidBelts", groupAsteroidBelt) }},
		{"stargates", func() error { return celestialsFromMember(ctx, src, w, lookups, "mapStargates", groupStargate) }},
		{"secondary suns", func() error { return celestialsFromMember(ctx, src, w, lookups, "mapSecondarySuns", groupSecondarySun) }},
		{"stations", func() error { return celestialsFromMember(ctx, src, w, lookups, "npcStations", groupStation) }},
	}
	for _, p := range passes {
		if err := p.run(); err != nil {
			return res, fmt.Errorf("celestials/%s: %w", p.name, err)
		}
	}
	if err := w.flush(); err != nil {
		return res, err
	}
	res.Read = w.written
	res.Written = w.written

	tbl := Table{
		Name:        "celestials",
		PK:          []string{"item_id"},
		Columns:     celestialColumns,
		PruneAbsent: true,
	}

	// Celestials are a complete projection of the current map archive plus the
	// region/constellation/system tables loaded from that same build. A removed
	// moon, belt, gate, or station must not survive as a phantom map object.
	tx, err := conn.Begin(ctx)
	if err != nil {
		return res, fmt.Errorf("begin celestial snapshot merge: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	if _, err := tx.Exec(ctx, mergeSQL(tbl, staging)); err != nil {
		return res, fmt.Errorf("merge celestials: %w", err)
	}
	pruned, err := tx.Exec(ctx, pruneSQL(tbl, staging))
	if err != nil {
		return res, fmt.Errorf("prune celestials: %w", err)
	}
	res.Pruned = pruned.RowsAffected()
	if err := tx.Commit(ctx); err != nil {
		return res, fmt.Errorf("commit celestial snapshot merge: %w", err)
	}

	res.Duration = time.Since(start)
	res.Elapsed = res.Duration.Round(time.Millisecond).String()
	return res, nil
}

// celestialsFromDatabase emits regions, constellations and systems. These are
// already in their own tables, so they are copied across rather than re-parsed —
// which also guarantees the names match exactly.
func celestialsFromDatabase(ctx context.Context, pool *pgxpool.Pool, w *stagingWriter) error {
	// These rows have no type of their own, so production stores the group id in
	// type_id as well — a region is type 3, a constellation type 4, a system
	// type 5. Each also carries whichever containers sit above it: a
	// constellation knows its region, a system knows both.
	type src struct {
		sql   string
		group int32
	}
	for _, s := range []src{
		{`SELECT region_id, name, NULL::int, NULL::int, center_x, center_y, center_z,
		         NULL::float8, NULL::float8 FROM regions`, groupRegion},
		{`SELECT constellation_id, constellation_name, NULL::int, region_id, x, y, z,
		         NULL::float8, NULL::float8 FROM constellations`, groupConstellation},
		{`SELECT solar_system_id, system_name, constellation_id, region_id, x, y, z,
		         radius, security FROM solar_systems`, groupSystem},
	} {
		rows, err := pool.Query(ctx, s.sql)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id int32
			var name *string
			var constellationID, regionID *int32
			var x, y, z, radius, security *float64
			if err := rows.Scan(&id, &name, &constellationID, &regionID,
				&x, &y, &z, &radius, &security); err != nil {
				rows.Close()
				return err
			}
			if err := w.add([]any{
				id, name, s.group, s.group, nil,
				constellationID, regionID, nil,
				x, y, z, radius, security, nil, nil,
			}); err != nil {
				rows.Close()
				return err
			}
		}
		rows.Close()
		if rows.Err() != nil {
			return rows.Err()
		}
	}
	return nil
}

// celestialsFromMember handles the archive-sourced passes. The naming rule is
// the only thing that varies between them, so they share one walk.
func celestialsFromMember(ctx context.Context, src *Source, w *stagingWriter, l *celestialLookups, member string, group int32) error {
	if !src.Has(member) {
		// CCP has dropped members before; a missing one should not abort the
		// whole import of a table that is mostly present.
		return nil
	}

	return src.Stream(ctx, member, func(r Row) error {
		id, ok := r.Key()
		if !ok {
			return nil
		}
		itemID := int32(id)

		sysID := r.Int("solarSystemID")
		var si systemInfo
		if sysID != nil {
			si = l.systems[*sysID]
		}

		typeID := r.Int("typeID")
		celIdx := r.Int("celestialIndex")
		orbitIdx := r.Int("orbitIndex")
		orbitID := r.Int("orbitID")
		x, y, z := xyz(r, "position")
		radius := r.Float("radius")

		// The real group comes from the type where one exists — a planet's group
		// distinguishes gas giants from barren worlds. The constant is the
		// fallback for members whose types are not in inv_types.
		effGroup := group
		if typeID != nil {
			if g, ok := l.typeGroup[*typeID]; ok {
				effGroup = g
			}
		}

		var name *string
		switch group {
		case groupStar:
			if si.name != "" {
				n := si.name + " - Star"
				name = &n
			}
			// Stars sit at the system centre and carry no meaningful position.
			zero := 0.0
			x, y, z = &zero, &zero, &zero
			effGroup = groupStar

		case groupPlanet:
			if si.name != "" && celIdx != nil {
				n := si.name + " " + toRoman(*celIdx)
				name = &n
				l.names[itemID] = n
			}

		case groupMoon:
			if orbitID != nil && orbitIdx != nil {
				if parent, ok := l.names[*orbitID]; ok {
					n := fmt.Sprintf("%s - Moon %d", parent, *orbitIdx)
					name = &n
					// Recorded because stations orbit moons as well as planets.
					l.names[itemID] = n
				}
			}

		case groupAsteroidBelt:
			if orbitID != nil && orbitIdx != nil {
				if parent, ok := l.names[*orbitID]; ok {
					n := fmt.Sprintf("%s - Asteroid Belt %d", parent, *orbitIdx)
					name = &n
				}
			}
			// Belts have no meaningful radius; production stores 1.
			one := 1.0
			radius = &one

		case groupStargate:
			// Named for where it goes, not where it is.
			if dest := r.Map("destination"); dest != nil {
				if destSys := Row(dest).Int("solarSystemID"); destSys != nil {
					if ds, ok := l.systems[*destSys]; ok && ds.name != "" {
						n := fmt.Sprintf("Stargate (%s)", ds.name)
						name = &n
					}
				}
			}
			effGroup = groupStargate

		case groupSecondarySun:
			n := "Unknown Anomaly"
			name = &n
			effGroup = groupSecondarySun
			zero := 0.0
			if x == nil {
				x, y, z = &zero, &zero, &zero
			}
			// Not the containing system's security: production records 0 for
			// these, and a secondary sun is an anomaly rather than a place with
			// a security rating of its own.
			si.security = &zero

		case groupStation:
			// "<whatever it orbits> - <corp> <operation>"
			orbitID = r.Int("orbitID")
			if orbitID != nil {
				if parent, ok := l.names[*orbitID]; ok {
					corp := ""
					if owner := r.Int("ownerID"); owner != nil {
						corp = l.corpName[*owner]
					}
					op := ""
					if opID := r.Int("operationID"); opID != nil {
						op = l.opName[*opID]
					}
					if corp != "" || op != "" {
						n := fmt.Sprintf("%s - %s %s", parent, corp, op)
						name = &n
					}
				}
			}
			effGroup = groupStation
			// npcStations carries celestialIndex and orbitIndex, but production
			// leaves both NULL on the celestial row — a station's position in
			// space is expressed by what it orbits, and the indices belong to
			// that parent rather than to the station.
			celIdx, orbitIdx = nil, nil
		}

		return w.add([]any{
			itemID, name, typeID, effGroup, sysID,
			si.constellationID, si.regionID, orbitID,
			x, y, z, radius, si.security, celIdx, orbitIdx,
		})
	})
}

// stagingWriter buffers rows and flushes them with COPY.
//
// Chunked rather than accumulating all ~496 k rows: the moon pass alone is
// 344 k rows of fifteen values, and holding every pass in memory before writing
// would cost hundreds of megabytes for no benefit.
type stagingWriter struct {
	ctx     context.Context
	conn    *pgx.Conn
	table   string
	columns []string

	buf     [][]any
	written int64
}

const stagingFlushAt = 50_000

func (w *stagingWriter) add(row []any) error {
	if len(row) != len(w.columns) {
		return fmt.Errorf("row has %d values for %d columns", len(row), len(w.columns))
	}
	w.buf = append(w.buf, row)
	if len(w.buf) >= stagingFlushAt {
		return w.flush()
	}
	return nil
}

func (w *stagingWriter) flush() error {
	if len(w.buf) == 0 {
		return nil
	}
	n, err := w.conn.CopyFrom(w.ctx, pgx.Identifier{w.table}, w.columns, pgx.CopyFromRows(w.buf))
	if err != nil {
		return fmt.Errorf("copy into %s: %w", w.table, err)
	}
	w.written += n
	w.buf = w.buf[:0]
	return nil
}
