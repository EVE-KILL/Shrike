package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

func registerWarRoutes(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "wars",
		Method:      http.MethodGet,
		Path:        "/wars",
		Summary:     "Wars",
		Tags:        []string{"wars"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		limit := boundedQueryInt(req, "limit", 50, 1, 100)
		query := `
			SELECT war_id, declared, started, finished, mutual,
			       aggressor_alliance_id, aggressor_corporation_id,
			       defender_alliance_id, defender_corporation_id,
			       aggressor_ships_killed, defender_ships_killed
			FROM wars`
		args := []any{}
		if after, ok := optionalQueryNumber(req, "after"); ok && after != 0 {
			query += ` WHERE war_id > $1`
			args = append(args, after)
		}
		args = append(args, limit+1)
		query += ` ORDER BY war_id ASC LIMIT $` + numberString(len(args))
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		return paginatedRows(rows, limit, "war_id"), nil
	})

	registerLegacy(a, entityIDOperation(
		"war", "/wars/{id}", "War detail", "wars",
	), func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		war, err := queryMap(ctx, opts.DB, `
			SELECT * FROM wars WHERE war_id = $1 LIMIT 1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if war == nil {
			return foundOr404(nil, "War not found"), nil
		}
		allies, err := queryMaps(ctx, opts.DB, `
			SELECT wa.alliance_id, wa.corporation_id,
			       a.name AS alliance_name, c.name AS corporation_name
			FROM war_allies wa
			LEFT JOIN alliances a ON a.alliance_id = wa.alliance_id
			LEFT JOIN corporations c ON c.corporation_id = wa.corporation_id
			WHERE wa.war_id = $1`, id)
		if err != nil {
			return legacyPayload{}, err
		}

		since := time.Date(2003, 1, 1, 0, 0, 0, 0, time.UTC)
		if started, ok := war["started"].(time.Time); ok {
			since = started
		} else if declared, ok := war["declared"].(time.Time); ok {
			since = declared
		}
		totals, err := queryMap(ctx, opts.DB, `
			SELECT COUNT(*)::bigint AS total_kills,
			       COALESCE(SUM(total_value), 0)::double precision AS total_value
			FROM killmails
			WHERE war_id = $1 AND killmail_time >= $2`, id, since)
		if err != nil {
			return legacyPayload{}, err
		}
		topShips, err := queryMaps(ctx, opts.DB, `
			SELECT k.victim_ship_type_id AS ship_type_id,
			       COALESCE(t.name, 'Unknown') AS ship_name,
			       COUNT(*)::bigint AS count
			FROM killmails k
			LEFT JOIN inv_types t ON t.type_id = k.victim_ship_type_id
			WHERE k.war_id = $1 AND k.killmail_time >= $2
			GROUP BY k.victim_ship_type_id, t.name
			ORDER BY count DESC
			LIMIT 10`, id, since)
		if err != nil {
			return legacyPayload{}, err
		}

		aggressor := warEntity(
			war["aggressor_alliance_id"], war["aggressor_corporation_id"],
			war["aggressor_isk_destroyed"], war["aggressor_ships_killed"],
		)
		defender := warEntity(
			war["defender_alliance_id"], war["defender_corporation_id"],
			war["defender_isk_destroyed"], war["defender_ships_killed"],
		)
		// Resolve the four primary entity names in two small indexed queries.
		if err := resolveWarEntity(ctx, opts.DB, aggressor); err != nil {
			return legacyPayload{}, err
		}
		if err := resolveWarEntity(ctx, opts.DB, defender); err != nil {
			return legacyPayload{}, err
		}
		allyOutput := make([]map[string]any, 0, len(allies))
		for _, ally := range allies {
			if ally["alliance_id"] != nil {
				name := ally["alliance_name"]
				if name == nil || name == "" {
					name = "Unknown"
				}
				allyOutput = append(allyOutput, map[string]any{
					"id": ally["alliance_id"], "name": name, "type": "alliance",
				})
			} else {
				name := ally["corporation_name"]
				if name == nil || name == "" {
					name = "Unknown"
				}
				allyOutput = append(allyOutput, map[string]any{
					"id": ally["corporation_id"], "name": name, "type": "corporation",
				})
			}
		}
		return jsonPayload(map[string]any{
			"war": map[string]any{
				"war_id":          war["war_id"],
				"declared":        war["declared"],
				"started":         war["started"],
				"finished":        war["finished"],
				"retracted":       war["retracted"],
				"mutual":          falseIfNil(war["mutual"]),
				"open_for_allies": falseIfNil(war["open_for_allies"]),
				"aggressor":       aggressor,
				"defender":        defender,
				"allies":          allyOutput,
			},
			"stats": map[string]any{
				"total_kills": totals["total_kills"],
				"total_value": totals["total_value"],
				"top_ships":   topShips,
			},
		}), nil
	})
}

func numberString(value int) string {
	return fmt.Sprintf("%d", value)
}

func warEntity(allianceID, corporationID, iskDestroyed, shipsKilled any) map[string]any {
	result := map[string]any{
		"id":            0,
		"name":          "Unknown",
		"ticker":        "?",
		"type":          "corporation",
		"isk_destroyed": zeroIfNil(iskDestroyed),
		"ships_killed":  zeroIfNil(shipsKilled),
	}
	if allianceID != nil {
		result["id"], result["type"] = allianceID, "alliance"
	} else if corporationID != nil {
		result["id"] = corporationID
	}
	return result
}

func resolveWarEntity(ctx context.Context, db Database, entity map[string]any) error {
	id, _ := int64Value(entity["id"])
	if id == 0 {
		return nil
	}
	table, column := "corporations", "corporation_id"
	if entity["type"] == "alliance" {
		table, column = "alliances", "alliance_id"
	}
	row, err := queryMap(ctx, db,
		`SELECT name, ticker FROM `+table+` WHERE `+column+` = $1 LIMIT 1`, id)
	if err != nil {
		return err
	}
	if row != nil {
		if name, ok := stringValue(row["name"]); ok && name != "" {
			entity["name"] = name
		}
		if ticker, ok := stringValue(row["ticker"]); ok && ticker != "" {
			entity["ticker"] = ticker
		}
	}
	return nil
}
