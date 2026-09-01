package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog/log"
)

const (
	statsRankingsCacheTTL = time.Minute
	statsRankingsCache    = "public, max-age=60, s-maxage=60, stale-while-revalidate=60"
)

var statsRankingSections = map[string]struct{}{
	"largest":      {},
	"security":     {},
	"growth":       {},
	"newest":       {},
	"achievements": {},
	"eve-kill":     {},
}

// registerStatsRankingsRoute is deliberately called after the established
// response-schema snapshot. Rankings shares the stats domain and OpenAPI
// document, but it does not inherit the older API's injected $schema field.
func registerStatsRankingsRoute(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "stats-rankings",
		Method:      http.MethodGet,
		Path:        "/stats/rankings",
		Summary:     "Population and achievement rankings",
		Tags:        []string{"stats"},
	}, routeJSONCache(
		opts,
		statsRankingsCacheTTL,
		statsRankingsCache,
		statsRankingsHandler(opts),
	))
}

func statsRankingsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		section := req.Query.Get("section")
		if section == "" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"Missing section parameter",
			)
		}
		if _, ok := statsRankingSections[section]; !ok {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"Unknown section: "+section,
			)
		}

		entityType := req.Query.Get("entityType")
		if entityType == "" {
			entityType = "alliance"
		}
		limit, validLimit := statsRankingLimit(req.Query.Get("limit"))
		if !validLimit {
			return statsRankingPayload(nil), nil
		}

		var (
			entries []map[string]any
			err     error
		)
		switch section {
		case "largest":
			entries, err = queryLargestRankings(
				ctx, opts.DB, entityType, limit,
			)
		case "security":
			rank := req.Query.Get("rank")
			if rank == "" {
				rank = "pirate"
			}
			entries, err = querySecurityRankings(
				ctx, opts.DB, entityType, rank, limit,
			)
		case "growth":
			direction := req.Query.Get("direction")
			if direction == "" {
				direction = "growing"
			}
			days, validDays := statsRankingDays(req.Query.Get("days"))
			if !validDays {
				return statsRankingPayload(nil), nil
			}
			entries, err = queryGrowthRankings(
				ctx, opts.DB, entityType, direction, days, limit,
			)
		case "newest":
			entries, err = queryNewestRankings(
				ctx, opts.DB, entityType, limit,
			)
		case "achievements":
			entries, err = queryAchievementRankings(
				ctx, opts.DB, entityType, limit,
			)
		case "eve-kill":
			entries, err = queryEveKillRankings(
				ctx, opts.DB, entityType, req.Query.Get("window"), limit,
			)
		}
		if err != nil {
			// The frontend endpoint deliberately degrades individual boards to
			// an empty list. A broken snapshot must not take down the stats page.
			log.Warn().
				Err(err).
				Str("section", section).
				Str("entity_type", entityType).
				Msg("stats rankings query failed")
			return statsRankingPayload(nil), nil
		}
		return statsRankingPayload(entries), nil
	}
}

func queryEveKillRankings(ctx context.Context, db Database, entityType, window string, limit int) ([]map[string]any, error) {
	typeID := map[string]int16{
		"character": 0, "corporation": 1, "alliance": 2,
		"ship": 3, "system": 4, "region": 6,
	}[entityType]
	if _, ok := map[string]bool{"character": true, "corporation": true, "alliance": true, "ship": true, "system": true, "region": true}[entityType]; !ok {
		return []map[string]any{}, nil
	}
	windowID := map[string]int16{"weekly": 0, "ninety_days": 1, "all_time": 2}[window]
	if window == "" {
		windowID = 2
	} else if _, ok := map[string]bool{"weekly": true, "ninety_days": true, "all_time": true}[window]; !ok {
		return []map[string]any{}, nil
	}
	nameSQL := map[string]string{
		"character":   "LEFT JOIN characters e ON e.character_id = r.entity_id",
		"corporation": "LEFT JOIN corporations e ON e.corporation_id = r.entity_id",
		"alliance":    "LEFT JOIN alliances e ON e.alliance_id = r.entity_id",
		"ship":        "LEFT JOIN inv_types e ON e.type_id = r.entity_id",
		"system":      "LEFT JOIN solar_systems e ON e.solar_system_id = r.entity_id",
		"region":      "LEFT JOIN regions e ON e.region_id = r.entity_id",
	}[entityType]
	nameColumn := "e.name"
	if entityType == "system" {
		nameColumn = "e.system_name"
	}
	return queryMaps(ctx, db, fmt.Sprintf(`
		SELECT r.entity_id, coalesce(nullif(%s, ''), 'Unknown') AS name,
		       r.combat_points, r.achievement_points, r.eve_kill_rating,
		       r.combat_rank, r.achievement_rank, r.overall_rank,
		       r.population, r.updated_at
		FROM entity_rankings r
		%s
		WHERE r.entity_type = $1 AND r.ranking_window = $2
		ORDER BY r.overall_rank
		LIMIT $3`, nameColumn, nameSQL), typeID, windowID, limit)
}

