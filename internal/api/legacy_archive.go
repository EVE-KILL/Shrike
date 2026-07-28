package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const legacyArchiveKillSelect = `
	SELECT k.killmail_id, k.killmail_time,
	       COALESCE(k.total_value, 0)::double precision AS total_value,
	       k.victim_name, k.victim_corp, k.victim_alliance, k.victim_ship,
	       k.system_name, k.security, k.victim_character_id,
	       k.victim_corporation_id, k.victim_alliance_id,
	       k.victim_ship_type_id, k.solar_system_id, k.final_blow_name,
	       (
	         SELECT COUNT(*)::int
	         FROM old_killmail_attackers count_attacker
	         WHERE count_attacker.killmail_id = k.killmail_id
	       ) AS attacker_count,
	       final_blow.character_id AS final_blow_character_id,
	       final_blow.corporation_id AS final_blow_corporation_id,
	       final_blow.alliance_id AS final_blow_alliance_id,
	       final_blow.ship_type_id AS final_blow_ship_type_id,
	       ship.name AS resolved_ship_name,
	       final_ship.name AS final_blow_ship_name,
	       system.system_name AS resolved_system_name,
	       system.security AS resolved_system_security,
	       system.region_id, region.name AS region_name
	FROM old_killmails k
	LEFT JOIN LATERAL (
		SELECT character_id, corporation_id, alliance_id, ship_type_id
		FROM old_killmail_attackers
		WHERE killmail_id = k.killmail_id AND final_blow IS TRUE
		ORDER BY id
		LIMIT 1
	) final_blow ON TRUE
	LEFT JOIN inv_types ship ON ship.type_id = k.victim_ship_type_id
	LEFT JOIN inv_types final_ship ON final_ship.type_id = final_blow.ship_type_id
	LEFT JOIN solar_systems system ON system.solar_system_id = k.solar_system_id
	LEFT JOIN regions region ON region.region_id = system.region_id`

func registerLegacyArchiveRoutes(a huma.API, opts Options) {
	registerLegacyArchiveAutocomplete(a, opts)
	registerLegacyArchiveKills(a, opts)
	registerLegacyArchiveStats(a, opts)
	registerLegacyArchiveTop(a, opts)
	registerLegacyArchiveKill(a, opts)
}

func registerLegacyArchiveAutocomplete(a huma.API, opts Options) {
	handler := routeJSONCache(
		opts,
		5*time.Minute,
		"public, max-age=60, s-maxage=300, stale-while-revalidate=300",
		legacyArchiveAutocompleteHandler(opts),
	)
	registerLegacy(a, huma.Operation{
		OperationID: "legacy-archive-autocomplete",
		Method:      http.MethodGet,
		Path:        "/legacy/autocomplete",
		Summary:     "Search historical killmail names",
		Tags:        []string{"legacy archive"},
	}, handler)
}

type legacyAutocompleteField struct {
	Table string
	Name  string
	ID    string
	Extra string
}

var legacyAutocompleteFields = map[string]legacyAutocompleteField{
	"victim": {
		Table: "old_killmails", Name: "victim_name",
		ID: "victim_character_id",
	},
	"attacker": {
		Table: "old_killmail_attackers", Name: "name", ID: "character_id",
	},
	"corp": {
		Table: "old_killmails", Name: "victim_corp",
		ID: "victim_corporation_id",
	},
	"alliance": {
		Table: "old_killmails", Name: "victim_alliance",
		ID: "victim_alliance_id", Extra: ", 'None', 'NONE'",
	},
	"system": {
		Table: "old_killmails", Name: "system_name", ID: "solar_system_id",
	},
}

func legacyArchiveAutocompleteHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		search := strings.TrimSpace(req.Query.Get("q"))
		if len(search) < 2 {
			return jsonPayload([]map[string]any{}), nil
		}
		field := req.Query.Get("field")
		if field == "" {
			field = "victim"
		}
		spec, ok := legacyAutocompleteFields[field]
		if !ok {
			return jsonPayload([]map[string]any{}), nil
		}
		limit := boundedQueryInt(req, "limit", 10, 0, 20)
		rows, err := queryMaps(ctx, opts.DB, fmt.Sprintf(`
			SELECT DISTINCT %[1]s AS name, %[2]s AS id
			FROM %[3]s
			WHERE %[1]s ILIKE $1
			  AND %[1]s NOT IN ('Unknown', 'unknown', ''%[4]s)
			LIMIT $2`,
			spec.Name, spec.ID, spec.Table, spec.Extra,
		), "%"+search+"%", limit)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(nonNilLegacyArchiveRows(rows)), nil
	}
}

