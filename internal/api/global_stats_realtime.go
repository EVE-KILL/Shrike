package api

import (
	"context"
	"fmt"
	"math"
)

const (
	realtimeStatsMaximumHours = 24
	realtimeStatsMinimumMins  = 10
)

type realtimeGlobalStatsQuery struct {
	from, id, name, timestamp, aggregate, predicate, resultType string
	isk                                                         bool
}

// Every SQL fragment below is static and selected by a validated dataType.
// Window and limit values remain parameters.
var realtimeGlobalStatsQueries = map[string]realtimeGlobalStatsQuery{
	"characters": {
		"killmail_attackers a JOIN characters n ON n.character_id = a.character_id",
		"a.character_id", "n.name", "a.killmail_time", "COUNT(*)::bigint",
		"a.final_blow = TRUE AND a.character_id >= 90000000", "character", false,
	},
	"corporations": {
		"killmail_attackers a JOIN corporations n ON n.corporation_id = a.corporation_id",
		"a.corporation_id", "n.name", "a.killmail_time", "COUNT(*)::bigint",
		"a.final_blow = TRUE AND a.character_id >= 90000000 AND a.corporation_id >= 2000000", "corporation", false,
	},
	"alliances": {
		"killmail_attackers a JOIN alliances n ON n.alliance_id = a.alliance_id",
		"a.alliance_id", "n.name", "a.killmail_time", "COUNT(*)::bigint",
		"a.final_blow = TRUE AND a.character_id >= 90000000 AND a.alliance_id IS NOT NULL", "alliance", false,
	},
	"isk_destroyers_chars": {
		"killmail_attackers a JOIN killmails k ON k.killmail_id = a.killmail_id JOIN characters n ON n.character_id = a.character_id",
		"a.character_id", "n.name", "a.killmail_time",
		"COALESCE(SUM(k.total_value), 0)::double precision",
		"a.final_blow = TRUE AND a.character_id >= 90000000", "character", true,
	},
	"isk_destroyers_corps": {
		"killmail_attackers a JOIN killmails k ON k.killmail_id = a.killmail_id JOIN corporations n ON n.corporation_id = a.corporation_id",
		"a.corporation_id", "n.name", "a.killmail_time",
		"COALESCE(SUM(k.total_value), 0)::double precision",
		"a.final_blow = TRUE AND a.character_id >= 90000000 AND a.corporation_id >= 2000000", "corporation", true,
	},
	"isk_destroyers_alliances": {
		"killmail_attackers a JOIN killmails k ON k.killmail_id = a.killmail_id JOIN alliances n ON n.alliance_id = a.alliance_id",
		"a.alliance_id", "n.name", "a.killmail_time",
		"COALESCE(SUM(k.total_value), 0)::double precision",
		"a.final_blow = TRUE AND a.character_id >= 90000000 AND a.alliance_id IS NOT NULL", "alliance", true,
	},
	"biggest_losers": {
		"killmails k JOIN characters n ON n.character_id = k.victim_character_id",
		"k.victim_character_id", "n.name", "k.killmail_time",
		"COALESCE(SUM(k.total_value), 0)::double precision",
		"k.victim_character_id >= 90000000", "character", true,
	},
	"solo_killers": {
		"killmail_attackers a JOIN killmails k ON k.killmail_id = a.killmail_id JOIN characters n ON n.character_id = a.character_id",
		"a.character_id", "n.name", "a.killmail_time", "COUNT(*)::bigint",
		"a.final_blow = TRUE AND a.character_id >= 90000000 AND k.is_solo = TRUE",
		"character", false,
	},
	"top_points": {
		"killmail_attackers a JOIN killmails k ON k.killmail_id = a.killmail_id JOIN characters n ON n.character_id = a.character_id",
		"a.character_id", "n.name", "a.killmail_time",
		"COALESCE(SUM(a.points), 0)::double precision",
		"a.character_id >= 90000000", "character", false,
	},
	"systems": {
		"killmails k JOIN solar_systems n ON n.solar_system_id = k.solar_system_id",
		"k.solar_system_id", "n.system_name", "k.killmail_time", "COUNT(*)::bigint",
		"TRUE", "system", false,
	},
	"regions": {
		"killmails k JOIN regions n ON n.region_id = k.region_id",
		"k.region_id", "n.name", "k.killmail_time", "COUNT(*)::bigint",
		"k.region_id IS NOT NULL", "region", false,
	},
	"dangerous_systems": {
		"killmails k JOIN solar_systems n ON n.solar_system_id = k.solar_system_id",
		"k.solar_system_id", "n.system_name", "k.killmail_time",
		"COALESCE(SUM(k.total_value), 0)::double precision",
		"TRUE", "system", true,
	},
	"deadliest_regions": {
		"killmails k JOIN regions n ON n.region_id = k.region_id",
		"k.region_id", "n.name", "k.killmail_time",
		"COALESCE(SUM(k.total_value), 0)::double precision",
		"k.region_id IS NOT NULL", "region", true,
	},
	"ships": {
		"killmail_attackers a JOIN inv_types n ON n.type_id = a.ship_type_id",
		"a.ship_type_id", "n.name", "a.killmail_time", "COUNT(*)::bigint",
		"a.final_blow = TRUE AND a.character_id >= 90000000 AND a.ship_type_id IS NOT NULL", "ship", false,
	},
	"most_used_ships": {
		"killmail_attackers a JOIN inv_types n ON n.type_id = a.ship_type_id",
		"a.ship_type_id", "n.name", "a.killmail_time", "COUNT(*)::bigint",
		"a.character_id >= 90000000 AND a.ship_type_id IS NOT NULL", "ship", false,
	},
	"most_destroyed_ships": {
		"killmails k JOIN inv_types n ON n.type_id = k.victim_ship_type_id",
		"k.victim_ship_type_id", "n.name", "k.killmail_time", "COUNT(*)::bigint",
		"k.victim_ship_type_id IS NOT NULL", "ship", false,
	},
}

