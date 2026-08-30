package api

import (
	"context"
	"maps"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/eve"
)

const universeCacheTTL = 2 * time.Minute

type universeLoader func(
	context.Context,
	Database,
	int64,
) (map[string]any, error)

type universeRoute struct {
	Name      string
	Canonical string
	Alias     string
	Summary   string
	NotFound  string
	Load      universeLoader
}

var universeRoutes = []universeRoute{
	{
		Name: "region", Canonical: "/universe/regions/{id}",
		Alias: "/region/{id}", Summary: "Region page data",
		NotFound: "Region not found", Load: loadUniverseRegion,
	},
	{
		Name: "constellation", Canonical: "/universe/constellations/{id}",
		Alias: "/constellation/{id}", Summary: "Constellation page data",
		NotFound: "Constellation not found", Load: loadUniverseConstellation,
	},
	{
		Name: "system", Canonical: "/universe/systems/{id}",
		Alias: "/system/{id}", Summary: "Solar system page data",
		NotFound: "System not found", Load: loadUniverseSystem,
	},
	{
		Name: "type", Canonical: "/universe/types/{id}",
		Alias: "/item/{id}", Summary: "Inventory type page data",
		NotFound: "Item not found", Load: loadUniverseType,
	},
}

// registerUniverseRoutes exposes the rich aggregates used by entity
// pages. Static reference lists and simple records stay on the existing
// /sde/* operations; duplicating them in the frontend API would create two
// contracts for identical data. The old shipgroup page now redirects to the
// market browser, so it gets no compatibility route; /sde/groups,
// /sde/categories, and /market/* already cover its live replacement.
func registerUniverseRoutes(a huma.API, opts Options) {
	for _, route := range universeRoutes {
		handler := universeEntityHandler(opts, route.NotFound, route.Load)
		handler = routeJSONCache(
			opts,
			universeCacheTTL,
			"public, max-age=30, s-maxage=120, stale-while-revalidate=120",
			handler,
		)
		registerLegacy(a, huma.Operation{
			OperationID: "universe-" + route.Name,
			Method:      http.MethodGet,
			Path:        route.Canonical,
			Summary:     route.Summary,
			Tags:        []string{"universe"},
		}, handler)
		registerLegacy(a, huma.Operation{
			OperationID: route.Name + "-compat",
			Method:      http.MethodGet,
			Path:        route.Alias,
			Summary:     route.Summary,
			Tags:        []string{"universe"},
		}, handler)
	}
	registerUniverseKilllistRoutes(a, opts)
}

func universeEntityHandler(
	opts Options,
	notFound string,
	load universeLoader,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseUniverseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := load(ctx, opts.DB, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if body == nil {
			return legacyPayload{}, apiError(http.StatusNotFound, notFound)
		}
		return jsonPayload(body), nil
	}
}

func parseUniverseID(raw string) (int64, error) {
	id, err := parseID(raw)
	if err != nil || id < 1 || id > pgInt4Max {
		return 0, apiError(http.StatusBadRequest, "Invalid id")
	}
	return id, nil
}

func universeCutoff(days int) string {
	return time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")
}

func nonNilUniverseRows(rows []map[string]any) []map[string]any {
	if rows == nil {
		return []map[string]any{}
	}
	return rows
}

func firstUniverseRow(rows []map[string]any) map[string]any {
	if len(rows) == 0 {
		return nil
	}
	return rows[0]
}

func universeStats(rows []map[string]any) map[string]any {
	if row := firstUniverseRow(rows); row != nil {
		return row
	}
	return map[string]any{
		"kills": int64(0), "total_value": float64(0),
		"npc_kills": int64(0), "pod_kills": int64(0),
	}
}