func statsRankingPayload(entries []map[string]any) legacyPayload {
	if entries == nil {
		entries = []map[string]any{}
	}
	return jsonPayload(map[string]any{"entries": entries})
}

// statsRankingLimit reproduces Number(value) || 10 followed by Math.min(...,
// 50). Values PostgreSQL cannot accept as a LIMIT (negative or fractional
// numbers) are rejected before the query and retain the old observable empty
// result without manufacturing a database error.
func statsRankingLimit(raw string) (int, bool) {
	value := statsRankingNumberOr(raw, 10)
	if value > 50 {
		value = 50
	}
	if math.IsInf(value, -1) || value < 0 || math.Trunc(value) != value {
		return 0, false
	}
	return int(value), true
}

// statsRankingDays reproduces Number(value) || 7. Growth needs days+1 as a
// PostgreSQL LIMIT, so reject values that cannot be represented by that bound.
func statsRankingDays(raw string) (int64, bool) {
	value := statsRankingNumberOr(raw, 7)
	if math.IsInf(value, 0) || value < 1 || math.Trunc(value) != value ||
		value >= float64(math.MaxInt64) {
		return 0, false
	}
	return int64(value), true
}

func statsRankingNumberOr(raw string, fallback float64) float64 {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback
	}
	switch value {
	case "Infinity", "+Infinity", "Inf", "+Inf":
		return math.Inf(1)
	case "-Infinity", "-Inf":
		return math.Inf(-1)
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || number == 0 || math.IsNaN(number) {
		return fallback
	}
	return number
}

// rankingNameJoin contains the only entity-table identifiers used by snapshot
// rankings. entityType is always a query parameter; it is never interpolated.
func rankingNameJoin(entityType string) (string, string) {
	switch entityType {
	case "alliance":
		return `
			COALESCE(NULLIF(entity.name, ''), 'Unknown') AS name`,
			`LEFT JOIN alliances entity
			  ON entity.alliance_id = ranked.entity_id`
	case "corporation":
		return `
			COALESCE(NULLIF(entity.name, ''), 'Unknown') AS name`,
			`LEFT JOIN corporations entity
			  ON entity.corporation_id = ranked.entity_id`
	case "character":
		return `
			COALESCE(NULLIF(entity.name, ''), 'Unknown') AS name`,
			`LEFT JOIN characters entity
			  ON entity.character_id = ranked.entity_id`
	default:
		// resolveNames returned no names for unknown entity types in the
		// TypeScript endpoint, but snapshots could still produce rows.
		return `'Unknown'::text AS name`, ""
	}
}

