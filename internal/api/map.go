package api

import (
	"context"
	"fmt"
	"net/http"
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
			return jsonPayload(emptyMapScope(scope, regions)), nil
		}

		data, err := loadMapData(ctx, opts.DB, int32Slice(regionValues...))
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"scope": scope, "regions": regions,
			"systems": data.Systems, "jumps": data.Jumps,
			"externalJumps": data.ExternalJumps, "activity": data.Activity,
		}), nil
	}
}

func emptyMapScope(scope string, regions []map[string]any) map[string]any {
	return map[string]any{
		"scope": scope, "regions": regions,
		"systems": []any{}, "jumps": []any{},
		"externalJumps": []any{}, "activity": []any{},
	}
}

func mapRegionHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
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

		data, err := loadMapData(ctx, opts.DB, []int32{int32(id)})
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
			"region": region, "systems": data.Systems,
			"constellations": constellations, "jumps": data.Jumps,
			"externalJumps": data.ExternalJumps, "activity": data.Activity,
			"celestials": celestials,
		}), nil
	}
}

func loadMapData(ctx context.Context, db Database, regionIDs []int32) (mapData, error) {
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
			  AND timestamp >= now() - interval '24 hours'
			GROUP BY system_id
			ORDER BY system_id`, systemIDs)
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

func nonNilRows(rows []map[string]any) []map[string]any {
	if rows == nil {
		return []map[string]any{}
	}
	return rows
}
