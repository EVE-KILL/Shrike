package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type batchStatsRequest struct {
	IDs  []int64 `json:"ids"`
	Type string  `json:"type"`
	From string  `json:"from"`
	To   string  `json:"to"`
}

type batchEntityConfig struct {
	key        string
	idColumn   string
	entityType int
}

func registerBatchStatsRoutes(a huma.API, opts Options) {
	for _, config := range []batchEntityConfig{
		{key: "characters", idColumn: "character_id", entityType: entityCharacter},
		{key: "corporations", idColumn: "corporation_id", entityType: entityCorporation},
		{key: "alliances", idColumn: "alliance_id", entityType: entityAlliance},
	} {
		config := config
		registerLegacy(a, huma.Operation{
			OperationID: config.key + "-batch-stats",
			Method:      http.MethodPost,
			Path:        "/" + config.key + "/stats",
			Summary:     "Batch " + config.key + " statistics",
			Tags:        []string{"stats"},
		}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
			return batchStats(ctx, opts.DB, config, req)
		})
	}
}

func batchStats(
	ctx context.Context,
	db Database,
	config batchEntityConfig,
	req *legacyRequest,
) (legacyPayload, error) {
	var body batchStatsRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		return legacyPayload{}, err
	}
	if len(body.IDs) == 0 {
		return legacyPayload{}, apiError(http.StatusBadRequest, "Missing ids array")
	}
	if len(body.IDs) > 100 {
		return legacyPayload{}, apiError(http.StatusBadRequest, "Maximum 100 IDs per request")
	}

	statsType := body.Type
	if statsType == "" {
		statsType = req.Query.Get("type")
	}
	if statsType == "" {
		statsType = "alltime"
	}
	period := "alltime"
	periodType := 2
	from, to := "", ""
	if statsType == "weekly" {
		period = "weekly"
		periodType = 0
		from = statsWindowDate("7d")
	} else if statsType == "range" {
		from, to = body.From, body.To
		if from == "" {
			from = req.Query.Get("from")
		}
		if to == "" {
			to = req.Query.Get("to")
		}
		if from == "" || to == "" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"from and to required for type=range",
			)
		}
		period = from + " to " + to
		periodType = 0
	}

	ids := make([]int32, 0, len(body.IDs))
	for _, id := range body.IDs {
		ids = append(ids, int32(id))
	}
	stats, err := loadBatchScalarStats(
		ctx, db, config.entityType, ids, periodType, from, to,
	)
	if err != nil {
		return legacyPayload{}, err
	}
	ships, err := loadBatchTopShips(
		ctx, db, config.entityType, ids, periodType, from, to,
	)
	if err != nil {
		return legacyPayload{}, err
	}

	names, err := loadBatchNames(ctx, db, config, ids)
	if err != nil {
		return legacyPayload{}, err
	}
	shipNames, err := loadBatchShipNames(ctx, db, ships)
	if err != nil {
		return legacyPayload{}, err
	}

	results := make([]map[string]any, 0, len(body.IDs))
	for _, id := range body.IDs {
		s := stats[id]
		topShips := make([]map[string]any, 0, len(ships[id]))
		for _, ship := range ships[id] {
			name := shipNames[ship.DimID]
			if name == "" {
				name = "Unknown"
			}
			topShips = append(topShips, map[string]any{
				"ship_type_id": ship.DimID,
				"ship_name":    name,
				"kills":        ship.Kills,
				"losses":       ship.Losses,
			})
		}
		results = append(results, map[string]any{
			"id":             id,
			"name":           names[id],
			"kills":          s.Kills,
			"losses":         s.Losses,
			"solo_kills":     s.SoloKills,
			"npc_losses":     s.NPCLosses,
			"isk_destroyed":  s.ISKDestroyed,
			"isk_lost":       s.ISKLost,
			"points":         s.Points,
			"final_blows":    s.FinalBlows,
			"damage_dealt":   s.DamageDealt,
			"damage_taken":   s.DamageTaken,
			"efficiency":     efficiency(s.Kills, s.Losses),
			"isk_efficiency": iskEfficiency(s.ISKDestroyed, s.ISKLost),
			"topShips":       topShips,
		})
	}
	return jsonPayload(map[string]any{"period": period, "results": results}), nil
}

func statsWindowDate(window string) string {
	_, date := statsWindow(window)
	return date
}

