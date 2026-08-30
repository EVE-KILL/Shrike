package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

func registerSDERoutes(a huma.API, opts Options) {
	registerSDESystems(a, opts)
	registerSDEUniverse(a, opts)
	registerSDEInventory(a, opts)
	registerSDEDetails(a, opts)
	registerSDEStations(a, opts)
	registerSDESovereigntyAndPrices(a, opts)
	registerSDEKillRoutes(a, opts)
}

func registerSDESystems(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "sde-systems",
		Method:      http.MethodGet,
		Path:        "/sde/systems",
		Summary:     "Solar systems",
		Tags:        []string{"sde"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		limit := boundedQueryInt(req, "limit", 50, 1, 100)
		where, args := []string{}, []any{}
		add := func(column, value string) {
			if n, ok := optionalQueryNumber(req, value); ok && n != 0 {
				args = append(args, n)
				where = append(where, fmt.Sprintf("%s = $%d", column, len(args)))
			}
		}
		add("region_id", "region_id")
		add("constellation_id", "constellation_id")
		if name := strings.TrimSpace(req.Query.Get("name")); name != "" {
			args = append(args, name+"%")
			where = append(where, fmt.Sprintf("system_name ILIKE $%d", len(args)))
		}
		add("solar_system_id", "after")

		query := `SELECT solar_system_id, system_name, constellation_id, region_id, security
			FROM solar_systems`
		if len(where) > 0 {
			query += " WHERE " + strings.Join(where, " AND ")
		}
		args = append(args, limit+1)
		query += fmt.Sprintf(" ORDER BY solar_system_id ASC LIMIT $%d", len(args))
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		return paginatedRows(rows, limit, "solar_system_id"), nil
	})

	registerLegacy(a, sdeIDOperation("sde-system", "/sde/systems/{id}", "Solar system detail"),
		func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
			id, err := parseID(req.Param("id"))
			if err != nil {
				return legacyPayload{}, err
			}
			row, err := queryMap(ctx, opts.DB, `
				SELECT s.solar_system_id, s.system_name, s.constellation_id,
				       c.constellation_name, s.region_id, r.name AS region_name,
				       s.security, s.security_class
				FROM solar_systems s
				LEFT JOIN constellations c ON c.constellation_id = s.constellation_id
				LEFT JOIN regions r ON r.region_id = s.region_id
				WHERE s.solar_system_id = $1
				LIMIT 1`, id)
			if err != nil {
				return legacyPayload{}, err
			}
			return foundOr404(row, "Solar system not found"), nil
		})
}

func registerSDEUniverse(a huma.API, opts Options) {
	registerSimpleSDEList(a, opts, simpleSDEList{
		ID: "sde-regions", Path: "/sde/regions", Summary: "Regions",
		Query: `SELECT region_id, name, faction_id FROM regions ORDER BY name ASC`,
	})
	registerLegacy(a, sdeIDOperation("sde-region", "/sde/regions/{id}", "Region detail"),
		func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
			id, err := parseID(req.Param("id"))
			if err != nil {
				return legacyPayload{}, err
			}
			row, err := queryMap(ctx, opts.DB, `
				SELECT r.region_id, r.name, r.description, r.faction_id,
				       f.name AS faction_name
				FROM regions r
				LEFT JOIN factions f ON f.faction_id = r.faction_id
				WHERE r.region_id = $1 LIMIT 1`, id)
			if err != nil {
				return legacyPayload{}, err
			}
			return foundOr404(row, "Region not found"), nil
		})

	registerLegacy(a, huma.Operation{
		OperationID: "sde-constellations",
		Method:      http.MethodGet,
		Path:        "/sde/constellations",
		Summary:     "Constellations",
		Tags:        []string{"sde"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		query := `SELECT constellation_id, constellation_name, region_id, faction_id
			FROM constellations`
		args := []any{}
		if regionID, ok := optionalQueryNumber(req, "region_id"); ok && regionID != 0 {
			query += ` WHERE region_id = $1`
			args = append(args, regionID)
		}
		query += ` ORDER BY constellation_name ASC`
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"data": rows}), nil
	})
	registerLegacy(a, sdeIDOperation("sde-constellation", "/sde/constellations/{id}", "Constellation detail"),
		func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
			id, err := parseID(req.Param("id"))
			if err != nil {
				return legacyPayload{}, err
			}
			row, err := queryMap(ctx, opts.DB, `
				SELECT c.constellation_id, c.constellation_name, c.region_id,
				       r.name AS region_name, c.faction_id
				FROM constellations c
				LEFT JOIN regions r ON r.region_id = c.region_id
				WHERE c.constellation_id = $1 LIMIT 1`, id)
			if err != nil {
				return legacyPayload{}, err
			}
			return foundOr404(row, "Constellation not found"), nil
		})

	for _, spec := range []simpleSDEList{
		{ID: "sde-factions", Path: "/sde/factions", Summary: "Factions", Query: `SELECT * FROM factions ORDER BY name ASC`},
		{ID: "sde-races", Path: "/sde/races", Summary: "Races", Query: `SELECT * FROM races ORDER BY race_name ASC`},
		{ID: "sde-bloodlines", Path: "/sde/bloodlines", Summary: "Bloodlines", Query: `SELECT * FROM bloodlines ORDER BY bloodline_name ASC`},
	} {
		registerSimpleSDEList(a, opts, spec)
	}
	for _, spec := range []simpleSDEByID{
		{ID: "sde-faction", Path: "/sde/factions/{id}", Summary: "Faction detail", Table: "factions", Column: "faction_id", NotFound: "Faction not found"},
		{ID: "sde-race", Path: "/sde/races/{id}", Summary: "Race detail", Table: "races", Column: "race_id", NotFound: "Race not found"},
		{ID: "sde-bloodline", Path: "/sde/bloodlines/{id}", Summary: "Bloodline detail", Table: "bloodlines", Column: "bloodline_id", NotFound: "Bloodline not found"},
	} {
		registerSimpleSDEByID(a, opts, spec)
	}
}

