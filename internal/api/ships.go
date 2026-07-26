package api

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

const (
	fittingWindowDays  = 90
	defaultFitFamilies = 20
)

func registerShipRoutes(a huma.API, opts Options) {
	registerLegacy(a, entityIDOperation(
		"ship-fittings", "/ships/{id}/fittings",
		"Popular ship fittings", "ships",
	), func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		limit := boundedQueryInt(req, "limit", defaultFitFamilies, 1, 50)
		moduleIDs := parseFittingModuleIDs(req.Query.Get("modules"))
		body, err := loadShipFittings(ctx, opts.DB, id, limit, moduleIDs)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(body), nil
	})
}

func parseFittingModuleIDs(raw string) []int32 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	result := []int32{}
	seen := map[int32]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		number, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number <= 0 {
			continue
		}
		id := int32(number)
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
		if len(result) == 5 {
			break
		}
	}
	return result
}

func loadShipFittings(
	ctx context.Context,
	db Database,
	shipID int64,
	limit int,
	moduleIDs []int32,
) (map[string]any, error) {
	var echoedModules any
	if moduleIDs != nil {
		echoedModules = moduleIDs
	}
	moduleRoots, err := resolveModuleRoots(ctx, db, moduleIDs)
	if err != nil {
		return nil, err
	}
	rare, err := queryMap(ctx, db,
		`SELECT type_id FROM custom_prices WHERE type_id = $1 LIMIT 1`, shipID)
	if err != nil {
		return nil, err
	}
	isRare := rare != nil

	var matchingHashes []string
	if moduleRoots != nil {
		rows, err := queryMaps(ctx, db, `
			WITH ship_fits AS (
			  SELECT DISTINCT kf.fit_hash
			  FROM killmail_fittings kf
			  WHERE kf.ship_type_id = $1
			    AND kf.kill_time >= now() - INTERVAL '90 days'
			)
			SELECT fi.fit_hash
			FROM fitting_items fi
			JOIN ship_fits sf ON sf.fit_hash = fi.fit_hash
			LEFT JOIN inv_types t ON t.type_id = fi.type_id
			LEFT JOIN inv_types ct ON ct.type_id = fi.charge_type_id
			CROSS JOIN LATERAL (VALUES
			  (COALESCE(t.variation_parent_type_id, fi.type_id)),
			  (COALESCE(ct.variation_parent_type_id, fi.charge_type_id))
			) AS roots(root)
			WHERE roots.root IS NOT NULL
			  AND roots.root = ANY($2::int[])
			GROUP BY fi.fit_hash
			HAVING COUNT(DISTINCT roots.root) = $3`,
			shipID, moduleRoots, len(moduleRoots))
		if err != nil {
			return nil, err
		}
		matchingHashes = make([]string, 0, len(rows))
		for _, row := range rows {
			hash, _ := stringValue(row["fit_hash"])
			matchingHashes = append(matchingHashes, hash)
		}
		if len(matchingHashes) == 0 {
			return map[string]any{
				"ship_type_id":  shipID,
				"window_days":   fittingWindowDays,
				"module_filter": echoedModules,
				"is_rare_hull":  isRare,
				"hull_cost":     nil,
				"families":      []any{},
			}, nil
		}
	}

	families, err := loadFittingFamilies(
		ctx, db, shipID, limit, isRare, matchingHashes, moduleRoots != nil,
	)
	if err != nil {
		return nil, err
	}
	if len(families) == 0 {
		return map[string]any{
			"ship_type_id":  shipID,
			"window_days":   fittingWindowDays,
			"module_filter": echoedModules,
			"is_rare_hull":  isRare,
			"families":      []any{},
		}, nil
	}

	canonicalHashes := make([]string, 0, len(families))
	familyHashes := make([]string, 0, len(families))
	for _, family := range families {
		canonical, _ := stringValue(family["canonical_fit_hash"])
		hash, _ := stringValue(family["family_hash"])
		canonicalHashes = append(canonicalHashes, canonical)
		familyHashes = append(familyHashes, hash)
	}
	items, err := queryMaps(ctx, db, `
		SELECT fi.fit_hash, fi.slot_group, fi.ordinal, fi.type_id, t.name
		FROM fitting_items fi
		LEFT JOIN inv_types t ON t.type_id = fi.type_id
		WHERE fi.fit_hash = ANY($1::text[])
		ORDER BY fi.fit_hash, fi.slot_group, fi.ordinal`, canonicalHashes)
	if err != nil {
		return nil, err
	}
	allianceRows, err := loadFittingAlliances(
		ctx, db, shipID, familyHashes, matchingHashes, moduleRoots != nil,
	)
	if err != nil {
		return nil, err
	}

	priceIDs := []int32{int32(shipID)}
	for _, item := range items {
		id, _ := int64Value(item["type_id"])
		priceIDs = append(priceIDs, int32(id))
	}
	priceIDs = int32Slice(anySlice(priceIDs)...)
	prices, err := loadFittingPrices(ctx, db, priceIDs)
	if err != nil {
		return nil, err
	}
	hullCost := any(nil)
	if price, ok := prices[shipID]; ok {
		hullCost = price
	}

	itemsByHash := map[string][]map[string]any{}
	fitCosts := map[string]float64{}
	for _, item := range items {
		hash, _ := stringValue(item["fit_hash"])
		typeID, _ := int64Value(item["type_id"])
		itemsByHash[hash] = append(itemsByHash[hash], map[string]any{
			"slot_group": item["slot_group"],
			"ordinal":    item["ordinal"],
			"type_id":    item["type_id"],
			"name":       item["name"],
		})
		fitCosts[hash] += prices[typeID]
	}
	alliancesByFamily := map[string][]map[string]any{}
	for _, row := range allianceRows {
		hash, _ := stringValue(row["family_hash"])
		uses, _ := int64Value(row["uses"])
		total, _ := int64Value(row["alliance_ship_total"])
		percentage := float64(0)
		if total > 0 {
			percentage = math.Round(float64(uses)/float64(total)*1000) / 10
		}
		alliancesByFamily[hash] = append(
			alliancesByFamily[hash],
			map[string]any{
				"alliance_id":            row["alliance_id"],
				"name":                   row["alliance_name"],
				"uses":                   uses,
				"pct_of_alliance_losses": percentage,
			},
		)
	}

	outputFamilies := make([]map[string]any, 0, len(families))
	for _, family := range families {
		hash, _ := stringValue(family["family_hash"])
		canonical, _ := stringValue(family["canonical_fit_hash"])
		total, _ := int64Value(family["total_uses"])
		canonicalUses, _ := int64Value(family["canonical_uses"])
		moduleList := itemsByHash[canonical]
		if moduleList == nil {
			moduleList = []map[string]any{}
		}
		topAlliances := alliancesByFamily[hash]
		if topAlliances == nil {
			topAlliances = []map[string]any{}
		}
		outputFamilies = append(outputFamilies, map[string]any{
			"family_hash":        hash,
			"canonical_fit_hash": canonical,
			"total_uses":         total,
			"canonical_uses":     canonicalUses,
			"variant_count":      total - canonicalUses,
			"last_used":          family["last_used"],
			"fit_cost":           fitCosts[canonical],
			"modules":            moduleList,
			"top_alliances":      topAlliances,
		})
	}
	return map[string]any{
		"ship_type_id":  shipID,
		"window_days":   fittingWindowDays,
		"module_filter": echoedModules,
		"is_rare_hull":  isRare,
		"hull_cost":     hullCost,
		"families":      outputFamilies,
	}, nil
}

