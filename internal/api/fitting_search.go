package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/eve-kill/shrike/internal/fitting"
)

type fittingSearchFilter struct {
	Kind, RoleID, Op, TypeName string
	TypeID                     int32
	Count                      int
}

type fittingStatQuery struct {
	Sort      string
	CapStable bool
}

func parseFittingStatQuery(req *legacyRequest) (fittingStatQuery, error) {
	result := fittingStatQuery{Sort: strings.TrimSpace(req.Query.Get("sort"))}
	if result.Sort == "" {
		result.Sort = "uses"
	}
	allowed := map[string]bool{"uses": true, "dps": true, "ehp": true, "alpha": true, "speed": true, "align": true, "repair": true, "npc_ehp": true}
	if !allowed[result.Sort] {
		return result, apiError(http.StatusBadRequest, "unsupported fitting sort")
	}
	result.CapStable = req.Query.Get("cap_stable") == "true"
	return result, nil
}

func (filter fittingSearchFilter) JSON() map[string]any {
	result := map[string]any{
		"kind": filter.Kind, "op": filter.Op, "count": filter.Count,
	}
	if filter.Kind == "role" {
		result["role_id"] = filter.RoleID
	} else {
		result["type_id"] = filter.TypeID
		if filter.TypeName != "" {
			result["type_name"] = filter.TypeName
		}
	}
	return result
}

func searchFittingsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		ship, err := parseSearchShip(req.Query.Get("ship"))
		if err != nil {
			return legacyPayload{}, err
		}
		filters, err := parseFittingSearchFilters(req.Query.Get("filters"))
		if err != nil {
			return legacyPayload{}, err
		}
		statQuery, err := parseFittingStatQuery(req)
		if err != nil {
			return legacyPayload{}, err
		}
		limit := fittingSearchPageNumber(req, "limit", 24, 1, 50)
		offset := fittingSearchPageNumber(req, "offset", 0, 0, math.MaxInt32)
		applied := fittingSearchFiltersJSON(filters)

		resolved, err := resolveFittingRoles(ctx, opts.DB)
		if err != nil {
			return legacyPayload{}, err
		}
		for _, filter := range filters {
			if filter.Kind == "role" && len(resolved[filter.RoleID]) == 0 &&
				filter.Count > 0 {
				return jsonPayload(emptyFittingSearchResponse(
					ship, offset, limit, applied,
				)), nil
			}
		}

		query, args, err := buildFittingSearchQuery(
			req, ship, filters, resolved, statQuery, limit, offset,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		total := int64(0)
		if len(rows) > 0 {
			total = int64OrZero(rows[0]["result_total"])
		}
		if total == 0 {
			return jsonPayload(emptyFittingSearchResponse(
				ship, offset, limit, applied,
			)), nil
		}
		fitRows := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			if stringOrEmpty(row["fit_hash"]) == "" {
				continue
			}
			delete(row, "result_total")
			delete(row, "sort_rank")
			fitRows = append(fitRows, row)
		}
		if len(fitRows) == 0 {
			response := emptyFittingSearchResponse(ship, offset, limit, applied)
			response["total"] = total
			return jsonPayload(response), nil
		}

		hashes := make([]string, 0, len(fitRows))
		familyHashSet := make(map[string]bool, len(fitRows))
		familyHashes := make([]string, 0, len(fitRows))
		for _, fit := range fitRows {
			hashes = append(hashes, stringOrEmpty(fit["fit_hash"]))
			familyHash := stringOrEmpty(fit["family_hash"])
			if familyHash != "" && !familyHashSet[familyHash] {
				familyHashSet[familyHash] = true
				familyHashes = append(familyHashes, familyHash)
			}
		}
		contents, err := loadCatalogueContents(
			ctx, opts.DB, hashes, []int32{ship},
		)
		if err != nil {
			return legacyPayload{}, err
		}
		contexts, err := loadFittingContexts(ctx, opts.DB, int64(ship), familyHashes)
		if err != nil {
			return legacyPayload{}, err
		}
		familyRows, err := queryMaps(ctx, opts.DB, `
			SELECT fit.family_hash,
			       COUNT(*)::int AS family_total_uses,
			       COUNT(DISTINCT fitting.fit_hash)::int AS variant_count
			FROM killmail_fittings fitting
			JOIN fittings fit ON fit.fit_hash = fitting.fit_hash
			WHERE fitting.ship_type_id = $1
			  AND fitting.kill_time >= NOW() - INTERVAL '90 days'
			  AND fit.family_hash = ANY($2::text[])
			GROUP BY fit.family_hash`, ship, familyHashes)
		if err != nil {
			return legacyPayload{}, err
		}
		familyTotals := make(map[string]map[string]any, len(familyRows))
		for _, row := range familyRows {
			familyTotals[stringOrEmpty(row["family_hash"])] = row
		}
		shipName := fitRows[0]["ship_name"]
		hullCost := any(nil)
		if price, ok := contents.Prices[int64(ship)]; ok {
			hullCost = price
		}
		fits := make([]map[string]any, 0, len(fitRows))
		for _, fit := range fitRows {
			hash := stringOrEmpty(fit["fit_hash"])
			familyHash := stringOrEmpty(fit["family_hash"])
			familyTotal := familyTotals[familyHash]
			fits = append(fits, map[string]any{
				"fit_hash": hash, "family_hash": familyHash,
				"ship_type_id": fit["ship_type_id"], "ship_name": shipName,
				"total_uses": fit["total_uses"], "last_used": fit["last_used"],
				"family_total_uses": int64OrZero(familyTotal["family_total_uses"]),
				"variant_count":     int64OrZero(familyTotal["variant_count"]),
				"fit_cost":          contents.CostByHash[hash], "hull_cost": hullCost,
				"modules": catalogueList(contents.ModulesByHash, hash),
				"drones":  catalogueList(contents.DronesByHash, hash),
				"context": contexts[familyHash],
				"stats":   fit["stats"],
			})
		}
		return jsonPayload(map[string]any{
			"ship_type_id": ship, "ship_name": shipName,
			"window_days": 90, "total": total,
			"has_more": int64(offset+len(fits)) < total,
			"offset":   offset, "limit": limit,
			"filters_applied": applied, "fits": fits,
		}), nil
	}
}

func fittingSearchAvailabilityHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		ship, err := parseSearchShip(req.Query.Get("ship"))
		if err != nil {
			return legacyPayload{}, err
		}
		filters, err := parseFittingSearchFilters(req.Query.Get("filters"))
		if err != nil {
			return legacyPayload{}, err
		}
		resolved, err := resolveFittingRoles(ctx, opts.DB)
		if err != nil {
			return legacyPayload{}, err
		}
		query, args, err := buildFittingAvailabilityQuery(req, ship, filters, resolved)
		if err != nil {
			return legacyPayload{}, err
		}
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		roleCounts := make(map[string]int64)
		typeCounts := make(map[string]int64)
		for _, row := range rows {
			key := stringOrEmpty(row["key"])
			if key == "" {
				continue
			}
			if stringOrEmpty(row["kind"]) == "role" {
				roleCounts[key] = int64OrZero(row["fit_count"])
			} else {
				typeCounts[key] = int64OrZero(row["fit_count"])
			}
		}
		return jsonPayload(map[string]any{
			"ship_type_id": ship, "window_days": 90,
			"role_counts": roleCounts, "type_counts": typeCounts,
		}), nil
	}
}

func fittingSearchDistributionsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		ship, err := parseSearchShip(req.Query.Get("ship"))
		if err != nil {
			return legacyPayload{}, err
		}
		filters, err := parseFittingSearchFilters(req.Query.Get("filters"))
		if err != nil {
			return legacyPayload{}, err
		}
		resolved, err := resolveFittingRoles(ctx, opts.DB)
		if err != nil {
			return legacyPayload{}, err
		}
		eligibility, args, err := buildFittingEligibilityCTE(req, ship, filters, resolved)
		if err != nil {
			return legacyPayload{}, err
		}
		metrics, err := fittingSearchDistributionMetrics(ctx, opts, eligibility, args)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"ship_type_id": ship, "window_days": 90, "metrics": metrics,
		}), nil
	}
}