func registerSDEInventory(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "sde-types",
		Method:      http.MethodGet,
		Path:        "/sde/types",
		Summary:     "Inventory types",
		Tags:        []string{"sde"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		limit := boundedQueryInt(req, "limit", 50, 1, 100)
		where, args := []string{}, []any{}
		if req.Query.Get("published") != "false" {
			where = append(where, "t.published IS TRUE")
		}
		if name := strings.TrimSpace(req.Query.Get("name")); name != "" {
			args = append(args, name+"%")
			where = append(where, fmt.Sprintf("t.name ILIKE $%d", len(args)))
		}
		for column, name := range map[string]string{
			"t.group_id":    "group_id",
			"g.category_id": "category_id",
			"t.type_id":     "after",
		} {
			if value, ok := optionalQueryNumber(req, name); ok && value != 0 {
				args = append(args, value)
				where = append(where, fmt.Sprintf("%s = $%d", column, len(args)))
			}
		}
		query := `SELECT t.type_id, t.name, t.group_id, g.name AS group_name,
		                 t.category_id, t.meta_group_id, t.published
			FROM inv_types t
			LEFT JOIN inv_groups g ON g.group_id = t.group_id`
		if len(where) > 0 {
			query += " WHERE " + strings.Join(where, " AND ")
		}
		// The after cursor is greater-than, not equality.
		query = strings.Replace(query, "t.type_id = $", "t.type_id > $", 1)
		args = append(args, limit+1)
		query += fmt.Sprintf(" ORDER BY t.type_id ASC LIMIT $%d", len(args))
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		return paginatedRows(rows, limit, "type_id"), nil
	})

	registerLegacy(a, sdeIDOperation("sde-type", "/sde/types/{id}", "Inventory type detail"),
		func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
			id, err := parseID(req.Param("id"))
			if err != nil {
				return legacyPayload{}, err
			}
			row, err := queryMap(ctx, opts.DB, `
				SELECT t.type_id, t.name, t.description, t.group_id,
				       g.name AS group_name, t.category_id,
				       c.name AS category_name, t.mass, t.volume, t.capacity,
				       t.published, t.market_group_id, t.meta_group_id, t.base_price
				FROM inv_types t
				LEFT JOIN inv_groups g ON g.group_id = t.group_id
				LEFT JOIN inv_categories c ON c.category_id = t.category_id
				WHERE t.type_id = $1 LIMIT 1`, id)
			if err != nil {
				return legacyPayload{}, err
			}
			return foundOr404(row, "Type not found"), nil
		})

	registerLegacy(a, huma.Operation{
		OperationID: "sde-groups",
		Method:      http.MethodGet,
		Path:        "/sde/groups",
		Summary:     "Inventory groups",
		Tags:        []string{"sde"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		limit := boundedQueryInt(req, "limit", 50, 1, 100)
		where, args := []string{}, []any{}
		if req.Query.Get("published") != "false" {
			where = append(where, "g.published IS TRUE")
		}
		if categoryID, ok := optionalQueryNumber(req, "category_id"); ok && categoryID != 0 {
			args = append(args, categoryID)
			where = append(where, fmt.Sprintf("g.category_id = $%d", len(args)))
		}
		if after, ok := optionalQueryNumber(req, "after"); ok && after != 0 {
			args = append(args, after)
			where = append(where, fmt.Sprintf("g.group_id > $%d", len(args)))
		}
		query := `SELECT g.group_id, g.name, g.category_id,
		                 c.name AS category_name, g.published
			FROM inv_groups g
			LEFT JOIN inv_categories c ON c.category_id = g.category_id`
		if len(where) > 0 {
			query += " WHERE " + strings.Join(where, " AND ")
		}
		args = append(args, limit+1)
		query += fmt.Sprintf(" ORDER BY g.group_id ASC LIMIT $%d", len(args))
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		return paginatedRows(rows, limit, "group_id"), nil
	})

	registerLegacy(a, sdeIDOperation("sde-group", "/sde/groups/{id}", "Inventory group detail"),
		func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
			id, err := parseID(req.Param("id"))
			if err != nil {
				return legacyPayload{}, err
			}
			row, err := queryMap(ctx, opts.DB, `
				SELECT g.group_id, g.name, g.category_id,
				       c.name AS category_name, g.published, g.icon_id
				FROM inv_groups g
				LEFT JOIN inv_categories c ON c.category_id = g.category_id
				WHERE g.group_id = $1 LIMIT 1`, id)
			if err != nil {
				return legacyPayload{}, err
			}
			return foundOr404(row, "Group not found"), nil
		})

	registerLegacy(a, huma.Operation{
		OperationID: "sde-categories",
		Method:      http.MethodGet,
		Path:        "/sde/categories",
		Summary:     "Inventory categories",
		Tags:        []string{"sde"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		query := `SELECT * FROM inv_categories`
		if req.Query.Get("published") != "false" {
			query += ` WHERE published IS TRUE`
		}
		query += ` ORDER BY name ASC`
		rows, err := queryMaps(ctx, opts.DB, query)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"data": rows}), nil
	})
	registerSimpleSDEByID(a, opts, simpleSDEByID{
		ID: "sde-category", Path: "/sde/categories/{id}", Summary: "Inventory category detail",
		Table: "inv_categories", Column: "category_id", NotFound: "Category not found",
	})

	registerLegacy(a, huma.Operation{
		OperationID: "sde-market-groups",
		Method:      http.MethodGet,
		Path:        "/sde/market-groups",
		Summary:     "Market groups",
		Tags:        []string{"sde"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		query := `SELECT * FROM inv_market_groups`
		args := []any{}
		if values, exists := req.Query["parent_id"]; exists {
			raw := ""
			if len(values) > 0 {
				raw = values[len(values)-1]
			}
			parentID, _ := strconv.ParseFloat(raw, 64)
			if parentID == 0 {
				query += ` WHERE parent_group_id IS NULL`
			} else {
				args = append(args, parentID)
				query += ` WHERE parent_group_id = $1`
			}
		}
		query += ` ORDER BY name ASC`
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"data": rows}), nil
	})
	registerSimpleSDEByID(a, opts, simpleSDEByID{
		ID: "sde-market-group", Path: "/sde/market-groups/{id}", Summary: "Market group detail",
		Table: "inv_market_groups", Column: "market_group_id", NotFound: "Market group not found",
	})
	registerSimpleSDEList(a, opts, simpleSDEList{
		ID: "sde-meta-groups", Path: "/sde/meta-groups", Summary: "Meta groups",
		Query: `SELECT * FROM inv_meta_groups ORDER BY name ASC`,
	})
	registerSimpleSDEList(a, opts, simpleSDEList{
		ID: "sde-flags", Path: "/sde/flags", Summary: "Inventory flags",
		Query: `SELECT * FROM inv_flags ORDER BY order_id ASC`,
	})
}

