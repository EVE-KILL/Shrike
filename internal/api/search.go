package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

var searchTypeRanks = map[string]int{
	"ship": 1, "shipgroup": 1, "item": 1, "group": 1,
	"region": 2, "constellation": 2,
	"system": 3, "faction": 4, "alliance": 5,
	"corporation": 6, "character": 7,
}

func registerSearchRoute(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "search",
		Method:      http.MethodGet,
		Path:        "/search",
		Summary:     "Search entities and universe data",
		Tags:        []string{"search"},
	}, searchHandler(opts))
}

func searchHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		query := strings.TrimSpace(req.Query.Get("q"))
		if query == "" {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Missing ?q= parameter")
		}
		typeSet := map[string]bool{}
		for value := range strings.SplitSeq(req.Query.Get("type"), ",") {
			if value = strings.TrimSpace(value); value != "" {
				typeSet[value] = true
			}
		}
		wants := func(entityType string) bool {
			return len(typeSet) == 0 || typeSet[entityType]
		}
		limit := boundedQueryInt(req, "limit", 25, math.MinInt, 50)
		start := time.Now()
		useTrigram := len(query) >= 3
		prefix := query + "%"

		parts := []string{}
		if wants("character") {
			parts = append(parts, `
				SELECT character_id AS entity_id, name, NULL::text AS ticker,
				       'character' AS type, corporation_id, alliance_id,
				       0 AS weight,
				       CASE WHEN $3 THEN similarity(name, $1) ELSE 0.5 END AS score
				FROM characters
				WHERE deleted IS NOT TRUE AND name ILIKE $2`)
		}
		if wants("corporation") {
			parts = append(parts, `
				SELECT corporation_id AS entity_id, name, ticker,
				       'corporation' AS type, NULL::integer AS corporation_id,
				       alliance_id, COALESCE(member_count, 0) AS weight,
				       CASE WHEN $3
				            THEN GREATEST(similarity(name, $1), similarity(ticker, $1))
				            ELSE 0.5 END AS score
				FROM corporations
				WHERE deleted IS NOT TRUE
				  AND (($3 AND (name % $1 OR ticker % $1))
				       OR name ILIKE $2 OR ticker ILIKE $2)`)
		}
		if wants("alliance") {
			parts = append(parts, `
				SELECT alliance_id AS entity_id, name, ticker,
				       'alliance' AS type, NULL::integer AS corporation_id,
				       NULL::integer AS alliance_id,
				       COALESCE(member_count, 0) AS weight,
				       CASE WHEN $3
				            THEN GREATEST(similarity(name, $1), similarity(ticker, $1))
				            ELSE 0.5 END AS score
				FROM alliances
				WHERE deleted IS NOT TRUE
				  AND (($3 AND (name % $1 OR ticker % $1))
				       OR name ILIKE $2 OR ticker ILIKE $2)`)
		}
		if wants("item") || wants("ship") {
			extra := ""
			if typeSet["ship"] && !typeSet["item"] {
				extra = " AND g.category_id = 6"
			} else if typeSet["item"] && !typeSet["ship"] {
				extra = " AND g.category_id != 6"
			}
			parts = append(parts, `
				SELECT t.type_id AS entity_id, t.name,
				       NULL::text AS ticker,
				       CASE WHEN g.category_id = 6 THEN 'ship' ELSE 'item' END AS type,
				       NULL::integer AS corporation_id,
				       NULL::integer AS alliance_id,
				       CASE WHEN g.category_id = 6 THEN 10 ELSE 0 END AS weight,
				       CASE WHEN $3 THEN similarity(t.name, $1) ELSE 0.5 END AS score
				FROM inv_types t
				JOIN inv_groups g ON g.group_id = t.group_id
				WHERE t.published IS TRUE
				  AND (($3 AND t.name % $1) OR t.name ILIKE $2)`+extra)
		}
		// Module groups are opt-in. They are useful in the advanced filter
		// picker, but noisy in a general search.
		if typeSet["group"] {
			parts = append(parts, `
				SELECT g.group_id AS entity_id, g.name, NULL::text AS ticker,
				       'group' AS type, NULL::integer AS corporation_id,
				       NULL::integer AS alliance_id, 0 AS weight,
				       CASE WHEN $3 THEN similarity(g.name, $1) ELSE 0.5 END AS score
				FROM inv_groups g
				WHERE g.published IS TRUE AND g.category_id = 7
				  AND g.name IS NOT NULL
				  AND (($3 AND g.name % $1) OR g.name ILIKE $2)`)
		}
		if wants("shipgroup") {
			parts = append(parts, `
				SELECT g.group_id AS entity_id, g.name, NULL::text AS ticker,
				       'shipgroup' AS type, NULL::integer AS corporation_id,
				       NULL::integer AS alliance_id, 0 AS weight,
				       CASE WHEN $3 THEN similarity(g.name, $1) ELSE 0.5 END AS score
				FROM inv_groups g
				WHERE g.published IS TRUE AND g.category_id = 6
				  AND g.name IS NOT NULL
				  AND (($3 AND g.name % $1) OR g.name ILIKE $2)`)
		}
		if wants("system") {
			parts = append(parts, searchUniversePart(
				"solar_system_id", "system_name", "solar_systems", "system",
			))
		}
		if wants("region") {
			parts = append(parts, searchUniversePart(
				"region_id", "name", "regions", "region",
			))
		}
		if wants("constellation") {
			parts = append(parts, searchUniversePart(
				"constellation_id", "constellation_name", "constellations", "constellation",
			))
		}
		if wants("faction") {
			parts = append(parts, searchUniversePart(
				"faction_id", "name", "factions", "faction",
			))
		}
		if len(parts) == 0 {
			return jsonPayload(map[string]any{
				"hits":             []any{},
				"query":            query,
				"processingTimeMs": 0,
				"total":            0,
				"entityCounts":     map[string]any{},
			}), nil
		}

		rows, err := queryMaps(ctx, opts.DB,
			"SELECT * FROM ("+strings.Join(parts, " UNION ALL ")+
				") sub ORDER BY score DESC LIMIT $4",
			query, prefix, useTrigram, limit,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		sort.SliceStable(rows, func(i, j int) bool {
			leftScore, _ := float64Value(rows[i]["score"])
			rightScore, _ := float64Value(rows[j]["score"])
			if math.Abs(leftScore-rightScore) < 0.1 {
				leftType, _ := stringValue(rows[i]["type"])
				rightType, _ := stringValue(rows[j]["type"])
				leftRank, rightRank := searchTypeRanks[leftType], searchTypeRanks[rightType]
				if leftRank == 0 {
					leftRank = 99
				}
				if rightRank == 0 {
					rightRank = 99
				}
				if leftRank != rightRank {
					return leftRank < rightRank
				}
				leftWeight, _ := float64Value(rows[i]["weight"])
				rightWeight, _ := float64Value(rows[j]["weight"])
				return leftWeight > rightWeight
			}
			return leftScore > rightScore
		})

		corpIDs, allianceIDs := searchRelatedIDs(rows)
		corps, err := loadSearchNames(ctx, opts.DB, "corporations", "corporation_id", corpIDs)
		if err != nil {
			return legacyPayload{}, err
		}
		alliances, err := loadSearchNames(ctx, opts.DB, "alliances", "alliance_id", allianceIDs)
		if err != nil {
			return legacyPayload{}, err
		}

		hits := make([]map[string]any, 0, len(rows))
		counts := map[string]int{}
		for _, row := range rows {
			entityType, _ := stringValue(row["type"])
			entityID, _ := int64Value(row["entity_id"])
			ticker := row["ticker"]
			if ticker == "" {
				ticker = nil
			}
			hit := map[string]any{
				"id":     fmt.Sprintf("%s_%d", entityType, entityID),
				"name":   row["name"],
				"ticker": ticker,
				"type":   entityType,
			}
			if entityType == "character" {
				hit["corporation_id"] = truthyNumberOrNil(row["corporation_id"])
				hit["alliance_id"] = truthyNumberOrNil(row["alliance_id"])
				if corpID, ok := int64Value(row["corporation_id"]); ok {
					if corp := corps[corpID]; corp != nil {
						hit["corporation_name"] = corp["name"]
						hit["corporation_ticker"] = corp["ticker"]
					}
				}
				if allianceID, ok := int64Value(row["alliance_id"]); ok {
					if alliance := alliances[allianceID]; alliance != nil {
						hit["alliance_name"] = alliance["name"]
						hit["alliance_ticker"] = alliance["ticker"]
					}
				}
			} else if entityType == "corporation" {
				hit["alliance_id"] = truthyNumberOrNil(row["alliance_id"])
				if allianceID, ok := int64Value(row["alliance_id"]); ok {
					if alliance := alliances[allianceID]; alliance != nil {
						hit["alliance_name"] = alliance["name"]
						hit["alliance_ticker"] = alliance["ticker"]
					}
				}
			}
			hits = append(hits, hit)
			counts[entityType]++
		}
		return jsonPayload(map[string]any{
			"hits":             hits,
			"query":            query,
			"processingTimeMs": time.Since(start).Milliseconds(),
			"total":            len(hits),
			"entityCounts":     counts,
		}), nil
	}
}