func fittingSearchDistributionMetrics(ctx context.Context, opts Options, eligibility string, args []any) ([]map[string]any, error) {
	metricNames := []string{"ehp", "dps", "repair", "speed"}
	queries := make([]databaseQuery, 0, len(metricNames))
	for _, metric := range metricNames {
		expression := fittingDistributionExpression("fs", metric)
		metricArgs := append(append([]any{}, args...), fitting.DistributionBuckets)
		bucketParameter := len(metricArgs)
		queries = append(queries, databaseQuery{SQL: fmt.Sprintf(`
			WITH %s,
			samples AS (
				SELECT fs.fit_hash,%s::double precision value,COUNT(*)::bigint observations
				FROM eligible
				JOIN fitting_stats fs USING (fit_hash)
				JOIN killmail_fittings kf USING (fit_hash)
				WHERE kf.kill_time>=NOW()-INTERVAL '90 days' AND %s>0
				GROUP BY fs.fit_hash,%s
			), weighted AS (
				SELECT samples.*,
				  SUM(observations) OVER (ORDER BY value,fit_hash ROWS UNBOUNDED PRECEDING) cumulative,
				  SUM(observations) OVER () total_observations
				FROM samples
			), summary AS (
				SELECT COUNT(*)::int fit_count,MAX(total_observations)::bigint observation_count,
				  MIN(value) minimum,MAX(value) maximum,
				  MIN(value) FILTER (WHERE cumulative>=total_observations*.01) p01,
				  MIN(value) FILTER (WHERE cumulative>=total_observations*.10) p10,
				  MIN(value) FILTER (WHERE cumulative>=total_observations*.25) p25,
				  MIN(value) FILTER (WHERE cumulative>=total_observations*.50) median,
				  MIN(value) FILTER (WHERE cumulative>=total_observations*.75) p75,
				  MIN(value) FILTER (WHERE cumulative>=total_observations*.90) p90,
				  MIN(value) FILTER (WHERE cumulative>=total_observations*.99) p99
				FROM weighted
			), assigned AS (
				SELECT samples.*,
				  CASE WHEN summary.p99<=summary.p01 THEN 1 ELSE LEAST($%d,GREATEST(1,width_bucket(samples.value,summary.p01,summary.p99,$%d))) END bucket
				FROM samples CROSS JOIN summary
			), aggregated AS (
				SELECT bucket,COUNT(*)::int fit_count,SUM(observations)::bigint observation_count
				FROM assigned GROUP BY bucket
			)
			SELECT summary.fit_count,summary.observation_count,summary.minimum,summary.maximum,
			  summary.p10,summary.p25,summary.median,summary.p75,summary.p90,series.bucket,
			  CASE WHEN summary.p99<=summary.p01 THEN summary.p01 ELSE summary.p01+(series.bucket-1)*(summary.p99-summary.p01)/$%d END lower_bound,
			  CASE WHEN summary.p99<=summary.p01 THEN summary.p99 ELSE summary.p01+series.bucket*(summary.p99-summary.p01)/$%d END upper_bound,
			  COALESCE(aggregated.fit_count,0)::int bucket_fit_count,
			  COALESCE(aggregated.observation_count,0)::bigint bucket_observation_count
			FROM summary
			CROSS JOIN LATERAL generate_series(1,CASE WHEN summary.p99<=summary.p01 THEN 1 ELSE $%d END) series(bucket)
			LEFT JOIN aggregated USING (bucket)
			WHERE summary.fit_count>0 ORDER BY series.bucket`, eligibility, expression, expression, expression,
			bucketParameter, bucketParameter, bucketParameter, bucketParameter, bucketParameter), Args: metricArgs})
	}
	results, err := queryMapsConcurrent(ctx, opts.DB, queries...)
	if err != nil {
		return nil, err
	}
	metrics := make([]map[string]any, 0, len(results))
	for index, rows := range results {
		if len(rows) == 0 {
			continue
		}
		buckets := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			buckets = append(buckets, map[string]any{
				"bucket": row["bucket"], "lower_bound": row["lower_bound"], "upper_bound": row["upper_bound"],
				"fit_count": row["bucket_fit_count"], "observation_count": row["bucket_observation_count"],
			})
		}
		row := rows[0]
		metrics = append(metrics, map[string]any{
			"metric": metricNames[index], "fit_count": row["fit_count"], "observation_count": row["observation_count"],
			"minimum": row["minimum"], "maximum": row["maximum"], "p10": row["p10"], "p25": row["p25"],
			"median": row["median"], "p75": row["p75"], "p90": row["p90"], "buckets": buckets,
		})
	}
	return metrics, nil
}