func loadUniverseRegion(
	ctx context.Context,
	db Database,
	id int64,
) (map[string]any, error) {
	cutoff := universeCutoff(7)
	result, err := queryMapsConcurrent(ctx, db,
		databaseQuery{
			SQL: `
				SELECT r.region_id, r.name, r.description, r.faction_id,
				       f.name AS faction_name,
				       (SELECT COUNT(*)::int FROM constellations c
				        WHERE c.region_id = r.region_id) AS constellation_count,
				       (SELECT COUNT(*)::int FROM solar_systems s
				        WHERE s.region_id = r.region_id) AS system_count
				FROM regions r
				LEFT JOIN factions f ON f.faction_id = r.faction_id
				WHERE r.region_id = $1
				LIMIT 1`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				WITH constellation_sov AS (
					SELECT c.constellation_id, c.constellation_name,
					       COUNT(DISTINCT s.solar_system_id)::int AS system_count,
					       (SELECT sov.alliance_id
					        FROM sovereignty sov
					        JOIN solar_systems ss
					          ON ss.solar_system_id = sov.system_id
					        WHERE ss.constellation_id = c.constellation_id
					          AND sov.alliance_id IS NOT NULL
					        GROUP BY sov.alliance_id
					        ORDER BY COUNT(*) DESC
					        LIMIT 1) AS alliance_id,
					       (SELECT sov.faction_id
					        FROM sovereignty sov
					        JOIN solar_systems ss
					          ON ss.solar_system_id = sov.system_id
					        WHERE ss.constellation_id = c.constellation_id
					          AND sov.faction_id IS NOT NULL
					          AND sov.alliance_id IS NULL
					        GROUP BY sov.faction_id
					        ORDER BY COUNT(*) DESC
					        LIMIT 1) AS faction_id
					FROM constellations c
					LEFT JOIN solar_systems s
					  ON s.constellation_id = c.constellation_id
					WHERE c.region_id = $1
					GROUP BY c.constellation_id, c.constellation_name
				)
				SELECT cs.constellation_id, cs.constellation_name,
				       cs.system_count, cs.alliance_id,
				       a.name AS alliance_name, cs.faction_id,
				       f.name AS faction_name
				FROM constellation_sov cs
				LEFT JOIN alliances a ON a.alliance_id = cs.alliance_id
				LEFT JOIN factions f ON f.faction_id = cs.faction_id
				ORDER BY cs.constellation_name ASC`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				WITH agg AS (
					SELECT COALESCE(SUM(kills), 0)::bigint AS kills,
					       COALESCE(SUM(isk_destroyed), 0) AS total_value,
					       COALESCE(SUM(npc_losses), 0)::bigint AS npc_kills
					FROM stats
					WHERE entity_type = 6 AND entity_id = $1
					  AND period_type = 0 AND period_start >= $2::date
				), pods AS (
					SELECT COUNT(*)::bigint AS pod_kills
					FROM killmails
					WHERE region_id = $1
					  AND killmail_time >= $2::date
					  AND victim_ship_group_id = 29
				)
				SELECT agg.kills, agg.total_value, agg.npc_kills,
				       pods.pod_kills
				FROM agg CROSS JOIN pods`,
			Args: []any{id, cutoff},
		},
		databaseQuery{
			SQL: `
				SELECT sov.alliance_id, a.name AS alliance_name,
				       sov.faction_id, f.name AS faction_name,
				       COUNT(*)::int AS system_count
				FROM sovereignty sov
				JOIN solar_systems ss ON ss.solar_system_id = sov.system_id
				LEFT JOIN alliances a ON a.alliance_id = sov.alliance_id
				LEFT JOIN factions f ON f.faction_id = sov.faction_id
				WHERE ss.region_id = $1
				  AND (sov.alliance_id IS NOT NULL
				       OR sov.faction_id IS NOT NULL)
				GROUP BY sov.alliance_id, a.name, sov.faction_id, f.name
				ORDER BY system_count DESC
				LIMIT 15`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				SELECT ss.solar_system_id, ss.system_name, ss.security,
				       COALESCE(SUM(sd.kills), 0) AS kills,
				       COALESCE(SUM(sd.isk_destroyed), 0) AS total_value
				FROM stats sd
				JOIN solar_systems ss ON ss.solar_system_id = sd.entity_id
				WHERE sd.entity_type = 4 AND sd.period_type = 0
				  AND sd.period_start >= $2::date
				  AND ss.region_id = $1
				GROUP BY ss.solar_system_id, ss.system_name, ss.security
				HAVING SUM(sd.kills) > 0
				ORDER BY kills DESC
				LIMIT 10`,
			Args: []any{id, cutoff},
		},
	)
	if err != nil {
		return nil, err
	}
	region := firstUniverseRow(result[0])
	if region == nil {
		return nil, nil
	}
	return map[string]any{
		"region":          region,
		"constellations":  nonNilUniverseRows(result[1]),
		"stats":           universeStats(result[2]),
		"sovDistribution": nonNilUniverseRows(result[3]),
		"topSystems":      nonNilUniverseRows(result[4]),
	}, nil
}

func loadUniverseConstellation(
	ctx context.Context,
	db Database,
	id int64,
) (map[string]any, error) {
	cutoff := universeCutoff(7)
	result, err := queryMapsConcurrent(ctx, db,
		databaseQuery{
			SQL: `
				SELECT c.constellation_id, c.constellation_name,
				       c.region_id, r.name AS region_name, c.faction_id
				FROM constellations c
				LEFT JOIN regions r ON r.region_id = c.region_id
				WHERE c.constellation_id = $1
				LIMIT 1`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				SELECT ss.solar_system_id, ss.system_name, ss.security,
				       sov.alliance_id, a.name AS alliance_name,
				       sov.faction_id, f.name AS faction_name
				FROM solar_systems ss
				LEFT JOIN sovereignty sov ON sov.system_id = ss.solar_system_id
				LEFT JOIN alliances a ON a.alliance_id = sov.alliance_id
				LEFT JOIN factions f ON f.faction_id = sov.faction_id
				WHERE ss.constellation_id = $1
				ORDER BY ss.system_name ASC`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				SELECT sov.alliance_id, a.name AS alliance_name,
				       sov.faction_id, f.name AS faction_name,
				       COUNT(*)::int AS system_count
				FROM sovereignty sov
				JOIN solar_systems ss ON ss.solar_system_id = sov.system_id
				LEFT JOIN alliances a ON a.alliance_id = sov.alliance_id
				LEFT JOIN factions f ON f.faction_id = sov.faction_id
				WHERE ss.constellation_id = $1
				  AND (sov.alliance_id IS NOT NULL
				       OR sov.faction_id IS NOT NULL)
				GROUP BY sov.alliance_id, a.name, sov.faction_id, f.name
				ORDER BY system_count DESC
				LIMIT 10`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				WITH agg AS (
					SELECT COALESCE(SUM(kills), 0)::bigint AS kills,
					       COALESCE(SUM(isk_destroyed), 0) AS total_value,
					       COALESCE(SUM(npc_losses), 0)::bigint AS npc_kills
					FROM stats
					WHERE entity_type = 5 AND entity_id = $1
					  AND period_type = 0 AND period_start >= $2::date
				), pods AS (
					SELECT COUNT(*)::bigint AS pod_kills
					FROM killmails
					WHERE constellation_id = $1
					  AND killmail_time >= $2::date
					  AND victim_ship_group_id = 29
				)
				SELECT agg.kills, agg.total_value, agg.npc_kills,
				       pods.pod_kills
				FROM agg CROSS JOIN pods`,
			Args: []any{id, cutoff},
		},
	)
	if err != nil {
		return nil, err
	}
	constellation := firstUniverseRow(result[0])
	if constellation == nil {
		return nil, nil
	}
	return map[string]any{
		"constellation":   constellation,
		"systems":         nonNilUniverseRows(result[1]),
		"sovDistribution": nonNilUniverseRows(result[2]),
		"stats":           universeStats(result[3]),
	}, nil
}

