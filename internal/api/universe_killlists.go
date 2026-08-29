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
	"github.com/eve-kill/shrike/internal/stats"
)

const (
	universeKilllistCacheTTL     = time.Minute
	universeMostValuableCacheTTL = 5 * time.Minute
)

type locationKilllistRoute struct {
	Name        string
	Canonical   string
	Alias       string
	Column      string
	EntityType  stats.EntityType
	MemberQuery string
}

var locationKilllistRoutes = []locationKilllistRoute{
	{
		Name: "region", Canonical: "/universe/regions/{id}/killmails",
		Alias: "/region/{id}/killlist", Column: "region_id",
		EntityType: stats.EntityRegion,
	},
	{
		Name:       "constellation",
		Canonical:  "/universe/constellations/{id}/killmails",
		Alias:      "/constellation/{id}/killlist",
		Column:     "constellation_id",
		EntityType: stats.EntitySystem,
		MemberQuery: `
			SELECT solar_system_id FROM solar_systems
			WHERE constellation_id = $2`,
	},
	{
		Name: "system", Canonical: "/universe/systems/{id}/killmails",
		Alias: "/system/{id}/killlist", Column: "solar_system_id",
		EntityType: stats.EntitySystem,
	},
}

func registerUniverseKilllistRoutes(a huma.API, opts Options) {
	for _, route := range locationKilllistRoutes {
		handler := locationKilllistHandler(opts, route)
		handler = routeJSONCache(
			opts,
			universeKilllistCacheTTL,
			"public, max-age=30, s-maxage=60, stale-while-revalidate=60",
			handler,
		)
		registerLegacy(a, huma.Operation{
			OperationID: "universe-" + route.Name + "-killmails",
			Method:      http.MethodGet,
			Path:        route.Canonical,
			Summary:     "Killmails in this " + route.Name,
			Tags:        []string{"universe", "killmails"},
		}, handler)
		registerLegacy(a, huma.Operation{
			OperationID: route.Name + "-killlist-compat",
			Method:      http.MethodGet,
			Path:        route.Alias,
			Summary:     "Killmails in this " + route.Name,
			Tags:        []string{"universe", "killmails"},
		}, handler)
	}

	for _, route := range locationKilllistRoutes {
		handler := locationMostValuableHandler(opts, route)
		handler = routeJSONCache(
			opts,
			universeMostValuableCacheTTL,
			"public, max-age=60, s-maxage=300, stale-while-revalidate=300",
			handler,
		)
		canonical := strings.TrimSuffix(route.Canonical, "/killmails") +
			"/most-valuable"
		alias := strings.TrimSuffix(route.Alias, "/killlist") +
			"/most-valuable"
		registerLegacy(a, huma.Operation{
			OperationID: "universe-" + route.Name + "-most-valuable",
			Method:      http.MethodGet,
			Path:        canonical,
			Summary:     "Most valuable kills in this " + route.Name,
			Tags:        []string{"universe", "killmails"},
		}, handler)
		registerLegacy(a, huma.Operation{
			OperationID: route.Name + "-most-valuable-compat",
			Method:      http.MethodGet,
			Path:        alias,
			Summary:     "Most valuable kills in this " + route.Name,
			Tags:        []string{"universe", "killmails"},
		}, handler)
	}

	itemHandler := universeItemKilllistHandler(opts, true)
	itemHandler = routeJSONCache(
		opts,
		universeKilllistCacheTTL,
		"public, max-age=30, s-maxage=60, stale-while-revalidate=60",
		itemHandler,
	)
	registerLegacy(a, huma.Operation{
		OperationID: "universe-type-killmails",
		Method:      http.MethodGet,
		Path:        "/universe/types/{id}/killmails",
		Summary:     "Killmails involving an inventory type",
		Tags:        []string{"universe", "killmails"},
	}, itemHandler)
	registerLegacy(a, huma.Operation{
		OperationID: "item-killlist-compat",
		Method:      http.MethodGet,
		Path:        "/item/{id}/killlist",
		Summary:     "Killmails involving an inventory type",
		Tags:        []string{"universe", "killmails"},
	}, itemHandler)

	shipHandler := universeItemKilllistHandler(opts, false)
	shipHandler = routeJSONCache(
		opts,
		universeKilllistCacheTTL,
		"public, max-age=30, s-maxage=60, stale-while-revalidate=60",
		shipHandler,
	)
	registerLegacy(a, huma.Operation{
		OperationID: "ship-killlist-compat",
		Method:      http.MethodGet,
		Path:        "/ship/{id}/killlist",
		Summary:     "Killmails where this ship type was destroyed",
		Tags:        []string{"universe", "killmails"},
	}, shipHandler)
}

