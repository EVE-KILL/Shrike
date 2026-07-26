package api

import (
	"context"
	"math"
)

func trendingFittingsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		rows, err := queryMaps(ctx, opts.DB, `
			WITH fit_uses AS (
				SELECT kf.fit_hash, fit.family_hash, fit.ship_type_id,
				       COUNT(*)::int AS uses,
				       MAX(kf.kill_time) AS last_used
				FROM killmail_fittings kf
				JOIN fittings fit ON fit.fit_hash = kf.fit_hash
				WHERE kf.kill_time >= NOW() - INTERVAL '30 days'
				  AND kf.ship_type_id NOT IN (
				    SELECT type_id FROM inv_types
				    WHERE group_id IN (29, 237)
				  )
				GROUP BY kf.fit_hash, fit.family_hash, fit.ship_type_id
			),
			family_totals AS (
				SELECT family_hash, ship_type_id,
				       SUM(uses)::int AS total_uses
				FROM fit_uses
				GROUP BY family_hash, ship_type_id
				ORDER BY total_uses DESC
				LIMIT 12
			),
			canonical AS (
				SELECT DISTINCT ON (uses.family_hash, uses.ship_type_id)
				       uses.family_hash, uses.ship_type_id,
				       uses.fit_hash AS canonical_fit_hash,
				       uses.uses AS canonical_uses,
				       uses.last_used AS canonical_last_used
				FROM fit_uses uses
				JOIN family_totals totals
				  ON totals.family_hash = uses.family_hash
				 AND totals.ship_type_id = uses.ship_type_id
				ORDER BY uses.family_hash, uses.ship_type_id,
				         uses.uses DESC, uses.fit_hash
			)
			SELECT totals.family_hash, totals.ship_type_id,
			       ship.name AS ship_name, totals.total_uses,
			       canonical.canonical_fit_hash,
			       canonical.canonical_uses,
			       canonical.canonical_last_used AS last_used
			FROM family_totals totals
			JOIN canonical
			  ON canonical.family_hash = totals.family_hash
			 AND canonical.ship_type_id = totals.ship_type_id
			LEFT JOIN inv_types ship
			  ON ship.type_id = totals.ship_type_id
			ORDER BY totals.total_uses DESC`)
		if err != nil {
			return legacyPayload{}, err
		}
		if len(rows) == 0 {
			return jsonPayload(map[string]any{
				"window_days": fittingRecentDays,
				"families":    []map[string]any{},
			}), nil
		}
		hashes, hulls := catalogueRowKeys(rows)
		contents, err := loadCatalogueContents(ctx, opts.DB, hashes, hulls)
		if err != nil {
			return legacyPayload{}, err
		}
		families := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			hash := stringOrEmpty(row["canonical_fit_hash"])
			shipID := int64OrZero(row["ship_type_id"])
			totalUses := int64OrZero(row["total_uses"])
			canonicalUses := int64OrZero(row["canonical_uses"])
			hullCost := any(nil)
			if price, ok := contents.Prices[shipID]; ok {
				hullCost = price
			}
			families = append(families, map[string]any{
				"family_hash":  row["family_hash"],
				"ship_type_id": shipID, "ship_name": row["ship_name"],
				"canonical_fit_hash": hash,
				"total_uses":         totalUses, "canonical_uses": canonicalUses,
				"variant_count": totalUses - canonicalUses,
				"last_used":     row["last_used"], "fit_cost": contents.CostByHash[hash],
				"hull_cost": hullCost,
				"modules":   catalogueList(contents.ModulesByHash, hash),
				"drones":    catalogueList(contents.DronesByHash, hash),
			})
		}
		return jsonPayload(map[string]any{
			"window_days": fittingRecentDays, "families": families,
		}), nil
	}
}

func catalogueRowKeys(rows []map[string]any) ([]string, []int32) {
	hashes := make([]string, 0, len(rows))
	hulls := make([]int32, 0, len(rows))
	for _, row := range rows {
		hashes = append(hashes, stringOrEmpty(row["canonical_fit_hash"]))
		hulls = append(hulls, int32(int64OrZero(row["ship_type_id"])))
	}
	return hashes, hulls
}