func resolveModuleRoots(
	ctx context.Context,
	db Database,
	moduleIDs []int32,
) ([]int32, error) {
	if moduleIDs == nil {
		return nil, nil
	}
	rows, err := queryMaps(ctx, db, `
		SELECT type_id, variation_parent_type_id AS parent
		FROM inv_types WHERE type_id = ANY($1::int[])`, moduleIDs)
	if err != nil {
		return nil, err
	}
	parents := map[int32]int32{}
	for _, row := range rows {
		id, _ := int64Value(row["type_id"])
		parent, ok := int64Value(row["parent"])
		if !ok {
			parent = id
		}
		parents[int32(id)] = int32(parent)
	}
	roots := []int32{}
	seen := map[int32]struct{}{}
	for _, id := range moduleIDs {
		root := id
		if parent, ok := parents[id]; ok {
			root = parent
		}
		if _, exists := seen[root]; !exists {
			seen[root] = struct{}{}
			roots = append(roots, root)
		}
	}
	return roots, nil
}

func loadFittingFamilies(
	ctx context.Context,
	db Database,
	shipID int64,
	limit int,
	isRare bool,
	matchingHashes []string,
	filtered bool,
) ([]map[string]any, error) {
	args := []any{shipID}
	filter := ""
	if filtered {
		args = append(args, matchingHashes)
		filter = fmt.Sprintf(" AND kf.fit_hash = ANY($%d::text[])", len(args))
	}
	minUses := 3
	if isRare {
		minUses = 0
	}
	args = append(args, minUses, limit)
	return queryMaps(ctx, db, `
		WITH fit_uses AS (
		  SELECT kf.fit_hash, f.family_hash, COUNT(*)::int AS uses,
		         MAX(kf.kill_time) AS last_used
		  FROM killmail_fittings kf
		  JOIN fittings f ON f.fit_hash = kf.fit_hash
		  WHERE kf.ship_type_id = $1
		    AND kf.kill_time >= now() - INTERVAL '90 days'`+filter+`
		  GROUP BY kf.fit_hash, f.family_hash
		),
		canonical AS (
		  SELECT DISTINCT ON (family_hash)
		         family_hash, fit_hash AS canonical_fit_hash,
		         uses AS canonical_uses, last_used AS canonical_last_used
		  FROM fit_uses
		  ORDER BY family_hash, uses DESC, fit_hash
		),
		family_totals AS (
		  SELECT family_hash, SUM(uses)::int AS total_uses
		  FROM fit_uses GROUP BY family_hash
		)
		SELECT c.family_hash, c.canonical_fit_hash, ft.total_uses,
		       c.canonical_uses, c.canonical_last_used AS last_used
		FROM canonical c
		JOIN family_totals ft ON ft.family_hash = c.family_hash
		WHERE ft.total_uses >= $`+fmt.Sprintf("%d", len(args)-1)+`
		ORDER BY ft.total_uses DESC, c.canonical_last_used DESC
		LIMIT $`+fmt.Sprintf("%d", len(args)),
		args...,
	)
}