func locationKilllistHandler(
	opts Options,
	route locationKilllistRoute,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseUniverseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		limit := boundedQueryInt(req, "limit", 50, 10, 100)
		after, err := optionalPositiveInt64(req.Query.Get("after"))
		if err != nil {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid after")
		}
		page := boundedQueryInt(req, "page", 0, 0, math.MaxInt32)
		if page >= 1 && after == nil {
			return loadNumberedLocationKilllist(
				ctx, opts.DB, route, id, page, limit,
			)
		}
		return loadCursorLocationKilllist(
			ctx, opts.DB, route, id, after, limit,
		)
	}
}

func loadCursorLocationKilllist(
	ctx context.Context,
	db Database,
	route locationKilllistRoute,
	id int64,
	after *int64,
	limit int,
) (legacyPayload, error) {
	where := []string{"k." + route.Column + " = $1"}
	args := []any{id}
	if after != nil {
		args = append(args, *after)
		where = append(where, fmt.Sprintf("k.killmail_id < $%d", len(args)))
	}
	args = append(args, limit+1)
	rows, err := queryMaps(
		ctx,
		db,
		campaignKilllistSelect+
			" WHERE "+strings.Join(where, " AND ")+
			fmt.Sprintf(" ORDER BY k.killmail_id DESC LIMIT $%d", len(args)),
		args...,
	)
	if err != nil {
		return legacyPayload{}, err
	}
	rows, hasMore, cursor, err := finishUniverseKilllist(ctx, db, rows, limit)
	if err != nil {
		return legacyPayload{}, err
	}
	response := map[string]any{
		"kills": rows, "hasMore": hasMore, "cursor": cursor,
	}
	total, err := loadLocationKilllistTotal(ctx, db, route, id)
	if err != nil {
		return legacyPayload{}, err
	}
	if total > 0 {
		response["totalPages"] = max(1, int(math.Ceil(float64(total)/float64(limit))))
	}
	return jsonPayload(response), nil
}

