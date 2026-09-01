package api

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const (
	fittingPublicCache  = "public, max-age="
	fittingRecentDays   = 30
	fittingTrendingDays = 7
)

func registerFittingCatalogueRoutes(a huma.API, opts Options) {
	routes := []struct {
		id, canonical, alias, summary string
		ttl                           time.Duration
		handler                       legacyHandler
	}{
		{
			"fittings-community-latest", "/fittings/community/latest",
			"/fits/community-latest", "Latest public community fittings",
			time.Minute, communityFittingsHandler(opts, false),
		},
		{
			"fittings-community-top-rated", "/fittings/community/top-rated",
			"/fits/top-rated", "Top-rated public community fittings",
			time.Minute, communityFittingsHandler(opts, true),
		},
		{
			"fittings-trending", "/fittings/trending",
			"/fits/flavors-of-the-week", "Weekly ship fitting rankings",
			30 * time.Minute, trendingFittingsHandler(opts),
		},
		{
			"fittings-popular-ships", "/fittings/ships/popular",
			"/fits/popular-ships", "Ships participating in the most recent kills",
			10 * time.Minute, popularFittingShipsHandler(opts),
		},
		{
			"fittings-stats", "/fittings/stats",
			"/fits/quick-stats", "Fitting catalogue statistics",
			10 * time.Minute, fittingStatsHandler(opts),
		},
		{
			"fittings-roles", "/fittings/roles",
			"/fits/roles", "Fitting search role taxonomy",
			10 * time.Minute, fittingRolesHandler(opts),
		},
		{
			"fittings-search", "/fittings/search",
			"/fits/search", "Search killmail-derived ship fittings",
			2 * time.Minute, searchFittingsHandler(opts),
		},
		{
			"fittings-alliance-doctrines", "/fittings/doctrines/alliances",
			"/fits/top-alliance-doctrines", "Popular alliance doctrines",
			30 * time.Minute, allianceDoctrineHandler(opts),
		},
		{
			"fittings-ship-families", "/fittings/ships/{id}/families",
			"/item/{id}/fittings", "Popular fitting families for a ship",
			5 * time.Minute, shipFittingFamiliesHandler(opts),
		},
		{
			"fittings-ship-metadata", "/fittings/ships/{id}/metadata",
			"/item/{id}/fit-meta", "Module group usage for a ship",
			5 * time.Minute, shipFittingMetadataHandler(opts),
		},
	}
	for _, route := range routes {
		cacheControl := fittingPublicCache + strconv.Itoa(int(route.ttl.Seconds()))
		for aliasIndex, path := range []string{route.canonical, route.alias} {
			id := route.id
			if aliasIndex > 0 {
				id += "-legacy"
			}
			operation := huma.Operation{
				OperationID: id,
				Method:      http.MethodGet,
				Path:        path,
				Summary:     route.summary,
				Tags:        []string{"fittings"},
			}
			if route.id == "fittings-trending" {
				operation.Parameters = []*huma.Param{{
					Name: "mode", In: "query",
					Description: "Rank weekly hulls by kill participation, final blows, or observed losses.",
					Schema: &huma.Schema{Type: huma.TypeString, Default: "kills",
						Enum: []any{"kills", "final_blows", "losses"}},
				}}
			}
			if route.id == "fittings-popular-ships" {
				operation.Parameters = []*huma.Param{{
					Name: "mode", In: "query",
					Description: "Rank hulls by kill participation.",
					Schema: &huma.Schema{Type: huma.TypeString, Default: "kills",
						Enum: []any{"kills"}},
				}}
			}
			if route.id == "fittings-alliance-doctrines" {
				operation.Parameters = []*huma.Param{{
					Name: "entity_type", In: "query",
					Description: "Group doctrine losses by alliance or corporation.",
					Schema: &huma.Schema{Type: huma.TypeString, Default: "alliance",
						Enum: []any{"alliance", "corporation"}},
				}}
			}
			registerLegacy(a, operation, routeJSONCache(opts, route.ttl, cacheControl, route.handler))
		}
	}
}

