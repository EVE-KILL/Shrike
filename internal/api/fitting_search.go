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
)

type fittingSearchFilter struct {
	Kind, RoleID, Op, TypeName string
	TypeID                     int32
	Count                      int
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

		query, args := buildFittingSearchQuery(
			ship, filters, resolved, limit, offset,
		)
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
			fitRows = append(fitRows, row)
		}
		if len(fitRows) == 0 {
			response := emptyFittingSearchResponse(ship, offset, limit, applied)
			response["total"] = total
			return jsonPayload(response), nil
		}

		hashes := make([]string, 0, len(fitRows))
		for _, fit := range fitRows {
			hashes = append(hashes, stringOrEmpty(fit["fit_hash"]))
		}
		contents, err := loadCatalogueContents(
			ctx, opts.DB, hashes, []int32{ship},
		)
		if err != nil {
			return legacyPayload{}, err
		}
		shipName := fitRows[0]["ship_name"]
		hullCost := any(nil)
		if price, ok := contents.Prices[int64(ship)]; ok {
			hullCost = price
		}
		fits := make([]map[string]any, 0, len(fitRows))
		for _, fit := range fitRows {
			hash := stringOrEmpty(fit["fit_hash"])
			fits = append(fits, map[string]any{
				"fit_hash": hash, "family_hash": fit["family_hash"],
				"ship_type_id": fit["ship_type_id"], "ship_name": shipName,
				"total_uses": fit["total_uses"], "last_used": fit["last_used"],
				"fit_cost": contents.CostByHash[hash], "hull_cost": hullCost,
				"modules": catalogueList(contents.ModulesByHash, hash),
				"drones":  catalogueList(contents.DronesByHash, hash),
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
	if len(values) > 8 {
		// The TypeScript implementation reports this only after validating
		// each row. Keep that order below by checking again before returning.
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
	ship int32,
	filters []fittingSearchFilter,
	resolved resolvedFittingRoles,
	limit, offset int,
) (string, []any) {
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
		result_count AS (
			SELECT COUNT(*)::int AS total FROM matched
		),
		page AS (
			SELECT fit.fit_hash, fit.family_hash, fit.ship_type_id,
			       ship.name AS ship_name,
			       uses.total_uses, uses.last_used
			FROM matched
			JOIN ship_fit_uses uses USING (fit_hash)
			JOIN fittings fit USING (fit_hash)
			LEFT JOIN inv_types ship
			  ON ship.type_id = fit.ship_type_id
			ORDER BY uses.total_uses DESC, uses.last_used DESC
			LIMIT $%d OFFSET $%d
		)
		SELECT page.*, result_count.total AS result_total
		FROM result_count
		LEFT JOIN page ON TRUE
		ORDER BY page.total_uses DESC NULLS LAST,
		         page.last_used DESC NULLS LAST`,
		matched, limitPlaceholder, offsetPlaceholder,
	)
	return query, args
}