func loadNumberedLocationKilllist(
	ctx context.Context,
	db Database,
	route locationKilllistRoute,
	id int64,
	page int,
	limit int,
) (legacyPayload, error) {
	rollup, err := loadLocationKilllistRollup(ctx, db, route, id)
	if err != nil {
		return legacyPayload{}, err
	}
	total := int64(0)
	for _, day := range rollup {
		total += day.Count
	}
	totalPages := max(1, int(math.Ceil(float64(total)/float64(limit))))
	first := int64(page-1) * int64(limit)
	if first >= total {
		return jsonPayload(map[string]any{
			"kills": []map[string]any{}, "hasMore": false,
			"cursor": nil, "totalPages": totalPages,
		}), nil
	}

	rows := make([]map[string]any, 0, limit)
	remaining := int64(limit)
	cumulative := int64(0)
	for _, day := range rollup {
		if remaining == 0 {
			break
		}
		next := cumulative + day.Count
		if next <= first {
			cumulative = next
			continue
		}
		offset := int64(0)
		if first > cumulative {
			offset = first - cumulative
		}
		fetch := min(remaining, day.Count-offset)
		if fetch <= 0 {
			cumulative = next
			continue
		}
		chunk, err := loadLocationKilllistDay(
			ctx, db, route, id, day.Start, offset, fetch,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		rows = append(rows, chunk...)
		remaining -= int64(len(chunk))
		cumulative = next
	}
	if err := enrichUniverseKilllist(ctx, db, rows); err != nil {
		return legacyPayload{}, err
	}
	return jsonPayload(map[string]any{
		"kills": rows, "hasMore": page < totalPages,
		"cursor": nil, "totalPages": totalPages,
	}), nil
}

type locationKilllistDay struct {
	Start time.Time
	Count int64
}

func loadLocationKilllistRollup(
	ctx context.Context,
	db Database,
	route locationKilllistRoute,
	id int64,
) ([]locationKilllistDay, error) {
	where, args := locationStatsScope(route, id)
	rows, err := queryMaps(ctx, db, fmt.Sprintf(`
		SELECT period_start, COALESCE(SUM(kills), 0)::bigint AS count
		FROM stats
		WHERE %s AND period_type = $%d
		GROUP BY period_start
		HAVING SUM(kills) > 0
		ORDER BY period_start DESC`,
		where, len(args)+1,
	), append(args, stats.PeriodDaily)...)
	if err != nil {
		return nil, err
	}
	out := make([]locationKilllistDay, 0, len(rows))
	for _, row := range rows {
		start, ok := row["period_start"].(time.Time)
		count := int64OrZero(row["count"])
		if !ok || count <= 0 {
			continue
		}
		out = append(out, locationKilllistDay{Start: start.UTC(), Count: count})
	}
	return out, nil
}

func loadLocationKilllistTotal(
	ctx context.Context,
	db Database,
	route locationKilllistRoute,
	id int64,
) (int64, error) {
	where, args := locationStatsScope(route, id)
	row, err := queryMap(ctx, db, fmt.Sprintf(`
		SELECT COALESCE(SUM(kills), 0)::bigint AS total
		FROM stats
		WHERE %s AND period_type = $%d`,
		where, len(args)+1,
	), append(args, stats.PeriodDaily)...)
	if err != nil {
		return 0, err
	}
	return int64OrZero(row["total"]), nil
}

func locationStatsScope(
	route locationKilllistRoute,
	id int64,
) (string, []any) {
	if route.MemberQuery == "" {
		return "entity_type = $1 AND entity_id = $2",
			[]any{route.EntityType, id}
	}
	return "entity_type = $1 AND entity_id IN (" + route.MemberQuery + ")",
		[]any{route.EntityType, id}
}

func loadLocationKilllistDay(
	ctx context.Context,
	db Database,
	route locationKilllistRoute,
	id int64,
	start time.Time,
	offset int64,
	limit int64,
) ([]map[string]any, error) {
	end := start.AddDate(0, 0, 1)
	return queryMaps(ctx, db, campaignKilllistSelect+fmt.Sprintf(`
		WHERE k.%s = $1
		  AND k.killmail_time >= $2
		  AND k.killmail_time < $3
		ORDER BY k.killmail_time DESC, k.killmail_id DESC
		LIMIT $4 OFFSET $5`, route.Column),
		id, start, end, limit, offset,
	)
}

func universeItemKilllistHandler(
	opts Options,
	includeFitted bool,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseUniverseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		limit := boundedQueryInt(req, "limit", 50, 10, 100)
		after, err := optionalPositiveInt64(req.Query.Get("after"))
		if err != nil {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid after")
		}

		query, args := universeItemKilllistQuery(id, after, limit, includeFitted)
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		rows, hasMore, cursor, err := finishUniverseKilllist(
			ctx, opts.DB, rows, limit,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"kills": rows, "hasMore": hasMore, "cursor": cursor,
		}), nil
	}
}

func universeItemKilllistQuery(
	id int64,
	after *int64,
	limit int,
	includeFitted bool,
) (string, []any) {
	args := []any{id}
	cursor := ""
	if after != nil {
		args = append(args, *after)
		cursor = fmt.Sprintf(" AND killmail_id < $%d", len(args))
	}
	args = append(args, limit+1)
	limitParameter := fmt.Sprintf("$%d", len(args))
	if !includeFitted {
		where := "k.victim_ship_type_id = $1" +
			strings.ReplaceAll(cursor, "killmail_id", "k.killmail_id")
		return campaignKilllistSelect + " WHERE " + where +
			" ORDER BY k.killmail_id DESC LIMIT " + limitParameter, args
	}
	boundedSelect := strings.Replace(
		campaignKilllistSelect,
		"\n\tFROM killmails k",
		"\n\tFROM bounded_item_kills k",
		1,
	)
	return fmt.Sprintf(`
		WITH item_kills AS MATERIALIZED (
			(
				SELECT killmail_id
				FROM killmails
				WHERE victim_ship_type_id = $1%s
				ORDER BY killmail_id DESC
				LIMIT %s
			)
			UNION
			(
				SELECT DISTINCT killmail_id
				FROM killmail_items
				WHERE type_id = $1
				  AND parent_index IS NULL
				  AND (
				    flag_id BETWEEN 11 AND 34
				    OR flag_id BETWEEN 92 AND 99
				    OR flag_id BETWEEN 125 AND 132
				    OR flag_id = 87
				  )%s
				ORDER BY killmail_id DESC
				LIMIT %s
			)
			ORDER BY killmail_id DESC
			LIMIT %s
		),
		bounded_item_kills AS MATERIALIZED (
			SELECT k.*
			FROM item_kills
			JOIN killmails k ON k.killmail_id = item_kills.killmail_id
			ORDER BY k.killmail_id DESC
			LIMIT %s
		)
		%s
		ORDER BY k.killmail_id DESC
		LIMIT %s`,
		cursor, limitParameter, cursor, limitParameter, limitParameter,
		limitParameter, boundedSelect, limitParameter,
	), args
}

