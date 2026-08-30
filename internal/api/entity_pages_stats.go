package api

import (
	"context"
	"math"
	"strconv"
	"time"
)

const (
	dimDiesToCorporation = 21
	dimDiesToAlliance    = 22
	dimKilledCorporation = 31
	dimKilledAlliance    = 32
)

func loadEntityPageStats(
	ctx context.Context,
	opts Options,
	kind string,
	id int64,
	req *legacyRequest,
) (any, error) {
	entityType, ok := entityPageStatsType(kind)
	if !ok {
		return nil, apiError(400, "Invalid entity type")
	}
	days := entityPageQueryInt(req.Query.Get("days"), 0)
	window := entityPageDaysWindow(days)
	return loadEntityDashboardStats(ctx, opts.DB, kind, entityType, id, days, window)
}

func entityPageStatsType(kind string) (int, bool) {
	switch kind {
	case entityPageCharacter:
		return entityCharacter, true
	case entityPageCorporation:
		return entityCorporation, true
	case entityPageAlliance:
		return entityAlliance, true
	default:
		return 0, false
	}
}

func entityPageQueryInt(raw string, fallback int) int {
	if stringsTrimmed := stringTrimSpace(raw); stringsTrimmed != "" {
		value, err := strconv.ParseFloat(stringsTrimmed, 64)
		if err == nil && !math.IsNaN(value) && !math.IsInf(value, 0) {
			if value > float64(math.MaxInt) {
				return math.MaxInt
			}
			if value < float64(math.MinInt) {
				return math.MinInt
			}
			return int(value)
		}
	}
	return fallback
}

func stringTrimSpace(value string) string {
	start, end := 0, len(value)
	for start < end {
		switch value[start] {
		case ' ', '\t', '\r', '\n':
			start++
		default:
			goto trailing
		}
	}
trailing:
	for end > start {
		switch value[end-1] {
		case ' ', '\t', '\r', '\n':
			end--
		default:
			return value[start:end]
		}
	}
	return value[start:end]
}

func entityPageDaysWindow(days int) string {
	switch {
	case days <= 0:
		return "alltime"
	case days <= 1:
		return "1d"
	case days <= 7:
		return "7d"
	case days <= 14:
		return "14d"
	case days <= 30:
		return "30d"
	case days <= 90:
		return "90d"
	case days <= 180:
		return "180d"
	default:
		return "365d"
	}
}

