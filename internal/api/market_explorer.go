package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const (
	marketOrdersDefaultLimit = 100
	marketOrdersMaximumLimit = 200
	marketHistoryDefaultDays = 30
	marketHistoryMaximumDays = 90
)

func registerMarketExplorerRoutes(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "market-item-orders",
		Method:      http.MethodGet,
		Path:        "/market/items/{id}/orders",
		Summary:     "Current regional buy and sell orders for an item",
		Tags:        []string{"market"},
		Parameters: []*huma.Param{{
			Name: "id", In: "path", Required: true,
			Schema: intSchema(), Description: "Inventory type ID.",
		}},
	}, routeJSONCache(
		opts, time.Minute,
		"public, max-age=30, s-maxage=60, stale-while-revalidate=60",
		marketItemOrdersHandler(opts),
	))

	registerLegacy(a, huma.Operation{
		OperationID: "market-item-history",
		Method:      http.MethodGet,
		Path:        "/market/items/{id}/history",
		Summary:     "Corrected regional market history for an item",
		Tags:        []string{"market"},
		Parameters: []*huma.Param{{
			Name: "id", In: "path", Required: true,
			Schema: intSchema(), Description: "Inventory type ID.",
		}},
	}, routeJSONCache(
		opts, 10*time.Minute,
		"public, max-age=60, s-maxage=600, stale-while-revalidate=300",
		marketItemHistoryHandler(opts),
	))
}

func marketItemOrdersHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		typeID, err := parseID(req.Param("id"))
		if err != nil || typeID > pgInt4Max {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid type ID")
		}
		regionID, err := optionalMarketRegion(req.Query.Get("region_id"))
		if err != nil {
			return legacyPayload{}, err
		}
		limit := boundedMarketInt(req.Query.Get("limit"), marketOrdersDefaultLimit, 1, marketOrdersMaximumLimit)
		securitySQL, err := marketSecuritySQL(req.Query.Get("security"), "system.security")
		if err != nil {
			return legacyPayload{}, err
		}

		itemRows, err := queryMaps(ctx, opts.DB, `
			SELECT type_id, name, group_id, market_group_id
			FROM inv_types WHERE type_id = $1 AND published IS TRUE`, typeID)
		if err != nil {
			return legacyPayload{}, err
		}
		if len(itemRows) == 0 {
			return legacyPayload{}, apiError(http.StatusNotFound, "Market item not found")
		}

		args := []any{typeID, regionID, limit}
		base := `
			SELECT orders.order_id, orders.duration, orders.is_buy_order,
			       orders.issued, orders.location_id, orders.min_volume,
			       orders.price, orders.order_range, orders.system_id,
			       orders.type_id, orders.volume_remain, orders.volume_total,
			       orders.region_id, orders.constellation_id, orders.snapshot_at,
			       system.system_name, system.security,
			       CASE WHEN orders.region_id = 19000001 THEN 'Global PLEX Market'
			            ELSE region.name END AS region_name,
			       COALESCE(station.station_name, structure.name,
			                system.system_name, orders.location_id::text) AS location_name,
			       orders.issued + make_interval(days => orders.duration) AS expires_at
			FROM market_orders orders
			JOIN solar_systems system ON system.solar_system_id = orders.system_id
			LEFT JOIN regions region ON region.region_id = orders.region_id
			LEFT JOIN stations station ON station.station_id = orders.location_id
			LEFT JOIN structures structure ON structure.structure_id = orders.location_id
			WHERE orders.type_id = $1
			  AND ($2::int = 0 OR orders.region_id = $2)
			  AND %s
			  AND orders.is_buy_order IS %s
			ORDER BY orders.price %s, orders.order_id
			LIMIT $3`

		sellers, err := queryMaps(ctx, opts.DB, fmt.Sprintf(base, securitySQL, "FALSE", "ASC"), args...)
		if err != nil {
			return legacyPayload{}, err
		}
		buyers, err := queryMaps(ctx, opts.DB, fmt.Sprintf(base, securitySQL, "TRUE", "DESC"), args...)
		if err != nil {
			return legacyPayload{}, err
		}

		regions, err := queryMaps(ctx, opts.DB, `
			SELECT orders.region_id,
			       CASE WHEN orders.region_id = 19000001 THEN 'Global PLEX Market'
			            ELSE region.name END AS name,
			       count(*)::bigint AS order_count,
			       min(orders.price) FILTER (WHERE orders.is_buy_order IS FALSE) AS lowest_sell,
			       max(orders.price) FILTER (WHERE orders.is_buy_order IS TRUE) AS highest_buy
			FROM market_orders orders
			LEFT JOIN regions region ON region.region_id = orders.region_id
			WHERE orders.type_id = $1
			GROUP BY orders.region_id, region.name
			ORDER BY region.name`, typeID)
		if err != nil {
			return legacyPayload{}, err
		}

		var snapshot any
		if len(sellers) > 0 {
			snapshot = sellers[0]["snapshot_at"]
		} else if len(buyers) > 0 {
			snapshot = buyers[0]["snapshot_at"]
		} else {
			_ = opts.DB.QueryRow(ctx, `SELECT max(snapshot_at) FROM market_orders`).Scan(&snapshot)
		}

		return jsonPayload(map[string]any{
			"item":        itemRows[0],
			"snapshot_at": snapshot,
			"region_id":   regionID,
			"security":    normalizedMarketSecurity(req.Query.Get("security")),
			"sellers":     nonNilRows(sellers),
			"buyers":      nonNilRows(buyers),
			"regions":     nonNilRows(regions),
		}), nil
	}
}

func marketItemHistoryHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		typeID, err := parseID(req.Param("id"))
		if err != nil || typeID > pgInt4Max {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid type ID")
		}
		regionID, err := optionalMarketRegion(req.Query.Get("region_id"))
		if err != nil {
			return legacyPayload{}, err
		}
		days := boundedMarketInt(req.Query.Get("days"), marketHistoryDefaultDays, 1, marketHistoryMaximumDays)

		rows, err := queryMaps(ctx, opts.DB, `
			SELECT date,
			       CASE WHEN sum(COALESCE(volume, 0)) > 0
			            THEN sum(average * volume) / sum(volume) END AS average,
			       max(highest) AS highest, min(lowest) AS lowest,
			       sum(order_count)::bigint AS order_count,
			       sum(volume)::bigint AS volume,
			       max(http_last_modified) AS source_updated_at
			FROM market_region_history
			WHERE type_id = $1
			  AND ($2::int = 0 OR region_id = $2)
			  AND date >= current_date - ($3::int - 1)
			GROUP BY date
			ORDER BY date`, typeID, regionID, days)
		if err != nil {
			return legacyPayload{}, err
		}

		return jsonPayload(map[string]any{
			"type_id": typeID, "region_id": regionID, "days": days,
			"history": nonNilRows(rows),
		}), nil
	}
}

func optionalMarketRegion(raw string) (int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value <= 0 {
		return 0, apiError(http.StatusBadRequest, "Invalid region ID")
	}
	return int32(value), nil
}

func boundedMarketInt(raw string, fallback, minimum, maximum int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return min(max(value, minimum), maximum)
}

func normalizedMarketSecurity(raw string) []string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return []string{"high", "low", "null"}
	}
	seen := map[string]bool{}
	for value := range strings.SplitSeq(raw, ",") {
		value = strings.TrimSpace(value)
		if value == "high" || value == "low" || value == "null" {
			seen[value] = true
		}
	}
	result := make([]string, 0, 3)
	for _, value := range []string{"high", "low", "null"} {
		if seen[value] {
			result = append(result, value)
		}
	}
	return result
}

func marketSecuritySQL(raw, column string) (string, error) {
	values := normalizedMarketSecurity(raw)
	if strings.TrimSpace(raw) != "" && len(values) == 0 {
		return "", apiError(http.StatusBadRequest, "Invalid security filter")
	}
	if len(values) == 3 {
		return "TRUE", nil
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		switch value {
		case "high":
			parts = append(parts, column+" >= 0.5")
		case "low":
			parts = append(parts, column+" > 0 AND "+column+" < 0.5")
		case "null":
			parts = append(parts, column+" <= 0")
		}
	}
	return "(" + strings.Join(parts, " OR ") + ")", nil
}
