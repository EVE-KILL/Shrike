package api

import (
	"context"
	"math"
)

func shipFittingMetadataHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		shipID, err := parseUniverseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		results, err := queryMapsConcurrent(ctx, opts.DB,
			databaseQuery{
				SQL: `
					WITH ship_kills AS (
						SELECT kf.fit_hash, COUNT(*)::int AS uses
						FROM killmail_fittings kf
						WHERE kf.ship_type_id = $1
						  AND kf.kill_time >= NOW() - INTERVAL '90 days'
						GROUP BY kf.fit_hash
					),
					total_kills AS (
						SELECT COALESCE(SUM(uses), 0)::int AS total
						FROM ship_kills
					),
					fit_groups AS (
						SELECT DISTINCT kills.fit_hash,
						       groups.group_id,
						       groups.name AS group_name,
						       kills.uses
						FROM ship_kills kills
						JOIN fitting_items item
						  ON item.fit_hash = kills.fit_hash
						JOIN inv_types item_type
						  ON item_type.type_id = item.type_id
						JOIN inv_groups groups
						  ON groups.group_id = item_type.group_id
						WHERE item.slot_group BETWEEN 1 AND 5
					)
					SELECT groups.group_id,
					       groups.group_name,
					       SUM(groups.uses)::int AS kill_count,
					       ROUND(
					         SUM(groups.uses)::numeric /
					         NULLIF(total.total, 0) * 100, 1
					       ) AS pct
					FROM fit_groups groups
					CROSS JOIN total_kills total
					GROUP BY groups.group_id, groups.group_name, total.total
					HAVING SUM(groups.uses)::numeric /
					       NULLIF(total.total, 0) * 100 >= 5
					ORDER BY pct DESC
					LIMIT 15`,
				Args: []any{shipID},
			},
			databaseQuery{
				SQL: `
					SELECT COUNT(*)::bigint AS total
					FROM killmail_fittings
					WHERE ship_type_id = $1
					  AND kill_time >= NOW() - INTERVAL '90 days'`,
				Args: []any{shipID},
			},
		)
		if err != nil {
			return legacyPayload{}, err
		}
		total := int64(0)
		if len(results[1]) > 0 {
			total = int64OrZero(results[1][0]["total"])
		}
		groups := make([]map[string]any, 0, len(results[0]))
		for _, row := range results[0] {
			groups = append(groups, map[string]any{
				"group_id": row["group_id"], "name": row["group_name"],
				"kill_count": row["kill_count"], "pct": row["pct"],
			})
		}
		return jsonPayload(map[string]any{
			"ship_type_id": shipID, "window_days": 90,
			"total_kills": total, "groups": groups,
		}), nil
	}
}

func shipFittingFamiliesHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		shipID, err := parseUniverseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		rare, err := queryMap(ctx, opts.DB, `
			SELECT 1 FROM custom_prices
			WHERE type_id = $1 LIMIT 1`, shipID)
		if err != nil {
			return legacyPayload{}, err
		}
		isRare := rare != nil
		minUses := 3
		if isRare {
			minUses = 0
		}
		families, err := queryMaps(ctx, opts.DB, `
			WITH fit_uses AS (
				SELECT kf.fit_hash, fit.family_hash,
				       COUNT(*)::int AS uses,
				       MAX(kf.kill_time) AS last_used
				FROM killmail_fittings kf
				JOIN fittings fit ON fit.fit_hash = kf.fit_hash
				WHERE kf.ship_type_id = $1
				  AND kf.kill_time >= NOW() - INTERVAL '90 days'
				GROUP BY kf.fit_hash, fit.family_hash
			),
			canonical AS (
				SELECT DISTINCT ON (family_hash)
				       family_hash,
				       fit_hash AS canonical_fit_hash,
				       uses AS canonical_uses,
				       last_used AS canonical_last_used
				FROM fit_uses
				ORDER BY family_hash, uses DESC, fit_hash
			),
			family_totals AS (
				SELECT family_hash, SUM(uses)::int AS total_uses,
				       COUNT(*)::int AS variant_count
				FROM fit_uses
				GROUP BY family_hash
			)
			SELECT canonical.family_hash,
			       canonical.canonical_fit_hash,
			       totals.total_uses,
			       totals.variant_count,
			       canonical.canonical_uses,
			       canonical.canonical_last_used AS last_used
			FROM canonical
			JOIN family_totals totals
			  ON totals.family_hash = canonical.family_hash
			WHERE totals.total_uses >= $2
			ORDER BY totals.total_uses DESC,
			         canonical.canonical_last_used DESC
			LIMIT 20`, shipID, minUses)
		if err != nil {
			return legacyPayload{}, err
		}
		if len(families) == 0 {
			return jsonPayload(map[string]any{
				"ship_type_id": shipID, "window_days": 90,
				"is_rare_hull": isRare, "families": []map[string]any{},
			}), nil
		}
		hashes := make([]string, 0, len(families))
		familyHashes := make([]string, 0, len(families))
		for _, family := range families {
			hashes = append(
				hashes, stringOrEmpty(family["canonical_fit_hash"]),
			)
			familyHashes = append(
				familyHashes, stringOrEmpty(family["family_hash"]),
			)
		}
		contents, err := loadCatalogueContents(
			ctx, opts.DB, hashes, []int32{int32(shipID)},
		)
		if err != nil {
			return legacyPayload{}, err
		}
		allianceRows, err := queryMaps(ctx, opts.DB, `
			WITH family_fits AS (
				SELECT fit.family_hash, kf.victim_alliance_id
				FROM killmail_fittings kf
				JOIN fittings fit ON fit.fit_hash = kf.fit_hash
				WHERE kf.ship_type_id = $1
				  AND kf.kill_time >= NOW() - INTERVAL '90 days'
				  AND kf.victim_alliance_id IS NOT NULL
				  AND fit.family_hash = ANY($2::text[])
			),
			alliance_family_uses AS (
				SELECT family_hash, victim_alliance_id,
				       COUNT(*)::int AS uses
				FROM family_fits
				GROUP BY family_hash, victim_alliance_id
			),
			alliance_ship_total AS (
				SELECT victim_alliance_id,
				       COUNT(*)::int AS total_losses
				FROM killmail_fittings
				WHERE ship_type_id = $1
				  AND kill_time >= NOW() - INTERVAL '90 days'
				  AND victim_alliance_id IS NOT NULL
				GROUP BY victim_alliance_id
			),
			ranked AS (
				SELECT uses.family_hash,
				       uses.victim_alliance_id AS alliance_id,
				       uses.uses,
				       totals.total_losses AS alliance_ship_total,
				       ROW_NUMBER() OVER (
				         PARTITION BY uses.family_hash
				         ORDER BY uses.uses DESC, uses.victim_alliance_id
				       ) AS rank
				FROM alliance_family_uses uses
				JOIN alliance_ship_total totals
				  ON totals.victim_alliance_id = uses.victim_alliance_id
			)
			SELECT ranked.family_hash, ranked.alliance_id,
			       alliance.name AS alliance_name,
			       ranked.uses, ranked.alliance_ship_total
			FROM ranked
			LEFT JOIN alliances alliance
			  ON alliance.alliance_id = ranked.alliance_id
			WHERE ranked.rank <= 3
			ORDER BY ranked.family_hash, ranked.uses DESC`,
			shipID, familyHashes)
		if err != nil {
			return legacyPayload{}, err
		}
		contexts, err := loadFittingContexts(ctx, opts.DB, shipID, familyHashes)
		if err != nil {
			return legacyPayload{}, err
		}
		alliancesByFamily := make(map[string][]map[string]any)
		for _, row := range allianceRows {
			hash := stringOrEmpty(row["family_hash"])
			uses := int64OrZero(row["uses"])
			total := int64OrZero(row["alliance_ship_total"])
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
		output := make([]map[string]any, 0, len(families))
		for _, family := range families {
			familyHash := stringOrEmpty(family["family_hash"])
			canonical := stringOrEmpty(family["canonical_fit_hash"])
			totalUses := int64OrZero(family["total_uses"])
			canonicalUses := int64OrZero(family["canonical_uses"])
			topAlliances := alliancesByFamily[familyHash]
			if topAlliances == nil {
				topAlliances = []map[string]any{}
			}
			output = append(output, map[string]any{
				"family_hash":        familyHash,
				"canonical_fit_hash": canonical,
				"total_uses":         totalUses, "canonical_uses": canonicalUses,
				"variant_count": int64OrZero(family["variant_count"]),
				"last_used":     family["last_used"],
				"fit_cost":      contents.CostByHash[canonical],
				"modules":       catalogueList(contents.ModulesByHash, canonical),
				"drones":        catalogueList(contents.DronesByHash, canonical),
				"top_alliances": topAlliances,
				"context":       contexts[familyHash],
			})
		}
		hullCost := any(nil)
		if price, ok := contents.Prices[shipID]; ok {
			hullCost = price
		}
		return jsonPayload(map[string]any{
			"ship_type_id": shipID, "window_days": 90,
			"is_rare_hull": isRare, "hull_cost": hullCost,
			"families": output,
		}), nil
	}
}