func registerLegacyArchiveKills(a huma.API, opts Options) {
	handler := routeJSONCache(
		opts,
		time.Minute,
		"public, max-age=30, s-maxage=60, stale-while-revalidate=60",
		legacyArchiveKillsHandler(opts),
	)
	registerLegacy(a, huma.Operation{
		OperationID: "legacy-archive-kills",
		Method:      http.MethodGet,
		Path:        "/legacy/kills",
		Summary:     "Historical killmail list",
		Tags:        []string{"legacy archive", "killmails"},
	}, handler)
}

func legacyArchiveKillsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		limit := boundedQueryInt(req, "limit", 50, 10, 100)
		after, err := optionalPositiveInt64(req.Query.Get("after"))
		if err != nil {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid after")
		}
		sortColumn, descending := legacyArchiveSort(req.Query.Get("sort"))
		where := []string{}
		args := []any{}
		add := func(value any) string {
			args = append(args, value)
			return "$" + strconv.Itoa(len(args))
		}
		if after != nil {
			parameter := add(*after)
			operator := ">"
			if descending {
				operator = "<"
			}
			if sortColumn == "killmail_id" {
				where = append(where, "k.killmail_id "+operator+" "+parameter)
			} else {
				where = append(where, fmt.Sprintf(
					`(k.%[1]s, k.killmail_id) %[2]s (
						(SELECT %[1]s FROM old_killmails
						 WHERE killmail_id = %[3]s),
						%[3]s
					)`,
					sortColumn, operator, parameter,
				))
			}
		}
		for _, filter := range []struct {
			Query  string
			Column string
		}{
			{"victim", "k.victim_name"},
			{"corp", "k.victim_corp"},
			{"alliance", "k.victim_alliance"},
			{"system", "k.system_name"},
		} {
			if value := req.Query.Get(filter.Query); value != "" {
				where = append(where,
					filter.Column+" ILIKE "+add("%"+value+"%"))
			}
		}
		if raw := req.Query.Get("ship"); raw != "" {
			ships := strings.Split(raw, ",")
			parts := make([]string, 0, len(ships))
			for _, ship := range ships {
				if ship = strings.TrimSpace(ship); ship != "" {
					parts = append(parts,
						"k.victim_ship ILIKE "+add("%"+ship+"%"))
				}
			}
			if len(parts) != 0 {
				where = append(where, "("+strings.Join(parts, " OR ")+")")
			}
		}
		if value := req.Query.Get("from"); value != "" {
			where = append(where, "k.killmail_time >= "+add(value))
		}
		if value := req.Query.Get("to"); value != "" {
			where = append(where, "k.killmail_time <= "+add(value))
		}
		if value := req.Query.Get("attacker"); value != "" {
			where = append(where, `k.killmail_id IN (
				SELECT killmail_id FROM old_killmail_attackers
				WHERE name ILIKE `+add("%"+value+"%")+")")
		}

		query := legacyArchiveKillSelect
		if len(where) != 0 {
			query += " WHERE " + strings.Join(where, " AND ")
		}
		direction := "ASC"
		if descending {
			direction = "DESC"
		}
		args = append(args, limit+1)
		query += fmt.Sprintf(
			" ORDER BY k.%s %s, k.killmail_id %s LIMIT $%d",
			sortColumn, direction, direction, len(args),
		)
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		hasMore := len(rows) > limit
		if hasMore {
			rows = rows[:limit]
		}
		kills := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			attackerCount := int64OrZero(row["attacker_count"])
			shipName := row["resolved_ship_name"]
			if shipName == nil {
				shipName = row["victim_ship"]
			}
			systemName := row["resolved_system_name"]
			if systemName == nil {
				systemName = row["system_name"]
			}
			security := row["resolved_system_security"]
			if security == nil {
				security = row["security"]
			}
			kills = append(kills, map[string]any{
				"killmail_id":                 row["killmail_id"],
				"killmail_time":               row["killmail_time"],
				"total_value":                 zeroIfNil(row["total_value"]),
				"attacker_count":              attackerCount,
				"is_npc":                      false,
				"is_solo":                     attackerCount == 1,
				"ship_type_id":                row["victim_ship_type_id"],
				"ship_name":                   shipName,
				"ship_group_name":             nil,
				"victim_character_id":         row["victim_character_id"],
				"victim_character_name":       row["victim_name"],
				"victim_corporation_id":       row["victim_corporation_id"],
				"victim_corporation_name":     row["victim_corp"],
				"victim_alliance_id":          row["victim_alliance_id"],
				"victim_alliance_name":        row["victim_alliance"],
				"victim_faction_id":           nil,
				"final_blow_character_id":     row["final_blow_character_id"],
				"final_blow_character_name":   row["final_blow_name"],
				"final_blow_corporation_id":   row["final_blow_corporation_id"],
				"final_blow_corporation_name": nil,
				"final_blow_alliance_id":      row["final_blow_alliance_id"],
				"final_blow_alliance_name":    nil,
				"final_blow_ship_type_id":     row["final_blow_ship_type_id"],
				"final_blow_ship_name":        row["final_blow_ship_name"],
				"solar_system_id":             row["solar_system_id"],
				"solar_system_name":           systemName,
				"solar_system_security":       security,
				"region_id":                   row["region_id"],
				"region_name":                 row["region_name"],
			})
		}
		var cursor any
		if len(kills) != 0 {
			cursor = kills[len(kills)-1]["killmail_id"]
		}
		return jsonPayload(map[string]any{
			"kills": kills, "hasMore": hasMore, "cursor": cursor,
		}), nil
	}
}

