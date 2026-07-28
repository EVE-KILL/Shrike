package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/killtype"
	"github.com/eve-kill/shrike/internal/stats"
)

const (
	killlistCacheTTL         = time.Minute
	advancedKilllistCacheTTL = 30 * time.Second
	killboardTopCacheTTL     = 5 * time.Minute
	graphCacheTTL            = 2 * time.Minute
)

// registerKillboardRoutes installs the aggregate killboard views used by the
// site. They live in the same API catalogue as the lower-level /killmails
// operations; the names describe the domain rather than an access boundary.
func registerKillboardRoutes(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "killlist",
		Method:      http.MethodGet,
		Path:        "/killlist",
		Summary:     "Killboard kill list",
		Tags:        []string{"killboard", "killmails"},
		Responses: map[string]*huma.Response{
			"200": {
				Description: "OK",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: killlistFrontendResponseSchema(),
					},
				},
			},
		},
	}, routeJSONCache(
		opts,
		killlistCacheTTL,
		"public, max-age=30, s-maxage=60, stale-while-revalidate=60",
		killlistHandler(opts),
	))

	registerLegacy(a, huma.Operation{
		OperationID: "killlist-advanced",
		Method:      http.MethodGet,
		Path:        "/killlist/advanced",
		Summary:     "Advanced killmail search",
		Tags:        []string{"killboard", "killmails"},
	}, routeJSONCache(
		opts,
		advancedKilllistCacheTTL,
		"public, max-age=15, s-maxage=30, stale-while-revalidate=30",
		advancedKilllistHandler(opts),
	))

	registerLegacy(a, huma.Operation{
		OperationID: "kills-top",
		Method:      http.MethodGet,
		Path:        "/kills/top",
		Summary:     "Top killers and locations",
		Tags:        []string{"killboard", "statistics"},
	}, routeJSONCache(
		opts,
		killboardTopCacheTTL,
		"public, max-age=60, s-maxage=300, stale-while-revalidate=300",
		topKillsHandler(opts),
	))

	registerLegacy(a, huma.Operation{
		OperationID: "kills-most-valuable",
		Method:      http.MethodGet,
		Path:        "/kills/most-valuable",
		Summary:     "Most valuable recent losses",
		Tags:        []string{"killboard", "statistics"},
	}, routeJSONCache(
		opts,
		killboardTopCacheTTL,
		"public, max-age=60, s-maxage=300, stale-while-revalidate=300",
		mostValuableKillsHandler(opts),
	))

	registerLegacy(a, huma.Operation{
		OperationID: "graph",
		Method:      http.MethodGet,
		Path:        "/graph",
		Summary:     "Relationship graph analysis",
		Tags:        []string{"killboard", "graph"},
	}, routeJSONCache(
		opts,
		graphCacheTTL,
		"public, max-age=30, s-maxage=120, stale-while-revalidate=120",
		graphHandler(opts),
	))
}

func killlistFrontendResponseSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"kills":      arraySchema(killlistRowSchema()),
		"hasMore":    boolSchema(),
		"cursor":     nullable(intSchema()),
		"totalPages": intSchema(),
	}, "kills", "hasMore", "cursor")
}

func killlistHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		kind := strings.TrimSpace(req.Query.Get("type"))
		if kind == "" {
			kind = "latest"
		}
		limit := boundedQueryInt(req, "limit", 50, 10, 100)
		after, err := optionalPositiveInt64(req.Query.Get("after"))
		if err != nil {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid after")
		}
		page := boundedQueryInt(req, "page", 0, 0, math.MaxInt32)
		factions := parseKilllistFactionIDs(req.Query.Get("victimFactions"))

		predicate, rollupEligible := killtype.Predicates()[kind]
		where := []string{}
		args := []any{}
		if rollupEligible && predicate != "TRUE" {
			where = append(where, predicate)
		}
		if len(factions) > 0 {
			args = append(args, factions)
			where = append(where, fmt.Sprintf(
				"k.victim_faction_id = ANY($%d::int[])", len(args),
			))
		}

		if numberedPage, ok := killlistNumberedPage(
			rollupEligible, len(factions), page, after,
		); ok {
			return loadNumberedKilllist(
				ctx, opts.DB, kind, predicate, numberedPage, limit,
			)
		}
		if len(factions) > 0 && page >= 1 && after == nil {
			return loadBoundedKilllist(ctx, opts.DB, where, args, page, limit)
		}

		cursorWhere := append([]string(nil), where...)
		cursorArgs := append([]any(nil), args...)
		if after != nil {
			cursorArgs = append(cursorArgs, *after)
			cursorWhere = append(cursorWhere, fmt.Sprintf(
				"k.killmail_id < $%d", len(cursorArgs),
			))
		}
		cursorArgs = append(cursorArgs, limit+1)
		query := campaignKilllistSelect
		if len(cursorWhere) > 0 {
			query += " WHERE " + strings.Join(cursorWhere, " AND ")
		}
		query += fmt.Sprintf(
			" ORDER BY k.killmail_id DESC LIMIT $%d", len(cursorArgs),
		)
		rows, err := queryMaps(ctx, opts.DB, query, cursorArgs...)
		if err != nil {
			return legacyPayload{}, err
		}
		rows, hasMore, cursor, err := finishUniverseKilllist(
			ctx, opts.DB, rows, limit,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		response := map[string]any{
			"kills": rows, "hasMore": hasMore, "cursor": cursor,
		}

		total, err := killlistCursorTotal(
			ctx, opts.DB, kind, rollupEligible, len(factions),
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if total > 0 {
			response["totalPages"] = max(
				1, int(math.Ceil(float64(total)/float64(limit))),
			)
		}
		return jsonPayload(response), nil
	}
}

func killlistNumberedPage(
	rollupEligible bool,
	factionCount int,
	page int,
	after *int64,
) (int, bool) {
	if !rollupEligible || factionCount > 0 || after != nil {
		return 0, false
	}
	if page < 1 {
		page = 1
	}
	return page, true
}

func killlistCursorTotal(
	ctx context.Context,
	db Database,
	kind string,
	rollupEligible bool,
	factionCount int,
) (int64, error) {
	if !rollupEligible || factionCount > 0 {
		// Faction-filtered lists have no compact count rollup. Counting millions
		// of matching killmails made the first faction-war render wait 15+
		// seconds just to display a page count. The response already carries a
		// stable cursor and hasMore, which is the normal fallback for filters
		// without a precomputed total.
		return 0, nil
	}
	return killlistRollupTotal(ctx, db, kind)
}

type killlistRollupDay struct {
	Date  time.Time
	Count int64
}