func queryLargestRankings(
	ctx context.Context,
	db Database,
	entityType string,
	limit int,
) ([]map[string]any, error) {
	nameSelect, nameJoin := rankingNameJoin(entityType)
	corporationFilter := ""
	if entityType == "corporation" {
		corporationFilter = "AND snapshot.entity_id > 1999999"
	}
	rows, err := queryMaps(ctx, db, fmt.Sprintf(`
		WITH latest_date AS (
			SELECT MAX(date) AS date
			FROM entity_snapshots
			WHERE entity_type = $1
		),
		ranked AS (
			SELECT snapshot.entity_id, snapshot.member_count,
			       latest_date.date AS snapshot_date
			FROM entity_snapshots snapshot
			CROSS JOIN latest_date
			WHERE snapshot.entity_type = $1
			  AND snapshot.date = latest_date.date
			  AND snapshot.member_count IS NOT NULL
			  AND snapshot.member_count > 0
			  %s
			ORDER BY snapshot.member_count DESC
			LIMIT $2
		)
		SELECT ranked.entity_id, %s,
		       ranked.member_count,
		       ranked.member_count -
		         COALESCE(day_1.member_count, ranked.member_count) AS delta_1d,
		       ranked.member_count -
		         COALESCE(day_7.member_count, ranked.member_count) AS delta_7d,
		       ranked.member_count -
		         COALESCE(day_30.member_count, ranked.member_count) AS delta_30d
		FROM ranked
		%s
		LEFT JOIN LATERAL (
			SELECT history.member_count
			FROM entity_snapshots history
			WHERE history.entity_type = $1
			  AND history.entity_id = ranked.entity_id
			  AND history.date BETWEEN ranked.snapshot_date - 4
			                       AND ranked.snapshot_date - 1
			ORDER BY ABS(
			  history.date - (ranked.snapshot_date - 1)
			)
			LIMIT 1
		) day_1 ON TRUE
		LEFT JOIN LATERAL (
			SELECT history.member_count
			FROM entity_snapshots history
			WHERE history.entity_type = $1
			  AND history.entity_id = ranked.entity_id
			  AND history.date BETWEEN ranked.snapshot_date - 10
			                       AND ranked.snapshot_date - 4
			ORDER BY ABS(
			  history.date - (ranked.snapshot_date - 7)
			)
			LIMIT 1
		) day_7 ON TRUE
		LEFT JOIN LATERAL (
			SELECT history.member_count
			FROM entity_snapshots history
			WHERE history.entity_type = $1
			  AND history.entity_id = ranked.entity_id
			  AND history.date BETWEEN ranked.snapshot_date - 33
			                       AND ranked.snapshot_date - 27
			ORDER BY ABS(
			  history.date - (ranked.snapshot_date - 30)
			)
			LIMIT 1
		) day_30 ON TRUE
		ORDER BY ranked.member_count DESC`,
		corporationFilter, nameSelect, nameJoin,
	), entityType, limit)
	if err != nil {
		return nil, err
	}
	return buildLargestRankingEntries(rows, entityType), nil
}

func buildLargestRankingEntries(
	rows []map[string]any,
	entityType string,
) []map[string]any {
	entries := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, map[string]any{
			"id":           rankingInt(row["entity_id"]),
			"name":         rankingName(row["name"]),
			"member_count": rankingInt(row["member_count"]),
			"delta_1d":     rankingInt(row["delta_1d"]),
			"delta_7d":     rankingInt(row["delta_7d"]),
			"delta_30d":    rankingInt(row["delta_30d"]),
			"type":         entityType,
		})
	}
	return entries
}

func querySecurityRankings(
	ctx context.Context,
	db Database,
	entityType, rank string,
	limit int,
) ([]map[string]any, error) {
	nameSelect, nameJoin := rankingNameJoin(entityType)
	securityFilter := "snapshot.avg_sec_status > 0"
	if rank == "pirate" {
		securityFilter = "snapshot.avg_sec_status < 0"
	}
	rows, err := queryMaps(ctx, db, fmt.Sprintf(`
		WITH latest_date AS (
			SELECT MAX(date) AS date
			FROM entity_snapshots
			WHERE entity_type = $1
		),
		ranked AS (
			SELECT snapshot.entity_id, snapshot.member_count,
			       snapshot.avg_sec_status,
			       ABS(snapshot.avg_sec_status) *
			         LN(GREATEST(snapshot.member_count, 2)) AS weighted_score
			FROM entity_snapshots snapshot
			CROSS JOIN latest_date
			WHERE snapshot.entity_type = $1
			  AND snapshot.date = latest_date.date
			  AND snapshot.member_count >= 50
			  AND snapshot.avg_sec_status IS NOT NULL
			  AND %s
			ORDER BY weighted_score DESC
			LIMIT $2
		)
		SELECT ranked.entity_id, %s,
		       ranked.member_count, ranked.avg_sec_status,
		       ranked.weighted_score
		FROM ranked
		%s
		ORDER BY ranked.weighted_score DESC`,
		securityFilter, nameSelect, nameJoin,
	), entityType, limit)
	if err != nil {
		return nil, err
	}
	return buildSecurityRankingEntries(rows, entityType), nil
}