func allianceDoctrineHandler(opts Options) legacyHandler {
	return func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		rows, err := queryMaps(ctx, opts.DB, `
			WITH excluded_ship_types AS (
				SELECT type_id FROM inv_types WHERE group_id IN (29, 237)
			),
			active_alliances AS (
				SELECT kf.victim_alliance_id AS alliance_id,
				       COUNT(*)::int AS total_losses
				FROM killmail_fittings kf
				WHERE kf.kill_time >= NOW() - INTERVAL '30 days'
				  AND kf.victim_alliance_id IS NOT NULL
				  AND kf.ship_type_id NOT IN (
				    SELECT type_id FROM excluded_ship_types
				  )
				GROUP BY kf.victim_alliance_id
				ORDER BY total_losses DESC
				LIMIT 10
			),
			alliance_fit_uses AS (
				SELECT kf.victim_alliance_id AS alliance_id,
				       kf.fit_hash, fit.family_hash, fit.ship_type_id,
				       COUNT(*)::int AS uses,
				       MAX(kf.kill_time) AS last_used
				FROM killmail_fittings kf
				JOIN fittings fit ON fit.fit_hash = kf.fit_hash
				JOIN active_alliances active
				  ON active.alliance_id = kf.victim_alliance_id
				WHERE kf.kill_time >= NOW() - INTERVAL '30 days'
				  AND kf.ship_type_id NOT IN (
				    SELECT type_id FROM excluded_ship_types
				  )
				GROUP BY kf.victim_alliance_id, kf.fit_hash,
				         fit.family_hash, fit.ship_type_id
			),
			alliance_family_totals AS (
				SELECT alliance_id, family_hash, ship_type_id,
				       SUM(uses)::int AS family_uses
				FROM alliance_fit_uses
				GROUP BY alliance_id, family_hash, ship_type_id
			),
			alliance_top_family AS (
				SELECT DISTINCT ON (alliance_id)
				       alliance_id, family_hash, ship_type_id, family_uses
				FROM alliance_family_totals
				ORDER BY alliance_id, family_uses DESC, family_hash
			),
			canonical AS (
				SELECT DISTINCT ON (top.alliance_id)
				       top.alliance_id,
				       uses.fit_hash AS canonical_fit_hash,
				       uses.last_used
				FROM alliance_top_family top
				JOIN alliance_fit_uses uses
				  ON uses.alliance_id = top.alliance_id
				 AND uses.family_hash = top.family_hash
				 AND uses.ship_type_id = top.ship_type_id
				ORDER BY top.alliance_id, uses.uses DESC, uses.fit_hash
			)
			SELECT active.alliance_id,
			       alliance.name AS alliance_name,
			       active.total_losses, top.family_hash,
			       top.ship_type_id, ship.name AS ship_name,
			       canonical.canonical_fit_hash,
			       top.family_uses AS doctrine_uses,
			       canonical.last_used
			FROM active_alliances active
			JOIN alliance_top_family top
			  ON top.alliance_id = active.alliance_id
			JOIN canonical
			  ON canonical.alliance_id = active.alliance_id
			LEFT JOIN alliances alliance
			  ON alliance.alliance_id = active.alliance_id
			LEFT JOIN inv_types ship
			  ON ship.type_id = top.ship_type_id
			ORDER BY active.total_losses DESC`)
		if err != nil {
			return legacyPayload{}, err
		}
		if len(rows) == 0 {
			return jsonPayload(map[string]any{
				"window_days": fittingRecentDays,
				"doctrines":   []map[string]any{},
			}), nil
		}
		hashes, hulls := catalogueRowKeys(rows)
		contents, err := loadCatalogueContents(ctx, opts.DB, hashes, hulls)
		if err != nil {
			return legacyPayload{}, err
		}
		doctrines := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			hash := stringOrEmpty(row["canonical_fit_hash"])
			shipID := int64OrZero(row["ship_type_id"])
			total := int64OrZero(row["total_losses"])
			uses := int64OrZero(row["doctrine_uses"])
			share := float64(0)
			if total > 0 {
				share = math.Round(float64(uses)/float64(total)*1000) / 10
			}
			hullCost := any(nil)
			if price, ok := contents.Prices[shipID]; ok {
				hullCost = price
			}
			doctrines = append(doctrines, map[string]any{
				"alliance_id":   row["alliance_id"],
				"alliance_name": row["alliance_name"],
				"total_losses":  total, "family_hash": row["family_hash"],
				"ship_type_id": shipID, "ship_name": row["ship_name"],
				"canonical_fit_hash": hash, "doctrine_uses": uses,
				"doctrine_share": share, "last_used": row["last_used"],
				"fit_cost": contents.CostByHash[hash], "hull_cost": hullCost,
				"modules": catalogueList(contents.ModulesByHash, hash),
				"drones":  catalogueList(contents.DronesByHash, hash),
			})
		}
		return jsonPayload(map[string]any{
			"window_days": fittingRecentDays, "doctrines": doctrines,
		}), nil
	}
}