func loadUniverseSystem(
	ctx context.Context,
	db Database,
	id int64,
) (map[string]any, error) {
	statsCutoff := universeCutoff(7)
	activityCutoff := time.Now().UTC().Add(-24 * time.Hour)
	result, err := queryMapsConcurrent(ctx, db,
		databaseQuery{
			SQL: `
				SELECT s.solar_system_id, s.system_name, s.security,
				       s.security_class, s.faction_id, s.sun_type_id,
				       sun.name AS sun_type_name, s.border, s.fringe,
				       s.corridor, s.hub, s.international, s.regional,
				       s.constellation_id, c.constellation_name,
				       s.region_id, r.name AS region_name
				FROM solar_systems s
				LEFT JOIN constellations c
				  ON c.constellation_id = s.constellation_id
				LEFT JOIN regions r ON r.region_id = s.region_id
				LEFT JOIN inv_types sun ON sun.type_id = s.sun_type_id
				WHERE s.solar_system_id = $1
				LIMIT 1`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				SELECT c.item_id, c.item_name, c.type_id, c.group_id,
				       t.name AS type_name
				FROM celestials c
				LEFT JOIN inv_types t ON t.type_id = c.type_id
				WHERE c.solar_system_id = $1`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				SELECT s.station_id, s.station_name, s.type_id,
				       s.corporation_id, c.name AS corporation_name,
				       o.operation_name
				FROM stations s
				LEFT JOIN station_operations o
				  ON o.operation_id = s.operation_id
				LEFT JOIN corporations c
				  ON c.corporation_id = s.corporation_id
				WHERE s.solar_system_id = $1`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				SELECT DISTINCT dest_id AS to_solar_system_id,
				       s.system_name, s.security, s.region_id,
				       CASE WHEN s.region_id <> src.region_id
				            THEN true ELSE false END AS is_regional
				FROM (
					SELECT to_solar_system_id AS dest_id
					FROM solar_system_jumps
					WHERE from_solar_system_id = $1
					UNION
					SELECT from_solar_system_id AS dest_id
					FROM solar_system_jumps
					WHERE to_solar_system_id = $1
				) conns
				JOIN solar_systems s ON s.solar_system_id = conns.dest_id
				CROSS JOIN solar_systems src
				WHERE src.solar_system_id = $1
				ORDER BY s.system_name ASC`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				SELECT sov.alliance_id, a.name AS alliance_name,
				       sov.corporation_id, c.name AS corporation_name,
				       sov.faction_id, f.name AS faction_name
				FROM sovereignty sov
				LEFT JOIN alliances a ON a.alliance_id = sov.alliance_id
				LEFT JOIN corporations c
				  ON c.corporation_id = sov.corporation_id
				LEFT JOIN factions f ON f.faction_id = sov.faction_id
				WHERE sov.system_id = $1
				LIMIT 1`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				WITH agg AS (
					SELECT COALESCE(SUM(kills), 0)::bigint AS kills,
					       COALESCE(SUM(isk_destroyed), 0) AS total_value,
					       COALESCE(SUM(npc_losses), 0)::bigint AS npc_kills
					FROM stats
					WHERE entity_type = 4 AND entity_id = $1
					  AND period_type = 0 AND period_start >= $2::date
				), pods AS (
					SELECT COUNT(*)::bigint AS pod_kills
					FROM killmails
					WHERE solar_system_id = $1
					  AND killmail_time >= $2::date
					  AND victim_ship_group_id = 29
				)
				SELECT agg.kills, agg.total_value, agg.npc_kills,
				       pods.pod_kills
				FROM agg CROSS JOIN pods`,
			Args: []any{id, statsCutoff},
		},
		databaseQuery{
			SQL: `
				SELECT h.alliance_id, a.name AS alliance_name,
				       h.corporation_id, c.name AS corporation_name,
				       h.faction_id, f.name AS faction_name,
				       h.date_added AS date
				FROM sovereignty_history h
				LEFT JOIN alliances a ON a.alliance_id = h.alliance_id
				LEFT JOIN corporations c
				  ON c.corporation_id = h.corporation_id
				LEFT JOIN factions f ON f.faction_id = h.faction_id
				WHERE h.system_id = $1
				ORDER BY h.date_added DESC
				LIMIT 20`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				SELECT s.structure_id, s.name, s.owner_id,
				       c.name AS owner_name, s.type_id,
				       t.name AS type_name, s.is_market, s.last_seen
				FROM structures s
				LEFT JOIN corporations c ON c.corporation_id = s.owner_id
				LEFT JOIN inv_types t ON t.type_id = s.type_id
				WHERE s.solar_system_id = $1`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				SELECT timestamp, ship_kills, npc_kills, pod_kills, ship_jumps
				FROM system_activity
				WHERE system_id = $1 AND timestamp >= $2
				ORDER BY timestamp DESC`,
			Args: []any{id, activityCutoff},
		},
	)
	if err != nil {
		return nil, err
	}
	system := firstUniverseRow(result[0])
	if system == nil {
		return nil, nil
	}
	counts, celestials := buildUniverseCelestials(result[1])
	return map[string]any{
		"system":             system,
		"celestials":         counts,
		"celestialList":      celestials,
		"stations":           nonNilUniverseRows(result[2]),
		"connections":        nonNilUniverseRows(result[3]),
		"sovereignty":        firstUniverseRow(result[4]),
		"stats":              universeStats(result[5]),
		"sovereigntyHistory": nonNilUniverseRows(result[6]),
		"structures":         nonNilUniverseRows(result[7]),
		"activity":           buildUniverseSystemActivity(result[8]),
	}, nil
}

func buildUniverseCelestials(
	rows []map[string]any,
) (map[string]int, []map[string]any) {
	counts := map[string]int{
		"stars": 0, "planets": 0, "moons": 0, "belts": 0, "stargates": 0,
	}
	list := make([]map[string]any, 0)
	for _, row := range rows {
		groupID, ok := int64Value(row["group_id"])
		if !ok {
			continue
		}
		category := ""
		switch groupID {
		case 6:
			counts["stars"]++
			category = "star"
		case 7:
			counts["planets"]++
			category = "planet"
		case 8:
			counts["moons"]++
			category = "moon"
		case 9:
			counts["belts"]++
			category = "belt"
		case 10:
			counts["stargates"]++
		}
		if category == "" {
			continue
		}
		item := make(map[string]any, len(row)+1)
		maps.Copy(item, row)
		item["category"] = category
		list = append(list, item)
	}
	return counts, list
}

func buildUniverseSystemActivity(rows []map[string]any) map[string]any {
	rows = nonNilUniverseRows(rows)
	if len(rows) == 0 {
		return map[string]any{
			"latest": nil, "summary_24h": nil, "history": rows,
		}
	}

	latestRow := rows[0]
	latest := map[string]any{
		"ship_kills": universeIntegerOrZero(latestRow["ship_kills"]),
		"npc_kills":  universeIntegerOrZero(latestRow["npc_kills"]),
		"pod_kills":  universeIntegerOrZero(latestRow["pod_kills"]),
		"ship_jumps": universeIntegerOrZero(latestRow["ship_jumps"]),
		"timestamp":  latestRow["timestamp"],
	}
	summary := map[string]int64{
		"ship_kills": 0, "npc_kills": 0, "pod_kills": 0, "ship_jumps": 0,
	}
	for _, row := range rows {
		for key := range summary {
			if value, ok := int64Value(row[key]); ok {
				summary[key] += value
			}
		}
	}
	return map[string]any{
		"latest": latest, "summary_24h": summary, "history": rows,
	}
}

func universeIntegerOrZero(value any) any {
	if value == nil {
		return int64(0)
	}
	if _, ok := int64Value(value); ok {
		return value
	}
	return int64(0)
}

var universeShipAttributeNames = map[int64]string{
	263: "Shield HP", 265: "Armor HP", 9: "Structure HP",
	271: "Shield EM Resist", 272: "Shield Thermal Resist",
	273: "Shield Kinetic Resist", 274: "Shield Explosive Resist",
	267: "Armor EM Resist", 268: "Armor Thermal Resist",
	269: "Armor Kinetic Resist", 270: "Armor Explosive Resist",
	113: "Hull EM Resist", 110: "Hull Thermal Resist",
	109: "Hull Kinetic Resist", 111: "Hull Explosive Resist",
	479: "Shield Recharge Time",
	11:  "Powergrid Output", 48: "CPU Output", 1132: "Calibration",
	12: "High Slots", 13: "Med Slots", 14: "Low Slots",
	1137: "Rig Slots", 1154: "Launcher Hardpoints", 102: "Turret Hardpoints",
	37: "Max Velocity", 552: "Signature Radius", 70: "Inertia Modifier",
	161: "Warp Speed Multiplier", 600: "Warp Capacitor Need",
	482: "Capacitor Capacity", 55: "Capacitor Recharge Time",
	564: "Max Target Range", 192: "Max Locked Targets", 73: "Scan Resolution",
	76: "Radar Strength", 77: "Ladar Strength",
	78: "Magnetometric Strength", 79: "Gravimetric Strength",
	283: "Drone Bandwidth", 284: "Drone Capacity",
	38: "Cargo Capacity", 4: "Mass",
}

var universeShipAttributeGroups = []struct {
	Name string
	IDs  []int64
}{
	{
		"defense",
		[]int64{263, 265, 9, 271, 272, 273, 274, 267, 268, 269, 270, 113, 110, 109, 111, 479},
	},
	{
		"fitting",
		[]int64{11, 48, 1132, 12, 13, 14, 1137, 1154, 102},
	},
	{"navigation", []int64{37, 552, 70, 161}},
	{"capacitor", []int64{482, 55}},
	{"targeting", []int64{564, 192, 73, 76, 77, 78, 79}},
	{"drones", []int64{283, 284}},
}

var universeSkillAttributeIDs = []int64{182, 183, 184}
var universeSkillLevelAttributeIDs = []int64{277, 278, 279}

func loadUniverseType(
	ctx context.Context,
	db Database,
	id int64,
) (map[string]any, error) {
	cutoff := universeCutoff(90)
	result, err := queryMapsConcurrent(ctx, db,
		databaseQuery{
			SQL: `
				SELECT t.type_id, t.name, t.description, t.group_id,
				       g.category_id, t.mass, t.volume, t.capacity,
				       t.portion_size, t.packaged_volume, t.radius,
				       t.meta_group_id, t.market_group_id, t.race_id,
				       t.faction_id, t.base_price, t.published,
				       g.name AS group_name, c.name AS category_name,
				       mg.name AS meta_group_name
				FROM inv_types t
				LEFT JOIN inv_groups g ON g.group_id = t.group_id
				LEFT JOIN inv_categories c ON c.category_id = g.category_id
				LEFT JOIN inv_meta_groups mg
				  ON mg.meta_group_id = t.meta_group_id
				WHERE t.type_id = $1
				LIMIT 1`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				SELECT a.attribute_id, a.value, skill.name AS skill_name
				FROM type_dogma_attributes a
				LEFT JOIN inv_types skill
				  ON skill.type_id = CASE
				    WHEN a.attribute_id = ANY('{182,183,184}'::int[])
				     AND a.value BETWEEN 1 AND 2147483647
				     AND trunc(a.value) = a.value
				    THEN a.value::integer
				    ELSE NULL
				  END
				WHERE a.type_id = $1
				ORDER BY a.attribute_id ASC`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				SELECT to_char(date, 'YYYY-MM-DD') AS date,
				       average, highest, lowest, order_count, volume
				FROM prices
				WHERE type_id = $1 AND region_id = 10000002
				  AND date >= $2::date
				ORDER BY date DESC`,
			Args: []any{id, cutoff},
		},
		databaseQuery{
			SQL: `
				SELECT to_char(date, 'YYYY-MM-DD') AS date, price
				FROM custom_prices
				WHERE type_id = $1 AND date >= $2::date
				ORDER BY date DESC`,
			Args: []any{id, cutoff},
		},
		databaseQuery{
			SQL: `
				SELECT level_name, cost, payout
				FROM insurance_prices
				WHERE type_id = $1`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				SELECT m.material_type_id, t.name, m.quantity
				FROM type_materials m
				LEFT JOIN inv_types t ON t.type_id = m.material_type_id
				WHERE m.type_id = $1`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				WITH RECURSIVE breadcrumb AS (
					SELECT mg.market_group_id, mg.name,
					       mg.parent_group_id, 1 AS depth
					FROM inv_market_groups mg
					WHERE mg.market_group_id = (
						SELECT market_group_id
						FROM inv_types
						WHERE type_id = $1
					)
					UNION ALL
					SELECT parent.market_group_id, parent.name,
					       parent.parent_group_id, child.depth + 1
					FROM inv_market_groups parent
					JOIN breadcrumb child
					  ON parent.market_group_id = child.parent_group_id
					WHERE child.depth < 10
				)
				SELECT market_group_id, name
				FROM breadcrumb
				ORDER BY depth DESC`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				WITH root AS (
					SELECT COALESCE(variation_parent_type_id, type_id) AS type_id
					FROM inv_types
					WHERE type_id = $1
				)
				SELECT t.type_id, t.name, t.meta_group_id,
				       mg.name AS meta_group_name
				FROM inv_types t
				CROSS JOIN root
				LEFT JOIN inv_meta_groups mg
				  ON mg.meta_group_id = t.meta_group_id
				WHERE (t.type_id = root.type_id
				       OR t.variation_parent_type_id = root.type_id)
				  AND t.published IS TRUE
				ORDER BY t.meta_group_id ASC, t.type_id ASC`,
			Args: []any{id},
		},
	)
	if err != nil {
		return nil, err
	}
	typeRow := firstUniverseRow(result[0])
	if typeRow == nil {
		return nil, nil
	}

	attributes := result[1]
	values := make(map[int64]float64, len(attributes))
	skillNames := make(map[int64]any, len(universeSkillAttributeIDs))
	for _, row := range attributes {
		attributeID, idOK := int64Value(row["attribute_id"])
		value, valueOK := float64Value(row["value"])
		if !idOK || !valueOK {
			continue
		}
		values[attributeID] = value
		if row["skill_name"] != nil {
			skillNames[int64(value)] = row["skill_name"]
		}
	}

	categoryID, _ := int64Value(typeRow["category_id"])
	isShip := categoryID == 6
	item := universeTypeSummary(typeRow, values, isShip)
	shipAttributes, flatAttributes := buildUniverseTypeAttributes(
		attributes, values, isShip,
	)

	variations := nonNilUniverseRows(result[7])
	if len(variations) <= 1 {
		variations = []map[string]any{}
	}
	return map[string]any{
		"item":             item,
		"shipAttributes":   shipAttributes,
		"attributes":       flatAttributes,
		"requiredSkills":   buildUniverseRequiredSkills(values, skillNames),
		"materials":        buildUniverseMaterials(result[5]),
		"marketBreadcrumb": buildUniverseMarketBreadcrumb(result[6]),
		"variations":       variations,
		"pricing": map[string]any{
			"summary":       summarizeUniverseMarketPrices(result[2]),
			"history":       nonNilUniverseRows(result[2]),
			"insurance":     nonNilUniverseRows(result[4]),
			"customSummary": summarizeUniverseCustomPrices(result[3]),
			"customHistory": nonNilUniverseRows(result[3]),
		},
	}, nil
}