func buildSecurityRankingEntries(
	rows []map[string]any,
	entityType string,
) []map[string]any {
	entries := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, map[string]any{
			"id":             rankingInt(row["entity_id"]),
			"name":           rankingName(row["name"]),
			"member_count":   rankingInt(row["member_count"]),
			"avg_sec_status": rankingRound(row["avg_sec_status"], 2),
			"weighted_score": rankingRound(row["weighted_score"], 2),
			"type":           entityType,
		})
	}
	return entries
}

func queryGrowthRankings(
	ctx context.Context,
	db Database,
	entityType, direction string,
	days int64,
	limit int,
) ([]map[string]any, error) {
	nameSelect, nameJoin := rankingNameJoin(entityType)
	corporationFilter := ""
	if entityType == "corporation" {
		corporationFilter = "AND snapshot.entity_id > 1999999"
	}
	comparison := "< 0"
	order := "ASC"
	if direction == "growing" {
		comparison = "> 0"
		order = "DESC"
	}
	rows, err := queryMaps(ctx, db, fmt.Sprintf(`
		WITH selected_dates AS (
			SELECT DISTINCT date
			FROM entity_snapshots
			WHERE entity_type = $1
			ORDER BY date DESC
			LIMIT $2
		),
		date_bounds AS (
			SELECT MAX(date) AS latest_date,
			       MIN(date) AS previous_date,
			       COUNT(*) AS date_count
			FROM selected_dates
		),
		latest AS (
			SELECT snapshot.entity_id, snapshot.member_count
			FROM entity_snapshots snapshot
			CROSS JOIN date_bounds
			WHERE date_bounds.date_count >= 2
			  AND snapshot.entity_type = $1
			  AND snapshot.date = date_bounds.latest_date
			  AND snapshot.member_count IS NOT NULL
			  AND snapshot.member_count > 0
			  %s
		),
		previous AS (
			SELECT snapshot.entity_id, snapshot.member_count
			FROM entity_snapshots snapshot
			CROSS JOIN date_bounds
			WHERE date_bounds.date_count >= 2
			  AND snapshot.entity_type = $1
			  AND snapshot.date = date_bounds.previous_date
			  AND snapshot.member_count IS NOT NULL
			  AND snapshot.member_count > 0
		),
		ranked AS (
			SELECT latest.entity_id, latest.member_count,
			       previous.member_count AS prev_count,
			       latest.member_count - previous.member_count AS delta
			FROM latest
			INNER JOIN previous
			  ON previous.entity_id = latest.entity_id
			WHERE latest.member_count - previous.member_count %s
			ORDER BY delta %s
			LIMIT $3
		)
		SELECT ranked.entity_id, %s,
		       ranked.member_count, ranked.prev_count, ranked.delta
		FROM ranked
		%s
		ORDER BY ranked.delta %s`,
		corporationFilter, comparison, order,
		nameSelect, nameJoin, order,
	), entityType, days+1, limit)
	if err != nil {
		return nil, err
	}
	return buildGrowthRankingEntries(rows, entityType), nil
}

func buildGrowthRankingEntries(
	rows []map[string]any,
	entityType string,
) []map[string]any {
	entries := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, map[string]any{
			"id":           rankingInt(row["entity_id"]),
			"name":         rankingName(row["name"]),
			"member_count": rankingInt(row["member_count"]),
			"prev_count":   rankingInt(row["prev_count"]),
			"delta":        rankingInt(row["delta"]),
			"type":         entityType,
		})
	}
	return entries
}

type newestRankingSpec struct {
	query      string
	resultType string
}

func newestRankingQuery(entityType string) newestRankingSpec {
	switch entityType {
	case "alliance":
		return newestRankingSpec{
			query: `
				SELECT alliance_id AS entity_id, name, member_count,
				       date_founded
				FROM alliances
				WHERE date_founded IS NOT NULL
				ORDER BY date_founded DESC
				LIMIT $1`,
			resultType: "alliance",
		}
	case "character":
		return newestRankingSpec{
			query: `
				SELECT character_id AS entity_id, name,
				       0::integer AS member_count,
				       birthday AS date_founded
				FROM characters
				WHERE birthday IS NOT NULL
				ORDER BY birthday DESC
				LIMIT $1`,
			resultType: "character",
		}
	default:
		// This intentional fallback matches the TypeScript endpoint.
		return newestRankingSpec{
			query: `
				SELECT corporation_id AS entity_id, name, member_count,
				       date_founded
				FROM corporations
				WHERE date_founded IS NOT NULL
				ORDER BY date_founded DESC
				LIMIT $1`,
			resultType: "corporation",
		}
	}
}

