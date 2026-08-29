package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

var validGlobalStatsTypes = map[string]struct{}{
	"characters": {}, "corporations": {}, "alliances": {}, "factions": {},
	"ships": {}, "systems": {}, "regions": {},
	"isk_destroyers_chars": {}, "isk_destroyers_corps": {},
	"isk_destroyers_alliances": {}, "solo_killers": {},
	"top_points": {}, "dangerous_systems": {}, "deadliest_regions": {},
	"most_used_ships": {}, "most_destroyed_ships": {},
	"biggest_losers": {}, "pirate_characters": {},
	"carebear_characters": {},
}

func registerGlobalStatsRoute(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "global-stats",
		Method:      http.MethodGet,
		Path:        "/stats",
		Summary:     "Global statistics",
		Tags:        []string{"stats"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		dataType := req.Query.Get("dataType")
		if dataType == "" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Missing dataType parameter",
			)
		}
		limit := globalStatsLimit(req.Query.Get("limit"))
		daysValue := globalStatsDays(req.Query.Get("days"))
		if query, ok := realtimeGlobalStatsQueries[dataType]; daysValue < 1 && ok {
			entries, err := loadRealtimeGlobalStats(
				ctx, opts.DB, query, daysValue*24, limit,
			)
			if err != nil {
				return legacyPayload{}, err
			}
			return jsonPayload(map[string]any{"entries": entries}), nil
		}
		days := int(daysValue)
		since := time.Now().AddDate(0, 0, -days)

		var (
			entries []map[string]any
			err     error
		)
		if _, ok := validGlobalStatsTypes[dataType]; ok {
			entries, err = loadGlobalTopList(
				ctx, opts.DB, dataType,
				since.UTC().Format("2006-01-02"), limit,
			)
			if err == nil {
				entries, err = attachGlobalStatsPalettes(ctx, opts.DB, entries)
			}
		} else if strings.HasPrefix(dataType, "most_valuable_") {
			entries, err = loadMostValuable(
				ctx, opts.DB, dataType, since, limit,
			)
		} else {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Unknown dataType: "+dataType,
			)
		}
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"entries": entries}), nil
	})
}

func globalStatsLimit(raw string) int {
	return int(math.Min(numberOr(raw, 10), 100))
}

func globalStatsDays(raw string) float64 {
	return math.Min(numberOr(raw, 7), 90)
}

type globalStatsQuery struct {
	entityType int
	table      string
	idColumn   string
	nameColumn string
	metric     string
	resultType string
}

func loadGlobalTopList(
	ctx context.Context,
	db Database,
	dataType, since string,
	limit int,
) ([]map[string]any, error) {
	if dataType == "pirate_characters" || dataType == "carebear_characters" {
		return loadSecurityStatusList(ctx, db, dataType, limit)
	}
	configs := map[string]globalStatsQuery{
		"characters":               {0, "characters", "character_id", "name", "kills", "character"},
		"corporations":             {1, "corporations", "corporation_id", "name", "kills", "corporation"},
		"alliances":                {2, "alliances", "alliance_id", "name", "kills", "alliance"},
		"factions":                 {7, "factions", "faction_id", "name", "kills", "faction"},
		"ships":                    {3, "inv_types", "type_id", "name", "kills", "ship"},
		"most_used_ships":          {3, "inv_types", "type_id", "name", "kills", "ship"},
		"systems":                  {4, "solar_systems", "solar_system_id", "system_name", "kills", "system"},
		"regions":                  {6, "regions", "region_id", "name", "kills", "region"},
		"isk_destroyers_chars":     {0, "characters", "character_id", "name", "isk_destroyed", "character"},
		"isk_destroyers_corps":     {1, "corporations", "corporation_id", "name", "isk_destroyed", "corporation"},
		"isk_destroyers_alliances": {2, "alliances", "alliance_id", "name", "isk_destroyed", "alliance"},
		"solo_killers":             {0, "characters", "character_id", "name", "solo_kills", "character"},
		"top_points":               {0, "characters", "character_id", "name", "points", "character"},
		"most_destroyed_ships":     {3, "inv_types", "type_id", "name", "losses", "ship"},
		"biggest_losers":           {0, "characters", "character_id", "name", "isk_lost", "character"},
		"dangerous_systems":        {4, "solar_systems", "solar_system_id", "system_name", "isk_destroyed", "system"},
		"deadliest_regions":        {6, "regions", "region_id", "name", "isk_destroyed", "region"},
	}
	config := configs[dataType]
	entityFilter := globalStatsEntityFilter(dataType)
	countMetric := config.metric
	includeISK := dataType == "dangerous_systems" || dataType == "deadliest_regions"
	if includeISK {
		countMetric = "kills"
	}
	extraAggregate := ""
	extraSelect := ""
	if includeISK {
		extraAggregate = ", COALESCE(SUM(s.isk_destroyed), 0)::double precision AS isk"
		extraSelect = ", ranked.isk"
	}
	rows, err := queryMaps(ctx, db, fmt.Sprintf(`
		WITH ranked AS MATERIALIZED (
			SELECT s.entity_id,
			       COALESCE(SUM(s.%s), 0)::double precision AS count%s
			FROM stats s
			WHERE s.entity_type = $1 AND s.period_type = 0
			  AND s.period_start >= $2::date%s
			GROUP BY s.entity_id
			ORDER BY SUM(s.%s) DESC
			LIMIT $3
		)
		SELECT ranked.entity_id AS id, n.%s AS name, ranked.count%s
		FROM ranked
		INNER JOIN %s n ON ranked.entity_id = n.%s
		ORDER BY ranked.count DESC`,
		countMetric,
		extraAggregate,
		entityFilter,
		config.metric,
		config.nameColumn,
		extraSelect,
		config.table,
		config.idColumn,
	), config.entityType, since, limit)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row["name"] == nil || row["name"] == "" {
			row["name"] = "Unknown"
		}
		row["type"] = config.resultType
	}
	return rows, nil
}