func loadBatchScalarStats(
	ctx context.Context,
	db Database,
	entityType int,
	ids []int32,
	periodType int,
	from, to string,
) (map[int64]entityStats, error) {
	args := []any{entityType, ids, periodType}
	query := `
		SELECT entity_id,
		       COALESCE(SUM(kills), 0)::bigint AS kills,
		       COALESCE(SUM(losses), 0)::bigint AS losses,
		       COALESCE(SUM(solo_kills), 0)::bigint AS solo_kills,
		       COALESCE(SUM(solo_losses), 0)::bigint AS solo_losses,
		       COALESCE(SUM(npc_losses), 0)::bigint AS npc_losses,
		       COALESCE(SUM(final_blows), 0)::bigint AS final_blows,
		       COALESCE(SUM(points), 0)::bigint AS points,
		       COALESCE(SUM(isk_destroyed), 0)::double precision AS isk_destroyed,
		       COALESCE(SUM(isk_lost), 0)::double precision AS isk_lost,
		       COALESCE(SUM(damage_dealt), 0)::bigint AS damage_dealt,
		       COALESCE(SUM(damage_taken), 0)::bigint AS damage_taken,
		       COALESCE(SUM(sum_attacker_count), 0)::bigint AS sum_attacker_count
		FROM stats
		WHERE entity_type = $1 AND entity_id = ANY($2::int[])
		  AND period_type = $3`
	if from != "" {
		args = append(args, from)
		query += fmt.Sprintf(" AND period_start >= $%d::date", len(args))
	}
	if to != "" {
		args = append(args, to)
		query += fmt.Sprintf(" AND period_start <= $%d::date", len(args))
	}
	query += " GROUP BY entity_id"
	rows, err := queryMaps(ctx, db, query, args...)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]entityStats, len(ids))
	for _, row := range rows {
		id, _ := int64Value(row["entity_id"])
		result[id] = statsFromMap(row)
	}
	return result, nil
}

func loadBatchTopShips(
	ctx context.Context,
	db Database,
	entityType int,
	ids []int32,
	periodType int,
	from, to string,
) (map[int64][]entityBreakdown, error) {
	args := []any{entityType, ids, periodType, dimShipFlown, dimShipLost}
	dateFilter := ""
	if from != "" {
		args = append(args, from)
		dateFilter += fmt.Sprintf(" AND period_start >= $%d::date", len(args))
	}
	if to != "" {
		args = append(args, to)
		dateFilter += fmt.Sprintf(" AND period_start <= $%d::date", len(args))
	}
	rows, err := queryMaps(ctx, db, `
		WITH combined AS (
			SELECT entity_id, dim_id AS ship_type_id,
			       SUM(CASE WHEN dim_category = $4 THEN kills ELSE 0 END)::int AS kills,
			       SUM(CASE WHEN dim_category = $5 THEN losses ELSE 0 END)::int AS losses
			FROM stats_breakdowns
			WHERE entity_type = $1
			  AND entity_id = ANY($2::int[])
			  AND period_type = $3
			  AND dim_category IN ($4, $5)`+dateFilter+`
			GROUP BY entity_id, dim_id
		)
		SELECT entity_id, ship_type_id, kills, losses
		FROM (
			SELECT *, ROW_NUMBER() OVER (
				PARTITION BY entity_id
				ORDER BY kills + losses DESC, ship_type_id
			) AS rn
			FROM combined
		) ranked
		WHERE rn <= 5
		ORDER BY entity_id, rn`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	result := map[int64][]entityBreakdown{}
	for _, row := range rows {
		entityID, _ := int64Value(row["entity_id"])
		shipID, _ := int64Value(row["ship_type_id"])
		kills, _ := int64Value(row["kills"])
		losses, _ := int64Value(row["losses"])
		result[entityID] = append(result[entityID], entityBreakdown{
			DimID: shipID, Kills: kills, Losses: losses,
		})
	}
	return result, nil
}

func loadBatchNames(
	ctx context.Context,
	db Database,
	config batchEntityConfig,
	ids []int32,
) (map[int64]any, error) {
	rows, err := queryMaps(ctx, db,
		`SELECT `+config.idColumn+` AS id, name
		 FROM `+config.key+`
		 WHERE `+config.idColumn+` = ANY($1::int[])`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]any, len(rows))
	for _, row := range rows {
		id, _ := int64Value(row["id"])
		result[id] = row["name"]
	}
	return result, nil
}

func loadBatchShipNames(
	ctx context.Context,
	db Database,
	ships map[int64][]entityBreakdown,
) (map[int64]string, error) {
	values := []any{}
	for _, rows := range ships {
		for _, row := range rows {
			values = append(values, row.DimID)
		}
	}
	ids := int32Slice(values...)
	if len(ids) == 0 {
		return map[int64]string{}, nil
	}
	rows, err := queryMaps(ctx, db,
		`SELECT type_id AS id, name FROM inv_types
		 WHERE type_id = ANY($1::int[])`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]string, len(rows))
	for _, row := range rows {
		id, _ := int64Value(row["id"])
		name, _ := stringValue(row["name"])
		result[id] = name
	}
	return result, nil
}
