package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/sync/errgroup"
)

type mapData struct {
	Systems       []map[string]any
	Jumps         []map[string]any
	ExternalJumps []map[string]any
	Activity      []map[string]any
}

var mapScopeNames = []string{"new-eden", "zarzakh", "wormhole", "abyssal", "proving"}
var mapActivityWindows = map[int]bool{1: true, 6: true, 24: true, 168: true}

const (
	mapAIIDJumpRadius = 10
	mapAIIDMaxAnchors = 8
)

func registerMapRoutes(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "map-regions",
		Method:      http.MethodGet,
		Path:        "/map/regions",
		Summary:     "Regions grouped by map scope",
		Tags:        []string{"map"},
	}, routeJSONCache(
		opts, 6*time.Hour,
		"public, max-age=3600, s-maxage=21600, stale-while-revalidate=3600",
		mapRegionsHandler(opts),
	))

	registerLegacy(a, huma.Operation{
		OperationID: "map-scope",
		Method:      http.MethodGet,
		Path:        "/map/scope",
		Summary:     "Systems and connections in a map scope",
		Tags:        []string{"map"},
	}, routeJSONCache(
		opts, 10*time.Minute,
		"public, max-age=60, s-maxage=600, stale-while-revalidate=60",
		mapScopeHandler(opts),
	))

	registerLegacy(a, huma.Operation{
		OperationID: "map-region",
		Method:      http.MethodGet,
		Path:        "/map/region/{id}",
		Summary:     "Region map with systems, connections, and activity",
		Tags:        []string{"map"},
	}, routeJSONCache(
		opts, 10*time.Minute,
		"public, max-age=60, s-maxage=600, stale-while-revalidate=60",
		mapRegionHandler(opts),
	))

	registerLegacy(a, huma.Operation{
		OperationID: "map-sovereignty",
		Method:      http.MethodGet,
		Path:        "/map/sovereignty",
		Summary:     "Alliance sovereignty map",
		Tags:        []string{"map"},
	}, routeJSONCache(
		opts, 10*time.Minute,
		"public, max-age=60, s-maxage=600, stale-while-revalidate=60",
		mapSovereigntyHandler(opts),
	))

	registerLegacy(a, huma.Operation{
		OperationID: "map-aiid",
		Method:      http.MethodGet,
		Path:        "/map/aiid",
		Summary:     "Am I In Danger watch map",
		Tags:        []string{"map", "killmails"},
	}, routeJSONCache(
		opts, 30*time.Second,
		"public, max-age=15, s-maxage=30, stale-while-revalidate=30",
		mapAIIDHandler(opts),
	))
}

func mapRegionsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		rows, err := queryMaps(ctx, opts.DB, `
			SELECT r.region_id, r.name, COUNT(s.solar_system_id)::int AS system_count
			FROM regions r
			LEFT JOIN solar_systems s ON s.region_id = r.region_id
			GROUP BY r.region_id, r.name
			ORDER BY r.name ASC`)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(groupMapRegions(rows)), nil
	}
}

func groupMapRegions(rows []map[string]any) map[string]any {
	groups := map[string][]map[string]any{
		"kspace": {}, "pochven": {}, "zarzakh": {},
		"wormhole": {}, "abyssal": {}, "proving": {},
	}
	for _, row := range rows {
		id, ok := int64Value(row["region_id"])
		if !ok {
			continue
		}
		switch {
		case id == 10000070:
			groups["pochven"] = append(groups["pochven"], row)
		case id == 10001000:
			groups["zarzakh"] = append(groups["zarzakh"], row)
		case id >= 10000001 && id < 11000000:
			groups["kspace"] = append(groups["kspace"], row)
		case id >= 11000001 && id <= 11000033:
			groups["wormhole"] = append(groups["wormhole"], row)
		case id >= 12000001 && id <= 12000005:
			groups["abyssal"] = append(groups["abyssal"], row)
		case id >= 14000001:
			groups["proving"] = append(groups["proving"], row)
		}
	}
	return map[string]any{
		"kspace": groups["kspace"], "pochven": groups["pochven"],
		"zarzakh": groups["zarzakh"], "wormhole": groups["wormhole"],
		"abyssal": groups["abyssal"], "proving": groups["proving"],
	}
}

