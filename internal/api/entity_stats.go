package api

import (
	"context"
	"fmt"
	"math"
	"time"
)

const (
	entityCharacter   = 0
	entityCorporation = 1
	entityAlliance    = 2

	dimShipFlown = 0
	dimShipLost  = 1
	dimSystem    = 10
)

type entityStats struct {
	Kills            int64
	Losses           int64
	SoloKills        int64
	SoloLosses       int64
	NPCLosses        int64
	FinalBlows       int64
	Points           int64
	ISKDestroyed     float64
	ISKLost          float64
	DamageDealt      int64
	DamageTaken      int64
	SumAttackerCount int64
}

type entityBreakdown struct {
	DimID            int64
	Kills            int64
	Losses           int64
	ISKDestroyed     float64
	ISKLost          float64
	LastKillmailID   any
	LastKillmailTime any
}

func loadEntityStats(
	ctx context.Context,
	db Database,
	entityType int,
	entityID int64,
	window string,
) (entityStats, error) {
	periodType, fromDate := statsWindow(window)
	query := `
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
	args := []any{entityType, entityID, periodType}
	if fromDate != "" {
		query += ` AND period_start >= $4::date`
		args = append(args, fromDate)
	}
	row, err := queryMap(ctx, db, query, args...)
	if err != nil {
		return entityStats{}, err
	}
	return statsFromMap(row), nil
}

func loadRangeStats(
	ctx context.Context,
	db Database,
	entityType int,
	entityID int64,
	from, to string,
) (entityStats, error) {
	row, err := queryMap(ctx, db, `
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
		WHERE entity_type = $1 AND entity_id = $2 AND period_type = 0
		  AND period_start >= $3::date AND period_start <= $4::date`,
		entityType, entityID, from, to,
	)
	if err != nil {
		return entityStats{}, err
	}
	return statsFromMap(row), nil
}

func statsFromMap(row map[string]any) entityStats {
	if row == nil {
		return entityStats{}
	}
	kills, _ := int64Value(row["kills"])
	losses, _ := int64Value(row["losses"])
	soloKills, _ := int64Value(row["solo_kills"])
	soloLosses, _ := int64Value(row["solo_losses"])
	npcLosses, _ := int64Value(row["npc_losses"])
	finalBlows, _ := int64Value(row["final_blows"])
	points, _ := int64Value(row["points"])
	iskDestroyed, _ := float64Value(row["isk_destroyed"])
	iskLost, _ := float64Value(row["isk_lost"])
	damageDealt, _ := int64Value(row["damage_dealt"])
	damageTaken, _ := int64Value(row["damage_taken"])
	sumAttackerCount, _ := int64Value(row["sum_attacker_count"])
	return entityStats{
		Kills: kills, Losses: losses, SoloKills: soloKills,
		SoloLosses: soloLosses, NPCLosses: npcLosses,
		FinalBlows: finalBlows, Points: points,
		ISKDestroyed: iskDestroyed, ISKLost: iskLost,
		DamageDealt: damageDealt, DamageTaken: damageTaken,
		SumAttackerCount: sumAttackerCount,
	}
}

func loadEntityBreakdowns(
	ctx context.Context,
	db Database,
	entityType int,
	entityID int64,
	dimension int,
	window string,
	limit int,
	orderBy string,
) ([]entityBreakdown, error) {
	periodType, fromDate := statsWindow(window)
	allowedOrder := map[string]string{
		"kills":         "SUM(kills)",
		"losses":        "SUM(losses)",
		"isk_destroyed": "SUM(isk_destroyed)",
		"isk_lost":      "SUM(isk_lost)",
	}
	orderColumn := allowedOrder[orderBy]
	if orderColumn == "" {
		orderColumn = "SUM(kills)"
	}
	query := `
		SELECT dim_id,
		       COALESCE(SUM(kills), 0)::bigint AS kills,
		       COALESCE(SUM(losses), 0)::bigint AS losses,
		       COALESCE(SUM(isk_destroyed), 0)::double precision AS isk_destroyed,
		       COALESCE(SUM(isk_lost), 0)::double precision AS isk_lost,
		       MAX(last_killmail_id) AS last_killmail_id,
		       MAX(last_killmail_time) AS last_killmail_time
		FROM stats_breakdowns
		WHERE entity_type = $1 AND entity_id = $2
		  AND dim_category = $3 AND period_type = $4`
	args := []any{entityType, entityID, dimension, periodType}
	if fromDate != "" {
		query += ` AND period_start >= $5::date`
		args = append(args, fromDate)
	}
	args = append(args, limit)
	query += fmt.Sprintf(
		" GROUP BY dim_id ORDER BY %s DESC LIMIT $%d",
		orderColumn, len(args),
	)
	rows, err := queryMaps(ctx, db, query, args...)
	if err != nil {
		return nil, err
	}
	return breakdownsFromMaps(rows), nil
}

func loadRangeBreakdowns(
	ctx context.Context,
	db Database,
	entityType int,
	entityID int64,
	dimension int,
	from, to, orderBy string,
	limit int,
) ([]entityBreakdown, error) {
	orderColumn := "SUM(kills)"
	if orderBy == "losses" {
		orderColumn = "SUM(losses)"
	}
	rows, err := queryMaps(ctx, db, `
		SELECT dim_id,
		       COALESCE(SUM(kills), 0)::bigint AS kills,
		       COALESCE(SUM(losses), 0)::bigint AS losses,
		       COALESCE(SUM(isk_destroyed), 0)::double precision AS isk_destroyed,
		       COALESCE(SUM(isk_lost), 0)::double precision AS isk_lost,
		       MAX(last_killmail_id) AS last_killmail_id,
		       MAX(last_killmail_time) AS last_killmail_time
		FROM stats_breakdowns
		WHERE entity_type = $1 AND entity_id = $2
		  AND dim_category = $3 AND period_type = 0
		  AND period_start >= $4::date AND period_start <= $5::date
		GROUP BY dim_id
		ORDER BY `+orderColumn+` DESC
		LIMIT $6`,
		entityType, entityID, dimension, from, to, limit,
	)
	if err != nil {
		return nil, err
	}
	return breakdownsFromMaps(rows), nil
}

func breakdownsFromMaps(rows []map[string]any) []entityBreakdown {
	result := make([]entityBreakdown, 0, len(rows))
	for _, row := range rows {
		dimID, _ := int64Value(row["dim_id"])
		kills, _ := int64Value(row["kills"])
		losses, _ := int64Value(row["losses"])
		iskDestroyed, _ := float64Value(row["isk_destroyed"])
		iskLost, _ := float64Value(row["isk_lost"])
		result = append(result, entityBreakdown{
			DimID: dimID, Kills: kills, Losses: losses,
			ISKDestroyed: iskDestroyed, ISKLost: iskLost,
			LastKillmailID:   row["last_killmail_id"],
			LastKillmailTime: row["last_killmail_time"],
		})
	}
	return result
}

func statsWindow(window string) (periodType int, fromDate string) {
	if window == "alltime" {
		return 2, ""
	}
	days := map[string]int{
		"1d": 1, "7d": 7, "14d": 14, "30d": 30,
		"90d": 90, "180d": 180, "365d": 365,
	}[window]
	if days == 0 {
		days = 1
	}
	return 0, time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")
}

func efficiency(kills, losses int64) int64 {
	if kills+losses == 0 {
		return 0
	}
	return int64(math.Floor(float64(kills)/float64(kills+losses)*100 + 0.5))
}

func iskEfficiency(destroyed, lost float64) int64 {
	if destroyed+lost == 0 {
		return 0
	}
	return int64(math.Floor(destroyed/(destroyed+lost)*100 + 0.5))
}

func scalarStatsMap(stats entityStats, includeDamage bool) map[string]any {
	result := map[string]any{
		"kills":          stats.Kills,
		"losses":         stats.Losses,
		"solo_kills":     stats.SoloKills,
		"npc_losses":     stats.NPCLosses,
		"isk_destroyed":  stats.ISKDestroyed,
		"isk_lost":       stats.ISKLost,
		"points":         stats.Points,
		"final_blows":    stats.FinalBlows,
		"efficiency":     efficiency(stats.Kills, stats.Losses),
		"isk_efficiency": iskEfficiency(stats.ISKDestroyed, stats.ISKLost),
	}
	if includeDamage {
		result["damage_dealt"] = stats.DamageDealt
		result["damage_taken"] = stats.DamageTaken
	}
	return result
}

func loadBreakdownNames(
	ctx context.Context,
	db Database,
	rows []entityBreakdown,
	table, idColumn, nameColumn string,
) (map[int64]string, error) {
	values := make([]any, 0, len(rows))
	for _, row := range rows {
		values = append(values, row.DimID)
	}
	ids := int32Slice(values...)
	result := map[int64]string{}
	if len(ids) == 0 {
		return result, nil
	}
	nameRows, err := queryMaps(ctx, db,
		`SELECT `+idColumn+` AS id, `+nameColumn+` AS name
		 FROM `+table+` WHERE `+idColumn+` = ANY($1::int[])`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	for _, row := range nameRows {
		id, _ := int64Value(row["id"])
		name, _ := stringValue(row["name"])
		result[id] = name
	}
	return result, nil
}

func assembleTopBreakdowns(
	ctx context.Context,
	db Database,
	flown, lost, systems []entityBreakdown,
) ([]map[string]any, []map[string]any, error) {
	allShips := append(append([]entityBreakdown{}, flown...), lost...)
	shipNames, err := loadBreakdownNames(ctx, db, allShips, "inv_types", "type_id", "name")
	if err != nil {
		return nil, nil, err
	}
	systemNames, err := loadBreakdownNames(ctx, db, systems, "solar_systems", "solar_system_id", "system_name")
	if err != nil {
		return nil, nil, err
	}
	losses := map[int64]int64{}
	for _, row := range lost {
		losses[row.DimID] = row.Losses
	}
	topShips := make([]map[string]any, 0, len(flown))
	for _, row := range flown {
		name := shipNames[row.DimID]
		if name == "" {
			name = "Unknown"
		}
		topShips = append(topShips, map[string]any{
			"ship_type_id": row.DimID,
			"ship_name":    name,
			"kills":        row.Kills,
			"losses":       losses[row.DimID],
		})
	}
	topSystems := make([]map[string]any, 0, len(systems))
	for _, row := range systems {
		name := systemNames[row.DimID]
		if name == "" {
			name = "Unknown"
		}
		topSystems = append(topSystems, map[string]any{
			"solar_system_id": row.DimID,
			"system_name":     name,
			"kills":           row.Kills,
			"losses":          row.Losses,
		})
	}
	return topShips, topSystems, nil
}

func loadTopMembers(
	ctx context.Context,
	db Database,
	membershipColumn string,
	entityID int64,
	window string,
) ([]map[string]any, error) {
	periodType, fromDate := statsWindow(window)
	query := `
		SELECT s.entity_id AS character_id, c.name,
		       COALESCE(SUM(s.kills), 0)::bigint AS kills,
		       COALESCE(SUM(s.losses), 0)::bigint AS losses,
		       COALESCE(SUM(s.isk_destroyed), 0)::double precision AS isk_destroyed,
		       COALESCE(SUM(s.isk_lost), 0)::double precision AS isk_lost
		FROM stats s
		INNER JOIN characters c ON c.character_id = s.entity_id
		WHERE s.entity_type = 0
		  AND c.` + membershipColumn + ` = $1
		  AND s.period_type = $2`
	args := []any{entityID, periodType}
	if fromDate != "" {
		query += ` AND s.period_start >= $3::date`
		args = append(args, fromDate)
	}
	query += ` GROUP BY s.entity_id, c.name
		ORDER BY SUM(s.kills) DESC
		LIMIT 10`
	rows, err := queryMaps(ctx, db, query, args...)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row["name"] == nil || row["name"] == "" {
			row["name"] = "Unknown"
		}
	}
	return rows, nil
}