func buildFittingAvailabilityQuery(req *legacyRequest, ship int32, filters []fittingSearchFilter, resolved resolvedFittingRoles) (string, []any, error) {
	eligibility, args, err := buildFittingEligibilityCTE(req, ship, filters, resolved)
	if err != nil {
		return "", nil, err
	}
	roleQueries := make([]string, 0, len(fittingRoleDefinitions))
	for _, role := range fittingRoleDefinitions {
		ids := resolved[role.ID]
		if len(ids) == 0 {
			continue
		}
		args = append(args, role.ID, ids)
		roleQueries = append(roleQueries, fmt.Sprintf(
			"SELECT $%d::text AS role_id, unnest($%d::int[]) AS type_id",
			len(args)-1, len(args),
		))
	}
	roleTypes := strings.Join(roleQueries, " UNION ALL ")
	query := fmt.Sprintf(`
		WITH %s,
		role_types AS (%s),
		type_counts AS (
			SELECT item.type_id, COUNT(DISTINCT eligible.fit_hash)::int AS fit_count
			FROM eligible JOIN fitting_items item USING (fit_hash)
			GROUP BY item.type_id
		),
		role_counts AS (
			SELECT role_types.role_id, COUNT(DISTINCT eligible.fit_hash)::int AS fit_count
			FROM eligible JOIN fitting_items item USING (fit_hash)
			JOIN role_types ON role_types.type_id=item.type_id
			GROUP BY role_types.role_id
		)
		SELECT 'type' AS kind, type_id::text AS key, fit_count FROM type_counts
		UNION ALL
		SELECT 'role' AS kind, role_id AS key, fit_count FROM role_counts`, eligibility, roleTypes)
	return query, args, nil
}

func buildFittingEligibilityCTE(req *legacyRequest, ship int32, filters []fittingSearchFilter, resolved resolvedFittingRoles) (string, []any, error) {
	args := []any{ship}
	having := make([]string, 0, len(filters))
	for _, filter := range filters {
		var ids []int32
		if filter.Kind == "role" {
			ids = resolved[filter.RoleID]
		} else {
			ids = []int32{filter.TypeID}
		}
		args = append(args, ids, filter.Count)
		having = append(having, fmt.Sprintf(
			"SUM(CASE WHEN item.type_id = ANY($%d::int[]) THEN item.quantity ELSE 0 END) %s $%d",
			len(args)-1, filter.Op, len(args),
		))
	}
	matched := "matched AS (SELECT fit_hash FROM ship_fit_uses)"
	if len(having) > 0 {
		matched = `matched AS (
			SELECT item.fit_hash FROM fitting_items item
			WHERE item.fit_hash IN (SELECT fit_hash FROM ship_fit_uses)
			GROUP BY item.fit_hash HAVING ` + strings.Join(having, " AND ") + `
		)`
	}
	args, filterSQL, _, _, err := buildFittingFilterSQL(req, "stats", "matched.fit_hash", args)
	if err != nil {
		return "", nil, err
	}
	statWhere := "TRUE"
	if filterSQL != "" {
		statWhere = strings.TrimPrefix(filterSQL, "AND ")
	}
	if req.Query.Get("cap_stable") == "true" {
		statWhere += " AND stats.cap_stable = true"
	}
	cte := fmt.Sprintf(`ship_fit_uses AS (
			SELECT DISTINCT fit_hash FROM killmail_fittings
			WHERE ship_type_id=$1 AND kill_time >= NOW() - INTERVAL '90 days'
		),
		%s,
		eligible AS (
			SELECT matched.fit_hash FROM matched
			LEFT JOIN fitting_stats stats USING (fit_hash)
			WHERE %s
		)`, matched, statWhere)
	return cte, args, nil
}

func parseSearchShip(raw string) (int32, error) {
	number, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if raw == "" || err != nil || math.IsNaN(number) ||
		math.IsInf(number, 0) || math.Trunc(number) != number ||
		number < 1 || number > math.MaxInt32 {
		return 0, apiError(
			http.StatusBadRequest, "ship query param is required",
		)
	}
	return int32(number), nil
}