func communityFittingsHandler(opts Options, topRated bool) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		limit := fittingFeedLimit(req.Query.Get("limit"))
		ratingFilter := ""
		order := "f.created_at DESC"
		if topRated {
			ratingFilter = " AND f.rating_count > 0"
			order = "f.rating_avg DESC, f.rating_count DESC, f.created_at DESC"
		}
		rows, err := queryMaps(ctx, opts.DB, `
			SELECT f.fit_id, f.name, f.description,
			       f.ship_type_id, ship.name AS ship_name,
			       f.owner_character_id, owner.name AS owner_name,
			       f.rating_avg, f.rating_count,
			       f.created_at, f.updated_at,
			       (
			         SELECT COUNT(*)::int
			         FROM user_fitting_items item
			         WHERE item.fit_id = f.fit_id
			       ) AS module_count
			FROM user_fittings f
			LEFT JOIN characters owner
			  ON owner.character_id = f.owner_character_id
			LEFT JOIN inv_types ship
			  ON ship.type_id = f.ship_type_id
			WHERE f.visibility = 3`+ratingFilter+`
			ORDER BY `+order+`
			LIMIT $1`, limit)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"fits": nonNilFittingRows(rows)}), nil
	}
}

func fittingFeedLimit(raw string) int {
	if raw == "" {
		return 12
	}
	number, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) ||
		math.Trunc(number) != number || number < 1 {
		return 12
	}
	if number > 50 {
		return 50
	}
	return int(number)
}

func popularFittingShipsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		mode := req.Query.Get("mode")
		if mode != "" && mode != "kills" {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid popular hull ranking mode")
		}
		rows, err := queryMaps(ctx, opts.DB, `
			WITH fit_evidence AS (
				SELECT kf.ship_type_id,
				       COUNT(DISTINCT f.family_hash)::int AS fit_count
				FROM killmail_fittings kf
				JOIN fittings f ON f.fit_hash = kf.fit_hash
				WHERE kf.kill_time >= NOW() - INTERVAL '30 days'
				GROUP BY kf.ship_type_id
			),
			ship_activity AS (
				SELECT daily.entity_id AS ship_type_id,
				       SUM(daily.kills)::int AS total_uses,
				       MAX(daily.period_start) AS last_used
				FROM stats daily
				JOIN inv_types ship ON ship.type_id = daily.entity_id
				WHERE daily.entity_type = 3
				  AND daily.period_type = 0
				  AND daily.period_start >= CURRENT_DATE - 29
				  AND ship.group_id NOT IN (29, 237)
				GROUP BY daily.entity_id
				ORDER BY total_uses DESC
				LIMIT 12
			)
			SELECT activity.ship_type_id, activity.total_uses,
			       COALESCE(evidence.fit_count, 0)::int AS fit_count,
			       activity.last_used,
			       ship.name AS ship_name, ship.group_id
			FROM ship_activity activity
			LEFT JOIN inv_types ship
			  ON ship.type_id = activity.ship_type_id
			LEFT JOIN fit_evidence evidence
			  ON evidence.ship_type_id = activity.ship_type_id
			ORDER BY activity.total_uses DESC`)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"window_days": fittingRecentDays,
			"ships":       nonNilFittingRows(rows),
		}), nil
	}
}

func fittingStatsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		row, err := queryMap(ctx, opts.DB, `
			SELECT
			  COALESCE((
			    SELECT reltuples::bigint
			    FROM pg_class WHERE relname = 'fittings'
			    LIMIT 1
			  ), 0) AS fittings_known,
			  COALESCE((
			    SELECT reltuples::bigint
			    FROM pg_class WHERE relname = 'killmail_fittings'
			    LIMIT 1
			  ), 0) AS killmails_analyzed,
			  (SELECT COUNT(*)::bigint FROM user_fittings
			   WHERE visibility = 3) AS community_fits,
			  (SELECT COUNT(*)::bigint
			   FROM user_fitting_ratings) AS ratings_cast`)
		if err != nil {
			return legacyPayload{}, err
		}
		if row == nil {
			row = map[string]any{
				"fittings_known": 0, "killmails_analyzed": 0,
				"community_fits": 0, "ratings_cast": 0,
			}
		}
		return jsonPayload(row), nil
	}
}

type fittingCatalogueContents struct {
	ModulesByHash map[string][]map[string]any
	DronesByHash  map[string][]map[string]any
	CostByHash    map[string]float64
	Prices        map[int64]float64
}

func loadCatalogueContents(
	ctx context.Context,
	db Database,
	fitHashes []string,
	hullIDs []int32,
) (fittingCatalogueContents, error) {
	result := fittingCatalogueContents{
		ModulesByHash: map[string][]map[string]any{},
		DronesByHash:  map[string][]map[string]any{},
		CostByHash:    map[string]float64{},
		Prices:        map[int64]float64{},
	}
	if len(fitHashes) == 0 {
		return result, nil
	}
	items, err := queryMaps(ctx, db, `
		SELECT item.fit_hash, item.slot_group, item.ordinal,
		       item.type_id, item_type.name,
		       item.charge_type_id, charge_type.name AS charge_name,
		       item.quantity
		FROM fitting_items item
		LEFT JOIN inv_types item_type
		  ON item_type.type_id = item.type_id
		LEFT JOIN inv_types charge_type
		  ON charge_type.type_id = item.charge_type_id
		WHERE item.fit_hash = ANY($1::text[])
		ORDER BY item.fit_hash, item.slot_group, item.ordinal`, fitHashes)
	if err != nil {
		return result, err
	}
	priceSet := make(map[int32]bool)
	for _, id := range hullIDs {
		priceSet[id] = true
	}
	for _, item := range items {
		priceSet[int32(int64OrZero(item["type_id"]))] = true
		if chargeID := int64OrZero(item["charge_type_id"]); chargeID > 0 {
			priceSet[int32(chargeID)] = true
		}
	}
	priceIDs := make([]int32, 0, len(priceSet))
	for id := range priceSet {
		priceIDs = append(priceIDs, id)
	}
	if len(priceIDs) > 0 {
		result.Prices, err = loadFittingPrices(ctx, db, priceIDs)
		if err != nil {
			return result, err
		}
	}
	for _, item := range items {
		hash := stringOrEmpty(item["fit_hash"])
		typeID := int64OrZero(item["type_id"])
		quantity := int64OrZero(item["quantity"])
		if quantity == 0 {
			quantity = 1
		}
		if int64OrZero(item["slot_group"]) == 6 {
			result.DronesByHash[hash] = append(
				result.DronesByHash[hash],
				map[string]any{
					"type_id": typeID, "name": item["name"],
					"quantity": quantity,
				},
			)
			result.CostByHash[hash] += result.Prices[typeID] * float64(quantity)
			continue
		}
		module := map[string]any{
			"slot_group": item["slot_group"], "ordinal": item["ordinal"],
			"type_id": typeID, "name": item["name"],
			"charge_type_id": item["charge_type_id"],
			"charge_name":    item["charge_name"],
		}
		result.ModulesByHash[hash] = append(result.ModulesByHash[hash], module)
		result.CostByHash[hash] += result.Prices[typeID]
		if chargeID := int64OrZero(item["charge_type_id"]); chargeID > 0 {
			result.CostByHash[hash] += result.Prices[chargeID]
		}
	}
	return result, nil
}

func catalogueList(
	values map[string][]map[string]any,
	key string,
) []map[string]any {
	if rows := values[key]; rows != nil {
		return rows
	}
	return []map[string]any{}
}