func legacyArchiveSort(raw string) (string, bool) {
	parts := strings.Split(raw, "_")
	field, direction := "id", "desc"
	if raw != "" {
		field = parts[0]
		if len(parts) > 1 {
			direction = parts[1]
		} else {
			direction = ""
		}
	}
	column := "killmail_id"
	switch field {
	case "value":
		column = "total_value"
	case "time":
		column = "killmail_time"
	}
	return column, direction != "asc"
}

func registerLegacyArchiveStats(a huma.API, opts Options) {
	handler := routeJSONCache(
		opts,
		time.Hour,
		"public, max-age=300, s-maxage=3600, stale-while-revalidate=3600",
		func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
			row, err := queryMap(ctx, opts.DB, `
				SELECT
				  COUNT(*)::int AS killmails,
				  (COUNT(DISTINCT victim_name)
				    FILTER (WHERE victim_name IS NOT NULL))::int AS characters,
				  (COUNT(DISTINCT victim_corp)
				    FILTER (WHERE victim_corp IS NOT NULL))::int AS corporations,
				  (COUNT(DISTINCT victim_alliance)
				    FILTER (WHERE victim_alliance IS NOT NULL))::int AS alliances
				FROM old_killmails`)
			if err != nil {
				return legacyPayload{}, err
			}
			return jsonPayload(map[string]any{
				"killmails":    int64OrZero(row["killmails"]),
				"characters":   int64OrZero(row["characters"]),
				"corporations": int64OrZero(row["corporations"]),
				"alliances":    int64OrZero(row["alliances"]),
			}), nil
		},
	)
	registerLegacy(a, huma.Operation{
		OperationID: "legacy-archive-stats",
		Method:      http.MethodGet,
		Path:        "/legacy/stats",
		Summary:     "Historical archive counts",
		Tags:        []string{"legacy archive"},
	}, handler)
}

type legacyTopSpec struct {
	Name      string
	ID        string
	Excluded  []string
	Condition string
	Type      string
}

var legacyTopSpecs = map[string]legacyTopSpec{
	"characters": {
		Name: "victim_name", ID: "victim_character_id",
		Excluded: []string{"Unknown", "unknown", ""}, Type: "character",
	},
	"corporations": {
		Name: "victim_corp", ID: "victim_corporation_id",
		Excluded: []string{
			"Unknown", "unknown", "NONE", "None", "",
			"Republic University", "School of Applied Knowledge",
			"Science and Trade Institute", "Royal Amarr Institute",
			"Center for Advanced Studies", "Federal Navy Academy",
			"University of Caille", "Hedion University", "Imperial Academy",
			"Republic Military School", "Pator Tech School",
			"State War Academy",
		}, Type: "corporation",
	},
	"alliances": {
		Name: "victim_alliance", ID: "victim_alliance_id",
		Excluded: []string{"None", "Unknown", "NONE", "unknown", ""},
		Type:     "alliance",
	},
	"ships": {
		Name: "victim_ship", ID: "victim_ship_type_id",
		Excluded:  []string{"Unknown", "unknown", ""},
		Condition: "victim_ship != 'Capsule'", Type: "ship",
	},
	"systems": {
		Name: "system_name", ID: "solar_system_id",
		Excluded: []string{"Unknown", "unknown", ""}, Type: "system",
	},
}