func parseFittingSearchFilters(raw string) ([]fittingSearchFilter, error) {
	if raw == "" {
		return []fittingSearchFilter{}, nil
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, apiError(http.StatusBadRequest, "filters must be valid JSON")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, apiError(http.StatusBadRequest, "filters must be valid JSON")
	}
	values, ok := decoded.([]any)
	if !ok {
		return nil, apiError(http.StatusBadRequest, "filters must be an array")
	}
	result := make([]fittingSearchFilter, 0, len(values))
	allowedOps := map[string]bool{">=": true, "<=": true, "=": true, ">": true, "<": true}
	for _, value := range values {
		row, ok := value.(map[string]any)
		if !ok || row == nil {
			return nil, apiError(http.StatusBadRequest, "filter must be an object")
		}
		op, ok := row["op"].(string)
		if !ok || !allowedOps[op] {
			display := "undefined"
			if value, exists := row["op"]; exists {
				display = fmt.Sprint(value)
			}
			return nil, apiError(
				http.StatusBadRequest,
				fmt.Sprintf(`unsupported op "%s"`, display),
			)
		}
		count, ok := exactJSONInteger(row["count"])
		if !ok || count < 0 || count > 16 {
			return nil, apiError(
				http.StatusBadRequest, "count must be an integer 0..16",
			)
		}
		if roleID, ok := row["role_id"].(string); ok {
			if _, exists := fittingRoleByID(roleID); !exists {
				return nil, apiError(
					http.StatusBadRequest,
					fmt.Sprintf(`unknown role_id "%s"`, roleID),
				)
			}
			result = append(result, fittingSearchFilter{
				Kind: "role", RoleID: roleID, Op: op, Count: int(count),
			})
			continue
		}
		typeID, validTypeID := exactJSONInteger(row["type_id"])
		if validTypeID && typeID > 0 && typeID <= math.MaxInt32 {
			typeName, _ := row["type_name"].(string)
			result = append(result, fittingSearchFilter{
				Kind: "type", TypeID: int32(typeID),
				TypeName: typeName, Op: op, Count: int(count),
			})
			continue
		}
		return nil, apiError(
			http.StatusBadRequest, "filter must include role_id or type_id",
		)
	}
	if len(result) > 8 {
		return nil, apiError(http.StatusBadRequest, "too many filters (max 8)")
	}
	return result, nil
}

func fittingSearchPageNumber(
	req *legacyRequest,
	name string,
	fallback, minimum, maximum int,
) int {
	if !req.Query.Has(name) {
		return fallback
	}
	raw := strings.TrimSpace(req.Query.Get(name))
	number := float64(0)
	var err error
	if raw != "" {
		number, err = strconv.ParseFloat(raw, 64)
	}
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return fallback
	}
	number = math.Trunc(number)
	if number < float64(minimum) {
		return minimum
	}
	if number > float64(maximum) {
		return maximum
	}
	return int(number)
}

func fittingSearchFiltersJSON(filters []fittingSearchFilter) []map[string]any {
	result := make([]map[string]any, 0, len(filters))
	for _, filter := range filters {
		result = append(result, filter.JSON())
	}
	return result
}

func emptyFittingSearchResponse(
	ship int32,
	offset, limit int,
	filters []map[string]any,
) map[string]any {
	return map[string]any{
		"ship_type_id": ship, "window_days": 90,
		"total": 0, "has_more": false,
		"offset": offset, "limit": limit,
		"filters_applied": filters, "fits": []map[string]any{},
	}
}