func universeTypeSummary(
	row map[string]any,
	attributes map[int64]float64,
	isShip bool,
) map[string]any {
	item := make(map[string]any, 23)
	for _, key := range []string{
		"type_id", "name", "description", "group_id", "category_id",
		"mass", "volume", "capacity", "portion_size", "packaged_volume",
		"radius", "meta_group_id", "market_group_id", "race_id",
		"faction_id", "base_price", "published", "group_name",
		"category_name", "meta_group_name",
	} {
		item[key] = row[key]
	}
	item["tech_level"] = nil
	if value, ok := attributes[422]; ok {
		item["tech_level"] = value
	}
	item["meta_level"] = nil
	if value, ok := attributes[633]; ok {
		item["meta_level"] = value
	}
	item["is_ship"] = isShip
	return item
}

func buildUniverseTypeAttributes(
	rows []map[string]any,
	values map[int64]float64,
	isShip bool,
) (any, []map[string]any) {
	if isShip {
		grouped := map[string][]map[string]any{}
		for _, group := range universeShipAttributeGroups {
			items := make([]map[string]any, 0, len(group.IDs))
			for _, attributeID := range group.IDs {
				value, ok := values[attributeID]
				if !ok || skipUniverseTypeAttribute(attributeID) {
					continue
				}
				name := universeShipAttributeNames[attributeID]
				if name == "" {
					name = "Attribute " + strconv.FormatInt(attributeID, 10)
				}
				items = append(items, map[string]any{
					"id": attributeID, "name": name, "value": value,
				})
			}
			if len(items) > 0 {
				grouped[group.Name] = items
			}
		}
		return grouped, []map[string]any{}
	}

	flat := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		attributeID, idOK := int64Value(row["attribute_id"])
		value, valueOK := float64Value(row["value"])
		if !idOK || !valueOK || value == 0 ||
			skipUniverseTypeAttribute(attributeID) {
			continue
		}
		flat = append(flat, map[string]any{
			"id": attributeID, "value": value,
		})
	}
	return nil, flat
}