func validateRealtimeGlobalStatsWindow(hours float64) (int, error) {
	if hours > realtimeStatsMaximumHours {
		return 0, fmt.Errorf(
			"real-time stats window cannot exceed %dh (got %gh); "+
				"use daily aggregated stats for longer periods",
			realtimeStatsMaximumHours, hours,
		)
	}
	minutes := int(math.Floor(hours*60 + 0.5))
	if hours < float64(realtimeStatsMinimumMins)/60 {
		return 0, fmt.Errorf(
			"real-time stats window must be at least %d minutes (got %dm)",
			realtimeStatsMinimumMins, minutes,
		)
	}
	return minutes, nil
}

func loadRealtimeGlobalStats(
	ctx context.Context,
	db Database,
	query realtimeGlobalStatsQuery,
	hours float64,
	limit int,
) ([]map[string]any, error) {
	minutes, err := validateRealtimeGlobalStatsWindow(hours)
	if err != nil {
		return nil, err
	}
	rows, err := queryMaps(ctx, db, fmt.Sprintf(`
		SELECT %s AS id, %s AS name, %s AS metric
		FROM %s
		WHERE %s >= NOW() - ($1::double precision * INTERVAL '1 minute')
		  AND (%s)
		GROUP BY %s, %s
		ORDER BY metric DESC
		LIMIT $2`,
		query.id, query.name, query.aggregate, query.from,
		query.timestamp, query.predicate, query.id, query.name,
	), minutes, limit)
	if err != nil {
		return nil, err
	}
	return normalizeRealtimeGlobalStats(rows, query.resultType, query.isk), nil
}

func normalizeRealtimeGlobalStats(
	rows []map[string]any,
	resultType string,
	isk bool,
) []map[string]any {
	if rows == nil {
		return []map[string]any{}
	}
	for _, row := range rows {
		if row["name"] == nil || row["name"] == "" {
			row["name"] = "Unknown"
		}
		metric := row["metric"]
		delete(row, "metric")
		row["count"] = zeroIfNil(metric)
		if isk {
			row["isk"] = zeroIfNil(metric)
		}
		row["type"] = resultType
	}
	return rows
}
