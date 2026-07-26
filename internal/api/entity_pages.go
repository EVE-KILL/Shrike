package api

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const (
	entityPageCharacter   = "character"
	entityPageCorporation = "corporation"
	entityPageAlliance    = "alliance"
	entityPageFaction     = "faction"
)

type entityPageRoute struct {
	Name      string
	Canonical string
	Aliases   []string
	Types     map[string]bool
	Summary   string
	TTL       time.Duration
	Load      func(context.Context, Options, string, int64, *legacyRequest) (any, error)
}

var entityPageRoutes = []entityPageRoute{
	{
		Name: "detail", Canonical: "/entities/{type}/{id}",
		Aliases: []string{
			"/character/{id}", "/corporation/{id}",
			"/alliance/{id}", "/faction/{id}",
		},
		Types: entityPageTypeSet(
			entityPageCharacter, entityPageCorporation,
			entityPageAlliance, entityPageFaction,
		),
		Summary: "Entity profile", TTL: 2 * time.Minute,
		Load: loadEntityPageDetail,
	},
	{
		Name: "stats", Canonical: "/entities/{type}/{id}/stats",
		Aliases: []string{
			"/character/{id}/stats", "/corporation/{id}/stats",
			"/alliance/{id}/stats",
		},
		Types: entityPageTypeSet(
			entityPageCharacter, entityPageCorporation, entityPageAlliance,
		),
		Summary: "Entity dashboard statistics",
		TTL:     2 * time.Minute, Load: loadEntityPageStats,
	},
	{
		Name: "intel", Canonical: "/entities/{type}/{id}/intel",
		Aliases: []string{
			"/character/{id}/intel", "/corporation/{id}/intel",
			"/alliance/{id}/intel",
		},
		Types: entityPageTypeSet(
			entityPageCharacter, entityPageCorporation, entityPageAlliance,
		),
		Summary: "Entity intelligence",
		TTL:     time.Hour, Load: loadEntityPageIntel,
	},
	{
		Name: "achievements", Canonical: "/entities/{type}/{id}/achievements",
		Aliases: []string{"/character/{id}/achievements"},
		Types:   entityPageTypeSet(entityPageCharacter),
		Summary: "Character achievements",
		TTL:     2 * time.Minute, Load: loadEntityPageAchievements,
	},
	{
		Name: "members", Canonical: "/entities/{type}/{id}/members",
		Aliases: []string{
			"/corporation/{id}/members", "/alliance/{id}/members",
		},
		Types:   entityPageTypeSet(entityPageCorporation, entityPageAlliance),
		Summary: "Organization members",
		TTL:     2 * time.Minute, Load: loadEntityPageMembers,
	},
	{
		Name: "corporations", Canonical: "/entities/{type}/{id}/corporations",
		Aliases: []string{"/alliance/{id}/corporations"},
		Types:   entityPageTypeSet(entityPageAlliance),
		Summary: "Alliance corporations",
		TTL:     2 * time.Minute, Load: loadEntityPageCorporations,
	},
	{
		Name: "killlist", Canonical: "/entities/{type}/{id}/killlist",
		Aliases: []string{"/entity/{type}/{id}/killlist"},
		Types: entityPageTypeSet(
			entityPageCharacter, entityPageCorporation,
			entityPageAlliance, entityPageFaction,
		),
		Summary: "Entity killmail list",
		TTL:     time.Minute, Load: loadEntityPageKilllist,
	},
	{
		Name: "most-valuable", Canonical: "/entities/{type}/{id}/most-valuable",
		Aliases: []string{"/entity/{type}/{id}/most-valuable"},
		Types: entityPageTypeSet(
			entityPageCharacter, entityPageCorporation, entityPageAlliance,
		),
		Summary: "Most valuable entity kills",
		TTL:     5 * time.Minute, Load: loadEntityPageMostValuable,
	},
	{
		Name: "ship-classes", Canonical: "/entities/{type}/{id}/ship-classes",
		Aliases: []string{"/entity/{type}/{id}/ship-classes"},
		Types: entityPageTypeSet(
			entityPageCharacter, entityPageCorporation, entityPageAlliance,
		),
		Summary: "Entity losses by ship class",
		TTL:     30 * time.Minute, Load: loadEntityPageShipClasses,
	},
	{
		Name: "top-lists", Canonical: "/entities/{type}/{id}/top-lists",
		Aliases: []string{"/entity/{type}/{id}/top-lists"},
		Types: entityPageTypeSet(
			entityPageCharacter, entityPageCorporation, entityPageAlliance,
		),
		Summary: "Entity interaction top lists",
		TTL:     30 * time.Minute, Load: loadEntityPageTopLists,
	},
}