func buildFittingSearchQuery(
	req *legacyRequest,
	ship int32,
	filters []fittingSearchFilter,
	resolved resolvedFittingRoles,
	statQuery fittingStatQuery,
	limit, offset int,
) (string, []any, error) {
	args := []any{ship}
	slotSet := make(map[int32]bool)
	slotHintDisabled := false
	for _, filter := range filters {
		if filter.Kind != "role" {
			slotHintDisabled = true
			break
		}
		role, _ := fittingRoleByID(filter.RoleID)
		if len(role.SlotGroups) == 0 {
			slotHintDisabled = true
			break
		}
		for _, slot := range role.SlotGroups {
			slotSet[int32(slot)] = true
		}
	}
	if slotHintDisabled {
		clear(slotSet)
	}
	having := make([]string, 0, len(filters))
	for _, filter := range filters {
		if filter.Kind == "role" {
			typeIDs := resolved[filter.RoleID]
			if len(typeIDs) == 0 {
				continue
			}
			args = append(args, typeIDs)
			idsPlaceholder := len(args)
			args = append(args, filter.Count)
			countPlaceholder := len(args)
			having = append(having, fmt.Sprintf(
				"SUM(CASE WHEN item.type_id = ANY($%d::int[]) "+
					"THEN item.quantity ELSE 0 END) %s $%d",
				idsPlaceholder, filter.Op, countPlaceholder,
			))
		} else {
			args = append(args, filter.TypeID)
			typePlaceholder := len(args)
			args = append(args, filter.Count)
			countPlaceholder := len(args)
			having = append(having, fmt.Sprintf(
				"SUM(CASE WHEN item.type_id = $%d "+
					"THEN item.quantity ELSE 0 END) %s $%d",
				typePlaceholder, filter.Op, countPlaceholder,
			))
		}
	}
	matched := "matched AS (SELECT fit_hash FROM ship_fit_uses)"
	if len(filters) > 0 {
		slotClause := ""
		if len(slotSet) > 0 {
			slots := make([]int32, 0, len(slotSet))
			for slot := range slotSet {
				slots = append(slots, slot)
			}
			args = append(args, slots)
			slotClause = fmt.Sprintf(
				" AND item.slot_group = ANY($%d::int[])", len(args),
			)
		}
		havingClause := "TRUE"
		if len(having) > 0 {
			havingClause = strings.Join(having, " AND ")
		}
		matched = `
			matched AS (
				SELECT item.fit_hash
				FROM fitting_items item
				WHERE item.fit_hash IN (
				  SELECT fit_hash FROM ship_fit_uses
				)` + slotClause + `
				GROUP BY item.fit_hash
				HAVING ` + havingClause + `
			)`
	}
	args = append(args, limit)
	limitPlaceholder := len(args)
	args = append(args, offset)
	offsetPlaceholder := len(args)
	var filterSQL, profileID string
	var filterErr error
	args, filterSQL, profileID, _, filterErr = buildFittingFilterSQL(req, "stats", "matched.fit_hash", args)
	if filterErr != nil {
		return "", nil, filterErr
	}
	statWhere := []string{"TRUE"}
	if filterSQL != "" {
		statWhere = append(statWhere, strings.TrimPrefix(filterSQL, "AND "))
	}
	if statQuery.CapStable {
		statWhere = append(statWhere, "stats.cap_stable = true")
	}
	npcExpression := "NULL::double precision"
	if profileID != "" {
		profile, _ := fitting.NPCProfile(profileID)
		npcExpression = "(" + fitting.NPCDamageEHPExpression("stats", profile) + ")"
	}
	repairExpression := "GREATEST(COALESCE(stats.shield_effective_boost,0),COALESCE(stats.armor_effective_repair,0),COALESCE(stats.hull_effective_repair,0),COALESCE(stats.passive_shield_effective,0))"
	order := map[string]string{"uses": "uses.total_uses DESC, uses.last_used DESC", "dps": "stats.dps_with_reload DESC NULLS LAST", "ehp": "stats.ehp DESC NULLS LAST", "alpha": "stats.alpha DESC NULLS LAST", "speed": "stats.max_velocity DESC NULLS LAST", "align": "stats.align_time ASC NULLS LAST", "repair": repairExpression + " DESC", "npc_ehp": npcExpression + " DESC NULLS LAST"}[statQuery.Sort]
	query := fmt.Sprintf(`
		WITH ship_fit_uses AS (
			SELECT kf.fit_hash,
			       COUNT(*)::int AS total_uses,
			       MAX(kf.kill_time) AS last_used
			FROM killmail_fittings kf
			WHERE kf.ship_type_id = $1
			  AND kf.kill_time >= NOW() - INTERVAL '90 days'
			GROUP BY kf.fit_hash
		),
		%s,
		stats_matched AS (
			SELECT matched.fit_hash FROM matched
			LEFT JOIN fitting_stats stats USING (fit_hash)
			WHERE %s
		),
		result_count AS (
			SELECT COUNT(*)::int AS total FROM stats_matched
		),
		page AS (
			SELECT fit.fit_hash, fit.family_hash, fit.ship_type_id,
			       ship.name AS ship_name,
			       uses.total_uses, uses.last_used,
			       (to_jsonb(stats) - 'fit_hash' - 'ship_type_id') ||
			         jsonb_build_object('repair', %s, 'npc_profile', %s, 'npc_ehp', %s) AS stats,
			       ROW_NUMBER() OVER (ORDER BY %s) AS sort_rank
			FROM stats_matched
			JOIN ship_fit_uses uses USING (fit_hash)
			JOIN fittings fit USING (fit_hash)
			LEFT JOIN fitting_stats stats USING (fit_hash)
			LEFT JOIN inv_types ship
			  ON ship.type_id = fit.ship_type_id
			ORDER BY %s
			LIMIT $%d OFFSET $%d
		)
		SELECT page.*, result_count.total AS result_total
		FROM result_count
		LEFT JOIN page ON TRUE
		ORDER BY page.sort_rank NULLS LAST`,
		matched, strings.Join(statWhere, " AND "), repairExpression, "'"+strings.ReplaceAll(profileID, "'", "''")+"'", npcExpression, order, order,
		limitPlaceholder, offsetPlaceholder,
	)
	return query, args, nil
}