func regionInMapScope(scope string, id int64) bool {
	switch scope {
	case "new-eden":
		return id >= 10000001 && id < 11000000 && id != 10001000
	case "zarzakh":
		return id == 10001000
	case "wormhole":
		return id >= 11000001 && id <= 11000033
	case "abyssal":
		return id >= 12000001 && id <= 12000005
	case "proving":
		return id >= 14000001
	default:
		return false
	}
}

func mapScopeHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		activityHours, err := parseMapActivityHours(req.Query.Get("hours"))
		if err != nil {
			return legacyPayload{}, err
		}
		scope := strings.TrimSpace(req.Query.Get("type"))
		if scope == "" {
			scope = "new-eden"
		}
		valid := false
		for _, name := range mapScopeNames {
			valid = valid || scope == name
		}
		if !valid {
			return legacyPayload{}, apiError(http.StatusBadRequest,
				fmt.Sprintf("Unknown scope: %s. Valid: %s",
					scope, strings.Join(mapScopeNames, ", ")))
		}

		allRegions, err := queryMaps(ctx, opts.DB,
			`SELECT region_id, name FROM regions ORDER BY name ASC`)
		if err != nil {
			return legacyPayload{}, err
		}
		regions := make([]map[string]any, 0)
		regionValues := make([]any, 0)
		for _, row := range allRegions {
			id, ok := int64Value(row["region_id"])
			if ok && regionInMapScope(scope, id) {
				regions = append(regions, row)
				regionValues = append(regionValues, id)
			}
		}
		if len(regionValues) == 0 {
			return jsonPayload(emptyMapScope(scope, regions, activityHours)), nil
		}

		data, err := loadMapData(ctx, opts.DB, int32Slice(regionValues...), activityHours)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"scope": scope, "activity_hours": activityHours, "regions": regions,
			"systems": data.Systems, "jumps": data.Jumps,
			"externalJumps": data.ExternalJumps, "activity": data.Activity,
		}), nil
	}
}

func emptyMapScope(scope string, regions []map[string]any, activityHours int) map[string]any {
	return map[string]any{
		"scope": scope, "activity_hours": activityHours, "regions": regions,
		"systems": []any{}, "jumps": []any{},
		"externalJumps": []any{}, "activity": []any{},
	}
}

func mapRegionHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		activityHours, err := parseMapActivityHours(req.Query.Get("hours"))
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := parseID(req.Param("id"))
		if err != nil || id > pgInt4Max {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid ID")
		}
		region, err := queryMap(ctx, opts.DB,
			`SELECT region_id, name FROM regions WHERE region_id = $1 LIMIT 1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if region == nil {
			return legacyPayload{}, apiError(http.StatusNotFound, "Region not found")
		}

		data, err := loadMapData(ctx, opts.DB, []int32{int32(id)}, activityHours)
		if err != nil {
			return legacyPayload{}, err
		}
		constellations, err := queryMaps(ctx, opts.DB, `
			SELECT constellation_id, constellation_name
			FROM constellations
			WHERE region_id = $1
			ORDER BY constellation_name ASC`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		celestials := []map[string]any{}
		if len(data.Systems) > 0 {
			celestials, err = queryMaps(ctx, opts.DB, `
				SELECT solar_system_id AS system_id, group_id, x, z
				FROM celestials
				WHERE region_id = $1 AND group_id = ANY('{6,7}'::int[])
				ORDER BY solar_system_id, group_id`, id)
			if err != nil {
				return legacyPayload{}, err
			}
		}

		return jsonPayload(map[string]any{
			"region": region, "activity_hours": activityHours, "systems": data.Systems,
			"constellations": constellations, "jumps": data.Jumps,
			"externalJumps": data.ExternalJumps, "activity": data.Activity,
			"celestials": celestials,
		}), nil
	}
}

func mapSovereigntyHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		activityHours, err := parseMapActivityHours(req.Query.Get("hours"))
		if err != nil {
			return legacyPayload{}, err
		}

		// Keep the complete K-space topology as faint context. Only player-alliance
		// claims paint territory below, leaving empire and NPC nullsec as negative space.
		regions, err := queryMaps(ctx, opts.DB, `
			SELECT region_id, name
			FROM regions
			WHERE region_id >= 10000001
			  AND region_id < 11000000
			  AND region_id <> 10001000
			ORDER BY name`)
		if err != nil {
			return legacyPayload{}, err
		}
		regionValues := make([]any, 0, len(regions))
		for _, region := range regions {
			regionValues = append(regionValues, region["region_id"])
		}
		regionIDs := int32Slice(regionValues...)
		data, err := loadMapData(ctx, opts.DB, regionIDs, activityHours)
		if err != nil {
			return legacyPayload{}, err
		}

		claims, err := queryMaps(ctx, opts.DB, `
			SELECT sov.system_id, sov.alliance_id, sov.date_added,
			       COALESCE(a.name, 'Unknown Alliance') AS alliance_name,
			       COALESCE(a.ticker, '?') AS alliance_ticker,
			       COALESCE(a.member_count, 0)::int AS member_count
			FROM sovereignty sov
			JOIN solar_systems s ON s.solar_system_id = sov.system_id
			LEFT JOIN alliances a ON a.alliance_id = sov.alliance_id
			WHERE s.region_id = ANY($1::int[])
			  AND sov.alliance_id IS NOT NULL
			ORDER BY sov.system_id`, regionIDs)
		if err != nil {
			return legacyPayload{}, err
		}
		changes, err := queryMaps(ctx, opts.DB, `
			SELECT DISTINCT ON (h.system_id)
			       h.system_id, h.alliance_id, h.date_added,
			       a.name AS alliance_name,
			       a.ticker AS alliance_ticker
			FROM sovereignty_history h
			JOIN solar_systems s ON s.solar_system_id = h.system_id
			LEFT JOIN alliances a ON a.alliance_id = h.alliance_id
			WHERE s.region_id = ANY($1::int[])
			  AND h.date_added >= now() - interval '7 days'
			ORDER BY h.system_id, h.date_added DESC, h.id DESC`, regionIDs)
		if err != nil {
			return legacyPayload{}, err
		}
		snapshot, err := queryMap(ctx, opts.DB, `
			SELECT COALESCE(MAX(sov.date_added), now()) AS snapshot_at
			FROM sovereignty sov
			JOIN solar_systems s ON s.solar_system_id = sov.system_id
			WHERE s.region_id = ANY($1::int[])`, regionIDs)
		if err != nil {
			return legacyPayload{}, err
		}

		return jsonPayload(map[string]any{
			"scope": "sovereignty", "activity_hours": activityHours,
			"snapshot_at": snapshot["snapshot_at"], "regions": regions,
			"systems": data.Systems, "jumps": data.Jumps,
			"externalJumps": data.ExternalJumps, "activity": data.Activity,
			"sovereignty": claims, "changes": changes,
		}), nil
	}
}

func mapAIIDHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		activityHours, err := parseMapActivityHours(req.Query.Get("hours"))
		if err != nil {
			return legacyPayload{}, err
		}
		anchorIDs, err := parseMapSystemIDs(req.Query.Get("systems"))
		if err != nil {
			return legacyPayload{}, err
		}
		if len(anchorIDs) == 0 {
			return jsonPayload(emptyMapAIID(activityHours)), nil
		}

		systems, err := queryMaps(ctx, opts.DB, `
			WITH RECURSIVE neighborhood(system_id, distance) AS (
				SELECT anchor_id, 0
				FROM unnest($1::int[]) AS anchor_id
				UNION
				SELECT CASE
					WHEN jump.from_solar_system_id = neighborhood.system_id
					THEN jump.to_solar_system_id
					ELSE jump.from_solar_system_id
				END,
				neighborhood.distance + 1
				FROM neighborhood
				JOIN solar_system_jumps jump
				  ON jump.from_solar_system_id = neighborhood.system_id
				  OR jump.to_solar_system_id = neighborhood.system_id
				WHERE neighborhood.distance < $2
			), distances AS (
				SELECT system_id, MIN(distance)::int AS distance
				FROM neighborhood
				GROUP BY system_id
			)
			SELECT system.solar_system_id, system.system_name,
			       system.x, system.y, system.z, system.x2d, system.z2d,
			       system.security, system.region_id, system.constellation_id,
			       distances.distance,
			       system.solar_system_id = ANY($1::int[]) AS is_anchor
			FROM distances
			JOIN solar_systems system
			  ON system.solar_system_id = distances.system_id
			ORDER BY distances.distance, system.solar_system_id`,
			anchorIDs, mapAIIDJumpRadius)
		if err != nil {
			return legacyPayload{}, err
		}
		if len(systems) == 0 {
			return legacyPayload{}, apiError(http.StatusNotFound, "Systems not found")
		}

		values := make([]any, 0, len(systems))
		for _, system := range systems {
			values = append(values, system["solar_system_id"])
		}
		systemIDs := int32Slice(values...)
		data, err := loadMapDataForSystems(ctx, opts.DB, systems, systemIDs, activityHours)
		if err != nil {
			return legacyPayload{}, err
		}
		regions, err := queryMaps(ctx, opts.DB, `
			SELECT region.region_id, region.name, COUNT(*)::int AS system_count
			FROM solar_systems system
			JOIN regions region ON region.region_id = system.region_id
			WHERE system.solar_system_id = ANY($1::int[])
			GROUP BY region.region_id, region.name
			ORDER BY region.name`, systemIDs)
		if err != nil {
			return legacyPayload{}, err
		}

		killRows, err := queryMaps(ctx, opts.DB, campaignKilllistSelect+`
			WHERE k.solar_system_id = ANY($1::int[])
			  AND k.killmail_time >= now() - interval '24 hours'
			ORDER BY k.killmail_time DESC, k.killmail_id DESC
			LIMIT 101`, systemIDs)
		if err != nil {
			return legacyPayload{}, err
		}
		kills, _, _, err := finishUniverseKilllist(ctx, opts.DB, killRows, 100)
		if err != nil {
			return legacyPayload{}, err
		}

		anchors := make([]map[string]any, 0, len(anchorIDs))
		for _, system := range systems {
			if isAnchor, _ := system["is_anchor"].(bool); isAnchor {
				anchors = append(anchors, map[string]any{
					"solar_system_id": system["solar_system_id"],
					"system_name":     system["system_name"],
					"region_id":       system["region_id"],
				})
			}
		}

		return jsonPayload(map[string]any{
			"scope": "aiid", "activity_hours": activityHours,
			"jump_radius": mapAIIDJumpRadius, "anchors": anchors,
			"regions": regions, "systems": data.Systems, "jumps": data.Jumps,
			"externalJumps": data.ExternalJumps, "activity": data.Activity,
			"kills": kills,
		}), nil
	}
}

func emptyMapAIID(activityHours int) map[string]any {
	return map[string]any{
		"scope": "aiid", "activity_hours": activityHours,
		"jump_radius": mapAIIDJumpRadius, "anchors": []any{},
		"regions": []any{}, "systems": []any{}, "jumps": []any{},
		"externalJumps": []any{}, "activity": []any{}, "kills": []any{},
	}
}

func parseMapSystemIDs(raw string) ([]int32, error) {
	seen := map[int32]bool{}
	ids := make([]int32, 0, mapAIIDMaxAnchors)
	for part := range strings.SplitSeq(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.ParseInt(part, 10, 32)
		if err != nil || value <= 0 {
			return nil, apiError(http.StatusBadRequest, "systems must be comma-separated positive IDs")
		}
		id := int32(value)
		if seen[id] {
			continue
		}
		if len(ids) >= mapAIIDMaxAnchors {
			return nil, apiError(http.StatusBadRequest, fmt.Sprintf("systems accepts at most %d IDs", mapAIIDMaxAnchors))
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, nil
}

func loadMapData(ctx context.Context, db Database, regionIDs []int32, activityHours int) (mapData, error) {
	empty := mapData{
		Systems: []map[string]any{}, Jumps: []map[string]any{},
		ExternalJumps: []map[string]any{}, Activity: []map[string]any{},
	}
	if len(regionIDs) == 0 {
		return empty, nil
	}

	systems, err := queryMaps(ctx, db, `
		SELECT solar_system_id, system_name, x, y, z, x2d, z2d, security,
		       region_id, constellation_id
		FROM solar_systems
		WHERE region_id = ANY($1::int[])
		ORDER BY solar_system_id`, regionIDs)
	if err != nil {
		return empty, err
	}
	if len(systems) == 0 {
		return empty, nil
	}
	values := make([]any, 0, len(systems))
	for _, system := range systems {
		values = append(values, system["solar_system_id"])
	}
	systemIDs := int32Slice(values...)
	return loadMapDataForSystems(ctx, db, systems, systemIDs, activityHours)
}

func loadMapDataForSystems(ctx context.Context, db Database, systems []map[string]any, systemIDs []int32, activityHours int) (mapData, error) {
	empty := mapData{
		Systems: []map[string]any{}, Jumps: []map[string]any{},
		ExternalJumps: []map[string]any{}, Activity: []map[string]any{},
	}
	if len(systemIDs) == 0 {
		return empty, nil
	}

	var jumps, external, activity []map[string]any
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		var queryErr error
		jumps, queryErr = queryMaps(groupCtx, db, `
			SELECT from_solar_system_id, to_solar_system_id
			FROM solar_system_jumps
			WHERE from_solar_system_id = ANY($1::int[])
			  AND to_solar_system_id = ANY($1::int[])
			ORDER BY from_solar_system_id, to_solar_system_id`, systemIDs)
		return queryErr
	})
	group.Go(func() error {
		var queryErr error
		external, queryErr = queryMaps(groupCtx, db, `
			SELECT
				CASE WHEN j.from_solar_system_id = ANY($1::int[])
				     THEN j.from_solar_system_id ELSE j.to_solar_system_id
				END AS internal_system_id,
				ext.solar_system_id AS external_system_id,
				ext.system_name AS external_system_name,
				ext.region_id AS external_region_id,
				r.name AS external_region_name,
				ext.security AS external_security,
				ext.x AS external_x, ext.z AS external_z,
				ext.x2d AS external_x2d, ext.z2d AS external_z2d
			FROM solar_system_jumps j
			JOIN solar_systems ext
			  ON ext.solar_system_id = CASE
				WHEN j.from_solar_system_id = ANY($1::int[])
				THEN j.to_solar_system_id ELSE j.from_solar_system_id END
			LEFT JOIN regions r ON r.region_id = ext.region_id
			WHERE (j.from_solar_system_id = ANY($1::int[]))
			   <> (j.to_solar_system_id = ANY($1::int[]))
			ORDER BY internal_system_id, external_system_id`, systemIDs)
		return queryErr
	})
	group.Go(func() error {
		var queryErr error
		activity, queryErr = queryMaps(groupCtx, db, `
			SELECT system_id,
			       COALESCE(SUM(ship_kills), 0)::int AS ship_kills,
			       COALESCE(SUM(npc_kills), 0)::int AS npc_kills,
			       COALESCE(SUM(pod_kills), 0)::int AS pod_kills,
			       COALESCE(SUM(ship_jumps), 0)::int AS ship_jumps
			FROM system_activity
			WHERE system_id = ANY($1::int[])
			  AND timestamp >= now() - ($2 * interval '1 hour')
			GROUP BY system_id
			ORDER BY system_id`, systemIDs, activityHours)
		return queryErr
	})
	if err := group.Wait(); err != nil {
		return empty, err
	}
	return mapData{
		Systems: systems, Jumps: nonNilRows(jumps),
		ExternalJumps: nonNilRows(external), Activity: nonNilRows(activity),
	}, nil
}

func parseMapActivityHours(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 24, nil
	}
	hours, err := strconv.Atoi(raw)
	if err != nil || !mapActivityWindows[hours] {
		return 0, apiError(http.StatusBadRequest, "hours must be one of 1, 6, 24, or 168")
	}
	return hours, nil
}

func nonNilRows(rows []map[string]any) []map[string]any {
	if rows == nil {
		return []map[string]any{}
	}
	return rows
}