func entityPageTypeSet(values ...string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

// registerEntityPageRoutes installs the consolidated entity-page contract.
// The singular aliases are deliberately limited to routes the current Nuxt
// application calls. Dead /top and /fits wrappers are not carried forward.
func registerEntityPageRoutes(a huma.API, opts Options) {
	for _, route := range entityPageRoutes {
		route := route
		registerLegacy(a, huma.Operation{
			OperationID: "entity-page-" + route.Name,
			Method:      http.MethodGet,
			Path:        route.Canonical,
			Summary:     route.Summary,
			Tags:        []string{"entities"},
		}, cacheEntityPageRoute(opts, route, "", entityPageHandler(opts, route, "")))

		for _, alias := range route.Aliases {
			fixedType := entityTypeFromAlias(alias)
			operationType := fixedType
			if operationType == "" {
				operationType = "generic"
			}
			registerLegacy(a, huma.Operation{
				OperationID: "entity-page-" + route.Name + "-" + operationType + "-compat",
				Method:      http.MethodGet,
				Path:        alias,
				Summary:     route.Summary,
				Tags:        []string{"entities"},
			}, cacheEntityPageRoute(
				opts, route, fixedType, entityPageHandler(opts, route, fixedType),
			))
		}
	}

	resolve := routeJSONCache(
		opts, 24*time.Hour,
		"public, max-age=3600, s-maxage=86400, stale-while-revalidate=86400",
		entityResolveHandler(opts),
	)
	for i, path := range []string{"/entities/resolve", "/entity/resolve"} {
		suffix := ""
		if i > 0 {
			suffix = "-compat"
		}
		registerLegacy(a, huma.Operation{
			OperationID: "entity-resolve" + suffix,
			Method:      http.MethodGet,
			Path:        path,
			Summary:     "Resolve an entity name",
			Tags:        []string{"entities"},
		}, resolve)
	}
}

func entityTypeFromAlias(path string) string {
	for _, kind := range []string{
		entityPageCharacter, entityPageCorporation,
		entityPageAlliance, entityPageFaction,
	} {
		if strings.HasPrefix(path, "/"+kind+"/") {
			return kind
		}
	}
	return ""
}

func cacheEntityPageRoute(
	opts Options,
	route entityPageRoute,
	fixedType string,
	next legacyHandler,
) legacyHandler {
	cacheControl := fmt.Sprintf(
		"public, max-age=%d, s-maxage=%d, stale-while-revalidate=%d",
		maxInt64(30, int64(route.TTL.Seconds()/4)),
		maxInt64(60, int64(route.TTL.Seconds())),
		maxInt64(60, int64(route.TTL.Seconds())),
	)
	return routeJSONCacheBy(opts, route.TTL, cacheControl, func(req *legacyRequest) string {
		kind := fixedType
		if kind == "" {
			kind = req.Param("type")
		}
		return "entities:" + route.Name + ":" + kind + ":" +
			req.Param("id") + "?" + req.Query.Encode()
	}, next)
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func entityPageHandler(
	opts Options,
	route entityPageRoute,
	fixedType string,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		kind := fixedType
		if kind == "" {
			kind = strings.ToLower(strings.TrimSpace(req.Param("type")))
		}
		if !route.Types[kind] {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid entity type")
		}
		id, err := parseUniverseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := route.Load(ctx, opts, kind, id, req)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(body), nil
	}
}

type entityTable struct {
	Table, IDColumn, NameColumn string
}

var entityResolveTables = map[string]entityTable{
	"character":   {"characters", "character_id", "name"},
	"corporation": {"corporations", "corporation_id", "name"},
	"alliance":    {"alliances", "alliance_id", "name"},
	"faction":     {"factions", "faction_id", "name"},
	"system":      {"solar_systems", "solar_system_id", "system_name"},
	"constellation": {
		"constellations", "constellation_id", "constellation_name",
	},
	"region": {"regions", "region_id", "name"},
	"type":   {"inv_types", "type_id", "name"},
}

func entityResolveHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		kind := strings.ToLower(strings.TrimSpace(req.Query.Get("type")))
		table, ok := entityResolveTables[kind]
		if !ok {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid type")
		}
		id, err := parseUniverseID(req.Query.Get("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		// Identifiers are selected exclusively from the fixed map above.
		row, err := queryMap(ctx, opts.DB, fmt.Sprintf(
			"SELECT %s AS name FROM %s WHERE %s = $1 LIMIT 1",
			table.NameColumn, table.Table, table.IDColumn,
		), id)
		if err != nil {
			return legacyPayload{}, err
		}
		name, _ := stringValue(row["name"])
		if name == "" {
			return legacyPayload{}, apiError(http.StatusNotFound, "Not found")
		}
		return jsonPayload(map[string]any{
			"type": kind, "id": id, "name": name,
		}), nil
	}
}

func loadEntityPageDetail(
	ctx context.Context,
	opts Options,
	kind string,
	id int64,
	_ *legacyRequest,
) (any, error) {
	switch kind {
	case entityPageCharacter:
		return loadCharacterPage(ctx, opts.DB, id)
	case entityPageCorporation:
		return loadCorporationPage(ctx, opts.DB, id)
	case entityPageAlliance:
		return loadAlliancePage(ctx, opts.DB, id)
	case entityPageFaction:
		return loadFactionPage(ctx, opts.DB, id)
	default:
		return nil, apiError(http.StatusBadRequest, "Invalid entity type")
	}
}

const entityPageStatsSQL = `
	SELECT
		COALESCE(SUM(kills) FILTER (WHERE period_type = 2), 0)::bigint AS kills,
		COALESCE(SUM(losses) FILTER (WHERE period_type = 2), 0)::bigint AS losses,
		COALESCE(SUM(solo_kills) FILTER (WHERE period_type = 2), 0)::bigint AS solo_kills,
		COALESCE(SUM(solo_losses) FILTER (WHERE period_type = 2), 0)::bigint AS solo_losses,
		COALESCE(SUM(npc_losses) FILTER (WHERE period_type = 2), 0)::bigint AS npc_losses,
		COALESCE(SUM(final_blows) FILTER (WHERE period_type = 2), 0)::bigint AS final_blows,
		COALESCE(SUM(points) FILTER (WHERE period_type = 2), 0)::bigint AS points,
		COALESCE(SUM(isk_destroyed) FILTER (WHERE period_type = 2), 0)::double precision AS isk_destroyed,
		COALESCE(SUM(isk_lost) FILTER (WHERE period_type = 2), 0)::double precision AS isk_lost,
		COALESCE(SUM(damage_dealt) FILTER (WHERE period_type = 2), 0)::bigint AS damage_dealt,
		COALESCE(SUM(damage_taken) FILTER (WHERE period_type = 2), 0)::bigint AS damage_taken,
		COALESCE(SUM(kills) FILTER (
			WHERE period_type = 0 AND period_start >= CURRENT_DATE - 90
		), 0)::bigint AS recent_kills,
		COALESCE(SUM(losses) FILTER (
			WHERE period_type = 0 AND period_start >= CURRENT_DATE - 90
		), 0)::bigint AS recent_losses,
		COALESCE(SUM(isk_destroyed) FILTER (
			WHERE period_type = 0 AND period_start >= CURRENT_DATE - 90
		), 0)::double precision AS recent_isk_destroyed,
		COALESCE(SUM(isk_lost) FILTER (
			WHERE period_type = 0 AND period_start >= CURRENT_DATE - 90
		), 0)::double precision AS recent_isk_lost
	FROM stats
	WHERE entity_type = $1 AND entity_id = $2
	  AND (period_type = 2 OR (
		period_type = 0 AND period_start >= CURRENT_DATE - 90
	  ))`

func entityDetailStats(row map[string]any, damage bool) map[string]any {
	kills, _ := int64Value(row["kills"])
	losses, _ := int64Value(row["losses"])
	iskDestroyed, _ := float64Value(row["isk_destroyed"])
	iskLost, _ := float64Value(row["isk_lost"])
	result := map[string]any{
		"kills": kills, "losses": losses,
		"solo_kills":    int64OrZero(row["solo_kills"]),
		"npc_losses":    int64OrZero(row["npc_losses"]),
		"isk_destroyed": iskDestroyed, "isk_lost": iskLost,
		"points":         int64OrZero(row["points"]),
		"final_blows":    int64OrZero(row["final_blows"]),
		"efficiency":     efficiency(kills, losses),
		"isk_efficiency": iskEfficiency(iskDestroyed, iskLost),
	}
	if damage {
		result["damage_dealt"] = int64OrZero(row["damage_dealt"])
		result["damage_taken"] = int64OrZero(row["damage_taken"])
	}
	return result
}

func entityRecentStats(row map[string]any) map[string]any {
	return map[string]any{
		"kills":         int64OrZero(row["recent_kills"]),
		"losses":        int64OrZero(row["recent_losses"]),
		"isk_destroyed": float64OrZero(row["recent_isk_destroyed"]),
		"isk_lost":      float64OrZero(row["recent_isk_lost"]),
	}
}

func int64OrZero(value any) int64 {
	out, _ := int64Value(value)
	return out
}

func float64OrZero(value any) float64 {
	out, _ := float64Value(value)
	return out
}

func loadCharacterPage(
	ctx context.Context,
	db Database,
	id int64,
) (map[string]any, error) {
	results, err := queryMapsConcurrent(ctx, db,
		databaseQuery{
			SQL: `
				SELECT c.character_id, c.name, c.description,
				       c.custom_description, c.custom_description_format,
				       c.birthday, c.gender,
				       COALESCE(c.security_status, 0)::real AS security_status,
				       c.title, c.last_active,
				       c.corporation_id, corp.name AS corporation_name,
				       corp.ticker AS corporation_ticker, corp.palette,
				       c.alliance_id, ally.name AS alliance_name,
				       ally.ticker AS alliance_ticker,
				       c.faction_id, faction.name AS faction_name,
				       race.race_name, bloodline.bloodline_name
				FROM characters c
				LEFT JOIN corporations corp
				  ON corp.corporation_id = c.corporation_id
				LEFT JOIN alliances ally ON ally.alliance_id = c.alliance_id
				LEFT JOIN factions faction ON faction.faction_id = c.faction_id
				LEFT JOIN races race ON race.race_id = c.race_id
				LEFT JOIN bloodlines bloodline
				  ON bloodline.bloodline_id = c.bloodline_id
				WHERE c.character_id = $1
				LIMIT 1`,
			Args: []any{id},
		},
		databaseQuery{SQL: entityPageStatsSQL, Args: []any{entityCharacter, id}},
		databaseQuery{
			SQL: `
				WITH flown AS (
					SELECT dim_id, SUM(kills)::bigint AS kills
					FROM stats_breakdowns
					WHERE entity_type = 0 AND entity_id = $1
					  AND dim_category = 0 AND period_type = 2
					GROUP BY dim_id
					ORDER BY kills DESC
					LIMIT 10
				), lost AS (
					SELECT dim_id, SUM(losses)::bigint AS losses
					FROM stats_breakdowns
					WHERE entity_type = 0 AND entity_id = $1
					  AND dim_category = 1 AND period_type = 2
					GROUP BY dim_id
				)
				SELECT flown.dim_id AS ship_type_id,
				       COALESCE(t.name, 'Unknown') AS ship_name,
				       flown.kills, COALESCE(lost.losses, 0)::bigint AS losses
				FROM flown
				LEFT JOIN lost USING (dim_id)
				LEFT JOIN inv_types t ON t.type_id = flown.dim_id
				ORDER BY flown.kills DESC`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				SELECT h.record_id, h.corporation_id,
				       COALESCE(c.name, 'Unknown') AS corporation_name,
				       COALESCE(c.ticker, '???') AS corporation_ticker,
				       h.start_date
				FROM character_corporation_history h
				LEFT JOIN corporations c
				  ON c.corporation_id = h.corporation_id
				WHERE h.character_id = $1
				ORDER BY h.start_date DESC`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				WITH stints AS (
					SELECT corporation_id, start_date,
					       LEAD(start_date) OVER (ORDER BY start_date) AS end_date,
					       ROW_NUMBER() OVER (ORDER BY start_date DESC) AS rn
					FROM character_corporation_history
					WHERE character_id = $1
				), daily AS (
					SELECT s.rn, COALESCE(SUM(d.kills), 0) AS kills,
					       COALESCE(SUM(d.losses), 0) AS losses
					FROM stints s
					LEFT JOIN stats d
					  ON d.entity_type = 0 AND d.entity_id = $1
					 AND d.period_type = 0
					 AND d.period_start >= GREATEST(
					   s.start_date::date,
					   (CURRENT_DATE - INTERVAL '365 days')::date
					 )
					 AND d.period_start < COALESCE(
					   s.end_date::date, CURRENT_DATE + INTERVAL '1 day'
					 )
					GROUP BY s.rn
				), monthly AS (
					SELECT s.rn, COALESCE(SUM(m.kills), 0) AS kills,
					       COALESCE(SUM(m.losses), 0) AS losses
					FROM stints s
					LEFT JOIN stats m
					  ON m.entity_type = 0 AND m.entity_id = $1
					 AND m.period_type = 1
					 AND m.period_start >= GREATEST(
					   s.start_date::date,
					   (CURRENT_DATE - INTERVAL '18 months')::date
					 )
					 AND m.period_start < LEAST(
					   COALESCE(s.end_date::date, CURRENT_DATE + INTERVAL '1 day'),
					   (CURRENT_DATE - INTERVAL '365 days')::date
					 )
					GROUP BY s.rn
				), yearly AS (
					SELECT s.rn, COALESCE(SUM(y.kills), 0) AS kills,
					       COALESCE(SUM(y.losses), 0) AS losses
					FROM stints s
					LEFT JOIN stats y
					  ON y.entity_type = 0 AND y.entity_id = $1
					 AND y.period_type = 2
					 AND y.period_start >= s.start_date::date
					 AND y.period_start < LEAST(
					   COALESCE(s.end_date::date, CURRENT_DATE + INTERVAL '1 day'),
					   (CURRENT_DATE - INTERVAL '18 months')::date
					 )
					GROUP BY s.rn
				)
				SELECT d.rn,
				       (d.kills + m.kills + y.kills)::bigint AS kills,
				       (d.losses + m.losses + y.losses)::bigint AS losses
				FROM daily d
				JOIN monthly m USING (rn)
				JOIN yearly y USING (rn)
				ORDER BY d.rn`,
			Args: []any{id},
		},
	)
	if err != nil {
		return nil, err
	}
	character := firstEntityPageRow(results[0])
	if character == nil {
		return nil, apiError(http.StatusNotFound, "Character not found")
	}
	character["custom_description_html"] = renderEntityBio(
		stringOrEmpty(character["custom_description"]),
		stringOrEmpty(character["custom_description_format"]),
	)

	stats := firstOrEmpty(results[1])
	topShips := nonNilEntityPageRows(results[2])
	history := nonNilEntityPageRows(results[3])
	stints := make(map[int64]map[string]any, len(results[4]))
	for _, row := range results[4] {
		stints[int64OrZero(row["rn"])] = row
	}
	for i, row := range history {
		stint := stints[int64(i+1)]
		row["kills"] = int64OrZero(stint["kills"])
		row["losses"] = int64OrZero(stint["losses"])
		delete(row, "record_id")
	}

	return map[string]any{
		"character":                character,
		"stats":                    entityDetailStats(stats, true),
		"recentStats":              entityRecentStats(stats),
		"topShips":                 topShips,
		"corporationHistoryQueued": len(history) == 0,
		"corporationHistory":       history,
	}, nil
}

func loadCorporationPage(
	ctx context.Context,
	db Database,
	id int64,
) (map[string]any, error) {
	results, err := queryMapsConcurrent(ctx, db,
		databaseQuery{
			SQL: `
				SELECT c.corporation_id, c.name, c.ticker, c.description,
				       c.custom_description, c.custom_description_format,
				       c.date_founded, c.url,
				       COALESCE(c.member_count, 0)::int AS member_count,
				       COALESCE(c.tax_rate, 0)::real AS tax_rate,
				       c.lp_tax_rate, COALESCE(c.war_eligible, false) AS war_eligible,
				       c.friendly_fire, c.state, c.type, c.palette,
				       c.ceo_id, ceo.name AS ceo_name,
				       c.creator_id, creator.name AS creator_name,
				       c.alliance_id, ally.name AS alliance_name,
				       ally.ticker AS alliance_ticker,
				       c.faction_id, faction.name AS faction_name
				FROM corporations c
				LEFT JOIN characters ceo ON ceo.character_id = c.ceo_id
				LEFT JOIN characters creator
				  ON creator.character_id = c.creator_id
				LEFT JOIN alliances ally ON ally.alliance_id = c.alliance_id
				LEFT JOIN factions faction ON faction.faction_id = c.faction_id
				WHERE c.corporation_id = $1
				LIMIT 1`,
			Args: []any{id},
		},
		databaseQuery{SQL: entityPageStatsSQL, Args: []any{entityCorporation, id}},
		databaseQuery{
			SQL: `
				SELECT h.alliance_id,
				       CASE WHEN h.alliance_id IS NULL THEN NULL
				            ELSE COALESCE(a.name, 'Unknown') END AS alliance_name,
				       a.ticker AS alliance_ticker, h.start_date
				FROM corporation_alliance_history h
				LEFT JOIN alliances a ON a.alliance_id = h.alliance_id
				WHERE h.corporation_id = $1
				ORDER BY h.start_date DESC`,
			Args: []any{id},
		},
	)
	if err != nil {
		return nil, err
	}
	corporation := firstEntityPageRow(results[0])
	if corporation == nil {
		return nil, apiError(http.StatusNotFound, "Corporation not found")
	}
	corporation["custom_description_html"] = renderEntityBio(
		stringOrEmpty(corporation["custom_description"]),
		stringOrEmpty(corporation["custom_description_format"]),
	)
	stats := firstOrEmpty(results[1])
	return map[string]any{
		"corporation":     corporation,
		"stats":           entityDetailStats(stats, false),
		"recentStats":     entityRecentStats(stats),
		"allianceHistory": nonNilEntityPageRows(results[2]),
	}, nil
}

func loadAlliancePage(
	ctx context.Context,
	db Database,
	id int64,
) (map[string]any, error) {
	results, err := queryMapsConcurrent(ctx, db,
		databaseQuery{
			SQL: `
				SELECT a.alliance_id, a.name, a.ticker,
				       a.custom_description, a.custom_description_format,
				       a.date_founded,
				       (SELECT COUNT(*)::int FROM corporations c
				        WHERE c.alliance_id = a.alliance_id
				          AND c.deleted IS NOT TRUE) AS corporation_count,
				       (SELECT COUNT(*)::int FROM characters c
				        WHERE c.alliance_id = a.alliance_id
				          AND c.deleted IS NOT TRUE) AS member_count,
				       a.creator_id, creator.name AS creator_name,
				       a.executor_corporation_id,
				       executor.name AS executor_name,
				       executor.ticker AS executor_ticker,
				       executor.palette,
				       a.faction_id, faction.name AS faction_name
				FROM alliances a
				LEFT JOIN characters creator
				  ON creator.character_id = a.creator_id
				LEFT JOIN corporations executor
				  ON executor.corporation_id = a.executor_corporation_id
				LEFT JOIN factions faction ON faction.faction_id = a.faction_id
				WHERE a.alliance_id = $1
				LIMIT 1`,
			Args: []any{id},
		},
		databaseQuery{SQL: entityPageStatsSQL, Args: []any{entityAlliance, id}},
	)
	if err != nil {
		return nil, err
	}
	alliance := firstEntityPageRow(results[0])
	if alliance == nil {
		return nil, apiError(http.StatusNotFound, "Alliance not found")
	}
	alliance["custom_description_html"] = renderEntityBio(
		stringOrEmpty(alliance["custom_description"]),
		stringOrEmpty(alliance["custom_description_format"]),
	)
	stats := firstOrEmpty(results[1])
	return map[string]any{
		"alliance":    alliance,
		"stats":       entityDetailStats(stats, false),
		"recentStats": entityRecentStats(stats),
	}, nil
}

func loadFactionPage(
	ctx context.Context,
	db Database,
	id int64,
) (map[string]any, error) {
	if id < 500000 || id > 599999 {
		return nil, apiError(http.StatusNotFound, "Faction not found")
	}
	results, err := queryMapsConcurrent(ctx, db,
		databaseQuery{
			SQL: `
				SELECT faction_id, name, description, corporation_id,
				       militia_corporation_id, solar_system_id,
				       COALESCE(station_count, 0)::int AS station_count,
				       COALESCE(station_system_count, 0)::int AS station_system_count
				FROM factions WHERE faction_id = $1 LIMIT 1`,
			Args: []any{id},
		},
		databaseQuery{
			SQL: `
				SELECT COUNT(*)::bigint AS losses,
				       COALESCE(SUM(total_value), 0)::double precision AS isk_lost,
				       COUNT(*) FILTER (
				         WHERE killmail_time >= NOW() - INTERVAL '90 days'
				       )::bigint AS recent_losses,
				       COALESCE(SUM(total_value) FILTER (
				         WHERE killmail_time >= NOW() - INTERVAL '90 days'
				       ), 0)::double precision AS recent_isk_lost
				FROM killmails WHERE victim_faction_id = $1`,
			Args: []any{id},
		},
	)
	if err != nil {
		return nil, err
	}
	faction := firstEntityPageRow(results[0])
	if faction == nil {
		return nil, apiError(http.StatusNotFound, "Faction not found")
	}
	stats := firstOrEmpty(results[1])
	return map[string]any{
		"faction": faction,
		"stats": map[string]any{
			"losses":   int64OrZero(stats["losses"]),
			"isk_lost": float64OrZero(stats["isk_lost"]),
		},
		"recentStats": map[string]any{
			"losses":   int64OrZero(stats["recent_losses"]),
			"isk_lost": float64OrZero(stats["recent_isk_lost"]),
		},
	}, nil
}

func firstOrEmpty(rows []map[string]any) map[string]any {
	if len(rows) == 0 {
		return map[string]any{}
	}
	return rows[0]
}

func firstEntityPageRow(rows []map[string]any) map[string]any {
	if len(rows) == 0 {
		return nil
	}
	return rows[0]
}

func nonNilEntityPageRows(rows []map[string]any) []map[string]any {
	if rows == nil {
		return []map[string]any{}
	}
	return rows
}

func stringOrEmpty(value any) string {
	out, _ := stringValue(value)
	return out
}

func loadEntityPageAchievements(
	ctx context.Context,
	opts Options,
	_ string,
	id int64,
	_ *legacyRequest,
) (any, error) {
	rows, err := queryMaps(ctx, opts.DB, `
		SELECT achievement_id,
		       COALESCE(current_count, 0)::int AS current_count,
		       threshold,
		       COALESCE(completion_tiers, 0)::int AS completion_tiers,
		       COALESCE(is_completed, false) AS is_completed,
		       COALESCE(points, 0)::int AS points,
		       completed_at
		FROM entity_achievements
		WHERE entity_id = $1
		ORDER BY points DESC, is_completed DESC`, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"achievements": nonNilEntityPageRows(rows)}, nil
}

var (
	entityMarkdownLink = regexp.MustCompile(`\[([^\]\n]+)\]\((https?://[^)\s]+|/[^)\s]*)\)`)
	entityMarkdownBold = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	entityMarkdownEm   = regexp.MustCompile(`\*([^*\n]+)\*`)
	entityEVEURL       = regexp.MustCompile(`(?is)<url=([^>]+)>(.*?)</url>`)
	entityEVEURLPlain  = regexp.MustCompile(`(?is)<url>(.*?)</url>`)
	entityEVEShowInfo  = regexp.MustCompile(`(?is)<a\s+href=["']showinfo:(\d+)(?://(\d+))?["'][^>]*>([^<]+)</a>`)
	entityEVEKill      = regexp.MustCompile(`(?is)<a\s+href=["']killReport:(\d+)(?::[A-Fa-f0-9]+)?["'][^>]*>([^<]+)</a>`)
	entityEVEWar       = regexp.MustCompile(`(?is)<a\s+href=["']warReport:(\d+)["'][^>]*>([^<]+)</a>`)
)

// renderEntityBio intentionally emits a smaller safe subset than the old
// DOMPurify pipeline. It preserves links and the common Markdown emphasis
// users rely on, but never forwards raw user HTML, inline CSS, images, embeds,
// or event attributes. That is a deliberate security tightening and keeps the
// resulting frontend contract (sanitized HTML or null) intact.
func renderEntityBio(text, format string) any {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if format == "eve_html" {
		return renderSafeEVEBio(text)
	}
	escaped := html.EscapeString(text)
	escaped = entityMarkdownLink.ReplaceAllStringFunc(escaped, func(match string) string {
		parts := entityMarkdownLink.FindStringSubmatch(match)
		return safeEntityAnchor(parts[2], parts[1])
	})
	escaped = entityMarkdownBold.ReplaceAllString(escaped, "<strong>$1</strong>")
	escaped = entityMarkdownEm.ReplaceAllString(escaped, "<em>$1</em>")
	paragraphs := strings.Split(strings.ReplaceAll(escaped, "\r\n", "\n"), "\n\n")
	for i, paragraph := range paragraphs {
		paragraphs[i] = "<p>" + strings.ReplaceAll(paragraph, "\n", "<br>") + "</p>"
	}
	return strings.Join(paragraphs, "\n")
}

func renderSafeEVEBio(text string) string {
	type replacement struct {
		token, html string
	}
	var replacements []replacement
	tokenize := func(re *regexp.Regexp, input string, build func([]string) string) string {
		return re.ReplaceAllStringFunc(input, func(match string) string {
			parts := re.FindStringSubmatch(match)
			token := fmt.Sprintf("ENTITYBIOLINKTOKEN%dX", len(replacements))
			replacements = append(replacements, replacement{token, build(parts)})
			return token
		})
	}
	text = tokenize(entityEVEShowInfo, text, func(parts []string) string {
		typeID, _ := strconv.ParseInt(parts[1], 10, 64)
		targetID := parts[2]
		path := "/item/" + parts[1]
		if targetID != "" {
			switch typeID {
			case 2:
				path = "/corporation/" + targetID
			case 3:
				path = "/region/" + targetID
			case 4:
				path = "/constellation/" + targetID
			case 5:
				path = "/system/" + targetID
			case 16159:
				path = "/alliance/" + targetID
			default:
				path = "/item/" + targetID
			}
		}
		return safeEntityAnchor(path, html.EscapeString(parts[3]))
	})
	text = tokenize(entityEVEKill, text, func(parts []string) string {
		return safeEntityAnchor("/kill/"+parts[1], html.EscapeString(parts[2]))
	})
	text = tokenize(entityEVEWar, text, func(parts []string) string {
		return safeEntityAnchor("/war/"+parts[1], html.EscapeString(parts[2]))
	})
	text = tokenize(entityEVEURL, text, func(parts []string) string {
		return safeEntityAnchor(html.UnescapeString(parts[1]), html.EscapeString(parts[2]))
	})
	text = tokenize(entityEVEURLPlain, text, func(parts []string) string {
		value := html.UnescapeString(parts[1])
		return safeEntityAnchor(value, html.EscapeString(parts[1]))
	})

	text = html.EscapeString(text)
	for _, item := range replacements {
		text = strings.ReplaceAll(text, item.token, item.html)
	}
	return strings.ReplaceAll(
		strings.ReplaceAll(text, "\r\n", "\n"), "\n", "<br>",
	)
}

func safeEntityAnchor(rawURL, label string) string {
	url := strings.TrimSpace(html.UnescapeString(rawURL))
	lower := strings.ToLower(url)
	if !(strings.HasPrefix(url, "/") ||
		strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(url, "#")) {
		return label
	}
	attrs := ` rel="noopener noreferrer nofollow"`
	if !strings.HasPrefix(url, "/") {
		attrs += ` target="_blank"`
	}
	return `<a href="` + html.EscapeString(url) + `"` + attrs + `>` + label + `</a>`
}

func sortedMapKeys(values map[int64]bool) []int64 {
	out := make([]int64, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