func skipUniverseTypeAttribute(id int64) bool {
	switch id {
	case 182, 183, 184, 277, 278, 279, 422, 633:
		return true
	default:
		return false
	}
}

func buildUniverseRequiredSkills(
	values map[int64]float64,
	names map[int64]any,
) []map[string]any {
	skills := make([]map[string]any, 0, len(universeSkillAttributeIDs))
	for i, skillAttribute := range universeSkillAttributeIDs {
		skillID := values[skillAttribute]
		level := values[universeSkillLevelAttributeIDs[i]]
		if skillID == 0 || level == 0 {
			continue
		}
		id := int64(skillID)
		skills = append(skills, map[string]any{
			"type_id": id, "name": names[id], "level": level,
		})
	}
	return skills
}

func buildUniverseMaterials(rows []map[string]any) []map[string]any {
	materials := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		materials = append(materials, map[string]any{
			"type_id":  row["material_type_id"],
			"name":     row["name"],
			"quantity": row["quantity"],
		})
	}
	return materials
}

func buildUniverseMarketBreadcrumb(rows []map[string]any) []map[string]any {
	breadcrumb := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		name, _ := stringValue(row["name"])
		breadcrumb = append(breadcrumb, map[string]any{
			"id": row["market_group_id"], "name": name,
			"slug": eve.Slugify(name),
		})
	}
	return breadcrumb
}