func finishUniverseKilllist(
	ctx context.Context,
	db Database,
	rows []map[string]any,
	limit int,
) ([]map[string]any, bool, any, error) {
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	if err := enrichUniverseKilllist(ctx, db, rows); err != nil {
		return nil, false, nil, err
	}
	var cursor any
	if len(rows) != 0 {
		cursor = rows[len(rows)-1]["killmail_id"]
	}
	return rows, hasMore, cursor, nil
}

func enrichUniverseKilllist(
	ctx context.Context,
	db Database,
	rows []map[string]any,
) error {
	paths, err := loadEntityKilllistMarketPaths(ctx, db, rows)
	if err != nil {
		return err
	}
	for _, row := range rows {
		groupID := int64OrZero(row["_ship_market_group_id"])
		row["ship_market_path"] = paths[groupID]
		delete(row, "_ship_market_group_id")
	}
	return nil
}

func locationMostValuableHandler(
	opts Options,
	route locationKilllistRoute,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseUniverseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		limit := boundedQueryInt(req, "limit", 8, 1, 32)
		days := boundedQueryInt(req, "days", 7, 1, 30)
		category := 0
		switch req.Query.Get("dataType") {
		case "most_valuable_ships":
			category = 6
		case "most_valuable_structures":
			category = 65
		}
		args := []any{id, time.Now().UTC().AddDate(0, 0, -days)}
		categoryFilter := ""
		if category != 0 {
			args = append(args, category)
			categoryFilter = fmt.Sprintf(`
				AND victim_ship_group_id IN (
					SELECT group_id FROM inv_groups WHERE category_id = $%d
				)`, len(args))
		}
		args = append(args, limit)
		rows, err := queryMaps(ctx, opts.DB, fmt.Sprintf(`
			WITH recent AS MATERIALIZED (
				SELECT killmail_id, killmail_hash, victim_ship_type_id,
				       total_value, victim_character_id,
				       victim_corporation_id, victim_alliance_id
				FROM killmails
				WHERE %s = $1 AND killmail_time >= $2%s
			)
			SELECT recent.killmail_id, recent.killmail_hash,
			       recent.victim_ship_type_id AS ship_type_id,
			       COALESCE(ship.name, 'Unknown') AS ship_name,
			       COALESCE(recent.total_value, 0)::double precision AS total_value,
			       recent.victim_character_id,
			       character.name AS victim_character_name,
			       corporation.name AS victim_corporation_name,
			       alliance.name AS victim_alliance_name
			FROM recent
			JOIN inv_types ship
			  ON ship.type_id = recent.victim_ship_type_id
			LEFT JOIN characters character
			  ON character.character_id = recent.victim_character_id
			LEFT JOIN corporations corporation
			  ON corporation.corporation_id = recent.victim_corporation_id
			LEFT JOIN alliances alliance
			  ON alliance.alliance_id = recent.victim_alliance_id
			ORDER BY recent.total_value DESC
			LIMIT $%d`,
			route.Column, categoryFilter, len(args),
		), args...)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"entries": nonNilUniverseRows(rows),
		}), nil
	}
}

func optionalPositiveInt64(raw string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return nil, fmt.Errorf("invalid positive integer")
	}
	return &value, nil
}
