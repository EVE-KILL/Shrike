package api

import (
	"context"
	"math"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// analyzeBodyLimit fits the documented 2500 IDs with room for whitespace.
const analyzeBodyLimit = 64 << 10

// analyzeRequest is both the decode target and the documented schema.
type analyzeRequest struct {
	CharacterIDs requestList[int64] `json:"character_ids" minItems:"1" maxItems:"2500" doc:"Characters to analyze, at most 2500 per request."`
}

func registerAnalyzeRoute(a huma.API, opts Options) {
	registerLegacyJSON(a, huma.Operation{
		OperationID: "character-analyze",
		Method:      http.MethodPost,
		Path:        "/characters/analyze",
		Summary:     "Analyze characters",
		Tags:        []string{"characters"},
	}, analyzeBodyLimit, func(
		ctx context.Context, req *legacyRequest, body *analyzeRequest,
	) (legacyPayload, error) {
		if len(body.CharacterIDs) == 0 {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"character_ids must be a non-empty array",
			)
		}
		if len(body.CharacterIDs) > 2500 {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"Maximum 2500 characters per request",
			)
		}
		ids := make([]int32, 0, len(body.CharacterIDs))
		for _, id := range body.CharacterIDs {
			if id <= 0 || id > math.MaxInt32 {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"All character_ids must be positive integers",
				)
			}
			ids = append(ids, int32(id))
		}
		data, err := analyzeCharacters(ctx, opts.DB, body.CharacterIDs, ids)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"data": data}), nil
	})
}

func analyzeCharacters(
	ctx context.Context,
	db Database,
	requestIDs []int64,
	ids []int32,
) ([]map[string]any, error) {
	queryResults, err := queryMapsConcurrent(ctx, db,
		databaseQuery{SQL: `
			SELECT entity_id,
			       COALESCE(SUM(kills), 0)::int AS kills,
			       COALESCE(SUM(losses), 0)::int AS losses,
			       COALESCE(SUM(solo_kills), 0)::int AS solo_kills,
			       COALESCE(SUM(sum_attacker_count), 0)::bigint AS sum_attacker_count
			FROM stats
			WHERE entity_type = 0
			  AND entity_id = ANY($1::int[])
			  AND period_type = 0
			  AND period_start >= CURRENT_DATE - 90
			GROUP BY entity_id`, Args: []any{ids}},
		databaseQuery{SQL: `
			SELECT cid AS character_id, ships.ship_type_id,
			       ships.ship_name, ships.kill_count
			FROM unnest($1::int[]) AS cid
			CROSS JOIN LATERAL (
			  SELECT a.ship_type_id, t.name AS ship_name, a.kill_count
			  FROM (
			    SELECT ship_type_id, COUNT(*) AS kill_count,
			           MAX(killmail_time) AS last_kill_time
			    FROM killmail_attackers
			    WHERE character_id = cid
			      AND ship_type_id IS NOT NULL
			      AND killmail_time >= NOW() - INTERVAL '90 days'
			    GROUP BY ship_type_id
			    ORDER BY last_kill_time DESC
			    LIMIT 5
			  ) a
			  JOIN inv_types t ON t.type_id = a.ship_type_id
			) ships`, Args: []any{ids}},
		databaseQuery{SQL: `
			SELECT cid AS character_id, loss.victim_ship_type_id,
			       loss.killmail_id, loss.killmail_time
			FROM unnest($1::int[]) AS cid
			CROSS JOIN LATERAL (
			  SELECT victim_ship_type_id, killmail_id, killmail_time
			  FROM killmails
			  WHERE victim_character_id = cid
			    AND killmail_time >= NOW() - INTERVAL '90 days'
			  ORDER BY killmail_id DESC
			  LIMIT 5
			) loss`, Args: []any{ids}},
		databaseQuery{SQL: `
			SELECT cid AS character_id, cyno.total_checked, cyno.cyno_losses
			FROM unnest($1::int[]) AS cid
			CROSS JOIN LATERAL (
			  SELECT COUNT(*)::int AS total_checked,
			         COUNT(*) FILTER (WHERE EXISTS (
			           SELECT 1 FROM killmail_items i
			           WHERE i.killmail_id = k.killmail_id
			             AND i.type_id IN (21096, 28646, 52694)
			         ))::int AS cyno_losses
			  FROM (
			    SELECT killmail_id FROM killmails
			    WHERE victim_character_id = cid
			      AND killmail_time >= NOW() - INTERVAL '90 days'
			    ORDER BY killmail_id DESC
			    LIMIT 50
			  ) k
			) cyno`, Args: []any{ids}},
	)
	if err != nil {
		return nil, err
	}
	statsRows, shipRows := queryResults[0], queryResults[1]
	lossRows, cynoRows := queryResults[2], queryResults[3]

	stats := map[int64]map[string]any{}
	for _, row := range statsRows {
		id, _ := int64Value(row["entity_id"])
		stats[id] = row
	}
	cynos := map[int64]map[string]any{}
	for _, row := range cynoRows {
		id, _ := int64Value(row["character_id"])
		cynos[id] = row
	}
	losses := map[[2]int64]map[string]any{}
	for _, row := range lossRows {
		characterID, _ := int64Value(row["character_id"])
		shipID, _ := int64Value(row["victim_ship_type_id"])
		key := [2]int64{characterID, shipID}
		if losses[key] == nil {
			losses[key] = map[string]any{
				"killmail_id":   row["killmail_id"],
				"killmail_time": row["killmail_time"],
			}
		}
	}
	ships := map[int64][]map[string]any{}
	for _, row := range shipRows {
		characterID, _ := int64Value(row["character_id"])
		shipID, _ := int64Value(row["ship_type_id"])
		kills, _ := int64Value(row["kill_count"])
		ships[characterID] = append(ships[characterID], map[string]any{
			"ship_type_id": shipID,
			"ship_name":    row["ship_name"],
			"kill_count":   kills,
			"last_loss":    losses[[2]int64{characterID, shipID}],
		})
	}

	result := make([]map[string]any, 0, len(requestIDs))
	for _, id := range requestIDs {
		s := stats[id]
		kills, _ := int64Value(s["kills"])
		lossCount, _ := int64Value(s["losses"])
		soloKills, _ := int64Value(s["solo_kills"])
		attackerCount, _ := int64Value(s["sum_attacker_count"])
		cyno := cynos[id]
		totalChecked, _ := int64Value(cyno["total_checked"])
		cynoLosses, _ := int64Value(cyno["cyno_losses"])
		characterShips := ships[id]
		if characterShips == nil {
			characterShips = []map[string]any{}
		}
		result = append(result, map[string]any{
			"character_id":      id,
			"total_kills":       kills,
			"total_losses":      lossCount,
			"efficiency":        roundedRatio(kills, kills+lossCount, 100),
			"gang_probability":  roundedRatio(kills-soloKills, kills, 100),
			"average_gang_size": roundedRatio(attackerCount, kills, 1),
			"last_5_ships":      characterShips,
			"cyno_probability":  roundedRatio(cynoLosses, totalChecked, 100),
		})
	}
	return result, nil
}

func roundedRatio(numerator, denominator int64, scale float64) float64 {
	if denominator == 0 {
		return 0
	}
	return math.Round(float64(numerator)/float64(denominator)*scale*100) / 100
}