func loadEntityDashboardStats(
	ctx context.Context,
	db Database,
	kind string,
	entityType int,
	id int64,
	days int,
	window string,
) (map[string]any, error) {
	periodType, fromDate := statsWindow(window)
	statsSQL := `
		SELECT
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
		WHERE entity_type = $1 AND entity_id = $2 AND period_type = $3`
	statsArgs := []any{entityType, id, periodType}
	if fromDate != "" {
		statsSQL += ` AND period_start >= $4::date`
		statsArgs = append(statsArgs, fromDate)
	}
	breakdownSQL := `
		WITH grouped AS (
			SELECT dim_category, dim_id,
			       COALESCE(SUM(kills), 0)::bigint AS kills,
			       COALESCE(SUM(losses), 0)::bigint AS losses,
			       COALESCE(SUM(isk_destroyed), 0)::double precision AS isk_destroyed,
			       COALESCE(SUM(isk_lost), 0)::double precision AS isk_lost
			FROM stats_breakdowns
			WHERE entity_type = $1 AND entity_id = $2
			  AND period_type = $3
			  AND dim_category = $4`
	if fromDate != "" {
		breakdownSQL += ` AND period_start >= $5::date`
	}
	breakdownSQL += `
			GROUP BY dim_category, dim_id
		)
		SELECT grouped.dim_category, grouped.dim_id,
		       grouped.kills, grouped.losses,
		       grouped.isk_destroyed, grouped.isk_lost,
		       CASE
		         WHEN grouped.dim_category IN (0, 1) THEN COALESCE(t.name, 'Unknown')
		         WHEN grouped.dim_category = 21 THEN COALESCE(c.name, 'Unknown')
		         WHEN grouped.dim_category = 22 THEN COALESCE(a.name, 'Unknown')
		       END AS name
		FROM grouped
		LEFT JOIN inv_types t
		  ON grouped.dim_category IN (0, 1) AND t.type_id = grouped.dim_id
		LEFT JOIN corporations c
		  ON grouped.dim_category = 21 AND c.corporation_id = grouped.dim_id
		LEFT JOIN alliances a
		  ON grouped.dim_category = 22 AND a.alliance_id = grouped.dim_id
		ORDER BY CASE
		  WHEN grouped.dim_category IN (1, 21, 22) THEN grouped.losses
		  ELSE grouped.kills
		END DESC, grouped.dim_id
		LIMIT 10`

	queries := []databaseQuery{
		{SQL: statsSQL, Args: statsArgs},
	}
	// Each category is an independent index range. Running the four exact
	// aggregates concurrently cuts cold-cache dashboard latency without a
	// second 60+ GB covering index on stats_breakdowns.
	for _, category := range []int16{
		dimShipFlown, dimShipLost, dimDiesToCorporation, dimDiesToAlliance,
	} {
		args := []any{entityType, id, periodType, category}
		if fromDate != "" {
			args = append(args, fromDate)
		}
		queries = append(queries, databaseQuery{SQL: breakdownSQL, Args: args})
	}
	breakdownResultEnd := len(queries)
	characterResultStart := breakdownResultEnd
	if kind == entityPageCharacter {
		since := time.Date(2003, time.January, 1, 0, 0, 0, 0, time.UTC)
		if days > 0 {
			// Bound arithmetic to time.Duration's useful range. The stats
			// window is already snapped to 365d, while this raw-table detail
			// only exists for the activity visualization.
			rawDays := min(days, 100000)
			since = time.Now().UTC().AddDate(0, 0, -rawDays)
		}
		queries = append(queries,
			databaseQuery{
				SQL: `
					WITH activity AS (
						SELECT EXTRACT(HOUR FROM a.killmail_time)::int AS hour,
						       EXTRACT(DOW FROM a.killmail_time)::int AS dow,
						       COUNT(*)::bigint AS kills, 0::bigint AS losses
						FROM killmail_attackers a
						WHERE a.character_id = $1
						  AND a.killmail_time >= $2
						GROUP BY hour, dow
						UNION ALL
						SELECT EXTRACT(HOUR FROM k.killmail_time)::int AS hour,
						       EXTRACT(DOW FROM k.killmail_time)::int AS dow,
						       0::bigint AS kills, COUNT(*)::bigint AS losses
						FROM killmails k
						WHERE k.victim_character_id = $1
						  AND k.killmail_time >= $2
						GROUP BY hour, dow
					)
					SELECT hour, dow, SUM(kills)::bigint AS kills,
					       SUM(losses)::bigint AS losses
					FROM activity
					GROUP BY hour, dow
					ORDER BY dow, hour`,
				Args: []any{id, since},
			},
			databaseQuery{
				SQL: `
					WITH mine AS MATERIALIZED (
						SELECT killmail_id
						FROM killmail_attackers
						WHERE character_id = $1 AND killmail_time >= $2
					), together AS (
						SELECT 0 AS target_type, a.corporation_id::bigint AS id,
						       COUNT(DISTINCT a.killmail_id)::bigint AS count
						FROM killmail_attackers a
						JOIN mine m USING (killmail_id)
						WHERE a.corporation_id IS NOT NULL
						  AND a.character_id IS NOT NULL
						  AND a.character_id <> $1
						GROUP BY a.corporation_id
						UNION ALL
						SELECT 1 AS target_type, a.alliance_id::bigint AS id,
						       COUNT(DISTINCT a.killmail_id)::bigint AS count
						FROM killmail_attackers a
						JOIN mine m USING (killmail_id)
						WHERE a.alliance_id IS NOT NULL
						  AND a.character_id IS NOT NULL
						  AND a.character_id <> $1
						GROUP BY a.alliance_id
					), ranked AS (
						SELECT together.*,
						       ROW_NUMBER() OVER (
						         PARTITION BY target_type
						         ORDER BY count DESC, id
						       ) AS rank
						FROM together
					)
					SELECT r.target_type, r.id, r.count,
					       CASE WHEN r.target_type = 0
						         THEN COALESCE(c.name, 'Unknown')
						         ELSE COALESCE(a.name, 'Unknown') END AS name
					FROM ranked r
					LEFT JOIN corporations c
					  ON r.target_type = 0 AND c.corporation_id = r.id
					LEFT JOIN alliances a
					  ON r.target_type = 1 AND a.alliance_id = r.id
					WHERE r.rank <= 10
					ORDER BY r.target_type, r.rank`,
				Args: []any{id, since},
			},
		)
	}

	results, err := queryMapsConcurrent(ctx, db, queries...)
	if err != nil {
		return nil, err
	}
	stats := firstOrEmpty(results[0])
	breakdowns := []map[string]any{}
	for _, rows := range results[1:breakdownResultEnd] {
		breakdowns = append(breakdowns, rows...)
	}
	topUsed := []map[string]any{}
	topLost := []map[string]any{}
	diesCorps := []map[string]any{}
	diesAlliances := []map[string]any{}
	for _, row := range breakdowns {
		category := int64OrZero(row["dim_category"])
		switch category {
		case dimShipFlown:
			topUsed = append(topUsed, map[string]any{
				"ship_type_id": row["dim_id"], "ship_name": row["name"],
				"count": row["kills"],
			})
		case dimShipLost:
			topLost = append(topLost, map[string]any{
				"ship_type_id": row["dim_id"], "ship_name": row["name"],
				"count": row["losses"],
			})
		case dimDiesToCorporation:
			diesCorps = append(diesCorps, map[string]any{
				"id": row["dim_id"], "name": row["name"],
				"count": row["losses"], "isk_value": row["isk_lost"],
			})
		case dimDiesToAlliance:
			diesAlliances = append(diesAlliances, map[string]any{
				"id": row["dim_id"], "name": row["name"],
				"count": row["losses"], "isk_value": row["isk_lost"],
			})
		}
	}

	kills := int64OrZero(stats["kills"])
	losses := int64OrZero(stats["losses"])
	iskDestroyed := float64OrZero(stats["isk_destroyed"])
	iskLost := float64OrZero(stats["isk_lost"])
	statsOut := map[string]any{
		"kills": kills, "losses": losses,
		"solo_kills":    int64OrZero(stats["solo_kills"]),
		"npc_losses":    int64OrZero(stats["npc_losses"]),
		"isk_destroyed": iskDestroyed, "isk_lost": iskLost,
		"points":         int64OrZero(stats["points"]),
		"final_blows":    int64OrZero(stats["final_blows"]),
		"efficiency":     efficiency(kills, losses),
		"isk_efficiency": iskEfficiency(iskDestroyed, iskLost),
	}
	out := map[string]any{
		"stats":              statsOut,
		"topShipsUsed":       topUsed,
		"topShipsLost":       topLost,
		"diesToCorporations": diesCorps,
		"diesToAlliances":    diesAlliances,
	}
	if kind != entityPageCharacter {
		return out, nil
	}

	statsOut["solo_losses"] = int64OrZero(stats["solo_losses"])
	statsOut["damage_dealt"] = int64OrZero(stats["damage_dealt"])
	statsOut["damage_taken"] = int64OrZero(stats["damage_taken"])
	if kills > 0 {
		blob := float64(int64OrZero(stats["sum_attacker_count"])) / float64(kills)
		statsOut["blob_factor"] = math.Round(blob*100) / 100
	} else {
		statsOut["blob_factor"] = float64(0)
	}

	heatMap := make(map[int]int64, 24)
	killsMatrix := make([][]int64, 7)
	lossesMatrix := make([][]int64, 7)
	for day := range 7 {
		killsMatrix[day] = make([]int64, 24)
		lossesMatrix[day] = make([]int64, 24)
	}
	for hour := range 24 {
		heatMap[hour] = 0
	}
	for _, row := range results[characterResultStart] {
		hour := int(int64OrZero(row["hour"]))
		day := int(int64OrZero(row["dow"]))
		if hour < 0 || hour >= 24 || day < 0 || day >= 7 {
			continue
		}
		killsAt := int64OrZero(row["kills"])
		lossesAt := int64OrZero(row["losses"])
		heatMap[hour] += killsAt
		killsMatrix[day][hour] = killsAt
		lossesMatrix[day][hour] = lossesAt
	}
	peakHour, peakCount := 0, int64(0)
	for hour := range 24 {
		if heatMap[hour] > peakCount {
			peakHour, peakCount = hour, heatMap[hour]
		}
	}
	switch {
	case peakHour >= 7 && peakHour <= 11:
		statsOut["active_timezone"] = "AUTZ"
	case peakHour >= 12 && peakHour <= 21:
		statsOut["active_timezone"] = "EUTZ"
	default:
		statsOut["active_timezone"] = "USTZ"
	}

	flownWithCorps := []map[string]any{}
	flownWithAlliances := []map[string]any{}
	for _, row := range results[characterResultStart+1] {
		item := map[string]any{
			"id": row["id"], "name": row["name"], "count": row["count"],
		}
		if int64OrZero(row["target_type"]) == 0 {
			flownWithCorps = append(flownWithCorps, item)
		} else {
			flownWithAlliances = append(flownWithAlliances, item)
		}
	}
	out["heatMap"] = heatMap
	out["activity"] = map[string]any{
		"kills": killsMatrix, "losses": lossesMatrix,
	}
	out["fliesWithCorporations"] = flownWithCorps
	out["fliesWithAlliances"] = flownWithAlliances
	return out, nil
}