func queryNewestRankings(
	ctx context.Context,
	db Database,
	entityType string,
	limit int,
) ([]map[string]any, error) {
	spec := newestRankingQuery(entityType)
	rows, err := queryMaps(ctx, db, spec.query, limit)
	if err != nil {
		return nil, err
	}
	return buildNewestRankingEntries(rows, spec.resultType), nil
}

func buildNewestRankingEntries(
	rows []map[string]any,
	resultType string,
) []map[string]any {
	entries := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, map[string]any{
			"id":           rankingInt(row["entity_id"]),
			"name":         rankingName(row["name"]),
			"member_count": rankingInt(row["member_count"]),
			"date_founded": row["date_founded"],
			"type":         resultType,
		})
	}
	return entries
}

func queryAchievementRankings(
	ctx context.Context,
	db Database,
	entityType string,
	limit int,
) ([]map[string]any, error) {
	if entityType == "character" {
		rows, err := queryMaps(ctx, db, `
			SELECT character_id AS entity_id, name,
			       achievement_points::int AS total_points
			FROM characters
			WHERE achievement_points > 0
			ORDER BY achievement_points DESC
			LIMIT $1`, limit)
		if err != nil {
			return nil, err
		}
		return buildCharacterAchievementRankingEntries(rows), nil
	}

	table := "corporations"
	idColumn := "corporation_id"
	if entityType == "alliance" {
		table = "alliances"
		idColumn = "alliance_id"
	}
	corporationFilter := ""
	if entityType == "corporation" {
		corporationFilter = "AND snapshot.entity_id > 1999999"
	}
	// table and idColumn come exclusively from the two choices above.
	rows, err := queryMaps(ctx, db, fmt.Sprintf(`
		WITH latest_date AS (
			SELECT MAX(date) AS date
			FROM entity_snapshots
			WHERE entity_type = $1
		)
		SELECT snapshot.entity_id, entity.name,
		       snapshot.member_count,
		       snapshot.total_achievement_points,
		       snapshot.avg_achievement_points
		FROM entity_snapshots snapshot
		CROSS JOIN latest_date
		INNER JOIN %s entity
		  ON entity.%s = snapshot.entity_id
		WHERE snapshot.entity_type = $1
		  AND snapshot.date = latest_date.date
		  AND snapshot.total_achievement_points IS NOT NULL
		  AND snapshot.total_achievement_points > 0
		  %s
		ORDER BY snapshot.avg_achievement_points DESC
		LIMIT $2`,
		table, idColumn, corporationFilter,
	), entityType, limit)
	if err != nil {
		return nil, err
	}
	return buildEntityAchievementRankingEntries(rows, entityType), nil
}

func buildCharacterAchievementRankingEntries(
	rows []map[string]any,
) []map[string]any {
	entries := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, map[string]any{
			"id":           rankingInt(row["entity_id"]),
			"name":         rankingName(row["name"]),
			"member_count": int64(0),
			"total_points": rankingInt(row["total_points"]),
			"avg_points":   float64(0),
			"type":         "character",
		})
	}
	return entries
}

func buildEntityAchievementRankingEntries(
	rows []map[string]any,
	entityType string,
) []map[string]any {
	entries := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, map[string]any{
			"id":           rankingInt(row["entity_id"]),
			"name":         rankingName(row["name"]),
			"member_count": rankingInt(row["member_count"]),
			"total_points": rankingInt(row["total_achievement_points"]),
			"avg_points":   rankingRound(row["avg_achievement_points"], 1),
			"type":         entityType,
		})
	}
	return entries
}

func rankingInt(value any) int64 {
	number, _ := int64Value(value)
	return number
}

func rankingName(value any) string {
	name, ok := stringValue(value)
	if !ok || name == "" {
		return "Unknown"
	}
	return name
}

func rankingRound(value any, places int) float64 {
	number, _ := float64Value(value)
	scale := math.Pow10(places)
	return math.Round(number*scale) / scale
}