func registerSDEDetails(a huma.API, opts Options) {
	registerLegacy(a, sdeIDOperation("sde-type-dogma", "/sde/types/{id}/dogma", "Type dogma"),
		func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
			id, err := parseID(req.Param("id"))
			if err != nil {
				return legacyPayload{}, err
			}
			attrs, err := queryMaps(ctx, opts.DB, `SELECT * FROM type_dogma_attributes WHERE type_id = $1`, id)
			if err != nil {
				return legacyPayload{}, err
			}
			effects, err := queryMaps(ctx, opts.DB, `SELECT * FROM type_dogma_effects WHERE type_id = $1`, id)
			if err != nil {
				return legacyPayload{}, err
			}
			return jsonPayload(map[string]any{"type_id": id, "attributes": attrs, "effects": effects}), nil
		})
	registerNestedSDEList(a, opts, "sde-type-materials", "/sde/types/{id}/materials",
		"Type manufacturing materials", "type_id", "materials",
		`SELECT * FROM type_materials WHERE type_id = $1`)
	registerNestedSDEList(a, opts, "sde-type-insurance", "/sde/types/{id}/insurance",
		"Type insurance prices", "type_id", "levels",
		`SELECT * FROM insurance_prices WHERE type_id = $1`)
	registerNestedSDEList(a, opts, "sde-system-jumps", "/sde/systems/{id}/jumps",
		"Solar system jumps", "solar_system_id", "jumps",
		`SELECT * FROM solar_system_jumps WHERE from_solar_system_id = $1`)
	registerNestedSDEList(a, opts, "sde-system-celestials", "/sde/systems/{id}/celestials",
		"Solar system celestials", "solar_system_id", "celestials",
		`SELECT * FROM celestials WHERE solar_system_id = $1 ORDER BY celestial_index ASC`)

	for _, spec := range []simpleSDEByID{
		{ID: "sde-celestial", Path: "/sde/celestials/{id}", Summary: "Celestial detail", Table: "celestials", Column: "item_id", NotFound: "Celestial not found"},
		{ID: "sde-npc-corporation", Path: "/sde/npc-corporations/{id}", Summary: "NPC corporation detail", Table: "npc_corporations", Column: "corporation_id", NotFound: "NPC corporation not found"},
	} {
		registerSimpleSDEByID(a, opts, spec)
	}
	registerSimpleSDEList(a, opts, simpleSDEList{
		ID: "sde-npc-corporations", Path: "/sde/npc-corporations", Summary: "NPC corporations",
		Query: `SELECT * FROM npc_corporations ORDER BY name ASC`,
	})
}

