package api

import (
	"context"
	"math"
	"net/http"
)

func trendingFittingsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		mode := req.Query.Get("mode")
		if mode == "" {
			mode = "kills"
		}
		if mode != "kills" && mode != "final_blows" && mode != "losses" {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid fitting ranking mode")
		}

		var query string
		if mode == "losses" {
			query = `
			WITH fit_uses AS (
				SELECT kf.fit_hash, fit.family_hash, fit.ship_type_id,
				       COUNT(*)::int AS uses,
				       MAX(kf.kill_time) AS last_used
				FROM killmail_fittings kf
				JOIN fittings fit ON fit.fit_hash = kf.fit_hash
				WHERE kf.kill_time >= NOW() - INTERVAL '7 days'
				  AND kf.ship_type_id NOT IN (
				    SELECT type_id FROM inv_types
				    WHERE group_id IN (29, 237)
				  )
				GROUP BY kf.fit_hash, fit.family_hash, fit.ship_type_id
			),
			family_totals AS (
				SELECT family_hash, ship_type_id,
				       SUM(uses)::int AS total_uses,
				       COUNT(*)::int AS variant_count
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
			       totals.variant_count,
			       totals.total_uses AS ranking_count,
			       canonical.canonical_fit_hash,
			       canonical.canonical_uses,
			       canonical.canonical_last_used AS last_used
			FROM family_totals totals
			JOIN canonical
			  ON canonical.family_hash = totals.family_hash
			 AND canonical.ship_type_id = totals.ship_type_id
			LEFT JOIN inv_types ship
			  ON ship.type_id = totals.ship_type_id
			ORDER BY totals.total_uses DESC`
		} else {
			shipActivitySQL := `
				SELECT daily.entity_id AS ship_type_id,
				       SUM(daily.kills)::int AS ranking_count
				FROM stats daily
				JOIN top_family top ON top.ship_type_id = daily.entity_id
				WHERE daily.entity_type = 3
				  AND daily.period_type = 0
				  AND daily.period_start >= CURRENT_DATE - 6
				GROUP BY daily.entity_id
				ORDER BY ranking_count DESC
				LIMIT 12`
			if mode == "final_blows" {
				shipActivitySQL = `
				SELECT attacker.ship_type_id,
				       COUNT(*)::int AS ranking_count
				FROM killmail_attackers attacker
				JOIN top_family top ON top.ship_type_id = attacker.ship_type_id
				WHERE attacker.killmail_time >= NOW() - INTERVAL '7 days'
				  AND attacker.character_id >= 90000000
				  AND attacker.final_blow IS TRUE
				GROUP BY attacker.ship_type_id
				ORDER BY ranking_count DESC
				LIMIT 12`
			}
			query = `
			WITH excluded_ship_types AS (
				SELECT type_id FROM inv_types WHERE group_id IN (29, 237)
			),
			fit_uses AS MATERIALIZED (
				SELECT kf.fit_hash, fit.family_hash, fit.ship_type_id,
				       COUNT(*)::int AS uses,
				       MAX(kf.kill_time) AS last_used
				FROM killmail_fittings kf
				JOIN fittings fit ON fit.fit_hash = kf.fit_hash
				WHERE kf.kill_time >= NOW() - INTERVAL '7 days'
				  AND kf.ship_type_id NOT IN (SELECT type_id FROM excluded_ship_types)
				GROUP BY kf.fit_hash, fit.family_hash, fit.ship_type_id
			),
			family_totals AS (
				SELECT family_hash, ship_type_id,
				       SUM(uses)::int AS total_uses,
				       COUNT(*)::int AS variant_count
				FROM fit_uses
				GROUP BY family_hash, ship_type_id
			),
			top_family AS (
				SELECT DISTINCT ON (ship_type_id)
				       ship_type_id, family_hash, total_uses, variant_count
				FROM family_totals
				ORDER BY ship_type_id, total_uses DESC, family_hash
			),
			canonical AS (
				SELECT DISTINCT ON (uses.ship_type_id)
				       uses.ship_type_id, uses.fit_hash AS canonical_fit_hash,
				       uses.uses AS canonical_uses, uses.last_used AS canonical_last_used
				FROM fit_uses uses
				JOIN top_family top
				  ON top.ship_type_id = uses.ship_type_id
				 AND top.family_hash = uses.family_hash
				ORDER BY uses.ship_type_id, uses.uses DESC, uses.fit_hash
			),
			ship_activity AS MATERIALIZED (` + shipActivitySQL + `
			)
			SELECT top.family_hash, activity.ship_type_id,
			       ship.name AS ship_name, top.total_uses, top.variant_count,
			       activity.ranking_count,
			       canonical.canonical_fit_hash,
			       canonical.canonical_uses,
			       canonical.canonical_last_used AS last_used
			FROM ship_activity activity
			JOIN top_family top ON top.ship_type_id = activity.ship_type_id
			JOIN canonical ON canonical.ship_type_id = activity.ship_type_id
			LEFT JOIN inv_types ship ON ship.type_id = activity.ship_type_id
			ORDER BY activity.ranking_count DESC`
		}

		rows, err := queryMaps(ctx, opts.DB, query)
		if err != nil {
			return legacyPayload{}, err
		}
		if len(rows) == 0 {
			return jsonPayload(map[string]any{
				"window_days":  fittingTrendingDays,
				"ranking_mode": mode,
				"families":     []map[string]any{},
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
				"ranking_count": int64OrZero(row["ranking_count"]),
				"variant_count": int64OrZero(row["variant_count"]),
				"last_used":     row["last_used"], "fit_cost": contents.CostByHash[hash],
				"hull_cost": hullCost,
				"modules":   catalogueList(contents.ModulesByHash, hash),
				"drones":    catalogueList(contents.DronesByHash, hash),
			})
		}
		return jsonPayload(map[string]any{
			"window_days":  fittingTrendingDays,
			"ranking_mode": mode, "families": families,
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
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		entityType := req.Query.Get("entity_type")
		if entityType == "" {
			entityType = "alliance"
		}
		if entityType != "alliance" && entityType != "corporation" {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid doctrine entity type")
		}

		entityColumn := "victim_alliance_id"
		entityTable := "alliances"
		entityIDColumn := "alliance_id"
		if entityType == "corporation" {
			entityColumn = "victim_corporation_id"
			entityTable = "corporations"
			entityIDColumn = "corporation_id"
		}

		rows, err := queryMaps(ctx, opts.DB, `
			WITH excluded_ship_types AS (
				SELECT type_id FROM inv_types WHERE group_id IN (29, 237)
			),
			active_entities AS (
				SELECT kf.`+entityColumn+` AS entity_id,
				       COUNT(*)::int AS total_losses
				FROM killmail_fittings kf
				WHERE kf.kill_time >= NOW() - INTERVAL '30 days'
				  AND kf.`+entityColumn+` IS NOT NULL
				  AND kf.ship_type_id NOT IN (
				    SELECT type_id FROM excluded_ship_types
				  )
				GROUP BY kf.`+entityColumn+`
				ORDER BY total_losses DESC
				LIMIT 10
			),
			entity_fit_uses AS (
				SELECT kf.`+entityColumn+` AS entity_id,
				       kf.fit_hash, fit.family_hash, fit.ship_type_id,
				       COUNT(*)::int AS uses,
				       MAX(kf.kill_time) AS last_used
				FROM killmail_fittings kf
				JOIN fittings fit ON fit.fit_hash = kf.fit_hash
				JOIN active_entities active
				  ON active.entity_id = kf.`+entityColumn+`
				WHERE kf.kill_time >= NOW() - INTERVAL '30 days'
				  AND kf.ship_type_id NOT IN (
				    SELECT type_id FROM excluded_ship_types
				  )
				GROUP BY kf.`+entityColumn+`, kf.fit_hash,
				         fit.family_hash, fit.ship_type_id
			),
			entity_family_totals AS (
				SELECT entity_id, family_hash, ship_type_id,
				       SUM(uses)::int AS family_uses
				FROM entity_fit_uses
				GROUP BY entity_id, family_hash, ship_type_id
			),
			entity_top_family AS (
				SELECT DISTINCT ON (entity_id)
				       entity_id, family_hash, ship_type_id, family_uses
				FROM entity_family_totals
				ORDER BY entity_id, family_uses DESC, family_hash
			),
			canonical AS (
				SELECT DISTINCT ON (top.entity_id)
				       top.entity_id,
				       uses.fit_hash AS canonical_fit_hash,
				       uses.last_used
				FROM entity_top_family top
				JOIN entity_fit_uses uses
				  ON uses.entity_id = top.entity_id
				 AND uses.family_hash = top.family_hash
				 AND uses.ship_type_id = top.ship_type_id
				ORDER BY top.entity_id, uses.uses DESC, uses.fit_hash
			)
			SELECT active.entity_id,
			       entity.name AS entity_name,
			       active.total_losses, top.family_hash,
			       top.ship_type_id, ship.name AS ship_name,
			       canonical.canonical_fit_hash,
			       top.family_uses AS doctrine_uses,
			       canonical.last_used
			FROM active_entities active
			JOIN entity_top_family top
			  ON top.entity_id = active.entity_id
			JOIN canonical
			  ON canonical.entity_id = active.entity_id
			LEFT JOIN `+entityTable+` entity
			  ON entity.`+entityIDColumn+` = active.entity_id
			LEFT JOIN inv_types ship
			  ON ship.type_id = top.ship_type_id
			ORDER BY active.total_losses DESC`)
		if err != nil {
			return legacyPayload{}, err
		}
		if len(rows) == 0 {
			return jsonPayload(map[string]any{
				"window_days": fittingRecentDays,
				"entity_type": entityType,
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
				"entity_id":    row["entity_id"],
				"entity_name":  row["entity_name"],
				"total_losses": total, "family_hash": row["family_hash"],
				"ship_type_id": shipID, "ship_name": row["ship_name"],
				"canonical_fit_hash": hash, "doctrine_uses": uses,
				"doctrine_share": share, "last_used": row["last_used"],
				"fit_cost": contents.CostByHash[hash], "hull_cost": hullCost,
				"modules": catalogueList(contents.ModulesByHash, hash),
				"drones":  catalogueList(contents.DronesByHash, hash),
			})
		}
		return jsonPayload(map[string]any{
			"window_days": fittingRecentDays, "entity_type": entityType,
			"doctrines": doctrines,
		}), nil
	}
}