func loadNumberedKilllist(
	ctx context.Context,
	db Database,
	kind, predicate string,
	page, limit int,
) (legacyPayload, error) {
	rollup, err := loadKilllistRollup(ctx, db, kind)
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
		chunk, err := loadKilllistDay(
			ctx, db, predicate, day.Date, offset, fetch,
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

func loadKilllistRollup(
	ctx context.Context,
	db Database,
	kind string,
) ([]killlistRollupDay, error) {
	rows, err := queryMaps(ctx, db, `
		SELECT date, count
		FROM kills_daily_count
		WHERE type = $1
		ORDER BY date DESC`, kind)
	if err != nil {
		return nil, err
	}
	result := make([]killlistRollupDay, 0, len(rows))
	for _, row := range rows {
		date, ok := row["date"].(time.Time)
		if !ok {
			if raw := stringOrEmpty(row["date"]); raw != "" {
				date, err = time.Parse("2006-01-02", raw)
				ok = err == nil
			}
		}
		count := int64OrZero(row["count"])
		if ok && count > 0 {
			result = append(result, killlistRollupDay{
				Date: date.UTC(), Count: count,
			})
		}
	}
	return result, nil
}

func loadKilllistDay(
	ctx context.Context,
	db Database,
	predicate string,
	start time.Time,
	offset, limit int64,
) ([]map[string]any, error) {
	where := []string{
		"k.killmail_time >= $1",
		"k.killmail_time < $2",
	}
	if predicate != "" && predicate != "TRUE" {
		where = append(where, predicate)
	}
	return queryMaps(ctx, db,
		campaignKilllistSelect+
			" WHERE "+strings.Join(where, " AND ")+
			` ORDER BY k.killmail_time DESC, k.killmail_id DESC
			  LIMIT $3 OFFSET $4`,
		start, start.AddDate(0, 0, 1), limit, offset,
	)
}

func loadBoundedKilllist(
	ctx context.Context,
	db Database,
	where []string,
	args []any,
	page, limit int,
) (legacyPayload, error) {
	total, err := countKilllistRows(ctx, db, where, args)
	if err != nil {
		return legacyPayload{}, err
	}
	totalPages := max(1, int(math.Ceil(float64(total)/float64(limit))))
	offset := int64(page-1) * int64(limit)
	if offset >= total {
		return jsonPayload(map[string]any{
			"kills": []map[string]any{}, "hasMore": false,
			"cursor": nil, "totalPages": totalPages,
		}), nil
	}
	queryArgs := append([]any(nil), args...)
	queryArgs = append(queryArgs, limit, offset)
	rows, err := queryMaps(ctx, db,
		campaignKilllistSelect+
			" WHERE "+strings.Join(where, " AND ")+
			fmt.Sprintf(
				" ORDER BY k.killmail_id DESC LIMIT $%d OFFSET $%d",
				len(queryArgs)-1, len(queryArgs),
			),
		queryArgs...,
	)
	if err != nil {
		return legacyPayload{}, err
	}
	if err := enrichUniverseKilllist(ctx, db, rows); err != nil {
		return legacyPayload{}, err
	}
	return jsonPayload(map[string]any{
		"kills": rows, "hasMore": page < totalPages,
		"cursor": nil, "totalPages": totalPages,
	}), nil
}

func countKilllistRows(
	ctx context.Context,
	db Database,
	where []string,
	args []any,
) (int64, error) {
	query := "SELECT COUNT(*)::bigint AS total FROM killmails k"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	row, err := queryMap(ctx, db, query, args...)
	if err != nil {
		return 0, err
	}
	return int64OrZero(row["total"]), nil
}

func killlistRollupTotal(
	ctx context.Context,
	db Database,
	kind string,
) (int64, error) {
	row, err := queryMap(ctx, db, `
		SELECT COALESCE(SUM(count), 0)::bigint AS total
		FROM kills_daily_count
		WHERE type = $1`, kind)
	if err != nil {
		return 0, err
	}
	return int64OrZero(row["total"]), nil
}

func parseKilllistFactionIDs(raw string) []int32 {
	seen := map[int32]struct{}{}
	result := []int32{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		value, err := strconv.ParseInt(part, 10, 32)
		if err != nil || value == 0 {
			continue
		}
		id := int32(value)
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func topKillsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		kind := strings.TrimSpace(req.Query.Get("type"))
		if kind == "" {
			kind = "latest"
		}
		dataType := strings.TrimSpace(req.Query.Get("dataType"))
		if dataType == "" {
			dataType = "characters"
		}
		limit := boundedQueryInt(req, "limit", 10, 1, 50)
		days := boundedQueryInt(req, "days", 7, 1, 365)
		since := time.Now().UTC().AddDate(0, 0, -days)

		rows, err := loadTopKills(
			ctx, opts.DB, kind, dataType, limit, since,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		rows, err = attachGlobalStatsPalettes(ctx, opts.DB, rows)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"entries": nonNilUniverseRows(rows),
		}), nil
	}
}

func loadTopKills(
	ctx context.Context,
	db Database,
	kind, dataType string,
	limit int,
	since time.Time,
) ([]map[string]any, error) {
	if (kind == "latest" || kind == "solo") &&
		(dataType == "characters" ||
			dataType == "corporations" ||
			dataType == "alliances") {
		return loadTopKillsFromStats(
			ctx, db, kind, dataType, limit, since,
		)
	}
	predicate, known := killtype.Predicates()[kind]
	if !known {
		predicate = "TRUE"
	}
	if kind == "npc" {
		return loadTopNPCLosses(ctx, db, dataType, limit, since)
	}

	specs := map[string]struct {
		Query      string
		EntityType string
	}{
		"characters": {
			Query: `
				SELECT a.character_id AS id, c.name,
				       COUNT(DISTINCT a.killmail_id)::bigint AS count
				FROM killmail_attackers a
				JOIN characters c ON c.character_id = a.character_id
				WHERE a.killmail_time >= $1
				  AND a.character_id IS NOT NULL
				  AND a.killmail_id IN (
				    SELECT k.killmail_id FROM killmails k
				    WHERE k.killmail_time >= $1 AND ` + predicate + `
				  )
				GROUP BY a.character_id, c.name
				ORDER BY count DESC
				LIMIT $2`,
			EntityType: "character",
		},
		"corporations": {
			Query: `
				SELECT a.corporation_id AS id, c.name,
				       COUNT(DISTINCT a.killmail_id)::bigint AS count
				FROM killmail_attackers a
				JOIN corporations c ON c.corporation_id = a.corporation_id
				WHERE a.killmail_time >= $1
				  AND a.corporation_id IS NOT NULL
				  AND a.killmail_id IN (
				    SELECT k.killmail_id FROM killmails k
				    WHERE k.killmail_time >= $1 AND ` + predicate + `
				  )
				GROUP BY a.corporation_id, c.name
				ORDER BY count DESC
				LIMIT $2`,
			EntityType: "corporation",
		},
		"alliances": {
			Query: `
				SELECT a.alliance_id AS id, alliance.name,
				       COUNT(DISTINCT a.killmail_id)::bigint AS count
				FROM killmail_attackers a
				JOIN alliances alliance
				  ON alliance.alliance_id = a.alliance_id
				WHERE a.killmail_time >= $1
				  AND a.alliance_id IS NOT NULL
				  AND a.killmail_id IN (
				    SELECT k.killmail_id FROM killmails k
				    WHERE k.killmail_time >= $1 AND ` + predicate + `
				  )
				GROUP BY a.alliance_id, alliance.name
				ORDER BY count DESC
				LIMIT $2`,
			EntityType: "alliance",
		},
		"ships": {
			Query: `
				SELECT k.victim_ship_type_id AS id, type.name,
				       COUNT(*)::bigint AS count
				FROM killmails k
				JOIN inv_types type
				  ON type.type_id = k.victim_ship_type_id
				WHERE k.killmail_time >= $1
				  AND k.victim_ship_type_id IS NOT NULL
				  AND ` + predicate + `
				GROUP BY k.victim_ship_type_id, type.name
				ORDER BY count DESC
				LIMIT $2`,
			EntityType: "ship",
		},
		"systems": {
			Query: `
				SELECT k.solar_system_id AS id, system.system_name AS name,
				       system.region_id, COUNT(*)::bigint AS count
				FROM killmails k
				JOIN solar_systems system
				  ON system.solar_system_id = k.solar_system_id
				WHERE k.killmail_time >= $1 AND ` + predicate + `
				GROUP BY k.solar_system_id, system.system_name,
				         system.region_id
				ORDER BY count DESC
				LIMIT $2`,
			EntityType: "system",
		},
		"regions": {
			Query: `
				SELECT k.region_id AS id, region.name,
				       COUNT(*)::bigint AS count
				FROM killmails k
				JOIN regions region ON region.region_id = k.region_id
				WHERE k.killmail_time >= $1
				  AND k.region_id IS NOT NULL
				  AND ` + predicate + `
				GROUP BY k.region_id, region.name
				ORDER BY count DESC
				LIMIT $2`,
			EntityType: "region",
		},
	}
	spec, ok := specs[dataType]
	if !ok {
		return []map[string]any{}, nil
	}
	rows, err := queryMaps(ctx, db, spec.Query, since, limit)
	if err != nil {
		return nil, err
	}
	normalizeTopKillRows(rows, spec.EntityType)
	return rows, nil
}

func loadTopKillsFromStats(
	ctx context.Context,
	db Database,
	kind, dataType string,
	limit int,
	since time.Time,
) ([]map[string]any, error) {
	metric := "kills"
	if kind == "solo" {
		metric = "solo_kills"
	}
	specs := map[string]struct {
		EntityType stats.EntityType
		Table      string
		ID         string
		Type       string
	}{
		"characters": {
			stats.EntityCharacter, "characters", "character_id", "character",
		},
		"corporations": {
			stats.EntityCorporation, "corporations", "corporation_id", "corporation",
		},
		"alliances": {
			stats.EntityAlliance, "alliances", "alliance_id", "alliance",
		},
	}
	spec := specs[dataType]
	rows, err := queryMaps(ctx, db, fmt.Sprintf(`
		SELECT s.entity_id AS id, entity.name,
		       COALESCE(SUM(s.%s), 0)::bigint AS count
		FROM stats s
		JOIN %s entity ON entity.%s = s.entity_id
		WHERE s.entity_type = $1 AND s.period_type = $2
		  AND s.period_start >= $3::date
		GROUP BY s.entity_id, entity.name
		ORDER BY count DESC
		LIMIT $4`, metric, spec.Table, spec.ID),
		spec.EntityType, stats.PeriodDaily,
		since.Format("2006-01-02"), limit,
	)
	if err != nil {
		return nil, err
	}
	normalizeTopKillRows(rows, spec.Type)
	return rows, nil
}

func loadTopNPCLosses(
	ctx context.Context,
	db Database,
	dataType string,
	limit int,
	since time.Time,
) ([]map[string]any, error) {
	specs := map[string]struct {
		Column, Table, ID, Type string
	}{
		"characters": {
			"victim_character_id", "characters", "character_id", "character",
		},
		"corporations": {
			"victim_corporation_id", "corporations", "corporation_id", "corporation",
		},
		"alliances": {
			"victim_alliance_id", "alliances", "alliance_id", "alliance",
		},
	}
	spec, ok := specs[dataType]
	if !ok {
		return []map[string]any{}, nil
	}
	rows, err := queryMaps(ctx, db, fmt.Sprintf(`
		SELECT k.%s AS id, entity.name, COUNT(*)::bigint AS count
		FROM killmails k
		JOIN %s entity ON entity.%s = k.%s
		WHERE k.%s IS NOT NULL
		  AND k.killmail_time >= $1
		  AND k.is_npc = true
		GROUP BY k.%s, entity.name
		ORDER BY count DESC
		LIMIT $2`,
		spec.Column, spec.Table, spec.ID, spec.Column,
		spec.Column, spec.Column,
	), since, limit)
	if err != nil {
		return nil, err
	}
	normalizeTopKillRows(rows, spec.Type)
	return rows, nil
}

func normalizeTopKillRows(rows []map[string]any, entityType string) {
	for _, row := range rows {
		if stringOrEmpty(row["name"]) == "" {
			row["name"] = "Unknown"
		}
		row["count"] = int64OrZero(row["count"])
		row["type"] = entityType
	}
}

func mostValuableKillsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		kind := strings.TrimSpace(req.Query.Get("type"))
		if kind == "" {
			kind = "latest"
		}
		limit := boundedQueryInt(req, "limit", 7, 1, 20)
		days := boundedQueryInt(req, "days", 7, 1, 365)
		predicate, known := killtype.Predicates()[kind]
		if !known {
			predicate = "TRUE"
		}
		rows, err := queryMaps(ctx, opts.DB, `
			WITH recent_kills AS MATERIALIZED (
				SELECT k.killmail_id, k.killmail_hash,
				       k.victim_ship_type_id, k.total_value,
				       k.victim_character_id, k.victim_corporation_id,
				       k.victim_alliance_id
				FROM killmails k
				WHERE k.killmail_time >= $1 AND `+predicate+`
			)
			SELECT recent.killmail_id, recent.killmail_hash,
			       recent.victim_ship_type_id AS ship_type_id,
			       type.name AS ship_name, recent.total_value,
			       recent.victim_character_id,
			       character.name AS victim_character_name,
			       corporation.name AS victim_corporation_name,
			       alliance.name AS victim_alliance_name
			FROM recent_kills recent
			JOIN inv_types type
			  ON type.type_id = recent.victim_ship_type_id
			LEFT JOIN characters character
			  ON character.character_id = recent.victim_character_id
			LEFT JOIN corporations corporation
			  ON corporation.corporation_id = recent.victim_corporation_id
			LEFT JOIN alliances alliance
			  ON alliance.alliance_id = recent.victim_alliance_id
			ORDER BY recent.total_value DESC
			LIMIT $2`,
			time.Now().UTC().AddDate(0, 0, -days), limit,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		for _, row := range rows {
			if stringOrEmpty(row["ship_name"]) == "" {
				row["ship_name"] = "Unknown"
			}
			row["total_value"] = float64OrZero(row["total_value"])
		}
		return jsonPayload(map[string]any{
			"entries": nonNilUniverseRows(rows),
		}), nil
	}
}

func sortedInt64Set(values map[int64]struct{}) []int64 {
	result := make([]int64, 0, len(values))
	for value := range values {
		if value > 0 {
			result = append(result, value)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