func globalStatsEntityFilter(dataType string) string {
	if dataType == "factions" {
		return ` AND EXISTS (
			SELECT 1 FROM factions
			WHERE factions.faction_id = s.entity_id
			  AND factions.militia_corporation_id IS NOT NULL
		)`
	}
	return ""
}

func loadSecurityStatusList(
	ctx context.Context,
	db Database,
	dataType string,
	limit int,
) ([]map[string]any, error) {
	operator, order := "< -5", "ASC"
	if dataType == "carebear_characters" {
		operator, order = "> 5", "DESC"
	}
	rows, err := queryMaps(ctx, db, `
		SELECT character_id AS id, name, security_status AS sec
		FROM characters
		WHERE security_status IS NOT NULL AND security_status `+operator+`
		  AND (corporation_id IS NULL OR (
		    corporation_id NOT BETWEEN 1000000 AND 1999999
		    AND corporation_id NOT IN (
		      109299958, 661107786, 924269309, 1069536620, 1838146481
		    )
		  ))
		ORDER BY security_status `+order+`
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row["name"] == nil || row["name"] == "" {
			row["name"] = "Unknown"
		}
		sec, _ := float64Value(row["sec"])
		row["sec"] = math.Round(sec*100) / 100
		row["count"] = 0
		row["type"] = "character"
	}
	return rows, nil
}

func loadMostValuable(
	ctx context.Context,
	db Database,
	dataType string,
	since time.Time,
	limit int,
) ([]map[string]any, error) {
	categoryFilter := ""
	categoryID := 0
	switch dataType {
	case "most_valuable_ships":
		categoryID = 6
	case "most_valuable_structures":
		categoryID = 65
	}
	args := []any{since}
	if categoryID != 0 {
		args = append(args, categoryID)
		categoryFilter = fmt.Sprintf(`
		  AND k.victim_ship_group_id IN (
		    SELECT group_id FROM inv_groups WHERE category_id = $%d
		  )`, len(args))
	}
	args = append(args, limit)
	rows, err := queryMaps(ctx, db, `
		SELECT k.killmail_id, k.killmail_hash, k.killmail_time,
		       k.victim_ship_type_id AS ship_type_id,
		       t.name AS ship_name, k.total_value,
		       k.victim_character_id, c.name AS victim_character_name,
		       co.name AS victim_corporation_name,
		       a.name AS victim_alliance_name
		FROM killmails k
		INNER JOIN inv_types t ON k.victim_ship_type_id = t.type_id
		LEFT JOIN characters c ON k.victim_character_id = c.character_id
		LEFT JOIN corporations co ON k.victim_corporation_id = co.corporation_id
		LEFT JOIN alliances a ON k.victim_alliance_id = a.alliance_id
		WHERE k.killmail_time >= $1`+categoryFilter+`
		ORDER BY k.total_value DESC
		LIMIT $`+fmt.Sprintf("%d", len(args)),
		args...,
	)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row["ship_name"] == nil || row["ship_name"] == "" {
			row["ship_name"] = "Unknown"
		}
		row["total_value"] = zeroIfNil(row["total_value"])
	}
	return rows, nil
}