func registerSDEStations(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "sde-stations",
		Method:      http.MethodGet,
		Path:        "/sde/stations",
		Summary:     "Stations",
		Tags:        []string{"sde"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		limit := boundedQueryInt(req, "limit", 50, 1, 100)
		where, args := []string{}, []any{}
		for column, name := range map[string]string{
			"solar_system_id": "solar_system_id",
			"region_id":       "region_id",
		} {
			if value, ok := optionalQueryNumber(req, name); ok && value != 0 {
				args = append(args, value)
				where = append(where, fmt.Sprintf("%s = $%d", column, len(args)))
			}
		}
		if after, ok := optionalQueryNumber(req, "after"); ok && after != 0 {
			args = append(args, after)
			where = append(where, fmt.Sprintf("station_id > $%d", len(args)))
		}
		query := `SELECT station_id, station_name, type_id, corporation_id,
		                 solar_system_id, region_id, security FROM stations`
		if len(where) > 0 {
			query += " WHERE " + strings.Join(where, " AND ")
		}
		args = append(args, limit+1)
		query += fmt.Sprintf(" ORDER BY station_id ASC LIMIT $%d", len(args))
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		return paginatedRows(rows, limit, "station_id"), nil
	})
	registerSimpleSDEByID(a, opts, simpleSDEByID{
		ID: "sde-station", Path: "/sde/stations/{id}", Summary: "Station detail",
		Table: "stations", Column: "station_id", NotFound: "Station not found",
	})
	registerSimpleSDEList(a, opts, simpleSDEList{
		ID: "sde-station-operations", Path: "/sde/station-operations", Summary: "Station operations",
		Query: `SELECT * FROM station_operations ORDER BY operation_name ASC`,
	})
	registerSimpleSDEByID(a, opts, simpleSDEByID{
		ID: "sde-station-operation", Path: "/sde/station-operations/{id}", Summary: "Station operation detail",
		Table: "station_operations", Column: "operation_id", NotFound: "Station operation not found",
	})

	registerLegacy(a, huma.Operation{
		OperationID: "sde-structures",
		Method:      http.MethodGet,
		Path:        "/sde/structures",
		Summary:     "Player structures",
		Tags:        []string{"sde"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		limit := boundedQueryInt(req, "limit", 50, 1, 100)
		where, args := []string{}, []any{}
		for column, name := range map[string]string{
			"solar_system_id": "solar_system_id",
			"region_id":       "region_id",
			"owner_id":        "owner_id",
		} {
			if value, ok := optionalQueryNumber(req, name); ok && value != 0 {
				args = append(args, value)
				where = append(where, fmt.Sprintf("%s = $%d", column, len(args)))
			}
		}
		if after, ok := optionalQueryNumber(req, "after"); ok && after != 0 {
			args = append(args, after)
			where = append(where, fmt.Sprintf("structure_id > $%d", len(args)))
		}
		query := `SELECT structure_id, name, owner_id, solar_system_id, region_id, type_id
			FROM structures`
		if len(where) > 0 {
			query += " WHERE " + strings.Join(where, " AND ")
		}
		args = append(args, limit+1)
		query += fmt.Sprintf(" ORDER BY structure_id ASC LIMIT $%d", len(args))
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		return paginatedRows(rows, limit, "structure_id"), nil
	})
	registerSimpleSDEByID(a, opts, simpleSDEByID{
		ID: "sde-structure", Path: "/sde/structures/{id}", Summary: "Structure detail",
		Table: "structures", Column: "structure_id", NotFound: "Structure not found",
	})
}

