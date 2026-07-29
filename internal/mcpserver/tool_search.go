package mcpserver

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type SearchInput struct {
	Query string      `json:"query" jsonschema:"Name or ticker to search for. Three or more characters enable fuzzy matching." minLength:"1" doc:"Name or ticker to search for."`
	Type  *EntityType `json:"type,omitempty" enum:"character,corporation,alliance,ship,item,system,region,constellation,faction" jsonschema:"Restrict results to one entity type."`
	Limit int         `json:"limit,omitempty" minimum:"1" maximum:"25" default:"10" jsonschema:"Maximum number of matches to return."`
}

type SearchHit struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	Ticker        *string    `json:"ticker"`
	Type          EntityType `json:"type"`
	URL           string     `json:"url"`
	CorporationID *int64     `json:"corporation_id,omitempty"`
	AllianceID    *int64     `json:"alliance_id,omitempty"`
}

type SearchOutput struct {
	Query string      `json:"query"`
	Count int         `json:"count"`
	Hits  []SearchHit `json:"hits"`
}

func registerSearchTool(registry *Registry) error {
	return addTool(registry, ToolDefinition{
		Name:  "search",
		Title: "Search EVE entities",
		Description: "Search EVE Online entities by name or ticker across characters, " +
			"corporations, alliances, ships, items, systems, regions, constellations, " +
			"and factions. Use this first when an entity is only known by name.",
	}, func(ctx context.Context, input SearchInput) (SearchOutput, error) {
		return search(ctx, registry.deps, input)
	})
}

func search(
	ctx context.Context,
	deps Dependencies,
	input SearchInput,
) (SearchOutput, error) {
	query := strings.TrimSpace(input.Query)
	output := SearchOutput{Query: query, Hits: []SearchHit{}}
	if query == "" {
		return output, nil
	}
	limit := input.Limit
	if limit == 0 {
		limit = 10
	}
	limit = clamp(limit, 1, 25)
	typeHint := ""
	if input.Type != nil {
		typeHint = string(*input.Type)
	}
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT *
		FROM (
			SELECT character_id::bigint AS entity_id, name::text AS name,
			       NULL::text AS ticker, 'character'::text AS type,
			       corporation_id::bigint, alliance_id::bigint,
			       0::bigint AS weight, similarity(name, $1) AS score
			FROM characters
			WHERE deleted IS NOT TRUE AND name ILIKE $2
			  AND ($3 = '' OR $3 = 'character')
			UNION ALL
			SELECT corporation_id, name, ticker, 'corporation',
			       corporation_id, alliance_id, COALESCE(member_count, 0),
			       GREATEST(similarity(name, $1), similarity(ticker, $1))
			FROM corporations
			WHERE deleted IS NOT TRUE
			  AND (name ILIKE $2 OR ticker ILIKE $2)
			  AND ($3 = '' OR $3 = 'corporation')
			UNION ALL
			SELECT alliance_id, name, ticker, 'alliance',
			       NULL::bigint, alliance_id, COALESCE(member_count, 0),
			       GREATEST(similarity(name, $1), similarity(ticker, $1))
			FROM alliances
			WHERE deleted IS NOT TRUE
			  AND (name ILIKE $2 OR ticker ILIKE $2)
			  AND ($3 = '' OR $3 = 'alliance')
			UNION ALL
			SELECT t.type_id, t.name, NULL::text,
			       CASE WHEN g.category_id = 6 THEN 'ship' ELSE 'item' END,
			       NULL::bigint, NULL::bigint,
			       CASE WHEN g.category_id = 6 THEN 10 ELSE 0 END,
			       similarity(t.name, $1)
			FROM inv_types t
			JOIN inv_groups g ON g.group_id = t.group_id
			WHERE t.published IS TRUE AND t.name ILIKE $2
			  AND ($3 = '' OR ($3 = 'ship' AND g.category_id = 6)
			       OR ($3 = 'item' AND g.category_id != 6))
			UNION ALL
			SELECT solar_system_id, system_name, NULL::text, 'system',
			       NULL::bigint, NULL::bigint, 0, similarity(system_name, $1)
			FROM solar_systems
			WHERE system_name ILIKE $2 AND ($3 = '' OR $3 = 'system')
			UNION ALL
			SELECT region_id, name, NULL::text, 'region',
			       NULL::bigint, NULL::bigint, 0, similarity(name, $1)
			FROM regions
			WHERE name ILIKE $2 AND ($3 = '' OR $3 = 'region')
			UNION ALL
			SELECT constellation_id, constellation_name, NULL::text,
			       'constellation', NULL::bigint, NULL::bigint, 0,
			       similarity(constellation_name, $1)
			FROM constellations
			WHERE constellation_name ILIKE $2
			  AND ($3 = '' OR $3 = 'constellation')
			UNION ALL
			SELECT faction_id, name, NULL::text, 'faction',
			       NULL::bigint, NULL::bigint, 0, similarity(name, $1)
			FROM factions
			WHERE name ILIKE $2 AND ($3 = '' OR $3 = 'faction')
		) matches
		ORDER BY score DESC
		LIMIT $4`, query, query+"%", typeHint, limit)
	if err != nil {
		return SearchOutput{}, fmt.Errorf("search entities: %w", err)
	}

	type rankedHit struct {
		hit    SearchHit
		score  float64
		weight int64
		rank   int
	}
	ranks := map[EntityType]int{
		EntityShip: 1, EntityItem: 1, EntityRegion: 2,
		EntityConstellation: 2, EntitySystem: 3, EntityFaction: 4,
		EntityAlliance: 5, EntityCorporation: 6, EntityCharacter: 7,
	}
	ranked := make([]rankedHit, 0, len(rows))
	for _, row := range rows {
		entityType := EntityType(valueString(row["type"]))
		hit := SearchHit{
			ID:     valueInt64(row["entity_id"]),
			Name:   valueString(row["name"]),
			Ticker: nullableString(row["ticker"]),
			Type:   entityType,
		}
		hit.URL = entityURL(deps.BaseURL, entityType, hit.ID)
		if entityType == EntityCharacter || entityType == EntityCorporation {
			hit.CorporationID = nullableInt64(row["corporation_id"])
			hit.AllianceID = nullableInt64(row["alliance_id"])
		}
		ranked = append(ranked, rankedHit{
			hit: hit, score: valueFloat64(row["score"]),
			weight: valueInt64(row["weight"]), rank: ranks[entityType],
		})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score-ranked[j].score < 0.1 &&
			ranked[j].score-ranked[i].score < 0.1 {
			if ranked[i].rank != ranked[j].rank {
				return ranked[i].rank < ranked[j].rank
			}
			return ranked[i].weight > ranked[j].weight
		}
		return ranked[i].score > ranked[j].score
	})
	for _, row := range ranked {
		output.Hits = append(output.Hits, row.hit)
	}
	output.Count = len(output.Hits)
	return output, nil
}