func summarizeUniverseMarketPrices(rows []map[string]any) map[string]any {
	if len(rows) == 0 {
		return nil
	}
	averageTotal := float64(0)
	volumeTotal := float64(0)
	highest := -math.MaxFloat64
	var lowest *float64
	for _, row := range rows {
		average, _ := float64Value(row["average"])
		averageTotal += average
		value, _ := float64Value(row["highest"])
		if value > highest {
			highest = value
		}
		value, ok := float64Value(row["lowest"])
		if ok && value > 0 && (lowest == nil || value < *lowest) {
			candidate := value
			lowest = &candidate
		}
		volume, _ := float64Value(row["volume"])
		volumeTotal += volume
	}
	var lowestValue any
	if lowest != nil {
		lowestValue = *lowest
	}
	return map[string]any{
		"latest":         rows[0]["average"],
		"latest_date":    rows[0]["date"],
		"average_90d":    math.Round(averageTotal / float64(len(rows))),
		"highest_90d":    highest,
		"lowest_90d":     lowestValue,
		"avg_volume_90d": math.Round(volumeTotal / float64(len(rows))),
	}
}

func summarizeUniverseCustomPrices(rows []map[string]any) map[string]any {
	if len(rows) == 0 {
		return nil
	}
	total := float64(0)
	highest := -math.MaxFloat64
	lowest := math.MaxFloat64
	for _, row := range rows {
		price, _ := float64Value(row["price"])
		total += price
		if price > highest {
			highest = price
		}
		if price < lowest {
			lowest = price
		}
	}
	return map[string]any{
		"latest": rows[0]["price"], "latest_date": rows[0]["date"],
		"average_90d": math.Round(total / float64(len(rows))),
		"highest_90d": highest, "lowest_90d": lowest,
	}
}