func registerSDESovereigntyAndPrices(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "sde-sovereignty",
		Method:      http.MethodGet,
		Path:        "/sde/sovereignty",
		Summary:     "Current sovereignty",
		Tags:        []string{"sde"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		query := `SELECT * FROM sovereignty`
		where, args := []string{}, []any{}
		for column, name := range map[string]string{"alliance_id": "alliance_id", "faction_id": "faction_id"} {
			if value, ok := optionalQueryNumber(req, name); ok && value != 0 {
				args = append(args, value)
				where = append(where, fmt.Sprintf("%s = $%d", column, len(args)))
			}
		}
		if len(where) > 0 {
			query += " WHERE " + strings.Join(where, " AND ")
		}
		query += ` ORDER BY system_id ASC`
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"data": rows}), nil
	})
	registerSimpleSDEByID(a, opts, simpleSDEByID{
		ID: "sde-sovereignty-system", Path: "/sde/sovereignty/{id}", Summary: "Solar system sovereignty",
		Table: "sovereignty", Column: "system_id", NotFound: "Sovereignty data not found",
	})
	registerNestedSDEList(a, opts, "sde-sovereignty-history", "/sde/sovereignty/{id}/history",
		"Sovereignty history", "system_id", "history",
		`SELECT * FROM sovereignty_history WHERE system_id = $1 ORDER BY date_added DESC`)

	registerLegacy(a, sdeIDOperation("sde-prices", "/sde/prices/{id}", "Market price history"),
		func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
			id, err := parseID(req.Param("id"))
			if err != nil {
				return legacyPayload{}, err
			}
			regionID := float64(10000002)
			if value, ok := optionalQueryNumber(req, "region_id"); ok && value != 0 {
				regionID = value
			}
			limit := boundedQueryInt(req, "limit", 30, 1, 365)
			rows, err := queryMaps(ctx, opts.DB, `
				SELECT type_id, region_id, to_char(date, 'YYYY-MM-DD') AS date,
				       average, highest, lowest, order_count, volume
				FROM prices
				WHERE type_id = $1 AND region_id = $2
				ORDER BY date DESC
				LIMIT $3`, id, regionID, limit)
			if err != nil {
				return legacyPayload{}, err
			}
			return jsonPayload(map[string]any{
				"type_id": id, "region_id": regionID, "prices": rows,
			}), nil
		})

	registerLegacy(a, huma.Operation{
		OperationID: "sde-custom-prices",
		Method:      http.MethodGet,
		Path:        "/sde/custom-prices",
		Summary:     "All custom price overrides",
		Tags:        []string{"sde"},
	}, func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		rows, err := queryMaps(ctx, opts.DB, `
			SELECT cp.type_id, t.name AS type_name,
			       to_char(cp.date, 'YYYY-MM-DD') AS valid_until, cp.price
			FROM custom_prices cp
			LEFT JOIN inv_types t ON t.type_id = cp.type_id
			ORDER BY cp.type_id ASC, cp.date ASC`)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"data": rows, "count": len(rows)}), nil
	})
}