func registerLegacyArchiveTop(a huma.API, opts Options) {
	handler := routeJSONCache(
		opts,
		time.Hour,
		"public, max-age=300, s-maxage=3600, stale-while-revalidate=3600",
		legacyArchiveTopHandler(opts),
	)
	registerLegacy(a, huma.Operation{
		OperationID: "legacy-archive-top",
		Method:      http.MethodGet,
		Path:        "/legacy/top",
		Summary:     "Historical archive top lists",
		Tags:        []string{"legacy archive"},
	}, handler)
}

func legacyArchiveTopHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		dataType := req.Query.Get("dataType")
		if dataType == "" {
			dataType = "characters"
		}
		spec, ok := legacyTopSpecs[dataType]
		if !ok {
			return jsonPayload(map[string]any{
				"entries": []map[string]any{},
			}), nil
		}
		limit := boundedQueryInt(req, "limit", 10, 0, 50)
		args := []any{spec.Excluded}
		where := []string{
			spec.Name + " IS NOT NULL",
			spec.Name + " != ALL($1::text[])",
		}
		if spec.Condition != "" {
			where = append(where, spec.Condition)
		}
		if raw := strings.TrimSpace(req.Query.Get("year")); raw != "" {
			year, err := strconv.ParseFloat(raw, 64)
			if err == nil && year != 0 && !math.IsNaN(year) {
				args = append(args, year)
				where = append(where,
					fmt.Sprintf("EXTRACT(YEAR FROM killmail_time) = $%d", len(args)))
			}
		}
		args = append(args, limit)
		rows, err := queryMaps(ctx, opts.DB, fmt.Sprintf(`
			SELECT %[1]s AS name, %[2]s AS id, COUNT(*)::int AS count
			FROM old_killmails
			WHERE %[3]s
			GROUP BY %[1]s, %[2]s
			ORDER BY count DESC
			LIMIT $%[4]d`,
			spec.Name, spec.ID, strings.Join(where, " AND "), len(args),
		), args...)
		if err != nil {
			return legacyPayload{}, err
		}
		entries := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			id := row["id"]
			if id == nil {
				id = 0
			}
			name := row["name"]
			if name == nil {
				name = "Unknown"
			}
			entries = append(entries, map[string]any{
				"id": id, "name": name, "count": row["count"], "type": spec.Type,
			})
		}
		return jsonPayload(map[string]any{"entries": entries}), nil
	}
}

func registerLegacyArchiveKill(a huma.API, opts Options) {
	handler := routeJSONCache(
		opts,
		time.Hour,
		"public, max-age=3600, s-maxage=3600",
		func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
			id, err := parseUniverseID(req.Param("id"))
			if err != nil {
				return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid ID")
			}
			results, err := queryMapsConcurrent(ctx, opts.DB,
				databaseQuery{
					SQL:  `SELECT * FROM old_killmails WHERE killmail_id = $1`,
					Args: []any{id},
				},
				databaseQuery{
					SQL: `
						SELECT * FROM old_killmail_attackers
						WHERE killmail_id = $1
						ORDER BY final_blow DESC, security_status DESC`,
					Args: []any{id},
				},
				databaseQuery{
					SQL: `
						SELECT * FROM old_killmail_items
						WHERE killmail_id = $1 ORDER BY name`,
					Args: []any{id},
				},
			)
			if err != nil {
				return legacyPayload{}, err
			}
			if len(results[0]) == 0 {
				return legacyPayload{}, apiError(
					http.StatusNotFound, "Killmail not found",
				)
			}
			return jsonPayload(map[string]any{
				"kill":      results[0][0],
				"attackers": nonNilLegacyArchiveRows(results[1]),
				"items":     nonNilLegacyArchiveRows(results[2]),
			}), nil
		},
	)
	registerLegacy(a, huma.Operation{
		OperationID: "legacy-archive-kill",
		Method:      http.MethodGet,
		Path:        "/legacy/kill/{id}",
		Summary:     "Historical killmail detail",
		Tags:        []string{"legacy archive", "killmails"},
	}, handler)
}

func nonNilLegacyArchiveRows(rows []map[string]any) []map[string]any {
	if rows == nil {
		return []map[string]any{}
	}
	return rows
}