func loadFittingAlliances(
	ctx context.Context,
	db Database,
	shipID int64,
	familyHashes, matchingHashes []string,
	filtered bool,
) ([]map[string]any, error) {
	args := []any{shipID, familyHashes}
	filter := ""
	if filtered {
		args = append(args, matchingHashes)
		filter = fmt.Sprintf(" AND kf.fit_hash = ANY($%d::text[])", len(args))
	}
	return queryMaps(ctx, db, `
		WITH family_fits AS (
		  SELECT f.family_hash, kf.victim_alliance_id
		  FROM killmail_fittings kf
		  JOIN fittings f ON f.fit_hash = kf.fit_hash
		  WHERE kf.ship_type_id = $1
		    AND kf.kill_time >= now() - INTERVAL '90 days'
		    AND kf.victim_alliance_id IS NOT NULL
		    AND f.family_hash = ANY($2::text[])`+filter+`
		),
		alliance_family_uses AS (
		  SELECT family_hash, victim_alliance_id, COUNT(*)::int AS uses
		  FROM family_fits GROUP BY family_hash, victim_alliance_id
		),
		alliance_ship_total AS (
		  SELECT victim_alliance_id, COUNT(*)::int AS total_losses
		  FROM killmail_fittings
		  WHERE ship_type_id = $1
		    AND kill_time >= now() - INTERVAL '90 days'
		    AND victim_alliance_id IS NOT NULL
		  GROUP BY victim_alliance_id
		),
		ranked AS (
		  SELECT afu.family_hash,
		         afu.victim_alliance_id AS alliance_id,
		         afu.uses, ast.total_losses AS alliance_ship_total,
		         ROW_NUMBER() OVER (
		           PARTITION BY afu.family_hash
		           ORDER BY afu.uses DESC, afu.victim_alliance_id
		         ) AS rnk
		  FROM alliance_family_uses afu
		  JOIN alliance_ship_total ast
		    ON ast.victim_alliance_id = afu.victim_alliance_id
		)
		SELECT r.family_hash, r.alliance_id,
		       a.name AS alliance_name, r.uses, r.alliance_ship_total
		FROM ranked r
		LEFT JOIN alliances a ON a.alliance_id = r.alliance_id
		WHERE r.rnk <= 3
		ORDER BY r.family_hash, r.uses DESC`,
		args...,
	)
}

func loadFittingPrices(
	ctx context.Context,
	db Database,
	ids []int32,
) (map[int64]float64, error) {
	jita, err := queryMaps(ctx, db, `
		SELECT DISTINCT ON (type_id) type_id, average AS price
		FROM prices
		WHERE type_id = ANY($1::int[]) AND region_id = 10000002
		ORDER BY type_id, date DESC`, ids)
	if err != nil {
		return nil, err
	}
	custom, err := queryMaps(ctx, db, `
		SELECT DISTINCT ON (type_id) type_id, price
		FROM custom_prices
		WHERE type_id = ANY($1::int[])
		ORDER BY type_id, date DESC`, ids)
	if err != nil {
		return nil, err
	}
	result := map[int64]float64{}
	for _, row := range append(jita, custom...) {
		id, _ := int64Value(row["type_id"])
		price, _ := float64Value(row["price"])
		result[id] = price
	}
	return result, nil
}

func anySlice(values []int32) []any {
	result := make([]any, len(values))
	for i, value := range values {
		result[i] = value
	}
	return result
}