func searchUniversePart(idColumn, nameColumn, table, entityType string) string {
	return fmt.Sprintf(`
		SELECT %s AS entity_id, %s AS name, NULL::text AS ticker,
		       '%s' AS type, NULL::integer AS corporation_id,
		       NULL::integer AS alliance_id, 0 AS weight,
		       CASE WHEN $3 THEN similarity(%s, $1) ELSE 0.5 END AS score
		FROM %s
		WHERE (($3 AND %s %% $1) OR %s ILIKE $2)`,
		idColumn, nameColumn, entityType, nameColumn, table, nameColumn, nameColumn,
	)
}

func searchRelatedIDs(rows []map[string]any) ([]int32, []int32) {
	corpValues, allianceValues := []any{}, []any{}
	for _, row := range rows {
		entityType, _ := stringValue(row["type"])
		if entityType == "character" {
			corpValues = append(corpValues, row["corporation_id"])
			allianceValues = append(allianceValues, row["alliance_id"])
		} else if entityType == "corporation" {
			allianceValues = append(allianceValues, row["alliance_id"])
		}
	}
	return int32Slice(corpValues...), int32Slice(allianceValues...)
}

func loadSearchNames(
	ctx context.Context,
	db Database,
	table, idColumn string,
	ids []int32,
) (map[int64]map[string]any, error) {
	result := map[int64]map[string]any{}
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := queryMaps(ctx, db,
		`SELECT `+idColumn+` AS id, name, ticker FROM `+table+
			` WHERE `+idColumn+` = ANY($1::int[])`, ids)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		id, _ := int64Value(row["id"])
		result[id] = row
	}
	return result, nil
}

func truthyNumberOrNil(value any) any {
	number, ok := float64Value(value)
	if !ok || number == 0 {
		return nil
	}
	return value
}