type simpleSDEList struct {
	ID      string
	Path    string
	Summary string
	Query   string
}

func registerSimpleSDEList(a huma.API, opts Options, spec simpleSDEList) {
	registerLegacy(a, huma.Operation{
		OperationID: spec.ID,
		Method:      http.MethodGet,
		Path:        spec.Path,
		Summary:     spec.Summary,
		Tags:        []string{"sde"},
	}, func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		rows, err := queryMaps(ctx, opts.DB, spec.Query)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"data": rows}), nil
	})
}

type simpleSDEByID struct {
	ID       string
	Path     string
	Summary  string
	Table    string
	Column   string
	NotFound string
}

func registerSimpleSDEByID(a huma.API, opts Options, spec simpleSDEByID) {
	registerLegacy(a, sdeIDOperation(spec.ID, spec.Path, spec.Summary),
		func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
			id, err := parseID(req.Param("id"))
			if err != nil {
				return legacyPayload{}, err
			}
			row, err := queryMap(ctx, opts.DB,
				`SELECT * FROM `+spec.Table+` WHERE `+spec.Column+` = $1 LIMIT 1`,
				id,
			)
			if err != nil {
				return legacyPayload{}, err
			}
			return foundOr404(row, spec.NotFound), nil
		})
}

func registerNestedSDEList(
	a huma.API,
	opts Options,
	id, path, summary, idKey, rowsKey, query string,
) {
	registerLegacy(a, sdeIDOperation(id, path, summary),
		func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
			entityID, err := parseID(req.Param("id"))
			if err != nil {
				return legacyPayload{}, err
			}
			rows, err := queryMaps(ctx, opts.DB, query, entityID)
			if err != nil {
				return legacyPayload{}, err
			}
			return jsonPayload(map[string]any{idKey: entityID, rowsKey: rows}), nil
		})
}

func sdeIDOperation(id, path, summary string) huma.Operation {
	return huma.Operation{
		OperationID: id,
		Method:      http.MethodGet,
		Path:        path,
		Summary:     summary,
		Tags:        []string{"sde"},
		Parameters: []*huma.Param{{
			Name:     "id",
			In:       "path",
			Required: true,
			Schema:   &huma.Schema{Type: huma.TypeString},
		}},
	}
}

func paginatedRows(rows []map[string]any, limit int, cursorKey string) legacyPayload {
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	var cursor any
	if len(rows) > 0 {
		cursor = rows[len(rows)-1][cursorKey]
	}
	return jsonPayload(map[string]any{
		"data": rows,
		"pagination": map[string]any{
			"hasMore": hasMore,
			"cursor":  cursor,
		},
	})
}

func foundOr404(row map[string]any, message string) legacyPayload {
	if row == nil {
		return legacyPayload{
			Status: http.StatusNotFound,
			Body:   map[string]any{"error": message},
		}
	}
	return jsonPayload(row)
}

func optionalQueryNumber(req *legacyRequest, name string) (float64, bool) {
	raw := strings.TrimSpace(req.Query.Get(name))
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(raw, 64)
	return value, err == nil
}

func boundedQueryInt(req *legacyRequest, name string, fallback, min, max int) int {
	value := numberOr(req.Query.Get(name), float64(fallback))
	if value < float64(min) {
		value = float64(min)
	}
	if value > float64(max) {
		value = float64(max)
	}
	return int(value)
}

func registerSDEKillRoutes(a huma.API, opts Options) {
	for _, spec := range []struct {
		id      string
		path    string
		summary string
		column  string
	}{
		{"sde-system-kills", "/sde/systems/{id}/kills", "Killmails in a solar system", "k.solar_system_id"},
		{"sde-region-kills", "/sde/regions/{id}/kills", "Killmails in a region", "k.region_id"},
	} {
		registerLegacy(a, sdeIDOperation(spec.id, spec.path, spec.summary),
			func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
				id, err := parseID(req.Param("id"))
				if err != nil {
					return legacyPayload{}, err
				}
				page := parsePagination(req.Query)
				where := []string{spec.column + " = $1"}
				return loadKilllistPage(ctx, opts.DB, where, []any{id}, page)
			})
	}
}
